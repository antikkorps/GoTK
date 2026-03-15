package measure

// Entry represents a single measurement log entry for one GoTK invocation.
type Entry struct {
	Timestamp    string  `json:"ts"`
	SessionID    string  `json:"session"`
	Command      string  `json:"command"`
	CommandType  string  `json:"cmd_type"`
	RawBytes     int     `json:"raw_bytes"`
	CleanBytes   int     `json:"clean_bytes"`
	RawTokens    int     `json:"raw_tokens"`
	CleanTokens  int     `json:"clean_tokens"`
	TokensSaved  int     `json:"tokens_saved"`
	ReductionPct float64 `json:"reduction_pct"`
	LinesRaw     int     `json:"lines_raw"`
	LinesClean   int     `json:"lines_clean"`
	ImportantLines int   `json:"important_lines"`
	QualityScore float64 `json:"quality_score"`
	Mode         string  `json:"mode"`
	Source       string  `json:"source"`
	Cached       bool    `json:"cached"`
	DurationUs   int64   `json:"duration_us"`
}
