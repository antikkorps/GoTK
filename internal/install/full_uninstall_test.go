package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanUninstall_DetectsPresentHooks(t *testing.T) {
	// Set up a fake HOME that contains a Claude settings file with a gotk
	// hook already installed, plus GoTK config files.
	home := t.TempDir()
	t.Setenv("HOME", home)

	globalSettings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(globalSettings), 0755); err != nil {
		t.Fatal(err)
	}
	settings := make(map[string]interface{})
	// findGotkPath resolves os.Executable(); whatever the test binary is,
	// PlanUninstall will look for "<testbin> hook". Install the same hookCmd
	// so the plan flags the global scope as Present.
	exe, err := findGotkPath()
	if err != nil {
		t.Fatal(err)
	}
	hookCmd := exe + " hook"
	addHook(settings, hookCmd)
	if err := writeSettings(globalSettings, settings); err != nil {
		t.Fatal(err)
	}

	// Config file + measure log.
	cfgPath := filepath.Join(home, ".config", "gotk", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("# test"), 0600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(home, ".local", "share", "gotk", "measure.jsonl")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanUninstall()
	if err != nil {
		t.Fatalf("PlanUninstall: %v", err)
	}

	var globalPresent bool
	for _, h := range plan.ClaudeHooks {
		if h.Scope == ScopeGlobal {
			globalPresent = h.Present
		}
	}
	if !globalPresent {
		t.Errorf("global scope should be flagged Present, got plan: %+v", plan.ClaudeHooks)
	}

	if !contains(plan.ConfigFiles, cfgPath) {
		t.Errorf("config.toml should be in plan, got: %v", plan.ConfigFiles)
	}
	if !contains(plan.ConfigFiles, logPath) {
		t.Errorf("measure.jsonl should be in plan, got: %v", plan.ConfigFiles)
	}
}

func TestExecuteUninstall_RemovesOnlyPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	exe, err := findGotkPath()
	if err != nil {
		t.Fatal(err)
	}
	hookCmd := exe + " hook"

	// Only install in ScopeLocal; the other two scopes should be Absent
	// and ExecuteUninstall should skip them cleanly.
	localPath, err := settingsFilePath(ScopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	// Run the test from a temp cwd so the local settings file lives somewhere
	// we control.
	oldCwd, _ := os.Getwd()
	project := t.TempDir()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd) //nolint:errcheck

	localPath, err = settingsFilePath(ScopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	settings := make(map[string]interface{})
	addHook(settings, hookCmd)
	if err := writeSettings(localPath, settings); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanUninstall()
	if err != nil {
		t.Fatal(err)
	}
	res := ExecuteUninstall(plan)

	if len(res.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", res.Errors)
	}
	if !contains(res.RemovedHooks, localPath) {
		t.Errorf("local hook should be removed, got: %v", res.RemovedHooks)
	}

	// The hook should actually be gone from the file.
	after, err := readSettings(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if isHookInstalled(after, hookCmd) {
		t.Errorf("hook still installed after uninstall")
	}
}

func TestExecuteUninstall_RemovesConfigFilesAndDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgPath := filepath.Join(home, ".config", "gotk", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("# test"), 0600); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanUninstall()
	if err != nil {
		t.Fatal(err)
	}
	res := ExecuteUninstall(plan)

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config.toml should be removed, got stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(cfgPath)); !os.IsNotExist(err) {
		t.Errorf("empty config dir should be removed, got stat err: %v", err)
	}
	if len(res.RemovedFiles) == 0 {
		t.Errorf("RemovedFiles should not be empty")
	}
}

func TestExecuteUninstall_LeavesNonEmptyConfigDir(t *testing.T) {
	// If the user has other files in ~/.config/gotk/ that we don't manage,
	// the uninstall must not remove the directory itself.
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "gotk")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte("# test"), 0600); err != nil {
		t.Fatal(err)
	}
	// Drop an unrelated file we should keep.
	strangerPath := filepath.Join(cfgDir, "user-notes.md")
	if err := os.WriteFile(strangerPath, []byte("personal notes"), 0600); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanUninstall()
	if err != nil {
		t.Fatal(err)
	}
	ExecuteUninstall(plan)

	// config.toml goes away but the dir survives because of user-notes.md.
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config.toml should be removed")
	}
	if _, err := os.Stat(cfgDir); os.IsNotExist(err) {
		t.Errorf("non-empty config dir should NOT be removed")
	}
	if _, err := os.Stat(strangerPath); os.IsNotExist(err) {
		t.Errorf("unrelated file should survive uninstall")
	}
}

func TestConfirm(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"no\n", false},
		{"\n", false}, // empty = default no
		{"maybe\n", false},
	}
	for _, tc := range cases {
		t.Run(strings.TrimSpace(tc.in), func(t *testing.T) {
			var out bytes.Buffer
			got, err := Confirm(strings.NewReader(tc.in), &out, "?")
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("Confirm(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestPrintPlan_ReportsAllScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	plan, err := PlanUninstall()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	PrintPlan(&out, plan)
	s := out.String()
	for _, label := range []string{"local", "project", "global", "Binary"} {
		if !strings.Contains(s, label) {
			t.Errorf("PrintPlan output missing %q, got:\n%s", label, s)
		}
	}
}

func TestPrintResult_IncludesRemovalCommand(t *testing.T) {
	plan := &UninstallPlan{BinaryPath: "/tmp/gotk"}
	res := &UninstallResult{}
	var out bytes.Buffer
	PrintResult(&out, plan, res)
	if !strings.Contains(out.String(), "rm /tmp/gotk") {
		t.Errorf("expected rm command in output, got:\n%s", out.String())
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
