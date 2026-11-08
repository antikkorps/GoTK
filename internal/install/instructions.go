package install

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// The hook filters output silently; without these instructions Claude does
// not know that markers are counts or how to get raw output back (#110).
//
//go:embed GOTK.md
var instructionsContent string

const (
	instructionsFile   = "GOTK.md"
	claudeMemoryFile   = "CLAUDE.md"
	instructionsImport = "@" + instructionsFile
)

// InstructionsChoice is what the user asked for on the command line.
type InstructionsChoice int

const (
	// InstructionsAsk prompts on a terminal and declines otherwise: the
	// user's CLAUDE.md is theirs, so nothing is written without a yes.
	InstructionsAsk InstructionsChoice = iota
	InstructionsYes
	InstructionsNo
)

// InstructionsState reports what is on disk.
type InstructionsState struct {
	FilePresent bool // GOTK.md exists
	Imported    bool // CLAUDE.md carries the @GOTK.md line
}

// instructionsDir is ~/.claude whatever the hook scope: the user-level
// CLAUDE.md is personal and never committed, unlike a project one.
func instructionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// ShouldInstallInstructions resolves the user's choice, prompting on w/r
// only when the choice is open and a terminal is attached.
func ShouldInstallInstructions(choice InstructionsChoice, interactive bool, r io.Reader, w io.Writer) (bool, error) {
	switch {
	case choice == InstructionsYes:
		return true, nil
	case choice == InstructionsNo || !interactive:
		return false, nil
	}
	line(w, "Add GoTK usage notes to ~/.claude/CLAUDE.md (a GOTK.md file plus an @GOTK.md import line)?")
	return Confirm(r, w, "Claude then knows its Bash output is filtered and how to get it raw.")
}

// InstallInstructions writes ~/.claude/GOTK.md and imports it.
func InstallInstructions(w io.Writer) error {
	dir, err := instructionsDir()
	if err != nil {
		return err
	}
	if err := installInstructionsAt(dir); err != nil {
		return err
	}
	linef(w, "GoTK instructions imported in %s\n", filepath.Join(dir, claudeMemoryFile))
	return nil
}

// UninstallInstructions removes the import line and ~/.claude/GOTK.md.
func UninstallInstructions(w io.Writer) error {
	dir, err := instructionsDir()
	if err != nil {
		return err
	}
	changed, err := uninstallInstructionsAt(dir)
	for _, c := range changed {
		linef(w, "GoTK instructions: %s\n", c)
	}
	return err
}

// InstructionsStatus prints the instructions state.
func InstructionsStatus(w io.Writer) error {
	dir, err := instructionsDir()
	if err != nil {
		return err
	}
	s := instructionsStateAt(dir)
	state := "not installed"
	switch {
	case s.Imported && s.FilePresent:
		state = "installed"
	case s.Imported:
		state = "import present but GOTK.md missing (re-run with --instructions)"
	case s.FilePresent:
		state = "GOTK.md present but not imported (re-run with --instructions)"
	}
	linef(w, "GoTK instructions (%s): %s\n", filepath.Join(dir, claudeMemoryFile), state)
	return nil
}

// InstructionsTarget is the full-uninstall view of the instructions.
type InstructionsTarget struct {
	Dir     string
	Present bool // GOTK.md or its import line exists
}

func planInstructions() InstructionsTarget {
	dir, err := instructionsDir()
	if err != nil {
		return InstructionsTarget{}
	}
	s := instructionsStateAt(dir)
	return InstructionsTarget{Dir: dir, Present: s.FilePresent || s.Imported}
}

func executeInstructions(t InstructionsTarget, res *UninstallResult) {
	if !t.Present {
		return
	}
	changed, err := uninstallInstructionsAt(t.Dir)
	res.InstructionChanges = append(res.InstructionChanges, changed...)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("removing GoTK instructions: %w", err))
	}
}

func printInstructionsPlan(w io.Writer, t InstructionsTarget) {
	line(w)
	line(w, "  Claude instructions:")
	if !t.Present {
		line(w, "    (none found)")
		return
	}
	linef(w, "    [x] %s and its @%s line in %s\n",
		filepath.Join(t.Dir, instructionsFile), instructionsFile, filepath.Join(t.Dir, claudeMemoryFile))
}
