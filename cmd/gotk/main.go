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
	gotkctx "github.com/antikkorps/GoTK/internal/ctx"
	"github.com/antikkorps/GoTK/internal/daemon"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/hook"
	"github.com/antikkorps/GoTK/internal/install"
	"github.com/antikkorps/GoTK/internal/learn"
	"github.com/antikkorps/GoTK/internal/mcp"
	"github.com/antikkorps/GoTK/internal/measure"
	"github.com/antikkorps/GoTK/internal/proxy"
	"github.com/antikkorps/GoTK/internal/watch"
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
			fmt.Fprintln(os.Stdout, out) //nolint:errcheck
			cleanBytes += len(out) + 1
		}
	}

	// Flush any pending buffered output (e.g., trailing dedup marker).
	if flushed := sf.Flush(); flushed != "" {
		fmt.Fprintln(os.Stdout, flushed) //nolint:errcheck
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
	abtest := false

	for _, a := range args {
		switch a {
		case "--json":
			jsonOutput = true
		case "--per-filter":
			perFilter = true
		case "--quality":
			quality = true
		case "--abtest":
			abtest = true
		}
	}

	if abtest {
		report := bench.RunABTest(cfg)
		if jsonOutput {
			fmt.Print(bench.FormatABTestJSON(report))
		} else {
			fmt.Print(bench.FormatABTest(report))
		}
		return
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

// runCtx handles the "gotk ctx" subcommand for context search.
func runCtx(args []string) {
	output, err := gotkctx.Run(cfg, args, maxLines, showStats)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(output)
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

// runLearn handles the "gotk learn" subcommand.
func runLearn(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "gotk learn: missing subcommand (run, suggest, status, clear)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  gotk learn run <command> [args...]   Observe a command's output")
		fmt.Fprintln(os.Stderr, "  gotk learn suggest                   Analyze and propose patterns")
		fmt.Fprintln(os.Stderr, "  gotk learn status                    Show observation statistics")
		fmt.Fprintln(os.Stderr, "  gotk learn clear                     Delete all observations")
		os.Exit(1)
	}

	storePath := learn.DefaultStorePath()

	switch args[0] {
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "gotk learn run: missing command")
			os.Exit(1)
		}
		cmdArgs := args[1:]

		// Execute the command
		result, err := exec.Run(cmdArgs[0], cmdArgs[1:]...)
		if err != nil && result == nil {
			fmt.Fprintf(os.Stderr, "gotk learn run: %v\n", err)
			os.Exit(1)
		}

		// Observe the raw output
		sessionID := learn.NewSessionID()
		collector := learn.NewCollector(strings.Join(cmdArgs, " "), sessionID)
		collector.Observe(result.Stdout)
		if result.Stderr != "" {
			collector.Observe(result.Stderr)
		}

		if err := learn.StoreWrite(storePath, collector.Observations()); err != nil {
			fmt.Fprintf(os.Stderr, "gotk learn: failed to write observations: %v\n", err)
			os.Exit(1)
		}

		// Also output the filtered result normally
		cmdType := detect.Identify(cmdArgs[0])
		chain := proxy.BuildChain(cfg, cmdType, maxLines)
		cleaned := chain.Apply(result.Stdout)
		fmt.Print(cleaned)
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}

		fmt.Fprintf(os.Stderr, "\n[gotk learn] observed %d lines from %q\n", collector.Count(), strings.Join(cmdArgs, " "))

	case "suggest":
		observations, err := learn.StoreRead(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk learn suggest: %v\n", err)
			os.Exit(1)
		}

		acfg := learn.AnalyzerConfig{
			MinSessions:   cfg.Learn.MinSessions,
			MinFrequency:  cfg.Learn.MinFrequency,
			MinNoiseScore: cfg.Learn.MinNoise,
			MaxBreadth:    0.30,
		}
		result := learn.Analyze(observations, acfg)
		fmt.Print(learn.FormatSuggestions(result))

	case "status":
		stats, err := learn.StoreStat(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk learn status: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(learn.FormatStatus(stats))

	case "clear":
		if err := learn.StoreClear(storePath); err != nil {
			fmt.Fprintf(os.Stderr, "gotk learn clear: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "gotk learn: observation store cleared.")

	default:
		fmt.Fprintf(os.Stderr, "gotk learn: unknown subcommand %q (use run, suggest, status, clear)\n", args[0])
		os.Exit(1)
	}
}

// observeForLearn performs passive observation when --learn flag is active.
func observeForLearn(command, rawOutput string) {
	storePath := learn.DefaultStorePath()
	sessionID := learn.NewSessionID()
	collector := learn.NewCollector(command, sessionID)
	collector.Observe(rawOutput)
	if err := learn.StoreWrite(storePath, collector.Observations()); err != nil {
		fmt.Fprintf(os.Stderr, "[gotk learn] warning: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "[gotk learn] observed %d lines\n", collector.Count())
}

// runDaemon handles the "gotk daemon" subcommand.
func runDaemon(args []string) {
	sub := "start"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "start":
		if err := daemon.Start(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "gotk daemon: %v\n", err)
			os.Exit(1)
		}

	case "status":
		daemon.Status()

	case "init":
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		for _, a := range args[1:] {
			switch a {
			case "--bash":
				shell = "bash"
			case "--zsh":
				shell = "zsh"
			}
		}
		if err := daemon.Init(shell); err != nil {
			fmt.Fprintf(os.Stderr, "gotk daemon init: %v\n", err)
			os.Exit(1)
		}

	case "summary":
		sessionID := ""
		for i, a := range args[1:] {
			if a == "--session" && i+2 < len(args) {
				sessionID = args[i+2]
			}
		}
		if sessionID == "" {
			sessionID = os.Getenv("GOTK_SESSION_ID")
		}
		if sessionID != "" {
			daemon.PrintSummary(os.Stderr, cfg.Measure.LogPath, sessionID)
		}

	case "stop":
		if os.Getenv("GOTK_DAEMON") == "1" {
			fmt.Fprintln(os.Stderr, "gotk daemon: type 'gotk exit' to leave the daemon session")
		} else {
			fmt.Fprintln(os.Stderr, "gotk daemon: no active session")
		}

	default:
		fmt.Fprintf(os.Stderr, "gotk daemon: unknown subcommand %q (use start, status, init, summary, stop)\n", sub)
		os.Exit(1)
	}
}

// runHook handles the "gotk hook" subcommand, invoked by Claude Code hooks.
func runHook() {
	if err := hook.Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "gotk hook: %v\n", err)
		os.Exit(1)
	}
}

// runInstall handles the "gotk install" subcommand.
func runInstall(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "gotk install: missing target (claude)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  gotk install claude [--global] [--project] [--uninstall] [--status]")
		os.Exit(1)
	}

	switch args[0] {
	case "claude":
		scope := install.ScopeProject
		uninstallFlag := false
		statusFlag := false

		for _, a := range args[1:] {
			switch a {
			case "--global":
				scope = install.ScopeGlobal
			case "--project":
				scope = install.ScopeProject
			case "--uninstall":
				uninstallFlag = true
			case "--status":
				statusFlag = true
			default:
				fmt.Fprintf(os.Stderr, "gotk install claude: unknown flag %q\n", a)
				os.Exit(1)
			}
		}

		var err error
		switch {
		case statusFlag:
			err = install.ClaudeStatus(scope)
		case uninstallFlag:
			err = install.ClaudeUninstall(scope)
		default:
			err = install.ClaudeInstall(scope)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "gotk install claude: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "gotk install: unknown target %q (supported: claude)\n", args[0])
		os.Exit(1)
	}
}

func printUsage() {
	usage := `GoTK - LLM Output Proxy

Clean command output before sending to LLMs. Reduces token usage by ~80%
while preserving errors, warnings, and semantically important content.

Best used as a CLI tool integrated with your AI coding assistant.
See "gotk help mcp" for MCP server mode (higher token overhead).

Usage:
  gotk [flags] <command> [args...]    Run a command with filtered output
  <command> | gotk [flags]            Filter piped input
  gotk -c "command"                   Shell-compatible execution
  gotk --shell                        Proxy shell mode (for SHELL= integration)

Subcommands:
  ctx       Context search with 5 output modes (gotk ctx pattern)
  learn     Project-specific pattern learning (run, suggest, status, clear)
  daemon    Start a filtered shell session (gotk daemon)
  install   Configure GoTK integration (gotk install claude)
  exec      Execute a command explicitly (gotk exec -- cmd args...)
  watch     Re-run command on file changes (gotk watch -- make test)
  bench     Run benchmark suite
  measure   Token consumption metrics (report, last, status, clear)
  help      Show help for a subcommand (gotk help watch)

Flags:
  -s, --stats          Show reduction statistics on stderr
  -m, --max-lines N    Max output lines (default: 50, keeps head+tail)
  --no-truncate        Disable line truncation
  --conservative       Minimal reduction, zero info loss
  --balanced           Default mode, good reduction
  --aggressive         Maximum reduction
  --mode MODE          Set filter mode explicitly
  --stream             Stream output line-by-line (real-time)
  --measure            Enable token consumption logging
  --learn              Passively observe output for pattern learning
  --profile PROFILE    LLM profile: claude, gpt, gemini
  --shell              Start proxy shell mode
  -c "command"         Execute single command
  --mcp                Start MCP server (JSON-RPC over stdio)
  -h, --help           Show this help
  -v, --version        Show version

Examples:
  gotk ctx BuildChain -t go                 Context search
  gotk grep -rn "func main" .              Direct mode
  gotk --stats go test ./...               With stats
  find . -name "*.go" | gotk               Pipe mode
  gotk --aggressive find / -name "*.log"   Maximum reduction
  gotk --stream make build                 Real-time filtering

AI tool integration (CLI mode, recommended for token efficiency):
  Claude Code  gotk install claude                          (auto hook)
  Aider        SHELL=gotk GOTK_SHELL=/bin/bash aider       (100% auto)
  Cursor       SHELL=gotk GOTK_SHELL=/bin/bash cursor .    (100% auto)

Config: ~/.config/gotk/config.toml | .gotk.toml | ./gotk.toml
Environment: GOTK_PASSTHROUGH=1 (bypass) | GOTK_SHELL (real shell)

Run "gotk help <subcommand>" for details. See also: man gotk`

	fmt.Println(strings.TrimSpace(usage))
}

func printSubcommandHelp(sub string) {
	helps := map[string]string{
		"ctx": `gotk ctx — Context Search

Usage:
  gotk ctx [flags] <pattern> [directory]

Search codebase and format results for LLM consumption. Five output modes
with built-in exclusions (node_modules, .git, vendor, etc.) and binary
file detection. Output is filtered through the GoTK pipeline.

Modes:
  (default)     Scan: file paths with match counts, indented matches
  -d [N]        Detail: N-line context windows with overlap merging (default: 3)
  --def         Def: language-aware declarations (func, class, struct, etc.)
  --tree        Tree: structural skeleton (imports, types, functions)
  --summary     Summary: directory breakdown table

Flags:
  -t, --type EXT     File extension filter (e.g., -t go, -t py)
  -g, --glob GLOB    Glob filter on filename (e.g., -g "*.test.js")
  -m, --max N        Max file results (0 = unlimited)
  -p, --path DIR     Root directory (alternative to positional arg)

Examples:
  gotk ctx BuildChain                       Scan mode (default)
  gotk ctx BuildChain -d 5                  Detail mode with 5 lines context
  gotk ctx BuildChain --def                 Definition mode
  gotk ctx BuildChain --tree                Tree/skeleton mode
  gotk ctx BuildChain --summary             Summary mode
  gotk --stats ctx BuildChain -t go -m 5     Filtered with stats
  gotk ctx "func.*Error" -t go              Regex search in Go files`,

		"exec": `gotk exec — Explicit command execution

Usage:
  gotk [flags] exec -- <command> [args...]

Execute a command with GoTK flags clearly separated from command arguments
using the -- delimiter.

Examples:
  gotk exec -- grep -rn "pattern" .
  gotk --aggressive exec -- find / -name "*.log"
  gotk --stats exec -- go test -v ./...`,

		"watch": `gotk watch — Watch files and re-run command on changes

Usage:
  gotk watch [watch-flags] -- <command> [args...]

Watch for file changes and automatically re-run the command with filtered
output. Useful for test-driven development with AI assistants.

Watch flags (before --):
  -i, --interval D    Polling interval (default: 2s, e.g., 5s, 500ms, 1m)
  -e, --ext EXT       File extension to watch (repeatable)
  -p, --path PATH     Path to watch (repeatable, default: ".")

Examples:
  gotk watch -- go test ./...
  gotk watch --ext .go -- go test ./...
  gotk watch --interval 5s -- make build
  gotk watch -e .py -p src -- python -m pytest
  gotk watch -e .go -e .mod -p ./internal -- go test ./internal/...`,

		"bench": `gotk bench — Run benchmark suite

Usage:
  gotk bench [flags]

Run benchmarks on realistic command output fixtures to measure reduction
rates, quality scores, and latency.

Flags:
  --json          Output results as JSON
  --per-filter    Show per-filter contribution breakdown
  --quality       Measure quality score (% of important lines preserved)
  --abtest        Compare conservative/balanced/aggressive modes side by side

Examples:
  gotk bench
  gotk bench --per-filter
  gotk bench --quality
  gotk bench --abtest
  gotk bench --json`,

		"measure": `gotk measure — Token consumption metrics

Usage:
  gotk measure <subcommand>

Track and analyze token savings from GoTK filtering.

Subcommands:
  report [--json] [--period D]    Show savings report (e.g., --period 7d)
  last [N]                        Show last N invocations (default: 10)
  status                          Show measurement status and log info
  clear                           Clear the measurement log

To enable measurement, use --measure flag or set measure.enabled = true
in your config file. Measurement is auto-enabled in MCP mode.

Examples:
  gotk --measure grep -rn "func" .
  gotk measure last
  gotk measure last 20
  gotk measure report
  gotk measure report --period 7d --json
  gotk measure status`,

		"learn": `gotk learn — Project-Specific Pattern Learning

Usage:
  gotk learn run <command> [args...]   Observe a command's output
  gotk learn suggest                   Analyze and propose always_remove patterns
  gotk learn status                    Show observation statistics
  gotk learn clear                     Delete all observations

Passive mode (observe while working normally):
  gotk --learn <command> [args...]     Same as normal execution + silent observation

How it works:
  1. Run your usual commands with "gotk learn run" a few times
  2. GoTK classifies each line (noise/debug/info/warning/error)
  3. Normalizes variable parts (hashes, timestamps, numbers, versions)
  4. Groups by normalized form and tracks frequency across sessions
  5. "gotk learn suggest" proposes regex patterns for .gotk.toml [rules]

Safety:
  - Patterns that match any warning/error line are never suggested
  - Minimum 80% noise confidence required (configurable)
  - Human review required: suggestions are not auto-applied

Config (.gotk.toml):
  [learn]
  min_sessions = 3       # Sessions needed before suggesting
  min_frequency = 0.05   # Minimum line frequency (5%)
  min_noise = 0.80       # Minimum noise confidence (80%)

Examples:
  gotk learn run go test ./...         Observe test output
  gotk learn run make build            Observe build output
  gotk learn suggest                   See pattern suggestions
  gotk --learn go test ./...           Passive observation`,

		"daemon": `gotk daemon — Filtered shell session

Usage:
  gotk daemon [start]     Start a daemon session (spawns filtered shell)
  gotk daemon status      Check if inside a daemon session
  gotk daemon init        Print shell init code (for eval "$(gotk daemon init)")
  gotk daemon stop        Instructions to exit the session

How it works:
  GoTK spawns your shell (bash or zsh) with hooks that intercept every
  command. Non-trivial command output is automatically piped through gotk,
  reducing noise while preserving errors, warnings, and important content.

  Interactive programs (vim, less, top, ssh, etc.) and trivial commands
  (cd, pwd, echo, etc.) are not intercepted.

  A [gotk] prefix is added to your prompt so you know filtering is active.
  Type "gotk exit" to leave and see a session summary with tokens saved.

  Measurement is automatically enabled during daemon sessions.

Manual init (advanced):
  eval "$(gotk daemon init)"          # Inject into current shell
  eval "$(gotk daemon init --zsh)"    # Force zsh init
  eval "$(gotk daemon init --bash)"   # Force bash init

Examples:
  gotk daemon                         # Start filtered shell
  gotk daemon status                  # Check if active
  # Inside session: all commands auto-filtered
  grep -rn "pattern" .                # Output is filtered
  vim file.go                         # Opens normally (interactive)
  gotk exit                           # Leave session, see summary`,

		"install": `gotk install — Configure GoTK integrations

Usage:
  gotk install claude [flags]

Targets:
  claude    Install GoTK as a Claude Code PreToolUse hook

Flags:
  --global      Install in ~/.claude/settings.json (all projects)
  --project     Install in .claude/settings.json (default, current project)
  --uninstall   Remove GoTK hook configuration
  --status      Check if GoTK hook is installed

How it works:
  GoTK registers as a PreToolUse hook for the Bash tool. When Claude Code
  runs a shell command, the hook wraps it with "| gotk" so the output is
  filtered before Claude sees it. This reduces token usage by ~80%.

  Trivial commands (cd, pwd, echo, etc.) are not wrapped.
  Commands already piped through gotk are not double-wrapped.

Examples:
  gotk install claude                  Install for current project
  gotk install claude --global         Install for all projects
  gotk install claude --status         Check installation status
  gotk install claude --uninstall      Remove hook`,

		"hook": `gotk hook — Claude Code hook handler (internal)

Usage:
  gotk hook

This subcommand is called automatically by Claude Code when configured as
a PreToolUse hook. It reads a JSON payload from stdin, wraps Bash commands
with "| gotk" for output filtering, and writes the response to stdout.

You normally don't call this directly. Use "gotk install claude" to set up
the hook configuration automatically.

JSON input format (from Claude Code):
  {
    "hook_event_name": "PreToolUse",
    "tool_name": "Bash",
    "tool_input": {"command": "grep -rn pattern ."}
  }

JSON output format (to Claude Code):
  {
    "updatedInput": {"command": "set -o pipefail; (grep -rn pattern .) | /path/to/gotk"}
  }`,

		"mcp": `gotk --mcp — MCP Server Mode (Model Context Protocol)

Usage:
  gotk --mcp

Start a JSON-RPC server over stdio that exposes GoTK tools to MCP-compatible
AI coding assistants (Claude Code, etc.).

Exposed tools:
  gotk_exec     Execute any command and return cleaned output
  gotk_filter   Filter pre-existing text through the cleaning pipeline
  gotk_read     Read a file with smart truncation and noise removal
  gotk_grep     Search file contents with grouped, compressed results

Setup:
  claude mcp add --transport stdio gotk -- gotk --mcp

Note: CLI mode (PostToolUse hook) is more token-efficient than MCP mode.
MCP adds JSON-RPC overhead and tool schema tokens on every LLM turn.
See docs/cli-vs-mcp.md for a detailed comparison.

Use MCP mode when:
  - Your AI tool doesn't support hooks or SHELL replacement
  - You want the AI to explicitly choose when to filter
  - You need gotk_read or gotk_grep specialized tools`,
	}

	if h, ok := helps[sub]; ok {
		fmt.Println(strings.TrimSpace(h))
	} else {
		fmt.Fprintf(os.Stderr, "gotk help: unknown subcommand %q\n\n", sub)
		printUsage()
	}
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
