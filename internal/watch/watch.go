package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	gotkerrors "github.com/antikkorps/GoTK/internal/errors"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// Config holds watch mode settings.
type Config struct {
	Command    []string       // Command and args to run
	Interval   time.Duration  // Polling interval (default: 2s)
	Debounce   time.Duration  // Debounce time after file change (default: 500ms)
	Paths      []string       // Paths to watch (default: ".")
	Extensions []string       // File extensions to watch (e.g., ".go", ".py")
	MaxLines   int            // Max output lines
	GoTKConfig *config.Config // GoTK filter config
}

// ignoredDirs contains directory names that should be skipped during file walking.
var ignoredDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
}

// snapshot maps file paths to their modification times.
type snapshot map[string]time.Time

// Run starts the watch loop. Blocks until ctx is cancelled.
func Run(ctx context.Context, cfg Config) error {
	if len(cfg.Command) == 0 {
		return &gotkerrors.ValidationError{Field: "watch command", Message: "no command specified"}
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.Debounce <= 0 {
		cfg.Debounce = 500 * time.Millisecond
	}
	if len(cfg.Paths) == 0 {
		cfg.Paths = []string{"."}
	}

	// Normalize extensions to include the leading dot.
	for i, ext := range cfg.Extensions {
		if ext != "" && ext[0] != '.' {
			cfg.Extensions[i] = "." + ext
		}
	}

	// Take initial snapshot.
	prev := takeSnapshot(cfg.Paths, cfg.Extensions)

	// Run the command immediately on start.
	runCommand(ctx, cfg)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			curr := takeSnapshot(cfg.Paths, cfg.Extensions)
			if snapshotChanged(prev, curr) {
				// Debounce: wait a short period, then re-snapshot to catch
				// batch saves that produce multiple writes in quick succession.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(cfg.Debounce):
				}
				// Re-snapshot after debounce to capture the final state.
				curr = takeSnapshot(cfg.Paths, cfg.Extensions)
				prev = curr
				runCommand(ctx, cfg)
			}
		}
	}
}

// runCommand clears the screen, prints a header, executes the command,
// filters its output, and prints a waiting message.
func runCommand(ctx context.Context, cfg Config) {
	// Clear screen.
	fmt.Fprint(os.Stdout, "\033[2J\033[H") //nolint:errcheck

	cmdStr := strings.Join(cfg.Command, " ")
	now := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stdout, "[gotk watch] %s — running: %s\n\n", now, cmdStr) //nolint:errcheck

	// Execute command with timeout from config.
	timeout := time.Duration(cfg.GoTKConfig.Security.CommandTimeout) * time.Second
	if timeout <= 0 {
		timeout = exec.DefaultTimeout
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := exec.RunWithTimeout(cmdCtx, cfg.Command[0], cfg.Command[1:]...)
	if err != nil && result == nil {
		fmt.Fprintf(os.Stderr, "[gotk watch] error: %v\n", err)
		fmt.Fprintln(os.Stdout, "\n[gotk watch] waiting for changes...") //nolint:errcheck
		return
	}

	// Build filter chain and apply.
	cmdType := detect.Identify(cfg.Command[0])
	if mapped, ok := cfg.GoTKConfig.Commands[cfg.Command[0]]; ok {
		cmdType = detect.Identify(mapped)
	}
	chain := proxy.BuildChain(cfg.GoTKConfig, cmdType, cfg.MaxLines)
	cleaned := chain.Apply(result.Stdout)

	fmt.Fprint(os.Stdout, cleaned) //nolint:errcheck

	// Pass through stderr unmodified.
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	// Show stats.
	rawBytes := len(result.Stdout)
	cleanBytes := len(cleaned)
	if rawBytes > 0 {
		saved := rawBytes - cleanBytes
		pct := saved * 100 / rawBytes
		fmt.Fprintf(os.Stderr, "\n[gotk watch] %d → %d bytes (-%d%%)\n", rawBytes, cleanBytes, pct)
	}

	if result.ExitCode != 0 {
		fmt.Fprintf(os.Stdout, "\n[gotk watch] exit code: %d\n", result.ExitCode) //nolint:errcheck
	}

	fmt.Fprintln(os.Stdout, "\n[gotk watch] waiting for changes...") //nolint:errcheck
}

// takeSnapshot walks the given paths and records modification times for files
// matching the extension filter. Hidden directories and common vendor
// directories are skipped.
func takeSnapshot(paths []string, extensions []string) snapshot {
	snap := make(snapshot)
	extSet := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		extSet[ext] = true
	}

	for _, root := range paths {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible paths
			}

			if info.IsDir() {
				name := info.Name()
				if isIgnoredDir(name) {
					return filepath.SkipDir
				}
				return nil
			}

			// Filter by extension if extensions are specified.
			if len(extSet) > 0 {
				ext := filepath.Ext(path)
				if !extSet[ext] {
					return nil
				}
			}

			snap[path] = info.ModTime()
			return nil
		})
	}

	return snap
}

// isIgnoredDir returns true if the directory name should be skipped.
func isIgnoredDir(name string) bool {
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}
	return ignoredDirs[name]
}

// snapshotChanged returns true if any file was added, removed, or modified
// between the two snapshots.
func snapshotChanged(prev, curr snapshot) bool {
	if len(prev) != len(curr) {
		return true
	}
	for path, modTime := range curr {
		if prevTime, ok := prev[path]; !ok || !prevTime.Equal(modTime) {
			return true
		}
	}
	return false
}
