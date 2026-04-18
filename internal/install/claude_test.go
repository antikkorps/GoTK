package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeInstall_NewFile(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")

	settings := make(map[string]interface{})
	hookCmd := "/usr/local/bin/gotk hook"
	addHook(settings, hookCmd)

	if err := writeSettings(settingsPath, settings); err != nil {
		t.Fatal(err)
	}

	// Read back and verify
	got, err := readSettings(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	if !isHookInstalled(got, hookCmd) {
		t.Error("hook should be installed")
	}
}

func TestClaudeInstall_ExistingSettings(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Write existing settings
	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(grep:*)"},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(settingsPath, data, 0644) //nolint:errcheck

	// Add hook
	settings, err := readSettings(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	hookCmd := "/usr/local/bin/gotk hook"
	addHook(settings, hookCmd)

	if err := writeSettings(settingsPath, settings); err != nil {
		t.Fatal(err)
	}

	// Verify both permissions and hook exist
	got, err := readSettings(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	if !isHookInstalled(got, hookCmd) {
		t.Error("hook should be installed")
	}
	if _, ok := got["permissions"]; !ok {
		t.Error("existing permissions should be preserved")
	}
}

func TestClaudeUninstall(t *testing.T) {
	settings := make(map[string]interface{})
	hookCmd := "/usr/local/bin/gotk hook"
	addHook(settings, hookCmd)

	if !isHookInstalled(settings, hookCmd) {
		t.Fatal("hook should be installed before removal")
	}

	removed := removeHook(settings, hookCmd)
	if !removed {
		t.Error("removeHook should return true")
	}

	if isHookInstalled(settings, hookCmd) {
		t.Error("hook should be removed")
	}

	// hooks key should be cleaned up
	if _, ok := settings["hooks"]; ok {
		t.Error("empty hooks map should be removed")
	}
}

func TestClaudeInstall_Idempotent(t *testing.T) {
	settings := make(map[string]interface{})
	hookCmd := "/usr/local/bin/gotk hook"

	addHook(settings, hookCmd)
	if !isHookInstalled(settings, hookCmd) {
		t.Fatal("first install failed")
	}

	// isHookInstalled should prevent double install
	if !isHookInstalled(settings, hookCmd) {
		t.Error("hook should still be detected as installed")
	}
}

func TestRemoveHook_NotInstalled(t *testing.T) {
	settings := make(map[string]interface{})
	removed := removeHook(settings, "/usr/local/bin/gotk hook")
	if removed {
		t.Error("should return false when hook is not installed")
	}
}

func TestReadSettings_NotExist(t *testing.T) {
	settings, err := readSettings("/nonexistent/path/settings.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) != 0 {
		t.Error("should return empty map for nonexistent file")
	}
}

func TestFindGotkPath(t *testing.T) {
	path, err := findGotkPath()
	if err != nil {
		t.Fatalf("findGotkPath() error: %v", err)
	}
	if path == "" {
		t.Error("findGotkPath() returned empty path")
	}
}

func TestSettingsFilePath_Local(t *testing.T) {
	path, err := settingsFilePath(ScopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(".claude", "settings.local.json") {
		t.Errorf("expected .claude/settings.local.json, got %s", path)
	}
}

func TestSettingsFilePath_Project(t *testing.T) {
	path, err := settingsFilePath(ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(".claude", "settings.json") {
		t.Errorf("expected .claude/settings.json, got %s", path)
	}
}

func TestSettingsFilePath_Global(t *testing.T) {
	path, err := settingsFilePath(ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".claude", "settings.json")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestSettingsFilePath_InvalidScope(t *testing.T) {
	_, err := settingsFilePath(Scope(99))
	if err == nil {
		t.Error("invalid scope should return error")
	}
}

func TestReadSettings_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644) //nolint:errcheck

	_, err := readSettings(path)
	if err == nil {
		t.Error("invalid JSON should return error")
	}
}

func TestWriteSettings_Permissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "settings.json")

	settings := map[string]interface{}{"test": true}
	if err := writeSettings(path, settings); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("settings file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestIsHookInstalled_TypeMismatches(t *testing.T) {
	// hooks is not a map
	settings := map[string]interface{}{"hooks": "not-a-map"}
	if isHookInstalled(settings, "gotk hook") {
		t.Error("should return false for non-map hooks")
	}

	// PreToolUse is not an array
	settings = map[string]interface{}{
		"hooks": map[string]interface{}{"PreToolUse": "not-array"},
	}
	if isHookInstalled(settings, "gotk hook") {
		t.Error("should return false for non-array PreToolUse")
	}

	// Group items are not maps
	settings = map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{"not-a-map"},
		},
	}
	if isHookInstalled(settings, "gotk hook") {
		t.Error("should return false for non-map group")
	}

	// Hooks list items are not maps
	settings = map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{"not-a-map"},
				},
			},
		},
	}
	if isHookInstalled(settings, "gotk hook") {
		t.Error("should return false for non-map hook entry")
	}
}

func TestRemoveHook_TypeMismatches(t *testing.T) {
	// hooks is not a map
	settings := map[string]interface{}{"hooks": "not-a-map"}
	if removeHook(settings, "gotk hook") {
		t.Error("should return false for non-map hooks")
	}

	// PreToolUse is not an array
	settings = map[string]interface{}{
		"hooks": map[string]interface{}{"PreToolUse": "not-array"},
	}
	if removeHook(settings, "gotk hook") {
		t.Error("should return false for non-array PreToolUse")
	}

	// Group is not a map
	settings = map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{"not-a-map"},
		},
	}
	if removeHook(settings, "gotk hook") {
		t.Error("should return false for non-map group")
	}
}

func TestRemoveHook_KeepsOtherHooksInGroup(t *testing.T) {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "gotk hook"},
						map[string]interface{}{"type": "command", "command": "other-hook"},
					},
				},
			},
		},
	}

	removed := removeHook(settings, "gotk hook")
	if !removed {
		t.Error("should have removed gotk hook")
	}

	// The group should still exist with the other hook
	hooks := settings["hooks"].(map[string]interface{})
	groups := hooks["PreToolUse"].([]interface{})
	if len(groups) != 1 {
		t.Errorf("should keep 1 group, got %d", len(groups))
	}
}

func TestAddHook_NonMapHooks(t *testing.T) {
	// hooks key exists but is not a map
	settings := map[string]interface{}{"hooks": "broken"}
	addHook(settings, "gotk hook")

	if !isHookInstalled(settings, "gotk hook") {
		t.Error("addHook should overwrite non-map hooks")
	}
}

func TestAddHook_NonArrayPreToolUse(t *testing.T) {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{"PreToolUse": "broken"},
	}
	addHook(settings, "gotk hook")

	if !isHookInstalled(settings, "gotk hook") {
		t.Error("addHook should overwrite non-array PreToolUse")
	}
}

func TestClaudeInstallAt_FullFlow(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	gotkPath := "/usr/local/bin/gotk"

	// Fresh install
	if err := claudeInstallAt(settingsPath, gotkPath); err != nil {
		t.Fatalf("claudeInstallAt error: %v", err)
	}

	// Verify installed
	settings, _ := readSettings(settingsPath)
	if !isHookInstalled(settings, gotkPath+" hook") {
		t.Error("hook should be installed")
	}

	// Idempotent: second install should succeed without error
	if err := claudeInstallAt(settingsPath, gotkPath); err != nil {
		t.Fatalf("second claudeInstallAt error: %v", err)
	}
}

func TestClaudeUninstallAt_FullFlow(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	gotkPath := "/usr/local/bin/gotk"

	// Install first
	claudeInstallAt(settingsPath, gotkPath) //nolint:errcheck

	// Uninstall
	if err := claudeUninstallAt(settingsPath, gotkPath); err != nil {
		t.Fatalf("claudeUninstallAt error: %v", err)
	}

	// Verify removed
	settings, _ := readSettings(settingsPath)
	if isHookInstalled(settings, gotkPath+" hook") {
		t.Error("hook should be removed")
	}

	// Uninstall when not installed
	if err := claudeUninstallAt(settingsPath, gotkPath); err != nil {
		t.Fatalf("second claudeUninstallAt error: %v", err)
	}
}

func TestClaudeStatusAt_Installed(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	gotkPath := "/usr/local/bin/gotk"

	claudeInstallAt(settingsPath, gotkPath) //nolint:errcheck

	if err := claudeStatusAt(settingsPath, gotkPath); err != nil {
		t.Fatalf("claudeStatusAt error: %v", err)
	}
}

func TestClaudeStatusAt_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	gotkPath := "/usr/local/bin/gotk"

	if err := claudeStatusAt(settingsPath, gotkPath); err != nil {
		t.Fatalf("claudeStatusAt error: %v", err)
	}
}

func TestClaudeStatusAt_NoFile(t *testing.T) {
	if err := claudeStatusAt("/nonexistent/settings.json", "/usr/bin/gotk"); err != nil {
		t.Fatalf("claudeStatusAt with missing file error: %v", err)
	}
}

func TestAddHook_PreservesExistingHooks(t *testing.T) {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Edit",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "some-other-hook",
						},
					},
				},
			},
		},
	}

	hookCmd := "/usr/local/bin/gotk hook"
	addHook(settings, hookCmd)

	hooks := settings["hooks"].(map[string]interface{})
	groups := hooks["PreToolUse"].([]interface{})
	if len(groups) != 2 {
		t.Errorf("expected 2 hook groups, got %d", len(groups))
	}

	if !isHookInstalled(settings, hookCmd) {
		t.Error("gotk hook should be installed")
	}
}
