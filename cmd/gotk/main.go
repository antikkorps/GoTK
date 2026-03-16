package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/antikkorps/GoTK/internal/bench"
	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/mcp"
	"github.com/antikkorps/GoTK/internal/measure"
	"github.com/antikkorps/GoTK/internal/proxy"
	"github.com/antikkorps/GoTK/internal/watch"
)

var (
	showStats       bool
	shellMode       bool
	shellCmd        string // -c "command"
	streamMode      bool
	measureFlag     bool   // --measure flag
	maxLines        int
	maxLinesExplicit bool // true if user set --max-lines or --no-truncate explicitly
	cfg             *config.Config
	mlog            *measure.Logger // nil if measurement disabled
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
			defer mlog.Close()
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
	case "measure":
		runMeasure(args[1:])
		os.Exit(0)
	case "bench":
		runBench(args[1:])
		os.Exit(0)
	case "watch":
		runWatch(args[1:])
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
	start := time.Now()
	chain := proxy.BuildChain(cfg, cmdType, maxLines)

	// Apply filters
	cleaned := chain.Apply(result.Stdout)
	logMeasurement(strings.Join(cmdArgs, " "), cmdType.String(), result.Stdout, cleaned, time.Since(start), false)

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

	// Create a context with timeout for streaming
	timeout := time.Duration(cfg.Security.CommandTimeout) * time.Second
	if timeout <= 0 {
		timeout = exec.DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch, wait := exec.RunStream(ctx, cmdArgs[0], cmdArgs[1:]...)

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

// runWatch parses watch-specific flags and starts the watch loop.
// Arguments before "--" are watch flags; arguments after are the command.
func runWatch(args []string) {
	var (
		interval   = 2 * time.Second
		extensions []string
		paths      []string
	)

	// Split args at "--" separator.
	var watchFlags, cmdArgs []string
	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}
	if dashIdx >= 0 {
		watchFlags = args[:dashIdx]
		cmdArgs = args[dashIdx+1:]
	} else {
		// No "--" — treat all args as command.
		cmdArgs = args
	}

	// Parse watch-specific flags.
	for i := 0; i < len(watchFlags); i++ {
		switch watchFlags[i] {
		case "--interval", "-i":
			if i+1 < len(watchFlags) {
				if d, err := time.ParseDuration(watchFlags[i+1]); err == nil {
					interval = d
				}
				i++
			}
		case "--ext", "-e":
			if i+1 < len(watchFlags) {
				extensions = append(extensions, watchFlags[i+1])
				i++
			}
		case "--path", "-p":
			if i+1 < len(watchFlags) {
				paths = append(paths, watchFlags[i+1])
				i++
			}
		default:
			fmt.Fprintf(os.Stderr, "gotk watch: unknown flag %q\n", watchFlags[i])
			os.Exit(1)
		}
	}

	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "gotk watch: missing command (use -- to separate flags from command)")
		os.Exit(1)
	}

	// Set up context with signal handling for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		// First signal: cancel context (stops current run).
		cancel()
		<-sigCh
		// Second signal: hard exit.
		os.Exit(130)
	}()

	wcfg := watch.Config{
		Command:    cmdArgs,
		Interval:   interval,
		Debounce:   500 * time.Millisecond,
		Paths:      paths,
		Extensions: extensions,
		MaxLines:   maxLines,
		GoTKConfig: cfg,
	}

	if err := watch.Run(ctx, wcfg); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "gotk watch: %v\n", err)
		os.Exit(1)
	}
}

// runBench handles the "gotk bench" subcommand.
func runBench(args []string) {
	jsonOutput := false
	perFilter := false
	quality := false

	for _, a := range args {
		switch a {
		case "--json":
			jsonOutput = true
		case "--per-filter":
			perFilter = true
		case "--quality":
			quality = true
		}
	}

	if quality {
		report := bench.MeasureQuality(cfg)
		if jsonOutput {
			fmt.Print(bench.FormatQualityJSON(report))
		} else {
			fmt.Print(bench.FormatQuality(report))
		}
		return
	}

	if perFilter {
		// Show per-filter contribution for every built-in fixture.
		report := bench.RunBenchmarks(cfg)
		fixtures := bench.AllFixtureInputs()
		for i, f := range fixtures {
			contributions := bench.MeasureFilters(cfg, f.Input, f.CmdType)
			if jsonOutput {
				fmt.Print(bench.FormatPerFilterJSON(report.Results[i].Name, contributions))
			} else {
				fmt.Print(bench.FormatPerFilter(report.Results[i].Name, contributions))
				fmt.Println()
			}
		}
		return
	}

	report := bench.RunBenchmarks(cfg)
	if jsonOutput {
		fmt.Print(bench.FormatReportJSON(report))
	} else {
		fmt.Print(bench.FormatReport(report))
	}
}

// logMeasurement logs a measurement entry if the logger is active.
func logMeasurement(command, cmdType, raw, cleaned string, dur time.Duration, cached bool) {
	if mlog == nil {
		return
	}
	rawTokens := measure.EstimateTokens(raw)
	cleanTokens := measure.EstimateTokens(cleaned)
	saved := rawTokens - cleanTokens
	var pct float64
	if rawTokens > 0 {
		pct = float64(saved) / float64(rawTokens) * 100
	}
	quality, important := measure.ComputeQualityScore(raw, cleaned)

	_ = mlog.Log(measure.Entry{
		Command:        command,
		CommandType:    cmdType,
		RawBytes:       len(raw),
		CleanBytes:     len(cleaned),
		RawTokens:      rawTokens,
		CleanTokens:    cleanTokens,
		TokensSaved:    saved,
		ReductionPct:   pct,
		LinesRaw:       measure.CountLines(raw),
		LinesClean:     measure.CountLines(cleaned),
		ImportantLines: important,
		QualityScore:   quality,
		Mode:           string(cfg.General.Mode),
		Source:         "cli",
		Cached:         cached,
		DurationUs:     dur.Microseconds(),
	})
}

// runMeasure handles the "gotk measure" subcommand.
func runMeasure(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "gotk measure: missing subcommand (report, last, status, clear)")
		os.Exit(1)
	}

	logPath := cfg.Measure.LogPath

	switch args[0] {
	case "status":
		fmt.Fprintf(os.Stderr, "Measurement: ")
		if cfg.Measure.Enabled {
			fmt.Fprintln(os.Stderr, "enabled")
		} else {
			fmt.Fprintln(os.Stderr, "disabled (use --measure flag or [measure] enabled=true in config)")
		}
		fmt.Fprintf(os.Stderr, "Log path: %s\n", logPath)

		info, err := os.Stat(logPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Log file: not found")
			return
		}
		entries, _ := measure.ReadEntries(logPath)
		fmt.Fprintf(os.Stderr, "Log size: %d bytes\n", info.Size())
		fmt.Fprintf(os.Stderr, "Entries:  %d\n", len(entries))

	case "report":
		jsonOutput := false
		period := "all"
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--json":
				jsonOutput = true
			case "--period":
				if i+1 < len(args) {
					period = args[i+1]
					i++
				}
			default:
				// Allow period as positional: gotk measure report 7d
				period = args[i]
			}
		}

		entries, err := measure.ReadEntries(logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk measure: cannot read log: %v\n", err)
			os.Exit(1)
		}

		entries = measure.FilterEntriesByPeriod(entries, period)
		report := measure.GenerateReport(entries, period)

		if jsonOutput {
			fmt.Print(measure.FormatReportJSON(report))
		} else {
			fmt.Print(measure.FormatReport(report))
		}

	case "last":
		n := 10
		if len(args) > 1 {
			if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil || n <= 0 {
				fmt.Fprintln(os.Stderr, "gotk measure last: invalid count, using default 10")
				n = 10
			}
		}

		entries, err := measure.ReadEntries(logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk measure: cannot read log: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(measure.FormatLast(entries, n))

	case "clear":
		if err := os.Truncate(logPath, 0); err != nil {
			fmt.Fprintf(os.Stderr, "gotk measure clear: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Measurement log cleared.")

	default:
		fmt.Fprintf(os.Stderr, "gotk measure: unknown subcommand %q (use report, status, last, clear)\n", args[0])
		os.Exit(1)
	}
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
  gotk watch [flags] -- <command> [args...] Watch mode (re-run on file changes)
  gotk bench [flags]                       Run benchmarks
  gotk measure report [--json] [--period]  Show token savings report
  gotk measure last [N]                    Show last N invocations (default: 10)
  gotk measure status                      Show measurement status
  gotk measure clear                       Clear measurement log

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
  gotk watch --ext .go -- go test ./...     Watch .go files, re-run tests
  gotk watch --interval 5s -- make build    Poll every 5s
  gotk watch -e .py -p src -- python -m pytest
  gotk bench                               Run all benchmarks
  gotk bench --per-filter                  Show per-filter breakdown
  gotk bench --quality                     Measure quality score (important lines preserved)
  gotk bench --json                        Output as JSON
  gotk --measure grep -rn "func" .        Run with measurement logging
  gotk measure last                       Show last 10 invocations
  gotk measure last 20                    Show last 20 invocations
  gotk measure report --period 7d         Show last 7 days report
  gotk measure report --json              JSON report output

Flags:
  -s, --stats        Show reduction statistics on stderr
  -m, --max-lines N  Max output lines (default: 50, keeps head+tail)
  --no-truncate      Disable line truncation
  --conservative     Minimal reduction, zero info loss (max-lines: 200, no truncation)
  --balanced         Default mode — good reduction, preserves important lines
  --aggressive       Maximum reduction, acceptable info loss (max-lines: 30)
  --mode MODE        Set filter mode (conservative, balanced, aggressive)
  --stream           Stream output line-by-line with real-time filtering
  --measure          Enable token consumption measurement logging
  --profile PROFILE  Set LLM profile: claude, gpt, gemini (auto-detected in MCP)
  --shell            Start proxy shell mode
  -c "command"       Execute single command through filter pipeline
  --mcp              Start MCP server (Model Context Protocol)
  -h, --help         Show this help
  -v, --version      Show version

Watch flags (before --):
  -i, --interval D   Polling interval (default: 2s, e.g., 5s, 500ms)
  -e, --ext EXT      File extension to watch (repeatable, e.g., -e .go -e .mod)
  -p, --path PATH    Path to watch (repeatable, default: ".")

Config (in order of precedence):
  ~/.config/gotk/config.toml    Global config
  .gotk.toml                    Project config (found by walking up to repo root)
  ./gotk.toml                   Local config (highest precedence)

Environment:
  GOTK_PASSTHROUGH=1    Disable filtering (escape hatch)
  GOTK_SHELL=/bin/bash  Shell used for -c and --shell execution`

	fmt.Println(strings.TrimSpace(usage))
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
