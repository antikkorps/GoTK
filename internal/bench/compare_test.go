package bench

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompare_IdenticalReportsPass(t *testing.T) {
	r := Report{
		AvgReduction: 70.00,
		Results: []Result{
			{Name: "grep large", Reduction: 95.5},
			{Name: "go test mixed", Reduction: 60.0},
		},
	}
	p := Compare(r, r, "linux", "windows", 0, 0)
	if !p.Pass {
		t.Fatalf("identical reports should pass, got %+v", p)
	}
	if p.AvgDrift != 0 || p.MaxDrift != 0 {
		t.Fatalf("expected zero drifts, got avg=%v max=%v", p.AvgDrift, p.MaxDrift)
	}
	if len(p.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(p.Results))
	}
}

func TestCompare_PerFixtureDriftFails(t *testing.T) {
	baseline := Report{
		AvgReduction: 80,
		Results: []Result{
			{Name: "grep large", Reduction: 95.0},
			{Name: "git log", Reduction: 65.0},
		},
	}
	candidate := Report{
		AvgReduction: 80,
		Results: []Result{
			{Name: "grep large", Reduction: 95.0},
			{Name: "git log", Reduction: 58.0}, // 7pp drop > 5pp
		},
	}
	p := Compare(baseline, candidate, "linux", "windows", 0, 0)
	if p.Pass {
		t.Fatalf("expected FAIL on 7pp drift, got PASS")
	}
	// First in sorted order should be the outlier.
	if p.Results[0].Name != "git log" {
		t.Fatalf("expected outlier 'git log' first, got %q", p.Results[0].Name)
	}
	if !p.Results[0].OverThreshold {
		t.Fatalf("git log result should be over threshold")
	}
}

func TestCompare_AvgDriftFails(t *testing.T) {
	baseline := Report{AvgReduction: 80, Results: []Result{{Name: "x", Reduction: 80}}}
	candidate := Report{AvgReduction: 73, Results: []Result{{Name: "x", Reduction: 80}}}
	// per-fixture drift = 0, but avg drift = -7pp > 5pp limit.
	p := Compare(baseline, candidate, "linux", "windows", 0, 0)
	if p.Pass {
		t.Fatalf("expected FAIL on avg drift of -7pp")
	}
}

func TestCompare_MissingFixtureFails(t *testing.T) {
	baseline := Report{
		AvgReduction: 80,
		Results: []Result{
			{Name: "a", Reduction: 80},
			{Name: "b", Reduction: 80},
		},
	}
	candidate := Report{
		AvgReduction: 80,
		Results: []Result{
			{Name: "a", Reduction: 80},
		},
	}
	p := Compare(baseline, candidate, "linux", "windows", 0, 0)
	if p.Pass {
		t.Fatalf("missing fixture should fail")
	}
	var found bool
	for _, r := range p.Results {
		if r.Name == "b" && r.MissingInCandidate {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected fixture 'b' flagged as missing in candidate")
	}
}

func TestLoadReportJSON_Roundtrip(t *testing.T) {
	original := Report{
		TotalRaw:     1000,
		TotalClean:   200,
		AvgReduction: 80.0,
		Results: []Result{
			{Name: "grep large", RawBytes: 500, CleanBytes: 50, Reduction: 90, LinesRaw: 10, LinesClean: 2},
			{Name: "git log", RawBytes: 500, CleanBytes: 150, Reduction: 70, LinesRaw: 5, LinesClean: 3},
		},
	}
	js := FormatReportJSON(original)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	if err := os.WriteFile(path, []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadReportJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AvgReduction != original.AvgReduction || len(loaded.Results) != len(original.Results) {
		t.Fatalf("roundtrip mismatch: %+v", loaded)
	}
	if loaded.Results[0].Name != "grep large" || loaded.Results[0].Reduction != 90 {
		t.Fatalf("first result mismatch: %+v", loaded.Results[0])
	}
}

func TestFormatParityJSON_ParseableAndContainsFields(t *testing.T) {
	baseline := Report{AvgReduction: 80, Results: []Result{{Name: "a", Reduction: 80}}}
	candidate := Report{AvgReduction: 80, Results: []Result{{Name: "a", Reduction: 80}}}
	p := Compare(baseline, candidate, "linux", "windows", 0, 0)
	js := FormatParityJSON(p)
	for _, want := range []string{`"baseline_label": "linux"`, `"candidate_label": "windows"`, `"pass": true`, `"results"`} {
		if !strings.Contains(js, want) {
			t.Fatalf("FormatParityJSON missing %q: %s", want, js)
		}
	}
}
