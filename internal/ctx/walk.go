package ctx

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// excludeDirs are directories skipped entirely during walk.
var excludeDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"coverage":     true,
	".next":        true,
	".cache":       true,
	".idea":        true,
	".vscode":      true,
}

// excludeFiles are filenames skipped during walk (lock files, etc.).
var excludeFiles = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"go.sum":            true,
	"Cargo.lock":        true,
	"pnpm-lock.yaml":    true,
	"composer.lock":     true,
	"Gemfile.lock":      true,
	"poetry.lock":       true,
}

// excludeExtensions are file extensions always skipped (lock files by extension).
var excludeExtensions = map[string]bool{
	".lock": true,
}

// binaryExtensions are file extensions treated as binary and skipped.
var binaryExtensions = map[string]bool{
	".exe": true, ".bin": true, ".o": true, ".a": true, ".so": true,
	".dylib": true, ".dll": true, ".class": true, ".jar": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".svg": true, ".webp": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".pyc": true, ".pyo": true, ".wasm": true,
}

// WalkFiles returns a list of file paths under opts.Dir that pass all filters.
func WalkFiles(opts Options) ([]string, error) {
	var files []string

	err := filepath.WalkDir(opts.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}

		name := d.Name()

		// Skip excluded directories
		if d.IsDir() {
			if excludeDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip excluded files by name
		if excludeFiles[name] {
			return nil
		}

		// Skip by extension
		ext := strings.ToLower(filepath.Ext(name))
		if excludeExtensions[ext] {
			return nil
		}

		// Skip binary files
		if binaryExtensions[ext] {
			return nil
		}

		// File type filter (e.g., -t go only includes .go files)
		if len(opts.FileTypes) > 0 {
			matched := false
			for _, ft := range opts.FileTypes {
				if ext == "."+ft {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		// Glob filter
		if opts.Glob != "" {
			matched, _ := filepath.Match(opts.Glob, name)
			if !matched {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}
