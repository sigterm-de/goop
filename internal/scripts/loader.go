package scripts

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"codeberg.org/daniel-ciaglia/goop/internal/logging"
)

// LoadResult is the combined outcome of loading built-in and user scripts.
type LoadResult struct {
	Scripts      []Script // Successfully loaded scripts from all sources
	SkippedFiles []string // Paths/names of files that were skipped (invalid or non-script)
	BuiltInCount int
	UserCount    int
}

// Loader discovers and parses scripts from an embedded asset FS and the user
// scripts directory.
type Loader interface {
	// Load reads all built-in and user scripts. Errors in individual files are
	// logged and skipped; Load MUST NOT return an error for a single bad script file.
	// Load returns an error only for system-level failures.
	Load(userScriptsDir string) (LoadResult, error)
}

type loader struct {
	builtinFS fs.FS // points at the scripts/ directory from assets.Scripts()
}

// NewLoader returns a production Loader backed by the provided embedded FS.
// Pass assets.Scripts() as builtinFS.
func NewLoader(builtinFS fs.FS) Loader {
	return &loader{builtinFS: builtinFS}
}

// Load implements Loader.
func (l *loader) Load(userScriptsDir string) (LoadResult, error) {
	var result LoadResult

	// ── Built-in scripts ──────────────────────────────────────────────────────
	if err := l.loadBuiltIns(&result); err != nil {
		return result, err
	}

	// ── User scripts ──────────────────────────────────────────────────────────
	if userScriptsDir != "" {
		l.loadUserScripts(userScriptsDir, &result)
	}

	return result, nil
}

func (l *loader) loadBuiltIns(result *LoadResult) error {
	return fs.WalkDir(l.builtinFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entry
		}
		// Skip the lib/ subdirectory — those are @boop/ module files, not scripts.
		if d.IsDir() {
			if d.Name() == "lib" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".js" {
			return nil
		}

		data, readErr := fs.ReadFile(l.builtinFS, path)
		if readErr != nil {
			logging.Log(logging.WARN, path, "cannot read embedded script: "+readErr.Error())
			result.SkippedFiles = append(result.SkippedFiles, path)
			return nil
		}

		script, parseErr := ParseHeader(string(data))
		if parseErr != nil {
			logging.Log(logging.WARN, path, "skipping: "+parseErr.Error())
			result.SkippedFiles = append(result.SkippedFiles, path)
			return nil
		}

		script.Source = BuiltIn
		script.FilePath = fmt.Sprintf("embedded:%s", path)
		result.Scripts = append(result.Scripts, script)
		result.BuiltInCount++
		return nil
	})
}

func (l *loader) loadUserScripts(dir string, result *LoadResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Log(logging.INFO, "", "user scripts dir does not exist: "+dir)
			return
		}
		logging.Log(logging.WARN, "", "cannot read user scripts dir: "+err.Error())
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".js" {
			continue
		}

		absPath := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			logging.Log(logging.WARN, entry.Name(), "cannot read user script: "+readErr.Error())
			result.SkippedFiles = append(result.SkippedFiles, entry.Name())
			continue
		}

		script, parseErr := ParseHeader(string(data))
		if parseErr != nil {
			logging.Log(logging.WARN, entry.Name(), "skipping: "+parseErr.Error())
			result.SkippedFiles = append(result.SkippedFiles, entry.Name())
			continue
		}

		script.Source = UserProvided
		script.FilePath = absPath
		result.Scripts = append(result.Scripts, script)
		result.UserCount++
	}
}
