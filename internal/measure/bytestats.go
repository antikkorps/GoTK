package measure

import "fmt"

// ByteStats holds raw vs filtered sizes for both output streams. Stderr must
// be counted: tools like cargo write everything there, and a stdout-only
// figure reported "0 → 0 bytes" for them (issue #96).
type ByteStats struct {
	RawOut, CleanOut int
	RawErr, CleanErr int
}

// String formats the reduction summary shown after "[gotk]". The per-stream
// breakdown appears only when stderr carried something.
func (s ByteStats) String() string {
	raw := s.RawOut + s.RawErr
	clean := s.CleanOut + s.CleanErr
	saved := raw - clean
	pct := 0
	if raw > 0 {
		pct = saved * 100 / raw
	}
	out := fmt.Sprintf("%d → %d bytes (-%d%%, saved %d bytes)", raw, clean, pct, saved)
	if s.RawErr > 0 {
		out += fmt.Sprintf(" [stdout %d → %d, stderr %d → %d]",
			s.RawOut, s.CleanOut, s.RawErr, s.CleanErr)
	}
	return out
}
