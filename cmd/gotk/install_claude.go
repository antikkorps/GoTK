package main

import (
	"fmt"
	"os"

	"github.com/antikkorps/GoTK/internal/install"
)

// installClaudeOpts holds the parsed `gotk install claude` flags.
type installClaudeOpts struct {
	scope        install.Scope
	uninstall    bool
	status       bool
	instructions install.InstructionsChoice
}

func parseScopeFlag(a string) (install.Scope, bool) {
	switch a {
	case "--global":
		return install.ScopeGlobal, true
	case "--project":
		return install.ScopeProject, true
	case "--local":
		return install.ScopeLocal, true
	}
	return install.ScopeLocal, false
}

func parseInstallClaudeFlags(args []string) installClaudeOpts {
	o := installClaudeOpts{scope: install.ScopeLocal}
	for _, a := range args {
		if sc, ok := parseScopeFlag(a); ok {
			o.scope = sc
			continue
		}
		switch a {
		case "--uninstall":
			o.uninstall = true
		case "--status":
			o.status = true
		case "--instructions":
			o.instructions = install.InstructionsYes
		case "--no-instructions":
			o.instructions = install.InstructionsNo
		default:
			fmt.Fprintf(os.Stderr, "gotk install claude: unknown flag %q\n", a)
			os.Exit(1)
		}
	}
	return o
}

func runInstallClaude(args []string) {
	o := parseInstallClaudeFlags(args)
	var err error
	switch {
	case o.status:
		err = claudeStatus(o.scope)
	case o.uninstall:
		err = claudeUninstall(o.scope)
	default:
		err = claudeInstall(o)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotk install claude: %v\n", err)
		fmt.Fprintln(os.Stderr, "  hint: use --status to check current state, --global for ~/.claude/settings.json")
		os.Exit(1)
	}
}

func claudeStatus(scope install.Scope) error {
	if err := install.ClaudeStatus(scope); err != nil {
		return err
	}
	return install.InstructionsStatus(os.Stderr)
}

// The instructions live in ~/.claude and serve every project, so only a
// global uninstall removes them; a local or project one leaves them for the
// hooks still installed elsewhere. `gotk uninstall` removes them too.
func claudeUninstall(scope install.Scope) error {
	if err := install.ClaudeUninstall(scope); err != nil {
		return err
	}
	if scope != install.ScopeGlobal {
		return nil
	}
	return install.UninstallInstructions(os.Stderr)
}

// claudeInstall adds the hook, then the instructions only if the user
// agrees: they edit the user's own CLAUDE.md, unlike the hook.
func claudeInstall(o installClaudeOpts) error {
	if err := install.ClaudeInstall(o.scope); err != nil {
		return err
	}
	// Both ends must be a terminal: with stderr redirected the question is
	// invisible and install would wait for an answer nobody knows to give.
	interactive := isTerminal(os.Stdin) && isTerminal(os.Stderr)
	ok, err := install.ShouldInstallInstructions(o.instructions, interactive, os.Stdin, os.Stderr)
	if err != nil {
		return err
	}
	if !ok {
		if o.instructions == install.InstructionsAsk && !interactive {
			fmt.Fprintln(os.Stderr, "Tip: add --instructions to also import GoTK usage notes in ~/.claude/CLAUDE.md")
		}
		return nil
	}
	if err := install.InstallInstructions(os.Stderr); err != nil {
		// Say what did succeed: the user should not reinstall the hook.
		return fmt.Errorf("hook installed, but instructions not written: %w", err)
	}
	return nil
}
