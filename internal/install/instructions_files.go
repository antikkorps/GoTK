package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antikkorps/GoTK/internal/paths"
)

// installInstructionsAt writes dir/GOTK.md and makes dir/CLAUDE.md import
// it. Both steps are idempotent, so re-running install refreshes the text
// after an upgrade without duplicating the import.
func installInstructionsAt(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := writeIfChanged(filepath.Join(dir, instructionsFile), instructionsContent); err != nil {
		return err
	}
	memPath := filepath.Join(dir, claudeMemoryFile)
	content, err := readOptional(memPath)
	if err != nil || hasImport(content) {
		return err
	}
	return writeKeepMode(memPath, prependImport(content))
}

// uninstallInstructionsAt removes the import line and GOTK.md, returning
// one "updated|deleted <path>" entry per file it changed. CLAUDE.md is
// deleted only when nothing at all is left, i.e. install created it.
func uninstallInstructionsAt(dir string) ([]string, error) {
	var changed []string
	memPath := filepath.Join(dir, claudeMemoryFile)
	content, err := readOptional(memPath)
	if err != nil {
		return nil, err
	}
	if stripped, found := removeImport(content); found {
		verb, err := rewriteOrDelete(memPath, stripped)
		if err != nil {
			return changed, fmt.Errorf("updating %s: %w", memPath, err)
		}
		changed = append(changed, verb+" "+memPath)
	}
	filePath := filepath.Join(dir, instructionsFile)
	if err := os.Remove(filePath); err == nil {
		changed = append(changed, "deleted "+filePath)
	} else if !os.IsNotExist(err) {
		return changed, err
	}
	return changed, nil
}

func rewriteOrDelete(path, content string) (string, error) {
	if content == "" {
		return "deleted", os.Remove(path)
	}
	return "updated", writeKeepMode(path, content)
}

func instructionsStateAt(dir string) InstructionsState {
	content, _ := readOptional(filepath.Join(dir, claudeMemoryFile))
	return InstructionsState{
		FilePresent: fileExists(filepath.Join(dir, instructionsFile)),
		Imported:    hasImport(content),
	}
}

// isImportLine accepts the bare import and path forms a user may have
// written by hand (@~/.claude/GOTK.md, @./GOTK.md), so install never adds
// a second import that would load the notes twice.
func isImportLine(line string) bool {
	l := strings.TrimSpace(line)
	return l == instructionsImport ||
		(strings.HasPrefix(l, "@") && strings.HasSuffix(l, "/"+instructionsFile))
}

// hasImport accepts the import on any line: a user who moved it chose
// that spot, and adding a second one would load the file twice.
func hasImport(content string) bool {
	for _, l := range strings.Split(content, "\n") {
		if isImportLine(l) {
			return true
		}
	}
	return false
}

// prependImport puts the import first, as RTK does, so it reads as tool
// setup rather than one of the user's rules. The file's line ending is kept.
func prependImport(content string) string {
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	if content == "" {
		return instructionsImport + eol
	}
	return instructionsImport + eol + eol + content
}

// removeImport drops every import line, plus the blank line prependImport
// put after it when the import opens the file.
func removeImport(content string) (string, bool) {
	lines := strings.SplitAfter(content, "\n")
	out := make([]string, 0, len(lines))
	found := false
	for i := 0; i < len(lines); i++ {
		if !isImportLine(lines[i]) {
			out = append(out, lines[i])
			continue
		}
		found = true
		if len(out) == 0 && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
			i++
		}
	}
	return strings.Join(out, ""), found
}

func readOptional(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(data), nil
}

func writeIfChanged(path, content string) error {
	if current, err := readOptional(path); err != nil || current == content {
		return err
	}
	return writeKeepMode(path, content)
}

// writeKeepMode preserves the permissions the user gave an existing file,
// and writes atomically: a crash mid-write must not truncate their rules.
func writeKeepMode(path, content string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := paths.WriteFileAtomic(path, []byte(content), mode); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
