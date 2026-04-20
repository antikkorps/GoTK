package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	home, err := os.UserHomeDir()
	if err == nil {
		// Config file (global, under XDG_CONFIG_HOME-style path).
		cfgFile := filepath.Join(home, ".config", "gotk", "config.toml")
		if fileExists(cfgFile) {
			plan.ConfigFiles = append(plan.ConfigFiles, cfgFile)
		}
		// Measurement log (XDG_DATA_HOME-style path).
		measureLog := filepath.Join(home, ".local", "share", "gotk", "measure.jsonl")
		if fileExists(measureLog) {
			plan.ConfigFiles = append(plan.ConfigFiles, measureLog)
		}
		// Their parent directories — only if they are GoTK-specific and
		// will end up empty after the files above are removed.
		for _, dir := range []string{
			filepath.Join(home, ".config", "gotk"),
			filepath.Join(home, ".local", "share", "gotk"),
		} {
			if dirExists(dir) {
				plan.ConfigDirs = append(plan.ConfigDirs, dir)
			}
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

// PrintPlan writes a human-readable summary of what the plan will do.
// It reports absent Claude hooks so the user can see we checked.
func PrintPlan(w io.Writer, plan *UninstallPlan) {
	fmt.Fprintln(w, "This will remove GoTK integrations from your system:")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "  Claude Code hooks:")
	for _, h := range plan.ClaudeHooks {
		mark := "[ ]"
		note := "(no hook found)"
		if h.Present {
			mark = "[x]"
			note = ""
		}
		label := scopeLabel(h.Scope)
		fmt.Fprintf(w, "    %s %s  %s %s\n", mark, label, h.Path, note)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  GoTK config / data files:")
	if len(plan.ConfigFiles) == 0 {
		fmt.Fprintln(w, "    (none found)")
	}
	for _, f := range plan.ConfigFiles {
		fmt.Fprintf(w, "    [x] %s\n", f)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Binary (NOT removed automatically — gotk can't delete itself while running):")
	fmt.Fprintf(w, "    %s\n", plan.BinaryPath)
}

// PrintResult writes a summary of what ExecuteUninstall actually did.
// It closes with the exact command the user needs to run to remove the
// binary, so nothing is left implicit.
func PrintResult(w io.Writer, plan *UninstallPlan, res *UninstallResult) {
	for _, p := range res.RemovedHooks {
		fmt.Fprintf(w, "Removed Claude hook from %s\n", p)
	}
	for _, p := range res.RemovedFiles {
		fmt.Fprintf(w, "Removed %s\n", p)
	}
	for _, p := range res.RemovedDirs {
		fmt.Fprintf(w, "Removed directory %s\n", p)
	}
	for _, err := range res.Errors {
		fmt.Fprintf(w, "error: %v\n", err)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Binary was not removed. Run this to finish the uninstall:")
	fmt.Fprintln(w)
	// Binaries under /usr/local/bin/ (or any root-owned path) need sudo;
	// otherwise a plain rm suffices. We test writability of the parent
	// directory as a proxy — good enough for a hint.
	prefix := ""
	if !parentWritable(plan.BinaryPath) {
		prefix = "sudo "
	}
	fmt.Fprintf(w, "  %srm %s\n", prefix, plan.BinaryPath)
}

// Confirm reads a yes/no answer from r. Empty input is treated as "no"
// so that accidental <enter> keeps the user's system intact.
func Confirm(r io.Reader, w io.Writer, prompt string) (bool, error) {
	fmt.Fprintf(w, "%s [y/N]: ", prompt)
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
