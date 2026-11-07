package proxy

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
)

func out(s string) Piece  { return Piece{Text: s} }
func errp(s string) Piece { return Piece{Stderr: true, Text: s} }

func TestInterleave_NothingFilteredRestoresOrder(t *testing.T) {
	raw := "1 out\n3 out\n5 out\n"
	blocks := []exec.StderrBlock{{AfterLine: 1, Text: "2 err\n"}, {AfterLine: 2, Text: "4 err\n"}}
	want := []Piece{out("1 out\n"), errp("2 err\n"), out("3 out\n"), errp("4 err\n"), out("5 out\n")}
	if got := Interleave(raw, raw, blocks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_DroppedLineAnchorsToPreviousSurvivor(t *testing.T) {
	raw := "out1\nout2\nout3\n"
	clean := "out1\nout3\n"
	blocks := []exec.StderrBlock{{AfterLine: 2, Text: "ERR\n"}}
	want := []Piece{out("out1\n"), errp("ERR\n"), out("out3\n")}
	if got := Interleave(raw, clean, blocks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_MatchIgnoresANSIAndSpacing(t *testing.T) {
	raw := "\x1b[31mred\x1b[0m   line\nnext\n"
	clean := "red line\nnext\n"
	blocks := []exec.StderrBlock{{AfterLine: 1, Text: "E\n"}}
	want := []Piece{out("red line\n"), errp("E\n"), out("next\n")}
	if got := Interleave(raw, clean, blocks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_UnmatchedLineTakesNextOrigin(t *testing.T) {
	raw := "a\nb\nb\nb\nc\n"
	clean := "a\nb\n... repeated 2 more times\nc\n"
	blocks := []exec.StderrBlock{{AfterLine: 4, Text: "E\n"}}
	// An unmatched line takes the origin of the next matched one: the block
	// may move up before the marker, never down past later stdout.
	want := []Piece{out("a\nb\n"), errp("E\n"), out("... repeated 2 more times\nc\n")}
	if got := Interleave(raw, clean, blocks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_BlockBeforeAnyStdoutComesFirst(t *testing.T) {
	raw := "\x1b[1mheader\x1b[0m\nbody\n"
	clean := "HEADER (rewritten)\nbody\n"
	blocks := []exec.StderrBlock{{AfterLine: 0, Text: "early\n"}}
	want := []Piece{errp("early\n"), out("HEADER (rewritten)\nbody\n")}
	if got := Interleave(raw, clean, blocks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_TruncatedMiddleAnchorsAfterHead(t *testing.T) {
	var rawLines []string
	for i := range 100 {
		rawLines = append(rawLines, fmt.Sprintf("line %d", i))
	}
	raw := strings.Join(rawLines, "\n") + "\n"
	clean := strings.Join(rawLines[:10], "\n") + "\n[... 80 lines truncated ...]\n" +
		strings.Join(rawLines[90:], "\n") + "\n"
	got := Interleave(raw, clean, []exec.StderrBlock{{AfterLine: 50, Text: "E\n"}})
	if len(got) != 3 || !strings.HasSuffix(got[0].Text, "line 9\n") ||
		!got[1].Stderr || !strings.HasPrefix(got[2].Text, "[... 80 lines truncated ...]\n") {
		t.Errorf("block should sit right after the head, got %+v", got)
	}
}

func TestInterleave_NoBlocksOrNoStdout(t *testing.T) {
	if got := Interleave("a\n", "a\n", nil); !reflect.DeepEqual(got, []Piece{out("a\n")}) {
		t.Errorf("no blocks: got %+v", got)
	}
	blocks := []exec.StderrBlock{{AfterLine: 0, Text: "only err\n"}}
	if got := Interleave("", "", blocks); !reflect.DeepEqual(got, []Piece{errp("only err\n")}) {
		t.Errorf("no stdout: got %+v", got)
	}
}

func TestInterleave_BlocksAfterLastLineGoAtEnd(t *testing.T) {
	blocks := []exec.StderrBlock{{AfterLine: 2, Text: "E1\n"}, {AfterLine: 2, Text: "E2\n"}}
	want := []Piece{out("x\ny"), errp("E1\n"), errp("E2\n")}
	if got := Interleave("x\ny", "x\ny", blocks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestWritePieces_SeparatesUnterminatedPieces(t *testing.T) {
	var o, e bytes.Buffer
	last := WritePieces(&o, &e, []Piece{out("x\ny"), errp("E"), out("z\n")})
	if o.String() != "x\ny\nz\n" || e.String() != "E\n" {
		t.Errorf("stdout %q, stderr %q", o.String(), e.String())
	}
	if last != "z\n" {
		t.Errorf("last = %q", last)
	}
}

func TestFilterStderrBlocks_FiltersEachBlockAndDropsEmpty(t *testing.T) {
	cfg := config.Default()
	blocks := []exec.StderrBlock{
		{AfterLine: 0, Text: "   Compiling a v1\n   Compiling b v1\n"},
		{AfterLine: 3, Text: "\x1b[0m"},
		{AfterLine: 5, Text: "token=ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx\n"},
	}
	got := FilterStderrBlocks(cfg, detect.CmdCargo, blocks)
	if len(got) != 2 {
		t.Fatalf("empty block should be dropped, got %+v", got)
	}
	if got[0].Text != "   ... 2 cargo progress lines (Compiling 2)\n" || got[0].AfterLine != 0 {
		t.Errorf("first block: %+v", got[0])
	}
	if strings.Contains(got[1].Text, "ghp_FAKE") || got[1].AfterLine != 5 {
		t.Errorf("second block: %+v", got[1])
	}
}

// Review finding: a rewritten line used to inherit the previous origin, which
// pushed a block past stdout written after it.
func TestInterleave_RewrittenLinesNeverPushBlockDown(t *testing.T) {
	raw := "AAA x\nBBB y\n"
	clean := "aaa\nbbb\n" // both rewritten beyond recognition
	got := Interleave(raw, clean, []exec.StderrBlock{{AfterLine: 1, Text: "E\n"}})
	want := []Piece{errp("E\n"), out("aaa\nbbb\n")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_CompressedPathsStillMatch(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip(err)
	}
	raw := cwd + "/a.go:1: x\n" + cwd + "/b.go:2: y\n"
	clean := "./a.go:1: x\n./b.go:2: y\n"
	got := Interleave(raw, clean, []exec.StderrBlock{{AfterLine: 1, Text: "E\n"}})
	want := []Piece{out("./a.go:1: x\n"), errp("E\n"), out("./b.go:2: y\n")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestInterleave_EmptyFilteredStdoutAddsNoBlankLine(t *testing.T) {
	got := Interleave("noise\n", "", []exec.StderrBlock{{AfterLine: 1, Text: "ERR\n"}})
	if want := []Piece{errp("ERR\n")}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// Review finding: a PEM key split across two blocks by an interleaved stdout
// line escaped the multi-line redaction pattern.
func TestFilterStderrBlocks_RedactsPrivateKeySplitAcrossBlocks(t *testing.T) {
	blocks := []exec.StderrBlock{
		{AfterLine: 0, Text: "-----BEGIN RSA PRIVATE KEY-----\nAAAASECRETBODY1\n"},
		{AfterLine: 1, Text: "BBBBSECRETBODY2\n-----END RSA PRIVATE KEY-----\n"},
		{AfterLine: 2, Text: "after\n"},
	}
	got := FilterStderrBlocks(config.Default(), detect.CmdGeneric, blocks)
	for _, b := range got {
		if strings.Contains(b.Text, "SECRETBODY") {
			t.Fatalf("private key body leaked: %+v", got)
		}
	}
	if len(got) != 2 || got[0].AfterLine != 0 || got[1].Text != "after\n" {
		t.Errorf("open PEM blocks should merge into the first: %+v", got)
	}
}

func TestMergeStreams_ArrivalOrderInOneText(t *testing.T) {
	res := &exec.Result{
		Stdout:       "1 out\n3 out",
		StderrBlocks: []exec.StderrBlock{{AfterLine: 1, Text: "2 err\n"}, {AfterLine: 2, Text: "4 err"}},
	}
	if got, want := MergeStreams(res), "1 out\n2 err\n3 out\n4 err"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
