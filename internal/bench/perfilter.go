package bench

import (
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/filter"
)

// FilterContribution measures how much each filter reduces the output.
type FilterContribution struct {
	FilterName  string
	BytesBefore int
	BytesAfter  int
	Reduction   float64
	Duration    time.Duration
}

// namedFilter pairs a filter function with its human-readable name.
type namedFilter struct {
	name string
	fn   filter.FilterFunc
}

// MeasureFilters runs input through each filter individually and measures its impact.
// Filters are applied sequentially (each filter gets the output of the previous one),
// matching how BuildChain works.
func MeasureFilters(cfg *config.Config, input string, cmdType detect.CmdType) []FilterContribution {
	filters := buildNamedFilters(cfg, cmdType)
	var contributions []FilterContribution

	current := input
	for _, nf := range filters {
		bytesBefore := len(current)

		start := time.Now()
		result := nf.fn(current)
		duration := time.Since(start)

		bytesAfter := len(result)
		reduction := 0.0
		if bytesBefore > 0 {
			reduction = float64(bytesBefore-bytesAfter) / float64(bytesBefore) * 100
		}

		contributions = append(contributions, FilterContribution{
			FilterName:  nf.name,
			BytesBefore: bytesBefore,
			BytesAfter:  bytesAfter,
			Reduction:   reduction,
			Duration:    duration,
		})

		current = result
	}

	return contributions
}

// buildNamedFilters constructs the same filter chain as proxy.BuildChain,
// but with human-readable names attached to each filter.
func buildNamedFilters(cfg *config.Config, cmdType detect.CmdType) []namedFilter {
	var filters []namedFilter

	if cfg.Filters.StripANSI {
		filters = append(filters, namedFilter{"StripANSI", filter.StripANSI})
	}
	if cfg.Filters.NormalizeWhitespace {
		filters = append(filters, namedFilter{"NormalizeWhitespace", filter.NormalizeWhitespace})
	}
	if cfg.Filters.Dedup {
		filters = append(filters, namedFilter{"Dedup", filter.Dedup})
	}

	// Command-specific filters
	cmdFilterNames := cmdTypeFilterNames(cmdType)
	cmdFilters := detect.FiltersFor(cmdType)
	for i, f := range cmdFilters {
		name := "CmdFilter"
		if i < len(cmdFilterNames) {
			name = cmdFilterNames[i]
		}
		filters = append(filters, namedFilter{name, f})
	}

	if cfg.Filters.TrimDecorative {
		filters = append(filters, namedFilter{"TrimEmpty", filter.TrimEmpty})
	}

	filters = append(filters, namedFilter{"CompressStackTraces", filter.CompressStackTraces})

	if cfg.Security.RedactSecrets {
		filters = append(filters, namedFilter{"RedactSecrets", filter.RedactSecrets})
	}

	filters = append(filters, namedFilter{"Summarize", filter.Summarize})

	if cfg.Filters.Truncate {
		filters = append(filters, namedFilter{"Truncate", filter.TruncateWithLimit(cfg.General.MaxLines)})
	}

	return filters
}

// cmdTypeFilterNames returns human-readable names for the command-specific
// filters returned by detect.FiltersFor.
func cmdTypeFilterNames(cmdType detect.CmdType) []string {
	switch cmdType {
	case detect.CmdGrep:
		return []string{"CompressPaths", "CompressGrepOutput"}
	case detect.CmdFind:
		return []string{"CompressPaths", "CompressFindOutput"}
	case detect.CmdGit:
		return []string{"CompressGitOutput"}
	case detect.CmdGoTool:
		return []string{"CompressPaths", "CompressGoOutput"}
	case detect.CmdLs:
		return []string{"CompressLsOutput"}
	case detect.CmdDocker:
		return []string{"CompressDockerOutput"}
	case detect.CmdNpm:
		return []string{"CompressNpmOutput"}
	case detect.CmdCargo:
		return []string{"CompressCargoOutput"}
	case detect.CmdMake:
		return []string{"CompressMakeOutput"}
	default:
		return []string{"CompressPaths"}
	}
}
