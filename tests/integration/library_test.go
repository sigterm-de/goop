// Package integration contains end-to-end tests for the scripts package.
// Test cases map to TC-L-xx entries in contracts/library.md.
package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"codeberg.org/sigterm-de/goop/assets"
	"codeberg.org/sigterm-de/goop/internal/scripts"
)

func newLoader() scripts.Loader {
	return scripts.NewLoader(assets.Scripts())
}

// TC-L-01: All embedded scripts are loaded.
func TestTC_L01_EmbeddedScriptsLoaded(t *testing.T) {
	result, err := newLoader().Load("")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if result.BuiltInCount < 60 {
		t.Errorf("expected BuiltInCount >= 60, got %d", result.BuiltInCount)
	}
	for _, s := range result.Scripts {
		if s.Source != scripts.BuiltIn {
			t.Errorf("non-builtin script in result when no userDir given: %s", s.Name)
		}
		if s.Name == "" {
			t.Errorf("script with empty Name: %+v", s)
		}
		if s.Description == "" {
			t.Errorf("script with empty Description: %s", s.Name)
		}
	}
	if len(result.SkippedFiles) != 0 {
		t.Errorf("unexpected skipped files: %v", result.SkippedFiles)
	}
}

// TC-L-02: User scripts are loaded from directory.
func TestTC_L02_UserScriptsLoaded(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "my-script.js", validScript("My Script", "Does something"))

	result, err := newLoader().Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if result.UserCount != 1 {
		t.Fatalf("expected UserCount=1, got %d", result.UserCount)
	}
	found := false
	for _, s := range result.Scripts {
		if s.Source == scripts.UserProvided && s.Name == "My Script" {
			found = true
		}
	}
	if !found {
		t.Error("user script 'My Script' not found in result")
	}
}

// TC-L-03: Invalid files are skipped without failure.
func TestTC_L03_InvalidFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "valid.js", validScript("Valid Script", "A valid one"))
	writeScript(t, dir, "no-header.js", `function main(state) {}`)
	writeScript(t, dir, "missing-name.js", "/**!\n * @description   No name\n */\nfunction main(state) {}")

	result, err := newLoader().Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if result.UserCount != 1 {
		t.Errorf("expected UserCount=1, got %d", result.UserCount)
	}
	skippedNames := result.SkippedFiles
	containsName := func(name string) bool {
		for _, s := range skippedNames {
			if filepath.Base(s) == name {
				return true
			}
		}
		return false
	}
	if !containsName("no-header.js") {
		t.Errorf("expected no-header.js in SkippedFiles; got %v", skippedNames)
	}
	if !containsName("missing-name.js") {
		t.Errorf("expected missing-name.js in SkippedFiles; got %v", skippedNames)
	}
}

// TC-L-04: Non-existent user scripts directory does not fail.
func TestTC_L04_MissingDirDoesNotFail(t *testing.T) {
	result, err := newLoader().Load("/tmp/goop-does-not-exist-12345")
	if err != nil {
		t.Fatalf("expected no error for missing dir, got: %v", err)
	}
	if result.UserCount != 0 {
		t.Errorf("expected UserCount=0, got %d", result.UserCount)
	}
	if result.BuiltInCount < 60 {
		t.Errorf("expected built-ins still available, got BuiltInCount=%d", result.BuiltInCount)
	}
}

// TC-L-05: Empty search returns all scripts.
func TestTC_L05_EmptySearchReturnsAll(t *testing.T) {
	result, _ := newLoader().Load("")
	lib := scripts.NewLibrary(result)

	all := lib.All()
	searched := lib.Search("")
	if len(searched) != len(all) {
		t.Errorf("Search('') returned %d, want %d", len(searched), len(all))
	}
}

// TC-L-06: Search filters by name.
func TestTC_L06_SearchFiltersByName(t *testing.T) {
	result, _ := newLoader().Load("")
	lib := scripts.NewLibrary(result)

	matches := lib.Search("base64")
	if len(matches) == 0 {
		t.Fatal("expected at least one result for 'base64'")
	}
	found := false
	for _, s := range matches {
		if s.Name == "Base64 Encode" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Base64 Encode' in results for 'base64', got: %v",
			scriptNames(matches))
	}
}

// TC-L-07: Search with no matches returns empty (non-nil) slice.
func TestTC_L07_NoMatchReturnsEmptySlice(t *testing.T) {
	result, _ := newLoader().Load("")
	lib := scripts.NewLibrary(result)

	matches := lib.Search("zzznomatch999")
	if matches == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d: %v", len(matches), scriptNames(matches))
	}
}

// TC-L-08: Name collision — both scripts loaded.
func TestTC_L08_NameCollisionBothLoaded(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "sort-lines.js", validScript("Sort lines", "Sorts lines"))

	result, err := newLoader().Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	count := 0
	for _, s := range result.Scripts {
		if s.Name == "Sort lines" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected 2 scripts named 'Sort lines' (1 builtin + 1 user), got %d", count)
	}
}

// TC-L-09: Ordering — bias controls position.
func TestTC_L09_BiasOrdering(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "z-script.js", biasScript("Z-Script", "Last alphabetically", -1.0))
	writeScript(t, dir, "a-script.js", validScript("A-Script", "First alphabetically"))

	result, err := newLoader().Load(dir)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	lib := scripts.NewLibrary(result)
	all := lib.All()

	zIdx, aIdx := -1, -1
	for i, s := range all {
		switch s.Name {
		case "Z-Script":
			zIdx = i
		case "A-Script":
			aIdx = i
		}
	}
	if zIdx < 0 || aIdx < 0 {
		t.Fatalf("could not find test scripts in library: %v", scriptNames(all))
	}
	if zIdx >= aIdx {
		t.Errorf("Z-Script (bias=-1) should appear before A-Script (bias=0), got zIdx=%d aIdx=%d",
			zIdx, aIdx)
	}
}

// TC-L-10: Metadata parsing — all fields.
func TestTC_L10_MetadataParsing(t *testing.T) {
	header := `/**!
 * @name          My Script
 * @description   Does something
 * @icon          <i class="fas fa-star"></i>
 * @tags          foo,bar, baz
 * @bias          -2.5
 */
function main(state) {}`

	script, err := scripts.ParseHeader(header)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}
	if script.Name != "My Script" {
		t.Errorf("Name: got %q, want %q", script.Name, "My Script")
	}
	if script.Description != "Does something" {
		t.Errorf("Description: got %q, want %q", script.Description, "Does something")
	}
	if script.Icon != `<i class="fas fa-star"></i>` {
		t.Errorf("Icon: got %q", script.Icon)
	}
	wantTags := []string{"foo", "bar", "baz"}
	if len(script.Tags) != len(wantTags) {
		t.Errorf("Tags length: got %d, want %d (%v)", len(script.Tags), len(wantTags), script.Tags)
	} else {
		for i, tag := range wantTags {
			if script.Tags[i] != tag {
				t.Errorf("Tags[%d]: got %q, want %q", i, script.Tags[i], tag)
			}
		}
	}
	if script.Bias != -2.5 {
		t.Errorf("Bias: got %v, want -2.5", script.Bias)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writeScript %s: %v", name, err)
	}
}

func validScript(name, description string) string {
	return "/**!\n * @name          " + name + "\n * @description   " + description + "\n */\n\nfunction main(state) { state.text = state.text; }"
}

func biasScript(name, description string, bias float64) string {
	return "/**!\n * @name          " + name + "\n * @description   " + description +
		"\n * @bias          " + fmt.Sprintf("%g", bias) + "\n */\n\nfunction main(state) {}"
}

func scriptNames(ss []scripts.Script) []string {
	names := make([]string, len(ss))
	for i, s := range ss {
		names[i] = s.Name
	}
	return names
}
