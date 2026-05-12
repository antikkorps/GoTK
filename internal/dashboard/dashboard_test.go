package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antikkorps/GoTK/internal/bench"
	"github.com/antikkorps/GoTK/internal/learn"
	"github.com/antikkorps/GoTK/internal/measure"
)

func TestSortedCommandTypes_OrderedByTokensSavedDesc(t *testing.T) {
	in := map[string]*measure.CommandTypeStats{
		"grep": {Invocations: 5, TokensSaved: 1000, AvgReduction: 90},
		"git":  {Invocations: 3, TokensSaved: 500, AvgReduction: 80},
		"go":   {Invocations: 10, TokensSaved: 2000, AvgReduction: 70},
	}
	rows := sortedCommandTypes(in)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	want := []string{"go", "grep", "git"}
	for i, n := range want {
		if rows[i].name != n {
			t.Errorf("row %d: want %s, got %s", i, n, rows[i].name)
		}
	}
}

func TestSortedCommandTypes_TiebreakerAlphabetical(t *testing.T) {
	in := map[string]*measure.CommandTypeStats{
		"zsh":  {TokensSaved: 100},
		"bash": {TokensSaved: 100},
		"fish": {TokensSaved: 100},
	}
	rows := sortedCommandTypes(in)
	want := []string{"bash", "fish", "zsh"}
	for i, n := range want {
		if rows[i].name != n {
			t.Errorf("row %d: want %s, got %s", i, n, rows[i].name)
		}
	}
}

func TestSortedCommandTypes_CapsAtEight(t *testing.T) {
	in := map[string]*measure.CommandTypeStats{}
	for i := range 20 {
		in[string(rune('a'+i))] = &measure.CommandTypeStats{TokensSaved: i}
	}
	rows := sortedCommandTypes(in)
	if len(rows) != 8 {
		t.Fatalf("expected top 8, got %d", len(rows))
	}
}

func TestFormatInt_ThousandsSeparator(t *testing.T) {
	cases := map[int]string{
		0:       "0",
		42:      "42",
		999:     "999",
		1000:    "1,000",
		12345:   "12,345",
		1234567: "1,234,567",
	}
	for in, want := range cases {
		if got := formatInt(in); got != want {
			t.Errorf("formatInt(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"abcdef", 6, "abcdef"},
		{"abcdef", 5, "ab..."},
		{"abcdef", 2, "ab"},
	}
	for _, c := range cases {
		if got := truncate(c.in, c.max); got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestKeyForPeriod(t *testing.T) {
	cases := map[string]string{
		"today": "1",
		"7d":    "7",
		"30d":   "3",
		"all":   "a",
		"weird": "?",
	}
	for in, want := range cases {
		if got := keyForPeriod(in); got != want {
			t.Errorf("keyForPeriod(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModel_PeriodSwitchKeys(t *testing.T) {
	m := New("/nonexistent/path")
	if m.period != "7d" {
		t.Fatalf("default period should be 7d, got %s", m.period)
	}

	cases := []struct {
		key  string
		want string
	}{
		{"1", "today"},
		{"7", "7d"},
		{"3", "30d"},
		{"0", "30d"},
		{"a", "all"},
	}
	for _, c := range cases {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(c.key)})
		got := updated.(Model).period
		if got != c.want {
			t.Errorf("key %q: want period %q, got %q", c.key, c.want, got)
		}
		m = updated.(Model)
	}
}

func TestModel_QuitKeys(t *testing.T) {
	m := New("/nonexistent/path")
	for _, key := range []string{"q", "ctrl+c", "esc"} {
		var msg tea.Msg
		switch key {
		case "q":
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
		case "ctrl+c":
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		}
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Errorf("key %q: expected a quit cmd, got nil", key)
			continue
		}
		// tea.Quit returns a tea.QuitMsg when invoked.
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("key %q: expected tea.QuitMsg, got %T", key, cmd())
		}
	}
}

func TestView_EmptyReportShowsHint(t *testing.T) {
	m := New("/nonexistent/path")
	out := m.View()
	if !strings.Contains(out, "No measurement entries") {
		t.Errorf("empty view should mention missing entries; got:\n%s", out)
	}
	if !strings.Contains(out, "q quit") {
		t.Errorf("empty view should include the footer hint; got:\n%s", out)
	}
}

func TestView_RendersTotalsWhenDataPresent(t *testing.T) {
	m := New("/nonexistent/path")
	m.report = measure.Report{
		Period:           "7d",
		TotalInvocations: 42,
		TotalTokensSaved: 12345,
		AvgReduction:     78.4,
		AvgQualityScore:  95.0,
		ByCommandType: map[string]*measure.CommandTypeStats{
			"grep": {Invocations: 10, TokensSaved: 5000, AvgReduction: 90},
		},
	}
	out := m.View()
	for _, want := range []string{"42", "12,345", "78.4%", "grep"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q; got:\n%s", want, out)
		}
	}
}

func TestRenderLearn_NilResult_ShowsHint(t *testing.T) {
	out := renderLearn(nil, nil)
	if !strings.Contains(out, "Learn patterns") {
		t.Errorf("missing title: %q", out)
	}
	if !strings.Contains(out, "no observations yet") {
		t.Errorf("missing empty-state hint: %q", out)
	}
}

func TestRenderLearn_StoreErrorRenders(t *testing.T) {
	out := renderLearn(nil, errFake("disk gone"))
	if !strings.Contains(out, "cannot read learn store") {
		t.Errorf("missing error label: %q", out)
	}
	if !strings.Contains(out, "disk gone") {
		t.Errorf("missing underlying error text: %q", out)
	}
}

func TestRenderLearn_NotReadyShowsAnalyzerMessage(t *testing.T) {
	res := &learn.AnalysisResult{NotReady: true, NotReadyMsg: "need more sessions"}
	out := renderLearn(res, nil)
	if !strings.Contains(out, "need more sessions") {
		t.Errorf("expected NotReadyMsg in output, got %q", out)
	}
}

func TestRenderLearn_CandidatesRanked(t *testing.T) {
	res := &learn.AnalysisResult{
		TotalLines: 1234,
		Sessions:   7,
		Candidates: []learn.Candidate{
			{Description: "alpha", Frequency: 0.4, NoiseScore: 0.9, MatchCount: 40},
			{Description: "beta", Frequency: 0.3, NoiseScore: 0.85, MatchCount: 30},
		},
	}
	out := renderLearn(res, nil)
	if !strings.Contains(out, "1234 observations") {
		t.Errorf("missing observation count: %q", out)
	}
	idxA := strings.Index(out, "alpha")
	idxB := strings.Index(out, "beta")
	if idxA < 0 || idxB < 0 || idxA > idxB {
		t.Errorf("expected alpha before beta in output, got %q", out)
	}
}

type errFakeT string

func (e errFakeT) Error() string { return string(e) }
func errFake(s string) error     { return errFakeT(s) }

func TestRenderPerFilter_NilSnapshotShowsHint(t *testing.T) {
	out := renderPerFilter(nil, nil)
	if !strings.Contains(out, "Per-filter compression") {
		t.Errorf("missing title: %q", out)
	}
	if !strings.Contains(out, "gotk bench --per-filter --persist") {
		t.Errorf("missing onboarding hint: %q", out)
	}
}

func TestRenderPerFilter_ErrorRenders(t *testing.T) {
	out := renderPerFilter(nil, errFake("disk gone"))
	if !strings.Contains(out, "cannot read snapshot") || !strings.Contains(out, "disk gone") {
		t.Errorf("missing error info: %q", out)
	}
}

func TestRenderPerFilter_EmptySnapshot(t *testing.T) {
	snap := &bench.PerFilterSnapshot{GeneratedAt: "2026-05-12T20:00:00Z"}
	out := renderPerFilter(snap, nil)
	if !strings.Contains(out, "snapshot empty") {
		t.Errorf("expected empty-snapshot hint, got %q", out)
	}
}

func TestRenderPerFilter_RanksRowsAndShowsSaved(t *testing.T) {
	snap := &bench.PerFilterSnapshot{
		GeneratedAt: "2026-05-12T20:00:00Z",
		Filters: []bench.FilterSnapshot{
			{FilterName: "StripANSI", BytesBefore: 10000, BytesAfter: 1000, Reduction: 90, Invocations: 21},
			{FilterName: "Dedup", BytesBefore: 5000, BytesAfter: 4000, Reduction: 20, Invocations: 21},
		},
	}
	out := renderPerFilter(snap, nil)
	if !strings.Contains(out, "StripANSI") || !strings.Contains(out, "Dedup") {
		t.Errorf("missing filter rows: %q", out)
	}
	if !strings.Contains(out, "9,000") {
		t.Errorf("expected formatted bytes-saved 9,000 in output, got %q", out)
	}
	idxA := strings.Index(out, "StripANSI")
	idxB := strings.Index(out, "Dedup")
	if idxA < 0 || idxB < 0 || idxA > idxB {
		t.Errorf("expected StripANSI before Dedup, got %q", out)
	}
}

func TestFormatSnapshotAge(t *testing.T) {
	if got := formatSnapshotAge(""); got != "" {
		t.Errorf("empty input: want \"\", got %q", got)
	}
	if got := formatSnapshotAge("not-a-date"); got != "not-a-date" {
		t.Errorf("unparseable input should round-trip, got %q", got)
	}
}
