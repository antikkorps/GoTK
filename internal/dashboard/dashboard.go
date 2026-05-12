// Package dashboard renders a live TUI summarising gotk activity from the
// measurement log. All data comes from the JSONL log written by
// internal/measure; this package is a pure visualisation layer.
package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/antikkorps/GoTK/internal/learn"
	"github.com/antikkorps/GoTK/internal/measure"
)

const (
	refreshInterval = 2 * time.Second
	recentCount     = 8
)

// periods are presented in the same order the user can cycle through.
var periods = []string{"today", "7d", "30d", "all"}

// Model is the bubbletea model for the dashboard.
type Model struct {
	logPath   string
	storePath string // learn observation store; empty → learn.DefaultStorePath()
	period    string
	report    measure.Report
	recent    []measure.Entry
	learn     *learn.AnalysisResult
	learnErr  error
	err       error
	width     int
	height    int
}

// New returns an initialised dashboard model that reads from logPath.
// If logPath is empty, measure.DefaultLogPath() is used.
func New(logPath string) Model {
	if logPath == "" {
		logPath = measure.DefaultLogPath()
	}
	return Model{
		logPath: logPath,
		period:  "7d",
	}
}

// Init is part of tea.Model. Kicks off the first data load and the refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(loadCmd(m.logPath, m.period), loadLearnCmd(m.storePath), tickCmd())
}

// dataMsg carries a freshly aggregated report + recent invocations.
type dataMsg struct {
	report measure.Report
	recent []measure.Entry
	err    error
}

// learnMsg carries the latest learn-store analysis (or the error from reading
// the store). A nil result with no error means the store doesn't exist yet,
// which we render as a friendly empty-state rather than a failure.
type learnMsg struct {
	result *learn.AnalysisResult
	err    error
}

// tickMsg fires on every refresh tick.
type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

// loadLearnCmd reads the learn observation store and runs the analyzer.
// A missing store is not an error; it returns a nil result so the pane can
// render a "no observations yet" hint.
func loadLearnCmd(storePath string) tea.Cmd {
	return func() tea.Msg {
		if storePath == "" {
			storePath = learn.DefaultStorePath()
		}
		obs, err := learn.StoreRead(storePath)
		if err != nil {
			return learnMsg{err: err}
		}
		if len(obs) == 0 {
			return learnMsg{result: nil}
		}
		return learnMsg{result: learn.Analyze(obs, learn.DefaultAnalyzerConfig())}
	}
}

func loadCmd(path, period string) tea.Cmd {
	return func() tea.Msg {
		entries, err := measure.ReadEntries(path)
		if err != nil {
			return dataMsg{err: err}
		}
		filtered := measure.FilterEntriesByPeriod(entries, period)
		report := measure.GenerateReport(filtered, period)

		recent := filtered
		if len(recent) > recentCount {
			recent = recent[len(recent)-recentCount:]
		}
		return dataMsg{report: report, recent: recent}
	}
}

// Update is part of tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "1":
			m.period = "today"
			return m, loadCmd(m.logPath, m.period)
		case "7":
			m.period = "7d"
			return m, loadCmd(m.logPath, m.period)
		case "3", "0":
			m.period = "30d"
			return m, loadCmd(m.logPath, m.period)
		case "a":
			m.period = "all"
			return m, loadCmd(m.logPath, m.period)
		case "r":
			return m, tea.Batch(loadCmd(m.logPath, m.period), loadLearnCmd(m.storePath))
		}

	case dataMsg:
		m.report = msg.report
		m.recent = msg.recent
		m.err = msg.err
		return m, nil

	case learnMsg:
		m.learn = msg.result
		m.learnErr = msg.err
		return m, nil

	case tickMsg:
		return m, tea.Batch(loadCmd(m.logPath, m.period), loadLearnCmd(m.storePath), tickCmd())
	}
	return m, nil
}

// styling
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	activeChip  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	dimChip     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	paneBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	headerStyle = lipgloss.NewStyle().Padding(0, 1)
)

// View is part of tea.Model.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("gotk dashboard: %v\n\nq to quit", m.err)
	}
	if m.report.TotalInvocations == 0 {
		return renderEmpty(m)
	}

	header := renderHeader(m.report, m.period)
	top := renderTopCommands(m.report)
	recent := renderRecent(m.recent)
	insights := renderInsights(m.report.Insights)
	learnPane := renderLearn(m.learn, m.learnErr)

	// Two-column body: top on the left, recent on the right.
	body := lipgloss.JoinHorizontal(lipgloss.Top, top, recent)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, insights, learnPane)
}

func renderEmpty(m Model) string {
	header := renderHeader(m.report, m.period)
	msg := fmt.Sprintf("No measurement entries in period %q.\nRun some commands through gotk, then press 'r' to refresh.\n\n%s",
		m.period, footerHint())
	return lipgloss.JoinVertical(lipgloss.Left, header, headerStyle.Render(msg))
}

func renderHeader(r measure.Report, period string) string {
	totals := fmt.Sprintf("%s   %s   %s   %s",
		labelStyle.Render("Tokens saved:")+" "+titleStyle.Render(formatInt(r.TotalTokensSaved)),
		labelStyle.Render("Invocations:")+" "+titleStyle.Render(fmt.Sprintf("%d", r.TotalInvocations)),
		labelStyle.Render("Avg reduction:")+" "+titleStyle.Render(fmt.Sprintf("%.1f%%", r.AvgReduction)),
		labelStyle.Render("Avg quality:")+" "+titleStyle.Render(fmt.Sprintf("%.1f%%", r.AvgQualityScore)),
	)
	chips := renderPeriodChips(period)
	hint := dimChip.Render(footerHint())
	return headerStyle.Render(strings.Join([]string{titleStyle.Render("gotk dashboard"), totals, chips, hint}, "\n"))
}

func renderPeriodChips(active string) string {
	var parts []string
	for _, p := range periods {
		key := keyForPeriod(p)
		label := fmt.Sprintf("[%s] %s", key, p)
		if p == active {
			parts = append(parts, activeChip.Render(label+" *"))
		} else {
			parts = append(parts, dimChip.Render(label))
		}
	}
	return labelStyle.Render("Period: ") + strings.Join(parts, "  ")
}

func keyForPeriod(p string) string {
	switch p {
	case "today":
		return "1"
	case "7d":
		return "7"
	case "30d":
		return "3"
	case "all":
		return "a"
	}
	return "?"
}

func footerHint() string {
	return "1/7/3/a switch period · r refresh · q quit"
}

func renderTopCommands(r measure.Report) string {
	rows := sortedCommandTypes(r.ByCommandType)
	var b strings.Builder
	b.WriteString(titleStyle.Render("Top commands by tokens saved"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%-12s %7s %10s %8s\n", "TYPE", "COUNT", "SAVED", "REDUC")
	if len(rows) == 0 {
		b.WriteString(labelStyle.Render("(no data)"))
	}
	for _, row := range rows {
		fmt.Fprintf(&b, "%-12s %7d %10s %7.1f%%\n",
			truncate(row.name, 12), row.stats.Invocations, formatInt(row.stats.TokensSaved), row.stats.AvgReduction)
	}
	return paneBorder.Width(46).Render(b.String())
}

type cmdRow struct {
	name  string
	stats *measure.CommandTypeStats
}

func sortedCommandTypes(m map[string]*measure.CommandTypeStats) []cmdRow {
	rows := make([]cmdRow, 0, len(m))
	for name, s := range m {
		rows = append(rows, cmdRow{name: name, stats: s})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].stats.TokensSaved != rows[j].stats.TokensSaved {
			return rows[i].stats.TokensSaved > rows[j].stats.TokensSaved
		}
		return rows[i].name < rows[j].name
	})
	if len(rows) > 8 {
		rows = rows[:8]
	}
	return rows
}

func renderRecent(entries []measure.Entry) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Recent invocations"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%-5s %-22s %6s %6s\n", "TIME", "COMMAND", "REDUC", "QUAL")
	if len(entries) == 0 {
		b.WriteString(labelStyle.Render("(no data)"))
	}
	// Most recent first.
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		timeStr := "??:??"
		if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
			timeStr = t.Local().Format("15:04")
		}
		cmd := truncate(e.Command, 22)
		fmt.Fprintf(&b, "%-5s %-22s %5.1f%% %5.1f%%\n",
			timeStr, cmd, e.ReductionPct, e.QualityScore)
	}
	return paneBorder.Width(46).Render(b.String())
}

func renderInsights(ins []measure.Insight) string {
	if len(ins) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Insights"))
	b.WriteString("\n")
	for _, x := range ins {
		prefix := "  [i]"
		style := infoStyle
		if x.Level == "warning" {
			prefix = "  [!]"
			style = warnStyle
		}
		b.WriteString(style.Render(prefix+" "+x.Message) + "\n")
	}
	return paneBorder.Render(b.String())
}

// renderLearn renders the learn-patterns pane. Three states:
//   - storeErr non-nil: render the error tersely (read failure)
//   - res nil: store missing or empty (most users) → friendly hint
//   - res.NotReady: not enough sessions yet → show the analyzer's own message
//   - otherwise: top candidates ranked by frequency
func renderLearn(res *learn.AnalysisResult, storeErr error) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Learn patterns"))
	b.WriteString("\n")

	if storeErr != nil {
		b.WriteString(warnStyle.Render("  [!] cannot read learn store: " + storeErr.Error()))
		return paneBorder.Render(b.String())
	}
	if res == nil {
		b.WriteString(labelStyle.Render("  (no observations yet — run `gotk --learn <cmd>` to collect samples)"))
		return paneBorder.Render(b.String())
	}
	if res.NotReady {
		b.WriteString(labelStyle.Render("  " + res.NotReadyMsg))
		return paneBorder.Render(b.String())
	}

	fmt.Fprintf(&b, "%s %d observations across %d sessions\n",
		labelStyle.Render("  store:"), res.TotalLines, res.Sessions)

	if len(res.Candidates) == 0 {
		b.WriteString(labelStyle.Render("  (no strong candidates yet — keep collecting)"))
		return paneBorder.Render(b.String())
	}

	fmt.Fprintf(&b, "  %-32s %6s %6s %5s\n", "PATTERN", "FREQ", "NOISE", "HITS")
	limit := min(len(res.Candidates), 5)
	for _, c := range res.Candidates[:limit] {
		fmt.Fprintf(&b, "  %-32s %5.1f%% %5.1f%% %5d\n",
			truncate(c.Description, 32), c.Frequency*100, c.NoiseScore*100, c.MatchCount)
	}
	return paneBorder.Render(b.String())
}

// formatInt renders an integer with thousands separators.
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// Run launches the dashboard. Blocks until the user quits.
func Run(logPath string) error {
	p := tea.NewProgram(New(logPath), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
