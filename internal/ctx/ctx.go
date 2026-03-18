package ctx

import (
	"fmt"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// Run executes a full context search: walk files, search, format, and filter output.
// Returns the formatted (and optionally filtered) result string plus raw for stats.
func Run(cfg *config.Config, args []string, maxLines int, showStats bool) (string, error) {
	opts, err := ParseFlags(args)
	if err != nil {
		return "", fmt.Errorf("gotk ctx: %w", err)
	}

	if opts.Pattern == "" && opts.Mode != ModeTree {
		return "", fmt.Errorf("gotk ctx: missing search pattern")
	}

	// Walk files
	files, err := WalkFiles(opts)
	if err != nil {
		return "", fmt.Errorf("gotk ctx: walk error: %w", err)
	}

	// Search
	results, err := Search(files, opts)
	if err != nil {
		return "", fmt.Errorf("gotk ctx: search error: %w", err)
	}

	// Format
	raw := Format(results, opts)

	// Apply GoTK filter chain for path compression, secret redaction, truncation
	chain := proxy.BuildChain(cfg, detect.CmdGeneric, maxLines)
	cleaned := chain.Apply(raw)

	if showStats {
		rawBytes := len(raw)
		cleanBytes := len(cleaned)
		saved := rawBytes - cleanBytes
		pct := 0
		if rawBytes > 0 {
			pct = saved * 100 / rawBytes
		}
		stats := fmt.Sprintf("\n[gotk] %d → %d bytes (-%d%%, saved %d bytes)\n",
			rawBytes, cleanBytes, pct, saved)
		return cleaned + stats, nil
	}

	return cleaned, nil
}
