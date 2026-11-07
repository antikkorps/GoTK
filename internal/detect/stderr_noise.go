package detect

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/antikkorps/GoTK/internal/filter"
)

// Stderr carries diagnostics, so the stderr chain never runs the stdout
// command filters. Some tools still print pure progress noise there (cargo
// writes everything to stderr, npm writes its npmrc warnings there). These
// filters are strict allowlists: a line is touched only if it matches a known
// noise shape exactly; everything else passes through byte-for-byte.
var stderrNoise = map[CmdType]filter.FilterFunc{
	CmdCargo: collapseCargoProgress,
	CmdNpm:   collapseNpmConfigWarnings,
	CmdNode:  collapseNpmConfigWarnings,
}

// StderrNoiseFilter returns the stderr noise allowlist filter for cmdType,
// or nil when the command has none.
func StderrNoiseFilter(cmdType CmdType) filter.FilterFunc {
	return stderrNoise[cmdType]
}

// Cargo right-aligns its status verb after leading spaces; requiring that
// indentation keeps "error: Compiling ..." style diagnostics out of reach.
var cargoProgressPattern = regexp.MustCompile(
	`^(\s+)(Compiling|Checking|Downloaded|Downloading|Fresh|Blocking) \S`)

// progressRun accumulates a run of consecutive cargo progress lines.
type progressRun struct {
	indent string
	lines  []string
	verbs  []string
	counts map[string]int
}

func (r *progressRun) add(line, indent, verb string) {
	if len(r.lines) == 0 {
		r.indent, r.counts = indent, map[string]int{}
	}
	if r.counts[verb] == 0 {
		r.verbs = append(r.verbs, verb)
	}
	r.counts[verb]++
	r.lines = append(r.lines, line)
}

// flush appends the run to out: verbatim when it is a single line (nothing
// to gain), otherwise as one summary line. It then resets the run.
func (r *progressRun) flush(out []string) []string {
	switch len(r.lines) {
	case 0:
		return out
	case 1:
		out = append(out, r.lines[0])
	default:
		parts := make([]string, len(r.verbs))
		for i, v := range r.verbs {
			parts[i] = fmt.Sprintf("%s %d", v, r.counts[v])
		}
		out = append(out, fmt.Sprintf("%s... %d cargo progress lines (%s)",
			r.indent, len(r.lines), strings.Join(parts, ", ")))
	}
	*r = progressRun{}
	return out
}

// collapseCargoProgress replaces each run of consecutive cargo progress
// lines with a count, keeping runs separate so warnings and errors stay in
// chronological position.
func collapseCargoProgress(input string) string {
	var out []string
	var run progressRun
	for _, line := range strings.Split(input, "\n") {
		if m := cargoProgressPattern.FindStringSubmatch(line); m != nil {
			run.add(line, m[1], m[2])
			continue
		}
		out = run.flush(out)
		out = append(out, line)
	}
	out = run.flush(out)
	return strings.Join(out, "\n")
}

// The full boilerplate is required: if npm ever changes the wording, the
// line may carry new information and is left alone.
var npmUnknownConfigPattern = regexp.MustCompile(
	`^npm warn Unknown (\S+) config "([^"]*)"\. This will stop working in the next major version of npm\.( See ` +
		"`npm help npmrc`" + ` for supported config options\.)?$`)

// npmConfigScope gathers the unknown-config warnings of one scope.
type npmConfigScope struct {
	keys  []string // distinct, in first-seen order
	lines int
}

// scanNpmConfigWarnings indexes the warnings by scope. npm re-prints the
// whole block in every process, so keys are de-duplicated.
func scanNpmConfigWarnings(lines []string) map[string]*npmConfigScope {
	scopes := map[string]*npmConfigScope{}
	for _, line := range lines {
		m := npmUnknownConfigPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		sc := scopes[m[1]]
		if sc == nil {
			sc = &npmConfigScope{}
			scopes[m[1]] = sc
		}
		sc.lines++
		if key := `"` + m[2] + `"`; !slices.Contains(sc.keys, key) {
			sc.keys = append(sc.keys, key)
		}
	}
	return scopes
}

// collapseNpmConfigWarnings merges the "Unknown <scope> config" warnings of
// each scope into one line listing the keys. The line sits at the first
// occurrence, so later repeats of the same scope move up: acceptable for
// config noise that carries no chronology. A scope with a single warning
// keeps its original line.
func collapseNpmConfigWarnings(input string) string {
	lines := strings.Split(input, "\n")
	scopes := scanNpmConfigWarnings(lines)
	out := make([]string, 0, len(lines))
	emitted := map[string]bool{}
	for _, line := range lines {
		m := npmUnknownConfigPattern.FindStringSubmatch(line)
		if m == nil || scopes[m[1]].lines == 1 {
			out = append(out, line)
			continue
		}
		if !emitted[m[1]] {
			emitted[m[1]] = true
			out = append(out, fmt.Sprintf("npm warn Unknown %s config %s (ignored by npm; see `npm help npmrc`)",
				m[1], strings.Join(scopes[m[1]].keys, ", ")))
		}
	}
	return strings.Join(out, "\n")
}
