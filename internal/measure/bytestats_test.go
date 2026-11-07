package measure

import "testing"

func TestByteStats_StdoutOnly(t *testing.T) {
	s := ByteStats{RawOut: 200, CleanOut: 50}
	want := "200 → 50 bytes (-75%, saved 150 bytes)"
	if got := s.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestByteStats_CountsStderrWithBreakdown(t *testing.T) {
	s := ByteStats{RawOut: 0, CleanOut: 0, RawErr: 400, CleanErr: 100}
	want := "400 → 100 bytes (-75%, saved 300 bytes) [stdout 0 → 0, stderr 400 → 100]"
	if got := s.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestByteStats_Empty(t *testing.T) {
	want := "0 → 0 bytes (-0%, saved 0 bytes)"
	if got := (ByteStats{}).String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
