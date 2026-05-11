package bench

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// DefaultPerFixtureDrift is the per-fixture pp drift above which parity fails.
const DefaultPerFixtureDrift = 5.0

// DefaultAvgDrift is the total-avg pp drift above which parity fails.
const DefaultAvgDrift = 5.0

// ParityResult is the comparison of one fixture between baseline and candidate.
type ParityResult struct {
	Name               string
	BaselineReduction  float64
	CandidateReduction float64
	Drift              float64 // candidate - baseline, in percentage points
	OverThreshold      bool
	MissingInBaseline  bool
	MissingInCandidate bool
}

// ParityReport is the full comparison between two bench reports.
type ParityReport struct {
	BaselineLabel   string
	CandidateLabel  string
	BaselineAvg     float64
	CandidateAvg    float64
	AvgDrift        float64 // candidate - baseline
	MaxDrift        float64 // max absolute per-fixture drift
	PerFixtureLimit float64
	AvgLimit        float64
	Results         []ParityResult
	Pass            bool
}

// reportJSON mirrors FormatReportJSON's output for parsing.
type reportJSON struct {
	TotalRaw      int          `json:"total_raw"`
	TotalClean    int          `json:"total_clean"`
	AvgReduction  float64      `json:"avg_reduction"`
	TotalDuration int64        `json:"total_duration_us"`
	Results       []resultJSON `json:"results"`
}

type resultJSON struct {
	Name       string  `json:"name"`
	RawBytes   int     `json:"raw_bytes"`
	CleanBytes int     `json:"clean_bytes"`
	Reduction  float64 `json:"reduction"`
	DurationUs int64   `json:"duration_us"`
	LinesRaw   int     `json:"lines_raw"`
	LinesClean int     `json:"lines_clean"`
}

// LoadReportJSON parses a bench --json file into a Report.
func LoadReportJSON(path string) (Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, fmt.Errorf("read %s: %w", path, err)
	}
	var rj reportJSON
	if err := json.Unmarshal(data, &rj); err != nil {
		return Report{}, fmt.Errorf("parse %s: %w", path, err)
	}
	r := Report{
		TotalRaw:     rj.TotalRaw,
		TotalClean:   rj.TotalClean,
		AvgReduction: rj.AvgReduction,
	}
	for _, x := range rj.Results {
		r.Results = append(r.Results, Result{
			Name:       x.Name,
			RawBytes:   x.RawBytes,
			CleanBytes: x.CleanBytes,
			Reduction:  x.Reduction,
			LinesRaw:   x.LinesRaw,
			LinesClean: x.LinesClean,
		})
	}
	return r, nil
}

// Compare diffs two bench reports against drift thresholds (in percentage points).
// baselineLabel/candidateLabel are display strings (e.g. "linux", "windows").
func Compare(baseline, candidate Report, baselineLabel, candidateLabel string, perFixtureLimit, avgLimit float64) ParityReport {
	if perFixtureLimit <= 0 {
		perFixtureLimit = DefaultPerFixtureDrift
	}
	if avgLimit <= 0 {
		avgLimit = DefaultAvgDrift
	}

	bIdx := indexResults(baseline.Results)
	cIdx := indexResults(candidate.Results)

	names := map[string]struct{}{}
	for n := range bIdx {
		names[n] = struct{}{}
	}
	for n := range cIdx {
		names[n] = struct{}{}
	}

	report := ParityReport{
		BaselineLabel:   baselineLabel,
		CandidateLabel:  candidateLabel,
		BaselineAvg:     baseline.AvgReduction,
		CandidateAvg:    candidate.AvgReduction,
		AvgDrift:        candidate.AvgReduction - baseline.AvgReduction,
		PerFixtureLimit: perFixtureLimit,
		AvgLimit:        avgLimit,
		Pass:            true,
	}

	for name := range names {
		b, bOK := bIdx[name]
		c, cOK := cIdx[name]
		pr := ParityResult{Name: name}
		switch {
		case !bOK:
			pr.MissingInBaseline = true
			pr.CandidateReduction = c.Reduction
			pr.OverThreshold = true
		case !cOK:
			pr.MissingInCandidate = true
			pr.BaselineReduction = b.Reduction
			pr.OverThreshold = true
		default:
			pr.BaselineReduction = b.Reduction
			pr.CandidateReduction = c.Reduction
			pr.Drift = c.Reduction - b.Reduction
			if math.Abs(pr.Drift) > perFixtureLimit {
				pr.OverThreshold = true
			}
		}
		if math.Abs(pr.Drift) > math.Abs(report.MaxDrift) {
			report.MaxDrift = pr.Drift
		}
		if pr.OverThreshold {
			report.Pass = false
		}
		report.Results = append(report.Results, pr)
	}

	sort.Slice(report.Results, func(i, j int) bool {
		// Outliers first (largest |drift|), then alphabetical.
		di, dj := math.Abs(report.Results[i].Drift), math.Abs(report.Results[j].Drift)
		if di != dj {
			return di > dj
		}
		return report.Results[i].Name < report.Results[j].Name
	})

	if math.Abs(report.AvgDrift) > avgLimit {
		report.Pass = false
	}

	return report
}

func indexResults(rs []Result) map[string]Result {
	m := make(map[string]Result, len(rs))
	for _, r := range rs {
		m[r.Name] = r
	}
	return m
}

// FormatParity renders a human-readable parity report.
func FormatParity(p ParityReport) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Bench parity: %s vs %s\n", p.BaselineLabel, p.CandidateLabel)
	fmt.Fprintf(&sb, "  thresholds: per-fixture %.1fpp, total-avg %.1fpp\n", p.PerFixtureLimit, p.AvgLimit)
	fmt.Fprintf(&sb, "  %s avg: %.2f%%   %s avg: %.2f%%   drift: %+.2fpp\n",
		p.BaselineLabel, p.BaselineAvg, p.CandidateLabel, p.CandidateAvg, p.AvgDrift)
	fmt.Fprintf(&sb, "  max per-fixture drift: %+.2fpp\n", p.MaxDrift)
	if p.Pass {
		sb.WriteString("  status: PASS\n")
	} else {
		sb.WriteString("  status: FAIL\n")
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "%-28s %10s %10s %10s\n", "Fixture", p.BaselineLabel, p.CandidateLabel, "Drift")
	for _, r := range p.Results {
		marker := "  "
		if r.OverThreshold {
			marker = "! "
		}
		switch {
		case r.MissingInBaseline:
			fmt.Fprintf(&sb, "%s%-26s %10s %10.2f %10s\n", marker, r.Name, "-", r.CandidateReduction, "missing")
		case r.MissingInCandidate:
			fmt.Fprintf(&sb, "%s%-26s %10.2f %10s %10s\n", marker, r.Name, r.BaselineReduction, "-", "missing")
		default:
			fmt.Fprintf(&sb, "%s%-26s %10.2f %10.2f %+10.2f\n", marker, r.Name, r.BaselineReduction, r.CandidateReduction, r.Drift)
		}
	}
	return sb.String()
}

// FormatParityJSON renders a parity report as JSON.
func FormatParityJSON(p ParityReport) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	fmt.Fprintf(&sb, "  \"baseline_label\": %s,\n", jsonString(p.BaselineLabel))
	fmt.Fprintf(&sb, "  \"candidate_label\": %s,\n", jsonString(p.CandidateLabel))
	fmt.Fprintf(&sb, "  \"baseline_avg\": %.2f,\n", p.BaselineAvg)
	fmt.Fprintf(&sb, "  \"candidate_avg\": %.2f,\n", p.CandidateAvg)
	fmt.Fprintf(&sb, "  \"avg_drift\": %.2f,\n", p.AvgDrift)
	fmt.Fprintf(&sb, "  \"max_drift\": %.2f,\n", p.MaxDrift)
	fmt.Fprintf(&sb, "  \"per_fixture_limit\": %.2f,\n", p.PerFixtureLimit)
	fmt.Fprintf(&sb, "  \"avg_limit\": %.2f,\n", p.AvgLimit)
	fmt.Fprintf(&sb, "  \"pass\": %t,\n", p.Pass)
	sb.WriteString("  \"results\": [\n")
	for i, r := range p.Results {
		sb.WriteString("    {\n")
		fmt.Fprintf(&sb, "      \"name\": %s,\n", jsonString(r.Name))
		fmt.Fprintf(&sb, "      \"baseline_reduction\": %.2f,\n", r.BaselineReduction)
		fmt.Fprintf(&sb, "      \"candidate_reduction\": %.2f,\n", r.CandidateReduction)
		fmt.Fprintf(&sb, "      \"drift\": %.2f,\n", r.Drift)
		fmt.Fprintf(&sb, "      \"over_threshold\": %t,\n", r.OverThreshold)
		fmt.Fprintf(&sb, "      \"missing_in_baseline\": %t,\n", r.MissingInBaseline)
		fmt.Fprintf(&sb, "      \"missing_in_candidate\": %t\n", r.MissingInCandidate)
		sb.WriteString("    }")
		if i < len(p.Results)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	return sb.String()
}
