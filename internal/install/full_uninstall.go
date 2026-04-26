package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/antikkorps/GoTK/internal/paths"
)

// UninstallPlan describes every GoTK artifact we can remove — the binary
// path, all three Claude hook scopes, and the user-level config/data
// directories. Callers can inspect it before any filesystem write happens.
//
// The plan is deliberately read-only: construction never mutates the
// system. Executing it lives in ExecuteUninstall so CLI consumers can
// render a summary, prompt the user, and then commit.
type UninstallPlan struct {
	BinaryPath  string
	ClaudeHooks []ClaudeHookTarget
	ConfigFiles []string
	ConfigDirs  []string
}

// ClaudeHookTarget records one Claude Code settings file we might touch.
// Present is false when either the file doesn't exist or it doesn't carry
// a GoTK hook; in that case ExecuteUninstall skips it silently.
type ClaudeHookTarget struct {
	Scope   Scope
	Path    string
	Present bool
}

// PlanUninstall inspects the system and returns an UninstallPlan.
// It never writes to disk.
func PlanUninstall() (*UninstallPlan, error) {
	binaryPath, err := findGotkPath()
	if err != nil {
		return nil, fmt.Errorf("locating gotk binary: %w", err)
	}

	plan := &UninstallPlan{BinaryPath: binaryPath}

	hookCmd := binaryPath + " hook"
	for _, scope := range []Scope{ScopeLocal, ScopeProject, ScopeGlobal} {
		path, err := settingsFilePath(scope)
		if err != nil {
			continue
		}
		target := ClaudeHookTarget{Scope: scope, Path: path}
		if settings, err := readSettings(path); err == nil && isHookInstalled(settings, hookCmd) {
			target.Present = true
		}
		plan.ClaudeHooks = append(plan.ClaudeHooks, target)
	}

	if cfgFile, ok := paths.ConfigFile(); ok && fileExists(cfgFile) {
		plan.ConfigFiles = append(plan.ConfigFiles, cfgFile)
	}
	if dataDir, ok := paths.DataDir(); ok {
		measureLog := filepath.Join(dataDir, "measure.jsonl")
		if fileExists(measureLog) {
			plan.ConfigFiles = append(plan.ConfigFiles, measureLog)
		}
	}
	// Parent directories — only if they are gotk-specific and will end up
	// empty after the files above are removed. On Windows, ConfigDir and
	// DataDir collapse to the same %AppData%/gotk path; the dedup happens
	// implicitly because dirExists guards the second append.
	seen := map[string]bool{}
	for _, fn := range []func() (string, bool){paths.ConfigDir, paths.DataDir} {
		if dir, ok := fn(); ok && !seen[dir] && dirExists(dir) {
			plan.ConfigDirs = append(plan.ConfigDirs, dir)
			seen[dir] = true
		}
	}

	return plan, nil
}

// UninstallResult summarizes what ExecuteUninstall actually did.
// Callers use this to print a success report to the user.
type UninstallResult struct {
	RemovedHooks []string
	RemovedFiles []string
	RemovedDirs  []string
	SkippedHooks []string
	Errors       []error
}

// ExecuteUninstall applies the plan: removes the Claude hook from each
// target that has one, deletes the config files, and removes any now-empty
// GoTK config directories.
//
// It NEVER touches the binary itself — gotk cannot delete its own running
// executable reliably across platforms. Callers must print the explicit
// `rm` command for the user to run afterwards.
func ExecuteUninstall(plan *UninstallPlan) *UninstallResult {
	res := &UninstallResult{}

	for _, target := range plan.ClaudeHooks {
		if !target.Present {
			res.SkippedHooks = append(res.SkippedHooks, target.Path)
			continue
		}
		settings, err := readSettings(target.Path)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("reading %s: %w", target.Path, err))
			continue
		}
		if removeHook(settings, plan.BinaryPath+" hook") {
			if err := writeSettings(target.Path, settings); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("writing %s: %w", target.Path, err))
				continue
			}
			res.RemovedHooks = append(res.RemovedHooks, target.Path)
		}
	}

	for _, f := range plan.ConfigFiles {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			res.Errors = append(res.Errors, fmt.Errorf("removing %s: %w", f, err))
			continue
		}
		res.RemovedFiles = append(res.RemovedFiles, f)
	}

	// Remove directories only if they are now empty. We never recurse —
	// the user may have their own files in there that we shouldn't touch.
	for _, d := range plan.ConfigDirs {
		if isDirEmpty(d) {
			if err := os.Remove(d); err != nil && !os.IsNotExist(err) {
				res.Errors = append(res.Errors, fmt.Errorf("removing directory %s: %w", d, err))
				continue
			}
			res.RemovedDirs = append(res.RemovedDirs, d)
		}
	}

	return res
}

// line prints an unformatted line + newline to w.
// Errors are intentionally discarded: these helpers render UI to stderr
// where there's nothing meaningful to recover from a write failure.
func line(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}

// linef prints a formatted line to w (no trailing newline unless the
// caller puts one in the format string).
func linef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// PrintPlan writes a human-readable summary of what the plan will do.
// It reports absent Claude hooks so the user can see we checked.
func PrintPlan(w io.Writer, plan *UninstallPlan) {
	line(w, "This will remove GoTK integrations from your system:")
	line(w)

	line(w, "  Claude Code hooks:")
	for _, h := range plan.ClaudeHooks {
		mark := "[ ]"
		note := "(no hook found)"
		if h.Present {
			mark = "[x]"
			note = ""
		}
		label := scopeLabel(h.Scope)
		linef(w, "    %s %s  %s %s\n", mark, label, h.Path, note)
	}

	line(w)
	line(w, "  GoTK config / data files:")
	if len(plan.ConfigFiles) == 0 {
		line(w, "    (none found)")
	}
	for _, f := range plan.ConfigFiles {
		linef(w, "    [x] %s\n", f)
	}

	line(w)
	line(w, "  Binary (NOT removed automatically — gotk can't delete itself while running):")
	linef(w, "    %s\n", plan.BinaryPath)
}

// PrintResult writes a summary of what ExecuteUninstall actually did.
// It closes with the exact command the user needs to run to remove the
// binary, so nothing is left implicit.
func PrintResult(w io.Writer, plan *UninstallPlan, res *UninstallResult) {
	for _, p := range res.RemovedHooks {
		linef(w, "Removed Claude hook from %s\n", p)
	}
	for _, p := range res.RemovedFiles {
		linef(w, "Removed %s\n", p)
	}
	for _, p := range res.RemovedDirs {
		linef(w, "Removed directory %s\n", p)
	}
	for _, err := range res.Errors {
		linef(w, "error: %v\n", err)
	}

	line(w)
	line(w, "Binary was not removed. Run this to finish the uninstall:")
	line(w)
	// Binaries under /usr/local/bin/ (or any root-owned path) need sudo;
	// otherwise a plain rm suffices. We test writability of the parent
	// directory as a proxy — good enough for a hint.
	prefix := ""
	if !parentWritable(plan.BinaryPath) {
		prefix = "sudo "
	}
	linef(w, "  %srm %s\n", prefix, plan.BinaryPath)
}

// Confirm reads a yes/no answer from r. Empty input is treated as "no"
// so that accidental <enter> keeps the user's system intact.
func Confirm(r io.Reader, w io.Writer, prompt string) (bool, error) {
	linef(w, "%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	ans := scanner.Text()
	return ans == "y" || ans == "Y" || ans == "yes" || ans == "YES", nil
}

func scopeLabel(s Scope) string {
	switch s {
	case ScopeLocal:
		return "local  "
	case ScopeProject:
		return "project"
	case ScopeGlobal:
		return "global "
	}
	return "?"
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isDirEmpty(p string) bool {
	entries, err := os.ReadDir(p)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// parentWritable reports whether the current process can create a file
// next to `path`. It's an approximation of "can I rm this without sudo?".
func parentWritable(path string) bool {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gotk-probe-*")
	if err != nil {
		return false
	}
	name := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(name)
	return true
}
