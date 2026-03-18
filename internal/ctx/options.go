package ctx

import (
	"fmt"
	"strconv"
	"strings"
)

// Mode determines how search results are formatted.
type Mode int

const (
	ModeScan    Mode = iota // Default: file + match count, indented matches
	ModeDetail              // Context windows around matches
	ModeDef                 // Language-aware definition search
	ModeTree                // Structural skeleton per file
	ModeSummary             // Directory breakdown table
)

// String returns the name of the mode.
func (m Mode) String() string {
	switch m {
	case ModeScan:
		return "scan"
	case ModeDetail:
		return "detail"
	case ModeDef:
		return "def"
	case ModeTree:
		return "tree"
	case ModeSummary:
		return "summary"
	default:
		return "scan"
	}
}

// Options holds all parameters for a context search.
type Options struct {
	Pattern    string   // Search pattern (regex)
	Dir        string   // Root directory to search (default: ".")
	Mode       Mode     // Output mode
	Context    int      // Lines of context for detail mode (default: 3)
	FileTypes  []string // File extensions to include (e.g., "go", "py")
	Glob       string   // Glob pattern to filter files (e.g., "*.go")
	MaxResults int      // Maximum number of file results (0 = unlimited)
	MaxLine    int      // Maximum characters per line in output (default: 120)
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		Dir:     ".",
		Mode:    ModeScan,
		Context: 3,
		MaxLine: 120,
	}
}

// ParseFlags parses ctx-specific flags from args and returns remaining positional args.
// Flags: -d N (detail context), -t ext (file type), -g glob, -m N (max results),
// --def, --tree, --summary, -p dir (path/directory)
func ParseFlags(args []string) (Options, error) {
	opts := DefaultOptions()
	var positional []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-d", "--detail":
			opts.Mode = ModeDetail
			// Optional numeric argument for context lines
			if i+1 < len(args) && isNumeric(args[i+1]) {
				n, _ := strconv.Atoi(args[i+1])
				opts.Context = n
				i++
			}
		case "--def":
			opts.Mode = ModeDef
		case "--tree":
			opts.Mode = ModeTree
		case "--summary":
			opts.Mode = ModeSummary
		case "-t", "--type":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("flag %s requires an argument", args[i])
			}
			i++
			opts.FileTypes = append(opts.FileTypes, strings.TrimPrefix(args[i], "."))
		case "-g", "--glob":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("flag %s requires an argument", args[i])
			}
			i++
			opts.Glob = args[i]
		case "-m", "--max":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("flag %s requires an argument", args[i])
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid max results: %s", args[i])
			}
			opts.MaxResults = n
		case "-p", "--path":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("flag %s requires an argument", args[i])
			}
			i++
			opts.Dir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return opts, fmt.Errorf("unknown flag: %s", args[i])
			}
			positional = append(positional, args[i])
		}
	}

	if len(positional) > 0 {
		opts.Pattern = positional[0]
	}
	if len(positional) > 1 {
		opts.Dir = positional[1]
	}

	return opts, nil
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
