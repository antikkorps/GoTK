package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/antikkorps/GoTK/internal/bench"
	gotkctx "github.com/antikkorps/GoTK/internal/ctx"
	"github.com/antikkorps/GoTK/internal/daemon"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/hook"
	"github.com/antikkorps/GoTK/internal/install"
	"github.com/antikkorps/GoTK/internal/learn"
	"github.com/antikkorps/GoTK/internal/measure"
	"github.com/antikkorps/GoTK/internal/proxy"
	"github.com/antikkorps/GoTK/internal/update"
	"github.com/antikkorps/GoTK/internal/watch"
)

// runUpdate handles the "gotk update" subcommand.
func runUpdate(args []string) {
	opts := update.Options{Current: Version}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--check":
			opts.CheckOnly = true
		case "--from-source":
			opts.FromSource = true
		case "--force":
			opts.Force = true
		case "-h", "--help":
			printSubcommandHelp("update")
			return
		default:
			fmt.Fprintf(os.Stderr, "gotk update: unknown flag %q\n", args[i])
			os.Exit(2)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := update.Run(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "gotk update: %v\n", err)
		os.Exit(1)
	}
}

// runConfig handles the "gotk config" subcommand.
func runConfig(args []string) {
	sub := "show"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "show":
		fmt.Print(cfg.Show())
	default:
		fmt.Fprintf(os.Stderr, "gotk config: unknown subcommand %q (use: show)\n", sub)
		os.Exit(1)
	}
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
		logInfo("\n[gotk] stream: %d → %d bytes (-%d%%, saved %d bytes)\n",
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

	if err := mlog.Log(measure.Entry{
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
	}); err != nil {
		logInfo("[gotk] measure: failed to log entry: %v\n", err)
	}
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
			fmt.Fprintln(os.Stderr, "disabled")
			fmt.Fprintln(os.Stderr, "  hint: enable with --measure flag or add [measure] enabled=true to config")
			fmt.Fprintln(os.Stderr, "  hint: run 'gotk config show' to see config file locations")
		}
		fmt.Fprintf(os.Stderr, "Log path: %s\n", logPath)

		info, err := os.Stat(logPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Log file: not found")
			return
		}
		entries, err := measure.ReadEntries(logPath)
		fmt.Fprintf(os.Stderr, "Log size: %d bytes\n", info.Size())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Entries:  error reading log: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Entries:  %d\n", len(entries))
		}

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
			fmt.Fprintf(os.Stderr, "gotk measure: cannot read log: %v\n  hint: run 'gotk measure status' to check log path and permissions\n", err)
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
			fmt.Fprintf(os.Stderr, "gotk measure: cannot read log: %v\n  hint: run 'gotk measure status' to check log path and permissions\n", err)
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
			fmt.Fprintf(os.Stderr, "gotk learn: failed to write observations: %v\n  hint: check write permissions for ~/.local/share/gotk/\n", err)
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

		logInfo("\n[gotk learn] observed %d lines from %q\n", collector.Count(), strings.Join(cmdArgs, " "))

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
		logInfo("[gotk learn] warning: %v\n", err)
		return
	}
	logInfo("[gotk learn] observed %d lines\n", collector.Count())
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
			fmt.Fprintln(os.Stderr, "  hint: ensure your shell is bash or zsh, set GOTK_SHELL if needed")
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
		fmt.Fprintln(os.Stderr, "  gotk install claude [--local | --project | --global] [--uninstall] [--status]")
		os.Exit(1)
	}

	switch args[0] {
	case "claude":
		scope := install.ScopeLocal
		uninstallFlag := false
		statusFlag := false

		for _, a := range args[1:] {
			switch a {
			case "--global":
				scope = install.ScopeGlobal
			case "--project":
				scope = install.ScopeProject
			case "--local":
				scope = install.ScopeLocal
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
			fmt.Fprintln(os.Stderr, "  hint: use --status to check current state, --global for ~/.claude/settings.json")
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "gotk install: unknown target %q (supported: claude)\n", args[0])
		os.Exit(1)
	}
}
