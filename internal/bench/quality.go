package bench

import (
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// QualityResult holds the quality measurement for a single fixture.
type QualityResult struct {
	Name           string
	TotalLines     int
	ImportantLines int // Warning + Error + Critical
	PreservedLines int // Important lines found in output
	Score          float64
	MissedLines    []string // Important lines that were lost
}

// QualityReport holds the aggregated quality report across all fixtures.
type QualityReport struct {
	Results            []QualityResult
	TotalImportant     int
	TotalPreserved     int
	OverallScore       float64
	AllFixturesPerfect bool
}

// MeasureQuality runs all fixtures and measures what percentage of
// semantically important lines (Warning, Error, Critical) survive filtering.
func MeasureQuality(cfg *config.Config) QualityReport {
	fixtures := allFixtures()
	report := QualityReport{}

	for _, f := range fixtures {
		input := f.gen()
		r := measureFixtureQuality(cfg, f.name, input, f.cmdType)
		report.Results = append(report.Results, r)
		report.TotalImportant += r.ImportantLines
		report.TotalPreserved += r.PreservedLines
	}

	if report.TotalImportant > 0 {
		report.OverallScore = float64(report.TotalPreserved) / float64(report.TotalImportant) * 100
	} else {
		report.OverallScore = 100
	}
	report.AllFixturesPerfect = report.TotalImportant == report.TotalPreserved

	return report
}

// measureFixtureQuality measures quality for a single fixture.
func measureFixtureQuality(cfg *config.Config, name string, input string, cmdType detect.CmdType) QualityResult {
	chain := proxy.BuildChain(cfg, cmdType, cfg.General.MaxLines)
	output := chain.Apply(input)
	normalizedFullOutput := normalizeLine(output)

	// Classify input lines
	inputLines, levels := classify.ClassifyLines(input)

	// Normalize output lines for matching (strip ANSI, trim whitespace)
	outputLines := strings.Split(output, "\n")
	normalizedOutput := make(map[string]bool, len(outputLines))
	for _, line := range outputLines {
		normalized := normalizeLine(line)
		if normalized != "" {
			normalizedOutput[normalized] = true
		}
	}

	result := QualityResult{
		Name:       name,
		TotalLines: len(inputLines),
	}

	for i, level := range levels {
		if level < classify.Warning {
			continue
		}
		result.ImportantLines++

		normalized := normalizeLine(inputLines[i])
		if normalized == "" {
			result.PreservedLines++
			continue
		}

		if isPreserved(normalized, normalizedOutput, normalizedFullOutput) {
			result.PreservedLines++
		} else {
			result.MissedLines = append(result.MissedLines, inputLines[i])
		}
	}

	if result.ImportantLines > 0 {
		result.Score = float64(result.PreservedLines) / float64(result.ImportantLines) * 100
	} else {
		result.Score = 100
	}

	return result
}

// isPreserved checks whether the semantic content of a line survives in the output.
// It handles cases where filters reformat lines (e.g., grep groups by file,
// stack traces are condensed) but preserve the key information.
func isPreserved(normalized string, outputLines map[string]bool, fullOutput string) bool {
	// Exact match
	if outputLines[normalized] {
		return true
	}

	// Output line contains this line (e.g., wrapped in summary)
	for line := range outputLines {
		if strings.Contains(line, normalized) {
			return true
		}
	}

	// Extract core content — filters may strip file path prefixes
	// (e.g., "src/file.go:22: TODO:value" → "22: TODO:value" or "TODO:value")
	core := extractCoreContent(normalized)
	if core != "" && strings.Contains(fullOutput, core) {
		return true
	}

	return false
}

// extractCoreContent extracts the semantically important part of a line,
// stripping common structural prefixes (file:line:, timestamps, log levels).
func extractCoreContent(line string) string {
	// Strip file:line: prefix (grep/compiler output)
	// Pattern: "path/file.go:123: content" → "content"
	parts := strings.SplitN(line, ":", 3)
	if len(parts) == 3 {
		content := strings.TrimSpace(parts[2])
		if content != "" {
			return content
		}
	}
	return ""
}

// normalizeLine strips ANSI codes and trims whitespace for comparison.
func normalizeLine(line string) string {
	return strings.TrimSpace(filter.StripANSI(line))
}
