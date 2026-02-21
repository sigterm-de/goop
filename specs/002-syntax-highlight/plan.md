# Implementation Plan: Editor Syntax Highlighting

**Branch**: `002-syntax-highlight` | **Date**: 2026-02-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-syntax-highlight/spec.md`

## Summary

Add automatic syntax highlighting to the goop editor. After each successful script execution, the editor content is analysed using a two-tier detector (structural heuristic + standard-library validation). If a format is identified with high confidence, the editor applies the corresponding GtkSourceView syntax highlighting and the status bar's new right-aligned syntax zone shows the language name. A user preference (default: enabled) controls the feature globally. The status bar is restructured into two independent zones: left for transient notifications (existing), right for the persistent syntax language indicator (new).

## Technical Context

**Language/Version**: Go 1.22+ (`CGO_ENABLED=1`, `CC=clang`)
**Primary Dependencies**:
- `libdb.so/gotk4-sourceview/pkg/gtksource/v5` — language manager, `Buffer.SetLanguage()`
- `github.com/diamondburned/gotk4/pkg/gtk/v4` — GTK4 UI
- `encoding/json`, `encoding/xml` (stdlib) — validation tier of detection
- `gopkg.in/yaml.v3` (already in go.mod) — YAML validation tier

**Storage**: `AppPreferences` JSON file via XDG (existing mechanism); one new field `syntax_auto_detect bool`
**Testing**: `go test -race ./...` with table-driven unit tests; 80% coverage gate
**Target Platform**: Linux + macOS (GTK4 desktop); no platform-specific code paths in this feature
**Performance Goals**: Detection completes in < 50 ms for content up to 1 MB (SC-005)
**Constraints**: All GTK API calls must remain on the main thread; detection runs synchronously on the main thread (content is small enough); zero new external dependencies
**Scale/Scope**: Single-user desktop application; detection state is in-process and ephemeral

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I — Code Quality | ✅ Pass | `gofmt`/`golangci-lint` will be enforced; all exported symbols will have doc comments; no magic values |
| II — Test-First | ✅ Pass | `detect_test.go` written before `detect.go`; `editor_test.go` / `statusbar_test.go` extended before production changes |
| III — Integration & Contract | ✅ Pass | New `tests/integration/syntax_detection_test.go` covers the full call path from script result → detection → editor highlight |
| IV — UX Consistency | ✅ Pass | Status bar split follows existing box/label pattern; no new CLI flags; no stdout output |
| V — Simplicity & YAGNI | ✅ Pass | `ui.Detect()` is a single focused function; stdlib only for parsing; no new external dependencies; no speculative abstractions |

**Complexity Tracking**: No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/002-syntax-highlight/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # N/A — desktop UI feature, no API contracts
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/ui/
├── detect.go           # New: two-tier content format detection (pure Go, no CGO)
├── detect_test.go      # New: table-driven unit tests for detection logic
├── editor.go           # Modified: add SetLanguage(langID), ClearLanguage()
├── statusbar.go        # Modified: split into notification zone + syntax zone
└── scriptpicker.go     # Modified: accept postScript func() callback; call after success

internal/app/
├── preferences.go      # Modified: add SyntaxAutoDetect bool field (default true)
├── settings.go         # Modified: add auto-detect checkbox to Editor section
└── window.go           # Modified: wire detection callback into ScriptPicker constructor

tests/integration/
└── syntax_detection_test.go   # New: end-to-end integration test
```

**Structure Decision**: Single project layout (existing). All changes are additive modifications to existing packages; no new packages required. The detection logic lives in `internal/ui` because it is consumed only by UI layer code and has no engine dependency.

## Phase 0: Research

See [research.md](research.md).

## Phase 1: Design

See [data-model.md](data-model.md) and [quickstart.md](quickstart.md).

No API contracts — this feature adds no HTTP endpoints, CLI commands, or inter-process interfaces.
All boundaries are internal Go function signatures documented in `quickstart.md`.
