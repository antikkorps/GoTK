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
	os.WriteFile(settingsPath, data, 0644)

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
