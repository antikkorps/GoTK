package daemon

import (
	"fmt"
	"io"
	"time"

	"github.com/antikkorps/GoTK/internal/measure"
)

// PrintSummary reads the measure log and prints a summary for the given session.
func PrintSummary(w io.Writer, logPath, sessionID string) {
	entries, err := measure.ReadEntries(logPath)
	if err != nil {
		return
	}

	var (
		count       int
		rawTokens   int
		savedTokens int
		totalDur    time.Duration
	)

	for _, e := range entries {
		if e.SessionID != sessionID {
			continue
		}
		count++
		rawTokens += e.RawTokens
		savedTokens += e.TokensSaved
		totalDur += time.Duration(e.DurationUs) * time.Microsecond
	}

	if count == 0 {
		fmt.Fprintf(w, "\n[gotk] session ended — no commands filtered\n") //nolint:errcheck
		return
	}

	pct := 0
	if rawTokens > 0 {
		pct = savedTokens * 100 / rawTokens
	}

	fmt.Fprintf(w, "\n[gotk] session summary\n")       //nolint:errcheck
	fmt.Fprintf(w, "  commands filtered: %d\n", count) //nolint:errcheck
	fmt.Fprintf(w, "  tokens processed:  %d\n", rawTokens)                        //nolint:errcheck
	fmt.Fprintf(w, "  tokens saved:      %d (-%d%%)\n", savedTokens, pct)        //nolint:errcheck
	fmt.Fprintf(w, "  filter time:       %s\n", totalDur.Round(time.Millisecond)) //nolint:errcheck
}
