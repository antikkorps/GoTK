package bench

import (
	"fmt"
	"strings"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// Result holds the benchmark results for a single fixture.
type Result struct {
	Name       string
	RawBytes   int
	CleanBytes int
	Reduction  float64 // percentage
	Duration   time.Duration
	LinesRaw   int
	LinesClean int
}

// Report holds the aggregated benchmark report.
type Report struct {
	Results       []Result
	TotalRaw      int
	TotalClean    int
	AvgReduction  float64
	TotalDuration time.Duration
}

// fixture describes a single benchmark test case.
type fixture struct {
	name    string
	gen     func() string
	cmdType detect.CmdType
}

// allFixtures returns the built-in benchmark fixtures.
func allFixtures() []fixture {
	return []fixture{
		{"grep large", generateGrepFixture, detect.CmdGrep},
		{"git log verbose", generateGitLogFixture, detect.CmdGit},
		{"git diff large", generateGitDiffFixture, detect.CmdGit},
		{"go test mixed", generateGoTestFixture, detect.CmdGoTool},
		{"find deep tree", generateFindFixture, detect.CmdFind},
		{"docker build", generateDockerBuildFixture, detect.CmdDocker},
		{"npm install", generateNpmInstallFixture, detect.CmdNpm},
		{"cargo build", generateCargoBuildFixture, detect.CmdCargo},
		{"make build", generateMakeBuildFixture, detect.CmdMake},
		{"stack traces", generateStackTraceFixture, detect.CmdGeneric},
		{"noisy log", generateNoisyLogFixture, detect.CmdGeneric},
		{"mixed errors", generateMixedErrorsFixture, detect.CmdGeneric},
		{"ctx scan", generateCtxScanFixture, detect.CmdGeneric},
		{"ctx detail", generateCtxDetailFixture, detect.CmdGeneric},
	}
}

// FixtureInput holds the generated input and command type for a fixture.
type FixtureInput struct {
	Name    string
	Input   string
	CmdType detect.CmdType
}

// AllFixtureInputs generates all fixture inputs and returns them.
func AllFixtureInputs() []FixtureInput {
	fixtures := allFixtures()
	result := make([]FixtureInput, len(fixtures))
	for i, f := range fixtures {
		result[i] = FixtureInput{
			Name:    f.name,
			Input:   f.gen(),
			CmdType: f.cmdType,
		}
	}
	return result
}

// RunBenchmarks runs all built-in benchmark fixtures and returns a report.
func RunBenchmarks(cfg *config.Config) Report {
	fixtures := allFixtures()
	report := Report{}

	for _, f := range fixtures {
		input := f.gen()
		r := RunSingleBenchmark(cfg, f.name, input, f.cmdType)
		report.Results = append(report.Results, r)
		report.TotalRaw += r.RawBytes
		report.TotalClean += r.CleanBytes
		report.TotalDuration += r.Duration
	}

	if report.TotalRaw > 0 {
		report.AvgReduction = float64(report.TotalRaw-report.TotalClean) / float64(report.TotalRaw) * 100
	}

	return report
}

// RunSingleBenchmark benchmarks a single input with a given command type.
func RunSingleBenchmark(cfg *config.Config, name string, input string, cmdType detect.CmdType) Result {
	chain := proxy.BuildChain(cfg, cmdType, cfg.General.MaxLines)

	start := time.Now()
	cleaned := chain.Apply(input)
	duration := time.Since(start)

	rawBytes := len(input)
	cleanBytes := len(cleaned)
	reduction := 0.0
	if rawBytes > 0 {
		reduction = float64(rawBytes-cleanBytes) / float64(rawBytes) * 100
	}

	return Result{
		Name:       name,
		RawBytes:   rawBytes,
		CleanBytes: cleanBytes,
		Reduction:  reduction,
		Duration:   duration,
		LinesRaw:   strings.Count(input, "\n") + 1,
		LinesClean: strings.Count(cleaned, "\n") + 1,
	}
}

// FormatReport formats a Report as a human-readable text table.
func FormatReport(report Report) string {
	var sb strings.Builder
	sb.WriteString("GoTK Benchmark Report\n")
	sb.WriteString("=====================\n\n")
	sb.WriteString(fmt.Sprintf("%-24s %10s %10s %10s %10s\n", "Name", "Raw", "Clean", "Reduction", "Time"))

	for _, r := range report.Results {
		sb.WriteString(fmt.Sprintf("%-24s %10s %10s %9.1f%% %10s\n",
			r.Name,
			formatBytes(r.RawBytes),
			formatBytes(r.CleanBytes),
			-r.Reduction,
			formatDuration(r.Duration),
		))
	}

	sb.WriteString(fmt.Sprintf("\n%-24s %10s %10s %9.1f%% %10s\n",
		"Total",
		formatBytes(report.TotalRaw),
		formatBytes(report.TotalClean),
		-report.AvgReduction,
		formatDuration(report.TotalDuration),
	))

	return sb.String()
}

// formatBytes formats a byte count with comma separators.
func formatBytes(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	remainder := len(s) % 3
	if remainder > 0 {
		result = append(result, s[:remainder]...)
	}
	for i := remainder; i < len(s); i += 3 {
		if len(result) > 0 {
			result = append(result, ',')
		}
		result = append(result, s[i:i+3]...)
	}
	return string(result)
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(d.Microseconds()))
	}
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
}
