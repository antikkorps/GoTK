package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// StoreFileName is the name of the observation store file.
	StoreFileName = ".gotk-learn.jsonl"
	// MaxStoreSize is the maximum store file size before rotation (1MB).
	MaxStoreSize = 1 * 1024 * 1024
)

// Observation is a single recorded line from command output.
type Observation struct {
	Timestamp  string `json:"ts"`
	Command    string `json:"cmd"`
	Normalized string `json:"norm"`
	Level      int    `json:"level"` // classify.Level as int
	SessionID  string `json:"session"`
}

// StoreWrite appends observations to the JSONL store file.
// It creates parent directories if needed and rotates when the file exceeds MaxStoreSize.
func StoreWrite(storePath string, observations []Observation) error {
	dir := filepath.Dir(storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("learn: create dir %s: %w", dir, err)
	}

	// Check if rotation is needed before writing
	if info, err := os.Stat(storePath); err == nil && info.Size() > MaxStoreSize {
		rotateStore(storePath)
	}

	f, err := os.OpenFile(storePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("learn: open %s: %w", storePath, err)
	}
	defer f.Close() //nolint:errcheck

	for _, obs := range observations {
		data, err := json.Marshal(obs)
		if err != nil {
			continue
		}
		fmt.Fprintf(f, "%s\n", data) //nolint:errcheck
	}
	return nil
}

// StoreRead reads all observations from the JSONL store file.
func StoreRead(storePath string) ([]Observation, error) {
	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var observations []Observation
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obs Observation
		if err := json.Unmarshal([]byte(line), &obs); err != nil {
			continue // skip malformed lines
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

// StoreStats returns summary statistics about the observation store.
type StoreStats struct {
	TotalObservations int
	Sessions          int
	Commands          int
	OldestEntry       string
	NewestEntry       string
	FileSize          int64
}

// StoreStat returns statistics about the store file.
func StoreStat(storePath string) (*StoreStats, error) {
	info, err := os.Stat(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StoreStats{}, nil
		}
		return nil, err
	}

	observations, err := StoreRead(storePath)
	if err != nil {
		return nil, err
	}

	sessions := make(map[string]bool)
	commands := make(map[string]bool)
	var oldest, newest string

	for _, obs := range observations {
		sessions[obs.SessionID] = true
		commands[obs.Command] = true
		if oldest == "" || obs.Timestamp < oldest {
			oldest = obs.Timestamp
		}
		if newest == "" || obs.Timestamp > newest {
			newest = obs.Timestamp
		}
	}

	return &StoreStats{
		TotalObservations: len(observations),
		Sessions:          len(sessions),
		Commands:          len(commands),
		OldestEntry:       oldest,
		NewestEntry:       newest,
		FileSize:          info.Size(),
	}, nil
}

// StoreClear removes the store file.
func StoreClear(storePath string) error {
	err := os.Remove(storePath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DefaultStorePath returns the store path next to .gotk.toml if found,
// or in the current working directory.
func DefaultStorePath() string {
	if projectDir := findProjectDir(); projectDir != "" {
		return filepath.Join(projectDir, StoreFileName)
	}
	return StoreFileName
}

// findProjectDir walks up from cwd looking for .gotk.toml and returns its directory.
func findProjectDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 50; i++ {
		candidate := filepath.Join(dir, ".gotk.toml")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// NewSessionID returns a unique session identifier.
func NewSessionID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixMilli(), os.Getpid())
}

// rotateStore keeps the most recent half of entries.
// Uses atomic file ops (write to temp file + rename) to prevent data loss.
func rotateStore(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	half := len(lines) / 2
	kept := strings.Join(lines[half:], "\n") + "\n"

	// Atomic write: temp file + rename to prevent corruption
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(kept), 0600); err == nil {
		_ = os.Rename(tmpPath, path)
	}
}
