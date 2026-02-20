# Contract: Scripts Library Interface

**Date**: 2026-02-20
**Branch**: `001-boop-core`
**Package**: `internal/scripts`
**Implements**: FR-002, FR-004, FR-005, FR-006, FR-013, FR-017

This contract defines the public interface of the `internal/scripts` package. It is the
acceptance test specification for `tests/contract/` and `tests/integration/library_test.go`.

---

## Package Responsibility

The `scripts` package owns:
1. Discovering and loading embedded (built-in) scripts from `assets/scripts/*.js`.
2. Discovering and loading user-provided scripts from the XDG config scripts directory.
3. Parsing the `/**!` header block to produce `Script` metadata.
4. Exposing the combined `ScriptLibrary` for display and fuzzy search.
5. Distinguishing built-in from user-provided scripts.

The `scripts` package does NOT own:
- JavaScript execution (that is `internal/engine`).
- UI display (that is `internal/ui`).
- Log file management (that is `internal/logging`).

---

## Types

### `ScriptSource`

```go
type ScriptSource int

const (
    BuiltIn      ScriptSource = iota // Embedded via go:embed at compile time
    UserProvided                      // Loaded from user config directory at startup
)
```

### `Script`

```go
type Script struct {
    Name        string
    Description string
    Icon        string    // FontAwesome HTML; empty string if not declared
    Tags        []string  // Empty slice if not declared
    Bias        float64   // Default 0.0
    Source      ScriptSource
    FilePath    string    // Virtual path for built-ins; absolute path for user scripts
    Content     string    // Full JavaScript source
}
```

### `LoadResult`

```go
type LoadResult struct {
    Scripts      []Script // Successfully loaded scripts (all sources)
    SkippedFiles []string // Paths of files that were skipped (with reasons logged)
    BuiltInCount int
    UserCount    int
}
```

---

## Interface

```go
// Loader discovers and parses scripts from embedded assets and the user scripts directory.
type Loader interface {
    // Load reads all built-in and user scripts. Errors in individual files are
    // logged and skipped; Load MUST NOT return an error for a single bad script file.
    // Load MUST return an error only for system-level failures (e.g., can't read
    // the embedded filesystem, config directory completely inaccessible).
    Load(userScriptsDir string) (LoadResult, error)
}

// Library provides the combined searchable set of scripts.
type Library interface {
    // All returns all scripts sorted by Bias ascending then Name ascending.
    // Built-in scripts precede user-provided scripts when Bias values are equal.
    All() []Script

    // Search returns scripts matching query using fuzzy matching on Name and Tags.
    // Returns All() when query is empty. Results are sorted by match score descending.
    Search(query string) []Script

    // Len returns the total number of loaded scripts.
    Len() int
}
```

---

## Header Parsing Contract

### Valid header

A file is a valid Boop script if and only if:
1. Its content starts with the exact string `/**!` (possibly preceded only by a UTF-8 BOM).
2. The `@name` field is present and non-empty after trimming.
3. The `@description` field is present and non-empty after trimming.

### Parsing rules

- Scan lines within the `/**!` ... `*/` block.
- For each line, match the pattern: optional whitespace, `*`, optional whitespace, `@key`, whitespace, value to end of line.
- `@name`: everything after `@name` trimmed.
- `@description`: everything after `@description` trimmed.
- `@icon`: everything after `@icon` trimmed (may contain HTML).
- `@tags`: split on `,`, trim each element; empty elements discarded.
- `@bias`: parse as `float64`; on parse error default to `0.0`.
- Unknown `@` keys are silently ignored.

### Skip conditions (logged as WARN)

A file MUST be skipped and logged as a warning (not an error) when:
- Does not start with `/**!`.
- Has `/**!` but `@name` or `@description` is missing or empty.
- Has `/**!` and valid metadata but contains a JavaScript syntax error (syntax check at
  load time — not execution; the file is skipped, not crashed on).

### Conflict handling

When a user script and a built-in script share the same `Name`:
- Both MUST be loaded.
- Both MUST appear in the library.
- Both MUST be visually distinguished in the UI (FR-013).
- Neither shadows or replaces the other.

---

## Library Ordering Contract

`All()` returns scripts ordered by:
1. `Bias` ascending (lower Bias = earlier in list).
2. Within equal Bias: built-in scripts before user-provided scripts.
3. Within equal Bias and Source: `Name` alphabetically ascending (case-insensitive).

`Search(query)` returns scripts ordered by fuzzy match score descending. Ties broken
by the same ordering as `All()`.

---

## Contract Test Cases

### TC-L-01: Embedded scripts are all loaded

```
Given: application is built with all Boop scripts in assets/scripts/
When:  Loader.Load(userScriptsDir="") is called
Then:  LoadResult.BuiltInCount >= 60
       All scripts in LoadResult.Scripts have Source == BuiltIn
       Every Script has non-empty Name and Description
```

### TC-L-02: User scripts are loaded from directory

```
Given: userScriptsDir contains a valid script file "my-script.js" with /**! header
When:  Loader.Load(userScriptsDir) is called
Then:  LoadResult.UserCount == 1
       LoadResult.Scripts contains a Script with Source == UserProvided
       That Script's Name matches the @name field in the file
```

### TC-L-03: Invalid files are skipped without failure

```
Given: userScriptsDir contains:
       - "valid.js" (valid Boop script)
       - "no-header.js" (valid JS but no /**! header)
       - "missing-name.js" (/**! header but no @name)
When:  Loader.Load(userScriptsDir) is called
Then:  LoadResult.UserCount == 1  (only valid.js loaded)
       LoadResult.SkippedFiles contains "no-header.js" and "missing-name.js"
       No error is returned
```

### TC-L-04: Non-existent user scripts directory does not fail

```
Given: userScriptsDir path does not exist
When:  Loader.Load(userScriptsDir) is called
Then:  No error is returned
       LoadResult.UserCount == 0
       Built-in scripts are still available
```

### TC-L-05: Empty search returns all scripts

```
Given: Library with N scripts loaded
When:  Library.Search("") is called
Then:  Result length == N
       Order matches Library.All()
```

### TC-L-06: Search filters by name

```
Given: Library with scripts including one named "Base64 Encode"
When:  Library.Search("base64") is called
Then:  "Base64 Encode" is in the result
       Result is sorted by match score descending
```

### TC-L-07: Search with no matches returns empty slice (not nil)

```
Given: Library with scripts
When:  Library.Search("zzznomatch999") is called
Then:  Result is an empty []Script (len == 0, not nil)
```

### TC-L-08: Name collision — both scripts loaded

```
Given: A built-in script with Name=="Sort Lines"
       A user script with Name=="Sort Lines"
When:  Loader.Load(userScriptsDir) is called
Then:  LoadResult.Scripts contains exactly 2 scripts with Name=="Sort Lines"
       One has Source==BuiltIn, one has Source==UserProvided
```

### TC-L-09: Ordering — bias controls position

```
Given: Script A with Name="Z-Script", Bias=-1.0 (built-in)
       Script B with Name="A-Script", Bias=0.0  (built-in)
When:  Library.All() is called
Then:  Script A appears before Script B (lower bias wins over alphabetical name order)
```

### TC-L-10: Metadata parsing — all fields

```
Given: A script file with header:
  /**!
   * @name          My Script
   * @description   Does something
   * @icon          <i class="fas fa-star"></i>
   * @tags          foo,bar, baz
   * @bias          -2.5
   */
When:  File is loaded
Then:  Script.Name == "My Script"
       Script.Description == "Does something"
       Script.Icon == `<i class="fas fa-star"></i>`
       Script.Tags == ["foo", "bar", "baz"]
       Script.Bias == -2.5
```
