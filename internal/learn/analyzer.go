package learn

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
)

// AnalyzerConfig controls the sensitivity of pattern analysis.
type AnalyzerConfig struct {
	MinSessions   int     // minimum sessions to have data from before suggesting (default: 3)
	MinFrequency  float64 // minimum line frequency to consider (default: 0.05 = 5%)
	MinNoiseScore float64 // minimum % of matching lines that are Noise/Debug (default: 0.80)
	MaxBreadth    float64 // maximum % of total lines a pattern can match (default: 0.30)
}

// DefaultAnalyzerConfig returns sensible defaults.
func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		MinSessions:   3,
		MinFrequency:  0.05,
		MinNoiseScore: 0.80,
		MaxBreadth:    0.30,
	}
}

// Candidate represents a proposed always_remove pattern.
type Candidate struct {
	Pattern     string   // regex pattern for always_remove
	Description string   // human-readable description
	Frequency   float64  // fraction of lines matched
	NoiseScore  float64  // fraction of matches classified as Noise/Debug
	MatchCount  int      // absolute number of matches
	Samples     []string // up to 3 example raw lines
	Commands    []string // which commands produced these lines
}

// AnalysisResult holds the output of Analyze.
type AnalysisResult struct {
	TotalLines  int
	Sessions    int
	Candidates  []Candidate
	NotReady    bool   // true if not enough data yet
	NotReadyMsg string // explanation if not ready
}

// Analyze examines collected observations and produces pattern candidates.
func Analyze(observations []Observation, acfg AnalyzerConfig) *AnalysisResult {
	if len(observations) == 0 {
		return &AnalysisResult{
			NotReady:    true,
			NotReadyMsg: "No observations collected yet. Run 'gotk learn run <command>' a few times first.",
		}
	}

	// Count unique sessions
	sessions := make(map[string]bool)
	for _, obs := range observations {
		sessions[obs.SessionID] = true
	}

	if len(sessions) < acfg.MinSessions {
		return &AnalysisResult{
			TotalLines: len(observations),
			Sessions:   len(sessions),
			NotReady:   true,
			NotReadyMsg: fmt.Sprintf(
				"Need at least %d sessions, have %d. Run more commands with 'gotk learn run <command>'.",
				acfg.MinSessions, len(sessions)),
		}
	}

	// Group observations by normalized form
	type group struct {
		normalized string
		count      int
		levels     []int
		commands   map[string]bool
		samples    []string // raw normalized lines as samples
	}

	groups := make(map[string]*group)
	for _, obs := range observations {
		g, ok := groups[obs.Normalized]
		if !ok {
			g = &group{
				normalized: obs.Normalized,
				commands:   make(map[string]bool),
			}
			groups[obs.Normalized] = g
		}
		g.count++
		g.levels = append(g.levels, obs.Level)
		g.commands[obs.Command] = true
		if len(g.samples) < 3 {
			g.samples = append(g.samples, obs.Normalized)
		}
	}

	totalLines := len(observations)
	var candidates []Candidate

	for _, g := range groups {
		freq := float64(g.count) / float64(totalLines)

		// Skip patterns that are too rare
		if freq < acfg.MinFrequency {
			continue
		}

		// Skip patterns that are too broad
		if freq > acfg.MaxBreadth {
			continue
		}

		// Calculate noise score: fraction of matching lines that are Noise or Debug
		noiseCount := 0
		hasWarningOrAbove := false
		for _, level := range g.levels {
			if level <= int(classify.Debug) {
				noiseCount++
			}
			if level >= int(classify.Warning) {
				hasWarningOrAbove = true
			}
		}
		noiseScore := float64(noiseCount) / float64(len(g.levels))

		// Safety: never suggest patterns that match any Warning+ lines
		if hasWarningOrAbove {
			continue
		}

		// Skip if noise score is too low
		if noiseScore < acfg.MinNoiseScore {
			continue
		}

		// Generate regex pattern
		pattern := ToRegex(g.normalized)

		// Validate the regex compiles
		if _, err := regexp.Compile(pattern); err != nil {
			continue
		}

		cmds := make([]string, 0, len(g.commands))
		for cmd := range g.commands {
			cmds = append(cmds, cmd)
		}
		sort.Strings(cmds)

		candidates = append(candidates, Candidate{
			Pattern:     pattern,
			Description: describePattern(g.normalized),
			Frequency:   freq,
			NoiseScore:  noiseScore,
			MatchCount:  g.count,
			Samples:     g.samples,
			Commands:    cmds,
		})
	}

	// Sort by frequency descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Frequency > candidates[j].Frequency
	})

	// Limit to top 20 suggestions
	if len(candidates) > 20 {
		candidates = candidates[:20]
	}

	return &AnalysisResult{
		TotalLines: totalLines,
		Sessions:   len(sessions),
		Candidates: candidates,
	}
}

// FormatSuggestions formats the analysis result as a human-readable string.
func FormatSuggestions(r *AnalysisResult) string {
	if r.NotReady {
		return r.NotReadyMsg + "\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "gotk learn: analyzed %d lines from %d sessions\n\n", r.TotalLines, r.Sessions)

	if len(r.Candidates) == 0 {
		b.WriteString("No noise patterns detected. Your output is already clean!\n")
		return b.String()
	}

	b.WriteString("Suggested patterns for always_remove:\n\n")

	for i, c := range r.Candidates {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, c.Description)
		fmt.Fprintf(&b, "     Pattern:   %s\n", c.Pattern)
		fmt.Fprintf(&b, "     Frequency: %.0f%% (%d lines), Noise confidence: %.0f%%\n",
			c.Frequency*100, c.MatchCount, c.NoiseScore*100)
		fmt.Fprintf(&b, "     Commands:  %s\n", strings.Join(c.Commands, ", "))
		for _, s := range c.Samples {
			fmt.Fprintf(&b, "     Example:   %q\n", truncateSample(s, 80))
		}
		b.WriteString("\n")
	}

	// Generate TOML snippet
	b.WriteString("Add to .gotk.toml:\n\n")
	b.WriteString("[rules]\n")
	b.WriteString("always_remove = [\n")
	for _, c := range r.Candidates {
		fmt.Fprintf(&b, "  %q,\n", c.Pattern)
	}
	b.WriteString("]\n")

	return b.String()
}

// FormatStatus formats the store statistics as a human-readable string.
func FormatStatus(stats *StoreStats) string {
	if stats.TotalObservations == 0 {
		return "gotk learn: no observations collected yet.\nRun 'gotk learn run <command>' to start observing.\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "gotk learn status:\n")
	fmt.Fprintf(&b, "  Observations: %d\n", stats.TotalObservations)
	fmt.Fprintf(&b, "  Sessions:     %d\n", stats.Sessions)
	fmt.Fprintf(&b, "  Commands:     %d\n", stats.Commands)
	fmt.Fprintf(&b, "  Store size:   %d bytes\n", stats.FileSize)
	if stats.OldestEntry != "" {
		fmt.Fprintf(&b, "  First entry:  %s\n", stats.OldestEntry)
		fmt.Fprintf(&b, "  Last entry:   %s\n", stats.NewestEntry)
	}
	return b.String()
}

// describePattern generates a short human-readable description from a normalized line.
func describePattern(normalized string) string {
	// Take the first ~60 chars of the normalized line
	s := normalized
	if len(s) > 60 {
		s = s[:60] + "..."
	}
	return s
}

// truncateSample truncates a sample line to maxLen characters.
func truncateSample(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
