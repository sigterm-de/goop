# Contract: Engine Package Interface

**Date**: 2026-02-20
**Branch**: `001-boop-core`
**Package**: `internal/engine`
**Implements**: FR-007, FR-009, FR-010, FR-015, FR-016, FR-018

This contract defines the public interface of the `internal/engine` package. It is the
acceptance test specification for `tests/contract/script_api_test.go` and all unit
tests within `internal/engine/*_test.go`.

---

## Package Responsibility

The `engine` package owns:
1. Creating a sandboxed JavaScript VM (goja runtime).
2. Constructing the `state` object and exposing it to the script.
3. Executing `main(state)` with a hard timeout.
4. Returning a `TransformationResult` to the caller — never panicking.
5. Providing module factories for `@boop/plist` and `@boop/yaml`.

The `engine` package does NOT own:
- GTK/UI interactions (those are in `internal/ui` and `internal/app`).
- Script loading or metadata parsing (those are in `internal/scripts`).
- Log file management (that is in `internal/logging`).

---

## Types

### `ExecutionInput`

```go
type ExecutionInput struct {
    ScriptSource string        // Full JS source text of the script
    ScriptName   string        // Display name (used in error messages and log entries)
    FullText     string        // Current full editor content
    SelectionText string       // Selected text (equals FullText if no selection)
    SelectionStart int         // 0-based character offset of selection start
    SelectionEnd   int         // 0-based character offset of selection end
    Timeout       time.Duration // Hard execution timeout (typically 5s)
}
```

### `MutationKind`

```go
type MutationKind int

const (
    MutationNone          MutationKind = iota // Script made no changes
    MutationReplaceDoc                        // state.fullText was written
    MutationReplaceSelect                     // state.text was written
    MutationInsertAtCursor                    // state.insert() was called
)
```

### `ExecutionResult`

```go
type ExecutionResult struct {
    Success      bool         // False if postError or unhandled exception occurred
    MutationKind MutationKind // Which mutation (if any) to apply; only valid when Success==true
    NewFullText  string       // Valid when MutationKind == MutationReplaceDoc
    NewText      string       // Valid when MutationKind == MutationReplaceSelect
    InsertText   string       // Valid when MutationKind == MutationInsertAtCursor
    ErrorMessage string       // Human-readable error; valid when Success==false
    TimedOut     bool         // True when failure was caused by the timeout
}
```

---

## Interface

```go
// Executor runs a single JavaScript script against a given input.
// Implementations MUST be safe to call from any goroutine.
// Each call creates a fresh JS runtime — no state persists between calls.
type Executor interface {
    Execute(ctx context.Context, input ExecutionInput) ExecutionResult
}
```

### Behaviour Contract

1. **Never panics**: `Execute` MUST recover from any internal panic and return a
   `ExecutionResult{Success: false, ErrorMessage: "internal engine error: ..."}`.

2. **Timeout**: If `input.Timeout` elapses before `main(state)` returns,
   `Execute` MUST return `ExecutionResult{Success: false, TimedOut: true,
   ErrorMessage: "Script execution timed out after <N>s"}`.

3. **Sandboxed**: The JS runtime MUST NOT expose any file system, network, or
   environment variable access (see `contracts/script-api.md` — Prohibited globals).

4. **Fresh VM per call**: A new goja runtime MUST be created for each `Execute` call.
   No module-level JavaScript variables persist between calls.

5. **postError semantics**: If the JS script calls `state.postError(msg)`:
   - `Execute` MUST return `ExecutionResult{Success: false, ErrorMessage: msg}`.
   - All mutations made before `postError` was called MUST be discarded.

6. **Unhandled exception semantics**: If `main(state)` throws an unhandled exception:
   - `Execute` MUST return `ExecutionResult{Success: false, ErrorMessage: <exception message>}`.
   - All mutations are discarded (identical behavior to `postError`).

7. **Context cancellation**: If `ctx` is cancelled before execution completes, `Execute`
   MUST interrupt the VM and return a timeout/cancelled result. `ctx.Done()` MUST be
   checked in addition to the `input.Timeout` timer.

8. **Determinism**: Given identical `ExecutionInput`, `Execute` MUST return identical
   `ExecutionResult` (scripts are pure text transformations — no randomness, no time
   dependency beyond the timeout).

---

## Module Registration Contract

The `engine` package MUST register these `require()` paths before running any script:

| Path | Go implementation | Module |
|------|-------------------|--------|
| `@boop/plist` | `howett.net/plist` | parse XML plist ↔ Go map ↔ JS object |
| `@boop/yaml` | `gopkg.in/yaml.v3` | parse YAML ↔ Go map ↔ JS object |

All other `require()` paths MUST return `Error: Cannot find module '<path>'`.

The `require` function itself MUST be registered (not poisoned to `undefined`) so that
scripts calling `require('@boop/plist')` succeed. Only non-`@boop/` paths fail.

---

## Contract Test Cases

These input/output pairs are the minimum required passing tests for `internal/engine`.

### TC-E-01: Simple text transformation

```
Input:  FullText="hello", SelectionText="hello", SelectionStart=0, SelectionEnd=5
Script: function main(state) { state.text = state.text.toUpperCase(); }
Output: Success=true, MutationKind=MutationReplaceSelect, NewText="HELLO"
```

### TC-E-02: FullText mutation

```
Input:  FullText="abc", SelectionText="abc", SelectionStart=0, SelectionEnd=3
Script: function main(state) { state.fullText = "XYZ"; }
Output: Success=true, MutationKind=MutationReplaceDoc, NewFullText="XYZ"
```

### TC-E-03: postError discards mutations

```
Input:  FullText="hello"
Script: function main(state) { state.fullText = "REPLACED"; state.postError("oops"); }
Output: Success=false, ErrorMessage="oops", MutationKind=MutationNone
```

### TC-E-04: Unhandled exception discards mutations

```
Input:  FullText="hello"
Script: function main(state) { state.fullText = "REPLACED"; throw new Error("boom"); }
Output: Success=false, ErrorMessage contains "boom", MutationKind=MutationNone
```

### TC-E-05: Timeout

```
Input:  FullText="x", Timeout=100ms
Script: function main(state) { while(true) {} }
Output: Success=false, TimedOut=true, ErrorMessage contains "timed out"
```

### TC-E-06: @boop/yaml round-trip

```
Input:  FullText='name: Alice\nage: 30'
Script: |
  var yaml = require('@boop/yaml');
  function main(state) {
    var obj = yaml.parse(state.fullText);
    state.fullText = JSON.stringify(obj);
  }
Output: Success=true, NewFullText contains '"name":"Alice"' and '"age":30'
```

### TC-E-07: Prohibited global access

```
Input:  FullText="x"
Script: function main(state) { var x = require('fs'); }
Output: Success=false, ErrorMessage contains "Cannot find module"
```

### TC-E-08: insert() at cursor

```
Input:  FullText="", SelectionText="", SelectionStart=0, SelectionEnd=0
Script: function main(state) { state.insert("HELLO"); }
Output: Success=true, MutationKind=MutationInsertAtCursor, InsertText="HELLO"
```

### TC-E-09: No mutation (read-only script)

```
Input:  FullText="hello"
Script: function main(state) { var x = state.fullText.length; }
Output: Success=true, MutationKind=MutationNone
```

### TC-E-10: btoa/atob globals

```
Input:  FullText="hello"
Script: function main(state) { state.text = btoa(state.text); }
Output: Success=true, NewText="aGVsbG8="
```
