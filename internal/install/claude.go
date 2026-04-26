// Package install handles auto-configuration of GoTK integrations.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// buildHookCmd builds the Claude Code hook command string from a binary
// path. Paths containing spaces (typical on Windows when gotk is installed
// under C:\Program Files\...) are wrapped in double quotes so the shell
// invoked by Claude Code parses the argv correctly. Double quotes work in
// both POSIX shells and cmd.exe / PowerShell.
func buildHookCmd(gotkPath string) string {
	if strings.ContainsRune(gotkPath, ' ') {
		return `"` + gotkPath + `" hook`
	}
	return gotkPath + " hook"
}

// Scope determines where the hook is installed.
//
// Claude Code resolves settings in three layers, from highest to lowest
// precedence: enterprise policy → user (~/.claude/settings.json) →
// project-shared (<project>/.claude/settings.json) → project-local
// (<project>/.claude/settings.local.json). The project-local file is
// expected to be gitignored and carries personal overrides.
type Scope int

const (
	// ScopeLocal writes to <project>/.claude/settings.local.json — the
	// gitignored, personal-override file. This is the default because it
	// is the safest: it never ends up in a teammate's diff, and it matches
	// Claude Code's own default for interactive settings changes.
	ScopeLocal Scope = iota
	// ScopeProject writes to <project>/.claude/settings.json — the shared,
	// commit-to-git variant. Use when every contributor on the project
	// should get the hook.
	ScopeProject
	// ScopeGlobal writes to ~/.claude/settings.json — applied across all
	// projects for this user.
	ScopeGlobal
)

// ClaudeInstall configures GoTK as a Claude Code PreToolUse hook.
func ClaudeInstall(scope Scope) error {
	gotkPath, err := findGotkPath()
	if err != nil {
		return fmt.Errorf("cannot locate gotk binary: %w", err)
	}
	settingsPath, err := settingsFilePath(scope)
	if err != nil {
		return err
	}
	return claudeInstallAt(settingsPath, gotkPath)
}

// claudeInstallAt is the core install logic, testable with arbitrary paths.
func claudeInstallAt(settingsPath, gotkPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	hookCmd := buildHookCmd(gotkPath)

	// Check if already installed
	if isHookInstalled(settings, hookCmd) {
		fmt.Fprintf(os.Stderr, "GoTK hook already configured in %s\n", settingsPath)
		return nil
	}

	// Add the hook
	addHook(settings, hookCmd)

	if err := writeSettings(settingsPath, settings); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "GoTK hook installed in %s\n", settingsPath)
	fmt.Fprintf(os.Stderr, "\nClaude Code will now filter all Bash output through GoTK.\n")
	fmt.Fprintf(os.Stderr, "To uninstall: gotk install claude --uninstall\n")
	return nil
}

// ClaudeUninstall removes the GoTK hook from Claude Code settings.
func ClaudeUninstall(scope Scope) error {
	gotkPath, err := findGotkPath()
	if err != nil {
		return fmt.Errorf("cannot locate gotk binary: %w", err)
	}
	settingsPath, err := settingsFilePath(scope)
	if err != nil {
		return err
	}
	return claudeUninstallAt(settingsPath, gotkPath)
}

// claudeUninstallAt is the core uninstall logic, testable with arbitrary paths.
func claudeUninstallAt(settingsPath, gotkPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	hookCmd := buildHookCmd(gotkPath)

	if removeHook(settings, hookCmd) {
		if err := writeSettings(settingsPath, settings); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "GoTK hook removed from %s\n", settingsPath)
	} else {
		fmt.Fprintf(os.Stderr, "GoTK hook not found in %s\n", settingsPath)
	}
	return nil
}

// ClaudeStatus checks if GoTK is installed as a Claude Code hook.
func ClaudeStatus(scope Scope) error {
	gotkPath, err := findGotkPath()
	if err != nil {
		return fmt.Errorf("cannot locate gotk binary: %w", err)
	}
	settingsPath, err := settingsFilePath(scope)
	if err != nil {
		return err
	}
	return claudeStatusAt(settingsPath, gotkPath)
}

// claudeStatusAt is the core status logic, testable with arbitrary paths.
func claudeStatusAt(settingsPath, gotkPath string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Settings file: %s (not found)\n", settingsPath)
		fmt.Fprintf(os.Stderr, "GoTK hook: not installed\n")
		return nil
	}

	hookCmd := buildHookCmd(gotkPath)
	if isHookInstalled(settings, hookCmd) {
		fmt.Fprintf(os.Stderr, "Settings file: %s\n", settingsPath)
		fmt.Fprintf(os.Stderr, "GoTK hook: installed\n")
	} else {
		fmt.Fprintf(os.Stderr, "Settings file: %s\n", settingsPath)
		fmt.Fprintf(os.Stderr, "GoTK hook: not installed\n")
	}
	return nil
}

func findGotkPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

func settingsFilePath(scope Scope) (string, error) {
	switch scope {
	case ScopeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".claude", "settings.json"), nil
	case ScopeProject:
		return filepath.Join(".claude", "settings.json"), nil
	case ScopeLocal:
		return filepath.Join(".claude", "settings.local.json"), nil
	default:
		return "", fmt.Errorf("unknown scope: %d", scope)
	}
}

func readSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func isHookInstalled(settings map[string]interface{}, hookCmd string) bool {
	hooks, ok := settings["hooks"]
	if !ok {
		return false
	}
	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		return false
	}
	preToolUse, ok := hooksMap["PreToolUse"]
	if !ok {
		return false
	}
	groups, ok := preToolUse.([]interface{})
	if !ok {
		return false
	}

	for _, g := range groups {
		group, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		hooksList, ok := group["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooksList {
			hook, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := hook["command"].(string); ok && cmd == hookCmd {
				return true
			}
		}
	}
	return false
}

func addHook(settings map[string]interface{}, hookCmd string) {
	hooks, ok := settings["hooks"]
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}
	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		hooksMap = make(map[string]interface{})
		settings["hooks"] = hooksMap
	}

	newGroup := map[string]interface{}{
		"matcher": "Bash",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookCmd,
			},
		},
	}

	preToolUse, ok := hooksMap["PreToolUse"]
	if !ok {
		hooksMap["PreToolUse"] = []interface{}{newGroup}
		return
	}
	groups, ok := preToolUse.([]interface{})
	if !ok {
		hooksMap["PreToolUse"] = []interface{}{newGroup}
		return
	}
	hooksMap["PreToolUse"] = append(groups, newGroup)
}

func removeHook(settings map[string]interface{}, hookCmd string) bool {
	hooks, ok := settings["hooks"]
	if !ok {
		return false
	}
	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		return false
	}
	preToolUse, ok := hooksMap["PreToolUse"]
	if !ok {
		return false
	}
	groups, ok := preToolUse.([]interface{})
	if !ok {
		return false
	}

	removed := false
	var remaining []interface{}
	for _, g := range groups {
		group, ok := g.(map[string]interface{})
		if !ok {
			remaining = append(remaining, g)
			continue
		}
		hooksList, ok := group["hooks"].([]interface{})
		if !ok {
			remaining = append(remaining, g)
			continue
		}

		var keptHooks []interface{}
		for _, h := range hooksList {
			hook, ok := h.(map[string]interface{})
			if !ok {
				keptHooks = append(keptHooks, h)
				continue
			}
			if cmd, ok := hook["command"].(string); ok && cmd == hookCmd {
				removed = true
				continue
			}
			keptHooks = append(keptHooks, h)
		}

		if len(keptHooks) > 0 {
			group["hooks"] = keptHooks
			remaining = append(remaining, group)
		}
	}

	if len(remaining) == 0 {
		delete(hooksMap, "PreToolUse")
		if len(hooksMap) == 0 {
			delete(settings, "hooks")
		}
	} else {
		hooksMap["PreToolUse"] = remaining
	}

	return removed
}
