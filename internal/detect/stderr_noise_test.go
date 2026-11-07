package detect_test

import (
	"path/filepath"
	"testing"

	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/testutil"
)

func stderrNoiseFor(t *testing.T, cmdType detect.CmdType) func(string) string {
	t.Helper()
	f := detect.StderrNoiseFilter(cmdType)
	if f == nil {
		t.Fatalf("StderrNoiseFilter(%s) = nil, want a filter", cmdType)
	}
	return f
}

func TestGolden_StderrCargo(t *testing.T) {
	dir := filepath.Join(testdataDir(), "stderr_cargo")
	testutil.RunGoldenTest(t, dir, stderrNoiseFor(t, detect.CmdCargo))
}

func TestGolden_StderrNpm(t *testing.T) {
	dir := filepath.Join(testdataDir(), "stderr_npm")
	testutil.RunGoldenTest(t, dir, stderrNoiseFor(t, detect.CmdNpm))
}

func TestStderrNoiseFilter_NilForCommandsWithoutAllowlist(t *testing.T) {
	for _, ct := range []detect.CmdType{detect.CmdGeneric, detect.CmdGrep, detect.CmdGoTool} {
		if detect.StderrNoiseFilter(ct) != nil {
			t.Errorf("StderrNoiseFilter(%s) should be nil", ct)
		}
	}
}

func TestStderrNoiseFilter_NodeSharesNpmConfigCollapse(t *testing.T) {
	// npx prints the same npmrc warnings as npm.
	in := "npm warn Unknown project config \"a\". This will stop working in the next major version of npm.\n" +
		"npm warn Unknown project config \"b\". This will stop working in the next major version of npm.\n"
	want := "npm warn Unknown project config \"a\", \"b\" (ignored by npm; see `npm help npmrc`)\n"
	if got := stderrNoiseFor(t, detect.CmdNode)(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCargoStderr_SingleProgressLineKeptVerbatim(t *testing.T) {
	in := "   Compiling myapp v0.1.0 (/x)\n    Finished `dev` profile in 0.5s\n"
	if got := stderrNoiseFor(t, detect.CmdCargo)(in); got != in {
		t.Errorf("single progress line must stay verbatim, got %q", got)
	}
}

func TestCargoStderr_RunsSplitByDiagnosticsStaySeparate(t *testing.T) {
	// Chronology matters: a warning between two runs must stay between them.
	in := "   Compiling a v1.0.0\n   Compiling b v1.0.0\n" +
		"warning: unused import\n" +
		"   Checking c v1.0.0\n   Checking d v1.0.0\n   Fresh e v1.0.0\n"
	want := "   ... 2 cargo progress lines (Compiling 2)\n" +
		"warning: unused import\n" +
		"   ... 3 cargo progress lines (Checking 2, Fresh 1)\n"
	if got := stderrNoiseFor(t, detect.CmdCargo)(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestCargoStderr_BlockingLockIsProgress(t *testing.T) {
	in := "    Blocking waiting for file lock on package cache\n" +
		"    Blocking waiting for file lock on build directory\n"
	want := "    ... 2 cargo progress lines (Blocking 2)\n"
	if got := stderrNoiseFor(t, detect.CmdCargo)(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCargoStderr_ErrorMentioningCompilingIsKept(t *testing.T) {
	// Only the indented cargo status form counts as progress.
	in := "error: Compiling failed for crate x\nerror: Compiling failed for crate y\n"
	if got := stderrNoiseFor(t, detect.CmdCargo)(in); got != in {
		t.Errorf("diagnostic lines must not be collapsed, got %q", got)
	}
}

func TestNpmStderr_NonStandardSuffixKept(t *testing.T) {
	// If npm changes the wording, we no longer know the line is pure noise.
	in := "npm warn Unknown project config \"a\". Something new and important.\n" +
		"npm warn Unknown project config \"b\". Something new and important.\n"
	if got := stderrNoiseFor(t, detect.CmdNpm)(in); got != in {
		t.Errorf("unknown wording must stay verbatim, got %q", got)
	}
}

func TestNpmStderr_RepeatedBlocksDedupKeys(t *testing.T) {
	// Each npm process re-prints the block; a script calling npm twice must
	// not list the keys twice.
	w := func(k string) string {
		return "npm warn Unknown project config \"" + k + "\". This will stop working in the next major version of npm.\n"
	}
	in := w("a") + w("b") + "> lint\n" + w("a") + w("b") + "done\n"
	want := "npm warn Unknown project config \"a\", \"b\" (ignored by npm; see `npm help npmrc`)\n> lint\ndone\n"
	if got := stderrNoiseFor(t, detect.CmdNpm)(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNpmStderr_SameKeyRepeatedTwiceIsFolded(t *testing.T) {
	w := "npm warn Unknown project config \"a\". This will stop working in the next major version of npm.\n"
	want := "npm warn Unknown project config \"a\" (ignored by npm; see `npm help npmrc`)\nok\n"
	if got := stderrNoiseFor(t, detect.CmdNpm)(w + "ok\n" + w); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
