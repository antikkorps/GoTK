package ctx

// Format dispatches to the appropriate formatter based on opts.Mode.
func Format(results []FileResult, opts Options) string {
	switch opts.Mode {
	case ModeDetail:
		return FormatDetail(results, opts)
	case ModeDef:
		return FormatDef(results, opts)
	case ModeTree:
		return FormatTree(results, opts)
	case ModeSummary:
		return FormatSummary(results, opts)
	default:
		return FormatScan(results, opts)
	}
}
