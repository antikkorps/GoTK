package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/mcp"
	"github.com/antikkorps/GoTK/internal/proxy"
)

var (
	showStats  bool
	shellMode  bool
	shellCmd   string // -c "command"
	streamMode bool
	cfg        *config.Config
)

func main() {
	// Load config (file-based defaults)
	cfg = config.Load()
	showStats = cfg.General.Stats
	filter.MaxLines = cfg.General.MaxLines
	shellMode = cfg.General.ShellMode

	args := os.Args[1:]

	// Extract gotk flags before command
	args = parseFlags(args)

	// Shell mode: -c "command" (sh-compatible single command execution)
	if shellCmd != "" {
		code := proxy.RunCommand(cfg, shellCmd)
		os.Exit(code)
	}

	// Shell mode: --shell (proxy shell loop)
	if shellMode {
		code := proxy.RunShell(cfg)
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
		input, _ := io.ReadAll(os.Stdin)
		raw := string(input)

		// Auto-detect command type from output format
		cmdType := detect.AutoDetect(raw)
		chain := proxy.BuildChain(cfg, cmdType)
		cleaned := chain.Apply(raw)
		printWithStats(raw, cleaned)
		return
	}

	// Handle subcommands
	var cmdArgs []string
	switch args[0] {
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
		mcp.Serve(cfg)
		os.Exit(0)
	case "--help", "-h":
		printUsage()
		os.Exit(0)
	case "--version", "-v":
		fmt.Println("gotk v0.1.0")
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
	chain := proxy.BuildChain(cfg, cmdType)

	// Apply filters
	cleaned := chain.Apply(result.Stdout)

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
					filter.MaxLines = n
				}
				i++
			}
		case "--no-truncate":
			filter.MaxLines = 0
		case "--shell":
			shellMode = true
		case "--stream":
			streamMode = true
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

// runStreaming executes a command in streaming mode, applying stream-compatible
// filters line-by-line as output arrives.
func runStreaming(cmdArgs []string) int {
	sf := filter.NewStreamFilter(filter.StreamConfig{
		StripANSI:           cfg.Filters.StripANSI,
		CompressPaths:       cfg.Filters.CompressPaths,
		Dedup:               cfg.Filters.Dedup,
		TrimDecorative:      cfg.Filters.TrimDecorative,
		NormalizeWhitespace: cfg.Filters.NormalizeWhitespace,
	})

	ch, wait := exec.RunStream(cmdArgs[0], cmdArgs[1:]...)

	rawBytes := 0
	cleanBytes := 0

	for r := range ch {
		rawBytes += len(r.Line) + 1 // +1 for newline

		if r.IsStderr {
			fmt.Fprintln(os.Stderr, r.Line)
			continue
		}

		out, emit := sf.ProcessLine(r.Line)
		if emit {
			fmt.Fprintln(os.Stdout, out)
			cleanBytes += len(out) + 1
		}
	}

	// Flush any pending buffered output (e.g., trailing dedup marker).
	if flushed := sf.Flush(); flushed != "" {
		fmt.Fprintln(os.Stdout, flushed)
		cleanBytes += len(flushed) + 1
	}

	code := wait()

	if showStats {
		saved := rawBytes - cleanBytes
		pct := 0
		if rawBytes > 0 {
			pct = saved * 100 / rawBytes
		}
		fmt.Fprintf(os.Stderr, "\n[gotk] stream: %d → %d bytes (-%d%%, saved %d bytes)\n",
			rawBytes, cleanBytes, pct, saved)
	}

	return code
}

func printUsage() {
	usage := `GoTK - LLM Output Proxy

Clean up command output before sending to LLMs.
Reduces token usage by removing noise from command output.

Usage:
  gotk [flags] <command> [args...]          Direct mode
  gotk [flags] exec -- <command> [args...]  Explicit mode
  <command> | gotk [flags]                  Pipe mode
  gotk --shell                              Proxy shell mode
  gotk -c "command"                         Shell-compatible execution
  gotk --mcp                                MCP server (JSON-RPC over stdio)

Examples:
  gotk grep -rn "func main" .
  gotk git log --oneline -20
  gotk find . -name "*.go"
  gotk go test ./...
  grep -rn "TODO" . | gotk --stats
  gotk --max-lines 100 find / -name "*.log"
  SHELL=/path/to/gotk gotk --shell          LLM integration
  gotk -c "grep -rn foo ."                  Single command
  gotk --stream make build                   Stream filtered output in real-time

Flags:
  -s, --stats        Show reduction statistics on stderr
  -m, --max-lines N  Max output lines (default: 50, keeps head+tail)
  --no-truncate      Disable line truncation
  --stream           Stream output line-by-line with real-time filtering
  --shell            Start proxy shell mode
  -c "command"       Execute single command through filter pipeline
  --mcp              Start MCP server (Model Context Protocol)
  -h, --help         Show this help
  -v, --version      Show version

Config:
  ~/.config/gotk/config.toml    Global config
  ./gotk.toml                   Local config (takes precedence)

Environment:
  GOTK_PASSTHROUGH=1    Disable filtering (escape hatch)
  GOTK_SHELL=/bin/bash  Shell used for -c and --shell execution`

	fmt.Println(strings.TrimSpace(usage))
}
