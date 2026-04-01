package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/mcp"
	"github.com/antikkorps/GoTK/internal/measure"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// Version is set at build time via -ldflags "-X main.Version=..."
var Version = "dev"

var (
	showStats        bool
	shellMode        bool
	shellCmd         string // -c "command"
	streamMode       bool
	measureFlag      bool // --measure flag
	learnFlag        bool // --learn flag for passive observation
	maxLines         int
	maxLinesExplicit bool // true if user set --max-lines or --no-truncate explicitly
	cfg              *config.Config
	mlog             *measure.Logger // nil if measurement disabled
)

func main() {
	// Set up graceful shutdown on SIGINT and SIGTERM
	setupSignalHandler()

	// Load config (file-based defaults)
	cfg = config.Load()
	showStats = cfg.General.Stats
	maxLines = cfg.General.MaxLines
	shellMode = cfg.General.ShellMode

	args := os.Args[1:]

	// Extract gotk flags before command
	args = parseFlags(args)

	// Apply profile then mode overrides.
	// Profile sets sensible defaults for the target LLM.
	// Mode overrides on top. Explicit --max-lines / --no-truncate wins over both.
	cfg.ApplyProfile()
	savedMaxLines := maxLines
	cfg.ApplyMode()
	if maxLinesExplicit {
		maxLines = savedMaxLines
		cfg.General.MaxLines = savedMaxLines
	} else {
		maxLines = cfg.General.MaxLines
	}

	// Initialize measurement logger if enabled
	if measureFlag {
		cfg.Measure.Enabled = true
	}
	if cfg.Measure.Enabled {
		l, err := measure.NewLogger(cfg.Measure.LogPath, measure.SessionID(), cfg.Measure.MaxLogSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gotk] warning: cannot init measure log: %v\n", err)
		} else {
			mlog = l
			defer mlog.Close() //nolint:errcheck
		}
	}

	// Shell mode: -c "command" (sh-compatible single command execution)
	if shellCmd != "" {
		code := proxy.RunCommand(cfg, shellCmd, maxLines)
		os.Exit(code)
	}

	// Shell mode: --shell (proxy shell loop)
	if shellMode {
		code := proxy.RunShell(cfg, maxLines)
		os.Exit(code)
	}

	// Check if stdin is a pipe
	isPipe := !isTerminal(os.Stdin)

	if len(args) == 0 && !isPipe {
		printUsage()
		os.Exit(0)
	}

	// Pipe mode: <command> | gotk
	if len(args) == 0 && isPipe {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		raw := string(input)

		// Auto-detect command type from output format
		cmdType := detect.AutoDetect(raw)
		start := time.Now()
		chain := proxy.BuildChain(cfg, cmdType, maxLines)
		cleaned := chain.Apply(raw)
		logMeasurement("pipe", cmdType.String(), raw, cleaned, time.Since(start), false)
		printWithStats(raw, cleaned)
		return
	}

	// Handle subcommands
	var cmdArgs []string
	switch args[0] {
	case "help":
		if len(args) > 1 {
			printSubcommandHelp(args[1])
		} else {
			printUsage()
		}
		os.Exit(0)
	case "measure":
		runMeasure(args[1:])
		os.Exit(0)
	case "bench":
		runBench(args[1:])
		os.Exit(0)
	case "ctx":
		runCtx(args[1:])
		os.Exit(0)
	case "learn":
		runLearn(args[1:])
		os.Exit(0)
	case "config":
		runConfig(args[1:])
		os.Exit(0)
	case "watch":
		runWatch(args[1:])
		os.Exit(0)
	case "hook":
		runHook()
		os.Exit(0)
	case "daemon":
		runDaemon(args[1:])
		os.Exit(0)
	case "install":
		runInstall(args[1:])
		os.Exit(0)
	case "exec":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "gotk exec: missing command")
			os.Exit(1)
		}
		if args[1] == "--" {
			cmdArgs = args[2:]
		} else {
			cmdArgs = args[1:]
		}
	case "--mcp":
		mcp.Version = Version
		mcp.Serve(cfg)
		os.Exit(0)
	case "--help", "-h":
		printUsage()
		os.Exit(0)
	case "--version", "-v":
		fmt.Printf("gotk %s\n", Version)
		os.Exit(0)
	default:
		cmdArgs = args
	}

	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "gotk: missing command")
		os.Exit(1)
	}

	// Stream mode: process output line-by-line in real-time.
	if streamMode {
		code := runStreaming(cmdArgs)
		os.Exit(code)
	}

	// Execute the command and capture output
	result, err := exec.Run(cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotk: %v\n", err)
		if result == nil {
			os.Exit(1)
		}
	}

	// Build filter chain based on detected command, using config
	cmdType := detect.Identify(cmdArgs[0])
	// Check custom command mappings
	if mapped, ok := cfg.Commands[cmdArgs[0]]; ok {
		cmdType = detect.Identify(mapped)
	}
	start := time.Now()
	chain := proxy.BuildChain(cfg, cmdType, maxLines)

	// Apply filters
	cleaned := chain.Apply(result.Stdout)
	logMeasurement(strings.Join(cmdArgs, " "), cmdType.String(), result.Stdout, cleaned, time.Since(start), false)

	// Passive learn: observe raw output if --learn flag is set
	if learnFlag && result.Stdout != "" {
		observeForLearn(strings.Join(cmdArgs, " "), result.Stdout)
	}

	// Output
	printWithStats(result.Stdout, cleaned)

	// Pass through stderr unmodified
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	os.Exit(result.ExitCode)
}

func parseFlags(args []string) []string {
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stats", "-s":
			showStats = true
		case "--max-lines", "-m":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					maxLines = n
					maxLinesExplicit = true
				}
				i++
			}
		case "--no-truncate":
			maxLines = 0
			maxLinesExplicit = true
		case "--shell":
			shellMode = true
		case "--stream":
			streamMode = true
		case "--measure":
			measureFlag = true
		case "--learn":
			learnFlag = true
		case "--aggressive":
			cfg.General.Mode = config.ModeAggressive
		case "--balanced":
			cfg.General.Mode = config.ModeBalanced
		case "--conservative":
			cfg.General.Mode = config.ModeConservative
		case "--mode":
			if i+1 < len(args) {
				cfg.General.Mode = config.ParseMode(args[i+1])
				i++
			}
		case "--profile":
			if i+1 < len(args) {
				cfg.Profile = config.ParseProfile(args[i+1])
				i++
			}
		case "-c":
			if i+1 < len(args) {
				shellCmd = args[i+1]
				i++
			}
		default:
			// Once we hit a non-flag, everything else is the command
			remaining = append(remaining, args[i:]...)
			return remaining
		}
	}
	return remaining
}

func printWithStats(raw, cleaned string) {
	fmt.Print(cleaned)

	if showStats {
		rawBytes := len(raw)
		cleanBytes := len(cleaned)
		saved := rawBytes - cleanBytes
		pct := 0
		if rawBytes > 0 {
			pct = saved * 100 / rawBytes
		}
		fmt.Fprintf(os.Stderr, "\n[gotk] %d → %d bytes (-%d%%, saved %d bytes)\n",
			rawBytes, cleanBytes, pct, saved)
	}
}

// isTerminal checks if a file is connected to a terminal.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// setupSignalHandler catches SIGINT and SIGTERM for graceful shutdown.
// It cleans up any running child processes and exits cleanly.
func setupSignalHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Fprintf(os.Stderr, "\n[gotk] received signal %s, shutting down...\n", sig)

		// Kill our entire process group so child processes are cleaned up.
		// Use negative PID to signal the process group.
		pgid, err := syscall.Getpgid(os.Getpid())
		if err == nil {
			// Send SIGTERM to the process group (excluding ourselves, since
			// we are already shutting down).
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
		}

		// Exit with 128 + signal number (Unix convention)
		switch sig {
		case syscall.SIGINT:
			os.Exit(130)
		case syscall.SIGTERM:
			os.Exit(143)
		default:
			os.Exit(1)
		}
	}()
}
