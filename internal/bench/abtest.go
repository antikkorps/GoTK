package bench

import (
	"fmt"
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/measure"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// ABModeResult holds the A/B test results for a single mode on a single fixture.
type ABModeResult struct {
	Mode           string
	RawTokens      int
	CleanTokens    int
	TokensSaved    int
	ReductionPct   float64
	QualityScore   float64
	ImportantLines int
	PreservedLines int
	MissedLines    []string
}

// ABFixtureResult holds all mode results for a single fixture.
type ABFixtureResult struct {
	Name    string
	Results []ABModeResult // one per mode: raw, conservative, balanced, aggressive
}

// ABReport holds the full A/B test report.
type ABReport struct {
	Fixtures []ABFixtureResult
	Summary  []ABModeSummary
}

// ABModeSummary holds aggregated stats for a single mode across all fixtures.
type ABModeSummary struct {
	Mode           string
	TotalRawTokens   int
	TotalCleanTokens int
	TotalSaved       int
	AvgReduction     float64
	AvgQuality       float64
	TotalImportant   int
	TotalPreserved   int
}

// modes to compare
var abModes = []struct {
	name string
	mode config.FilterMode
}{
	{"conservative", config.ModeConservative},
	{"balanced", config.ModeBalanced},
	{"aggressive", config.ModeAggressive},
}

// RunABTest runs all fixtures through each mode and compares results.
func RunABTest(baseCfg *config.Config) ABReport {
	fixtures := allFixtures()
	report := ABReport{}

	// Initialize summaries: raw + 3 modes
	summaries := make([]ABModeSummary, len(abModes)+1)
	summaries[0] = ABModeSummary{Mode: "raw (no filter)"}
	for i, m := range abModes {
		summaries[i+1] = ABModeSummary{Mode: m.name}
	}

	for _, f := range fixtures {
		input := f.gen()
		rawTokens := measure.EstimateTokens(input)

		fr := ABFixtureResult{Name: f.name}

		// Raw baseline (no filtering)
		important, _, _ := countImportantLines(input, input)
		rawResult := ABModeResult{
			Mode:           "raw (no filter)",
			RawTokens:      rawTokens,
			CleanTokens:    rawTokens,
			TokensSaved:    0,
			ReductionPct:   0,
			QualityScore:   100,
			ImportantLines: important,
			PreservedLines: important,
		}
		fr.Results = append(fr.Results, rawResult)
		summaries[0].TotalRawTokens += rawTokens
		summaries[0].TotalCleanTokens += rawTokens
		summaries[0].TotalImportant += important
		summaries[0].TotalPreserved += important
		summaries[0].AvgQuality += 100

		// Each filter mode
		for mi, m := range abModes {
			modeCfg := copyConfigWithMode(baseCfg, m.mode)
			chain := proxy.BuildChain(modeCfg, f.cmdType, modeCfg.General.MaxLines)
			cleaned := chain.Apply(input)

			cleanTokens := measure.EstimateTokens(cleaned)
			saved := rawTokens - cleanTokens
			pct := 0.0
			if rawTokens > 0 {
				pct = float64(saved) / float64(rawTokens) * 100
			}

			important, preserved, missed := countImportantLines(input, cleaned)
			quality := 100.0
			if important > 0 {
				quality = float64(preserved) / float64(important) * 100
			}

			mr := ABModeResult{
				Mode:           m.name,
				RawTokens:      rawTokens,
				CleanTokens:    cleanTokens,
				TokensSaved:    saved,
				ReductionPct:   pct,
				QualityScore:   quality,
				ImportantLines: important,
				PreservedLines: preserved,
				MissedLines:    missed,
			}
			fr.Results = append(fr.Results, mr)

			si := mi + 1
			summaries[si].TotalRawTokens += rawTokens
			summaries[si].TotalCleanTokens += cleanTokens
			summaries[si].TotalSaved += saved
			summaries[si].TotalImportant += important
			summaries[si].TotalPreserved += preserved
			summaries[si].AvgQuality += quality
		}

		report.Fixtures = append(report.Fixtures, fr)
	}

	// Compute averages
	n := float64(len(fixtures))
	for i := range summaries {
		if n > 0 {
			summaries[i].AvgQuality /= n
		}
		if summaries[i].TotalRawTokens > 0 {
			summaries[i].AvgReduction = float64(summaries[i].TotalSaved) / float64(summaries[i].TotalRawTokens) * 100
		}
	}
	report.Summary = summaries

	return report
}

// copyConfigWithMode creates a config copy with the given mode applied.
func copyConfigWithMode(base *config.Config, mode config.FilterMode) *config.Config {
	c := *base
	c.General.Mode = mode
	c.ApplyMode()
	return &c
}

// countImportantLines counts important lines in raw input and how many survive in output.
func countImportantLines(raw, output string) (important, preserved int, missed []string) {
	inputLines, levels := classify.ClassifyLines(raw)

	outputLines := strings.Split(output, "\n")
	normalizedOutput := make(map[string]bool, len(outputLines))
	for _, line := range outputLines {
		n := strings.TrimSpace(filter.StripANSI(line))
		if n != "" {
			normalizedOutput[n] = true
		}
	}
	normalizedFull := strings.TrimSpace(filter.StripANSI(output))

	for i, level := range levels {
		if level < classify.Warning {
			continue
		}
		important++

		normalized := strings.TrimSpace(filter.StripANSI(inputLines[i]))
		if normalized == "" {
			preserved++
			continue
		}

		if isPreserved(normalized, normalizedOutput, normalizedFull) {
			preserved++
		} else {
			missed = append(missed, inputLines[i])
		}
	}
	return
}

// FormatABTest formats the A/B test report as a human-readable table.
func FormatABTest(report ABReport) string {
	var b strings.Builder

	b.WriteString("GoTK A/B Test — Mode Comparison\n")
	b.WriteString(strings.Repeat("=", 78) + "\n\n")

	// Per-fixture detail
	for _, f := range report.Fixtures {
		fmt.Fprintf(&b, "  %s\n", f.Name)
		fmt.Fprintf(&b, "  %-16s %8s %8s %8s %7s %7s\n",
			"MODE", "RAW TOK", "CLEAN", "SAVED", "REDUC", "QUAL")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 58))
		for _, r := range f.Results {
			fmt.Fprintf(&b, "  %-16s %8d %8d %8d %6.1f%% %6.1f%%\n",
				r.Mode, r.RawTokens, r.CleanTokens, r.TokensSaved,
				r.ReductionPct, r.QualityScore)
		}
		b.WriteString("\n")
	}

	// Summary
	b.WriteString("Summary (all fixtures)\n")
	b.WriteString(strings.Repeat("=", 78) + "\n\n")
	fmt.Fprintf(&b, "  %-16s %10s %10s %10s %7s %7s\n",
		"MODE", "RAW TOK", "CLEAN TOK", "SAVED", "REDUC", "QUAL")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 64))
	for _, s := range report.Summary {
		fmt.Fprintf(&b, "  %-16s %10d %10d %10d %6.1f%% %6.1f%%\n",
			s.Mode, s.TotalRawTokens, s.TotalCleanTokens, s.TotalSaved,
			s.AvgReduction, s.AvgQuality)
	}

	// Quality warnings
	var warnings []string
	for _, f := range report.Fixtures {
		for _, r := range f.Results {
			if len(r.MissedLines) > 0 {
				for _, line := range r.MissedLines {
					warnings = append(warnings, fmt.Sprintf("  [%s/%s] %s", f.Name, r.Mode, line))
				}
			}
		}
	}
	if len(warnings) > 0 {
		fmt.Fprintf(&b, "\nMissed important lines (%d):\n", len(warnings))
		for _, w := range warnings {
			b.WriteString(w + "\n")
		}
	} else {
		b.WriteString("\nAll important lines preserved across all modes.\n")
	}

	return b.String()
}
