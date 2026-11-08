package install

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstallInstructions_CreatesBothFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".claude")
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "GOTK.md")); got != instructionsContent {
		t.Errorf("GOTK.md content mismatch:\n%s", got)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != "@GOTK.md\n" {
		t.Errorf("CLAUDE.md = %q, want the import line only", got)
	}
}

func TestInstallInstructions_PrependsImportAndKeepsContent(t *testing.T) {
	dir := t.TempDir()
	user := "# My rules\n\n- be terse\n"
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), user)

	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	want := "@GOTK.md\n\n" + user
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != want {
		t.Errorf("CLAUDE.md:\n got %q\nwant %q", got, want)
	}
}

func TestInstallInstructions_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "@RTK.md\n\n# Rules\n")
	for range 2 {
		if err := installInstructionsAt(dir); err != nil {
			t.Fatal(err)
		}
	}
	got := readFile(t, filepath.Join(dir, "CLAUDE.md"))
	if n := strings.Count(got, "@GOTK.md"); n != 1 {
		t.Errorf("import appears %d times:\n%s", n, got)
	}
}

func TestInstallInstructions_KeepsImportPlacedByUser(t *testing.T) {
	// A user who moved the import further down chose that spot: leave it.
	dir := t.TempDir()
	user := "# Rules\n\n@GOTK.md\n"
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), user)
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != user {
		t.Errorf("CLAUDE.md changed:\n%s", got)
	}
}

func TestInstallInstructions_RefreshesStaleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "GOTK.md"), "old text\n")
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "GOTK.md")); got != instructionsContent {
		t.Errorf("GOTK.md not refreshed:\n%s", got)
	}
}

func TestUninstallInstructions_RestoresUserFile(t *testing.T) {
	dir := t.TempDir()
	user := "# My rules\n\n- be terse\n"
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), user)
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}

	changed, err := uninstallInstructionsAt(dir)
	if err != nil || len(changed) == 0 {
		t.Fatalf("uninstall: changed=%v err=%v", changed, err)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != user {
		t.Errorf("CLAUDE.md:\n got %q\nwant %q", got, user)
	}
	if _, err := os.Stat(filepath.Join(dir, "GOTK.md")); !os.IsNotExist(err) {
		t.Errorf("GOTK.md still present: %v", err)
	}
}

func TestUninstallInstructions_DeletesFileItCreated(t *testing.T) {
	dir := t.TempDir()
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstallInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("empty CLAUDE.md left behind: %v", err)
	}
}

func TestUninstallInstructions_HandlesCRLF(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "@GOTK.md\r\n\r\n# Rules\r\n")
	if _, err := uninstallInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != "# Rules\r\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
}

func TestUninstallInstructions_NothingToRemove(t *testing.T) {
	dir := t.TempDir()
	user := "# Rules\n"
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), user)
	changed, err := uninstallInstructionsAt(dir)
	if err != nil || len(changed) != 0 {
		t.Fatalf("changed=%v err=%v, want none", changed, err)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != user {
		t.Errorf("CLAUDE.md changed: %q", got)
	}
}

func TestInstructionsState(t *testing.T) {
	dir := t.TempDir()
	if s := instructionsStateAt(dir); s.FilePresent || s.Imported {
		t.Errorf("empty dir: %+v", s)
	}
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if s := instructionsStateAt(dir); !s.FilePresent || !s.Imported {
		t.Errorf("after install: %+v", s)
	}
}

func TestShouldInstallInstructions(t *testing.T) {
	tests := []struct {
		name        string
		choice      InstructionsChoice
		interactive bool
		answer      string
		want        bool
		wantPrompt  bool
	}{
		{"flag yes", InstructionsYes, false, "", true, false},
		{"flag no", InstructionsNo, true, "y\n", false, false},
		{"ask, not a terminal", InstructionsAsk, false, "y\n", false, false},
		{"ask, user accepts", InstructionsAsk, true, "y\n", true, true},
		{"ask, user presses enter", InstructionsAsk, true, "\n", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := ShouldInstallInstructions(tt.choice, tt.interactive, strings.NewReader(tt.answer), &out)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if prompted := strings.Contains(out.String(), "[y/N]"); prompted != tt.wantPrompt {
				t.Errorf("prompted=%v, want %v (output %q)", prompted, tt.wantPrompt, out.String())
			}
		})
	}
}

func TestFullUninstall_RemovesInstructions(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, ".claude")
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# Rules\n")
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanUninstall()
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Instructions.Present {
		t.Fatalf("plan should flag instructions: %+v", plan.Instructions)
	}
	var out bytes.Buffer
	PrintPlan(&out, plan)
	if !strings.Contains(out.String(), "GOTK.md") {
		t.Errorf("plan output does not mention GOTK.md:\n%s", out.String())
	}

	res := ExecuteUninstall(plan)
	if len(res.Errors) != 0 {
		t.Fatalf("errors: %v", res.Errors)
	}
	if s := instructionsStateAt(dir); s.FilePresent || s.Imported {
		t.Errorf("instructions left after full uninstall: %+v", s)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != "# Rules\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
}

func TestInstallInstructions_PrependKeepsCRLF(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# Rules\r\n")
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != "@GOTK.md\r\n\r\n# Rules\r\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
}

func TestInstallInstructions_KeepsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	writeFile(t, path, "# Rules\n")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %v, want 0600", info.Mode().Perm())
	}
}

func TestInstallInstructions_RecognisesPathImports(t *testing.T) {
	for _, imp := range []string{"@~/.claude/GOTK.md", "@./GOTK.md"} {
		dir := t.TempDir()
		user := "# Rules\n" + imp + "\n"
		writeFile(t, filepath.Join(dir, "CLAUDE.md"), user)
		if err := installInstructionsAt(dir); err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, filepath.Join(dir, "CLAUDE.md")); got != user {
			t.Errorf("%s: second import added:\n%s", imp, got)
		}
	}
}

// A whitespace-only CLAUDE.md is still the user's file: only one install
// created from nothing may be deleted.
func TestUninstallInstructions_KeepsWhitespaceOnlyUserFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "\n")
	if err := installInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := uninstallInstructionsAt(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Errorf("user file deleted: %v", err)
	}
}

func TestUninstallInstructions_ReportsWhatChanged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "@GOTK.md\n\n# Rules\n")
	changed, err := uninstallInstructionsAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"updated " + filepath.Join(dir, "CLAUDE.md")}
	if strings.Join(changed, "|") != strings.Join(want, "|") {
		t.Errorf("changed = %v, want %v (GOTK.md was never there)", changed, want)
	}
}

func TestInstructionsStatus_ImportWithoutFile(t *testing.T) {
	home := fakeHome(t)
	writeFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "@GOTK.md\n")
	var out bytes.Buffer
	if err := InstructionsStatus(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "GOTK.md missing") {
		t.Errorf("status = %q", out.String())
	}
}
