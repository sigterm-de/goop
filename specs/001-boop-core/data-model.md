# Data Model: goop — Linux Text Transformation Tool

**Date**: 2026-02-20
**Branch**: `001-boop-core`

---

## Entities

### Script

Represents a single available transformation. Produced by the script loader from
either an embedded file or a file on disk.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `Name` | string | Required; non-empty | Display name parsed from `@name` header field |
| `Description` | string | Required; non-empty | Short description from `@description` header |
| `Icon` | string | Optional; default `""` | FontAwesome HTML snippet from `@icon` header |
| `Tags` | []string | Optional; default `[]` | Keywords from `@tags` header (comma-split, trimmed) |
| `Bias` | float64 | Optional; default `0.0` | Sort weight from `@bias`; lower = higher in list |
| `Source` | ScriptSource | Required | `BuiltIn` or `UserProvided` |
| `FilePath` | string | Required | Absolute path for user scripts; virtual path for built-ins |
| `Content` | string | Required; non-empty | Full JavaScript source text |

**ScriptSource** (enum): `BuiltIn | UserProvided`

**Identity**: Scripts are identified by `(Source, FilePath)`. Two scripts may share the
same `Name`; they are always distinct entities. When displayed, `UserProvided` scripts
show a visual badge to distinguish them (FR-013).

**Validation rules**:
- `Name` and `Description` MUST be non-empty for a script to be loaded.
- File MUST start with `/**!` header block; files without it are silently skipped.
- `Content` MUST be parsable JavaScript (syntax check at load time).
- Scripts failing validation are logged and skipped; remaining scripts load normally.

**State transitions**:
```
[File on disk] → parse metadata → [Script] → load into library → [Available]
[Invalid file] → validation fail → [Skipped] (logged)
```

---

### ScriptLibrary

The full collection of scripts available at runtime. Composed at startup; immutable
until the application restarts (FR-017 — no hot reload).

| Field | Type | Description |
|-------|------|-------------|
| `All` | []Script | All loaded scripts, ordered by `Bias` then `Name` |
| `BuiltInCount` | int | Number of embedded scripts |
| `UserCount` | int | Number of successfully loaded user scripts |

**Operations**:
- `Search(query string) []Script` — returns scripts matching `query` via fuzzy search;
  returns `All` when `query` is empty; results ordered by match score (FR-006).
- `ByIndex(i int) Script` — retrieve script at position `i` in `All`.

**Loading order**: Built-in scripts first (sorted by `Bias` then `Name`), followed by
user scripts (sorted by `Bias` then `Name`). This ensures built-in scripts appear at
the top of an unfiltered list.

---

### ScriptState

The mutable object passed to a script's `main(state)` function during execution.
Exposes the editor text and receives the script's output. Also tracks whether
`postError` was called, which triggers rollback.

| Field | Type | JS property | Description |
|-------|------|-------------|-------------|
| `FullText` | string | `state.fullText` | Complete editor content at execution start |
| `Text` | string | `state.text` | Selected text if any; full text if no selection |
| `Selection` | Selection | `state.selection` | Caret/selection bounds |
| `errorPosted` | bool | (internal) | Set true when `postError()` is called |
| `errorMessage` | string | (internal) | Error message from `postError()` |
| `insertPending` | string | (internal) | Text queued by `insert()` |

**Selection** sub-type:

| Field | Type | JS property | Description |
|-------|------|-------------|-------------|
| `Start` | int | `state.selection.start` | 0-based offset of selection start |
| `End` | int | `state.selection.end` | 0-based offset of selection end (equals Start if no selection) |

**Write semantics** (applied after `main()` returns, in priority order):
1. If `errorPosted == true`: discard all mutations; show `errorMessage` in UI; log it.
2. Else if `Text` was mutated: replace selection range (or full text if no selection).
3. Else if `FullText` was mutated: replace full document.
4. Else if `insertPending != ""`: insert at cursor position.
5. Else: no change (script read-only / informational).

**ScriptState is not persisted**. It is created fresh for each script execution from
the current editor content and discarded after the result is applied.

---

### TransformationResult

Produced by the engine after executing a script. Carries the outcome back to the UI.

| Field | Type | Description |
|-------|------|-------------|
| `Success` | bool | True if script completed without `postError` or unhandled exception |
| `NewFullText` | string | Replacement text for the full document (when `FullText` mutated) |
| `NewText` | string | Replacement text for selection range (when `Text` mutated) |
| `InsertText` | string | Text to insert at cursor (when `insert()` called) |
| `MutationKind` | MutationKind | Which of the four write semantics applies |
| `ErrorMessage` | string | Error message when `Success == false` |
| `ScriptName` | string | Name of the script that was run |

**MutationKind** (enum): `None | ReplaceSelection | ReplaceDocument | InsertAtCursor`

---

### LogEntry

Represents a single line written to `~/.config/goop/goop.log`.

| Field | Type | Description |
|-------|------|-------------|
| `Timestamp` | time.Time | UTC time of the event |
| `Level` | LogLevel | `INFO`, `WARN`, or `ERROR` |
| `ScriptName` | string | Script name; empty for non-script events (e.g., startup) |
| `Message` | string | Human-readable description of the event |

**LogLevel** (enum): `INFO | WARN | ERROR`

**Format** (one line per entry):
```
2026-02-20T14:23:01.004Z [WARN ] script=JSONPrettify Cannot find module '@boop/unknown'
2026-02-20T14:23:05.182Z [ERROR] script=MyBrokenScript Script execution timed out after 5s
```

Log file is appended to across sessions. No rotation in v1. Max file size is not
enforced in v1.

---

### UserConfiguration

User-controlled settings resolved at startup. Not persisted to a config file in v1 —
all values are derived from XDG environment variables and command-line flags.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ScriptsDir` | string | `~/.config/goop/scripts/` | Directory scanned for user `.js` scripts |
| `LogFilePath` | string | `~/.config/goop/goop.log` | Append-mode log file |
| `ScriptTimeout` | time.Duration | `5s` | Hard timeout for script execution (not user-configurable in v1) |

**Resolution**: `ScriptsDir` and `LogFilePath` are derived from `xdg.ConfigHome`. If
`XDG_CONFIG_HOME` is set, these paths change accordingly. The `ScriptsDir` is created
automatically if it does not exist (FR-005).

---

## Entity Relationships

```text
UserConfiguration
  └─ ScriptsDir ──────────────┐
                               ▼
ScriptLibrary ◄── loads ── Script (UserProvided, *n)
ScriptLibrary ◄── loads ── Script (BuiltIn, *~65)
ScriptLibrary.Search() ─────► []Script (filtered)

Window ──── selects ───► Script
         ──── reads ────► ScriptState (constructed from editor content)
         ──── runs ─────► Engine.Execute(script, state) ──► TransformationResult
         ──── applies ──► editor update (based on MutationKind)
         ──── logs ─────► LogEntry (on error/warn)
```

---

## State Transitions — Script Execution

```
[Idle]
  │ user selects script
  ▼
[Executing]
  │ engine runs main(state)
  ├─ success ──────────────► [Applying] ──► editor updated ──► [Idle]
  ├─ postError() called ───► [Error]    ──► text unchanged, error shown ──► [Idle]
  ├─ unhandled JS throw ───► [Error]    ──► text unchanged, error shown ──► [Idle]
  └─ timeout (5s) ──────────► [Error]   ──► text unchanged, timeout shown ──► [Idle]

During [Executing]: editor + script list are disabled; progress indicator is shown.
During [Error]: error message shown in UI; path to log file appended to message.
```
