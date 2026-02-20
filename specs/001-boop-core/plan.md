# Implementation Plan: goop — Linux Text Transformation Tool

**Branch**: `001-boop-core` | **Date**: 2026-02-20 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-boop-core/spec.md`

## Summary

goop is a Linux desktop text-transformation tool — a Go re-implementation of the
macOS Boop app (https://github.com/IvanMathy/Boop). Users paste text into a
GtkSourceView-powered editor, select a script from a fuzzy-searchable overlay panel,
and the script transforms the text in place via a sandboxed goja JavaScript runtime.

The application embeds all original Boop scripts at compile time via `go:embed`, loads
additional user scripts from `~/.config/goop/scripts/` at startup, and logs errors
to `~/.config/goop/goop.log`. The UI is built with GTK4 (gotk4 bindings),
targeting X11 and Wayland on Arch/Manjaro. Releases are automated via
go-semantic-release + goreleaser on Woodpecker CI, producing zip archives for
`linux/amd64` and `linux/arm64`.

## Technical Context

**Language/Version**: Go 1.22+; `CGO_ENABLED=1` (required by gotk4 GTK4 bindings)

**Primary Dependencies**:
- `github.com/diamondburned/gotk4/pkg/gtk/v4` — GTK4 UI bindings (CGo-based)
- `github.com/diamondburned/gotk4-sourceview/pkg/gtksource/v5` — GtkSourceView editor
- `github.com/dop251/goja` — ES6+ JavaScript engine (pure Go, no CGo)
- `github.com/dop251/goja_nodejs/require` — CommonJS `require()` for `@boop/` modules
- `github.com/sahilm/fuzzy` — fuzzy-match scoring for the script picker
- `github.com/adrg/xdg` — XDG Base Directory Specification path resolution
- `gopkg.in/yaml.v3` — YAML parsing for `@boop/yaml` module implementation
- `howett.net/plist` — Plist parsing for `@boop/plist` module implementation

**Storage**: Files only — no database.
- Bundled scripts: embedded in binary via `//go:embed assets/scripts`
- User scripts: `~/.config/goop/scripts/*.js` (loaded at startup, not watched)
- Log file: `~/.config/goop/goop.log` (append across sessions)

**Testing**: Go stdlib `testing` package; table-driven unit tests (`*_test.go`) per
package; `tests/contract/` for script API verification; `tests/integration/` for
end-to-end transform flows; `go test -race -coverprofile=coverage.out ./...`;
`golangci-lint run`

**Target Platform**: Linux desktop — X11 and Wayland; primary target Arch/Manjaro with
GTK 4.12; compatible with Ubuntu 24.04 (GTK 4.12) and Fedora 39+ (GTK 4.12)

**Project Type**: Single Go binary, CGo (dynamically linked against GTK4 system libs)

**Performance Goals**:
- App ready for user input within 2 seconds of launch
- Script picker search results update within 50ms of each keystroke
- Script execution begins displaying progress within 100ms of script selection
- 5-second hard timeout on script execution

**Constraints**:
- CGo required; binary dynamically links GTK4 + GtkSourceView (see Complexity Tracking)
- No static linking of GTK4 (upstream does not support it)
- Cross-compilation requires native CI runner per target architecture
- No network access; no database; no persistent text state between sessions
- Scripts loaded once at startup — restart required to pick up new user scripts

**Scale/Scope**: Single-user desktop app; ~60–65 embedded scripts; unlimited user
scripts (practical limit ~1000 before search performance degrades); 5-second script
timeout; no concurrent users

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality | ✅ PASS | `gofmt` + `golangci-lint` enforced in CI; `%w` error wrapping throughout; exported symbols documented |
| II. Test-First | ✅ PASS | Tasks ordered: contract tests written first (must fail), then implementation, then integration tests |
| III. Integration & Contract Testing | ✅ PASS | `tests/contract/` covers script API; `tests/integration/` covers transform flows and library loading |
| IV. UX Consistency | ✅ PASS | `stdout` = output only; errors to UI + log; exit codes: 0=success, 1=user error, 2=internal; `.desktop` file provided |
| V. Simplicity & YAGNI | ✅ PASS | All 8 runtime dependencies trace to a functional requirement; no speculative abstractions; no optional feature flags |
| Technical Standards | ✅ PASS | Go 1.22+; `golangci-lint`; `gofmt`; `go vet ./...`; 80%+ coverage gate in CI |
| Development Workflow | ✅ PASS | Conventional commits; semantic release via go-semantic-release; goreleaser for packaging; Woodpecker CI |

**Post-Design Constitution Check**: Re-evaluated after Phase 1 artifact generation.

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality | ✅ PASS | Interfaces defined in contracts prevent god-objects; each package has single responsibility |
| II. Test-First | ✅ PASS | Contract files define acceptance criteria that map directly to test cases |
| III. Integration & Contract Testing | ✅ PASS | `contracts/script-api.md` defines input→output pairs; `contracts/engine.md` + `library.md` define component boundaries |
| IV. UX Consistency | ✅ PASS | UX flows defined in quickstart.md; error → log file → show path in UI message |
| V. Simplicity | ✅ PASS | No repository pattern, no DI framework, no event bus introduced |

## Project Structure

### Documentation (this feature)

```text
specs/001-boop-core/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── script-api.md    # JavaScript Script API contract (Boop compatibility)
│   ├── engine.md        # Go engine package interface
│   └── library.md       # Go scripts package interface
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── goop/
    └── main.go                  # Entry point: GTK app init, arg parsing, run

internal/
├── app/
│   ├── app.go                   # gtk.Application setup, activate callback, lifecycle
│   └── window.go                # ApplicationWindow: layout, widget wiring, state
├── ui/
│   ├── editor.go                # GtkSourceView wrapper: get/set text, selection, undo
│   ├── scriptpicker.go          # Overlay panel: fuzzy search entry + filtered list
│   └── statusbar.go             # Status/error message display strip
├── engine/
│   ├── engine.go                # goja VM: sandbox setup, timeout, run script
│   ├── state.go                 # ScriptState struct + JS-callable Insert/PostError
│   └── modules.go               # @boop/plist and @boop/yaml module factories
├── scripts/
│   ├── loader.go                # Discover embedded + user scripts; go:embed source
│   ├── metadata.go              # Parse /**! header comment block → Script struct
│   └── library.go               # ScriptLibrary: full list + fuzzy search
└── logging/
    └── logging.go               # Open/append log file; XDG path; structured entries

assets/
└── scripts/                     # Bundled Boop scripts (~65 .js files)
    └── *.js

tests/
├── contract/
│   ├── script_api_test.go       # State object: fullText/text/selection/insert/postError
│   └── modules_test.go          # @boop/plist and @boop/yaml: parse + stringify
└── integration/
    ├── transform_test.go         # Load script → apply to text → verify output
    └── library_test.go           # Script discovery: embedded + user scripts, fuzzy search

.desktop/
└── goop.desktop              # XDG .desktop entry

.goreleaser.yml                  # goreleaser v2: zip archives, linux/amd64 + arm64
.semrelrc.yml                    # go-semantic-release + provider-gitea plugin
.woodpecker.yml                  # CI: lint → test → semantic-release → goreleaser
.golangci.yml                    # golangci-lint rules
go.mod
go.sum
```

**Structure Decision**: Single Go application following the `cmd/` + `internal/`
convention. `cmd/goop/main.go` contains only the entry point; all application logic
is in `internal/` sub-packages with clear single-responsibility boundaries. Embedded
scripts live in `assets/scripts/` and are referenced via `//go:embed assets/scripts`
in `internal/scripts/loader.go`. Unit tests live alongside each package as `*_test.go`
files; integration and contract tests that span package boundaries live in `tests/`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| Binary dynamically links GTK4 + GtkSourceView (FR-008 amended) | gotk4 uses CGo to bind GTK4 C libraries; GTK4 does not support static linking | Pure-Go UI toolkits (Fyne, Gio) use OpenGL rendering and do not respect system GTK themes — incompatible with "integration into modern desktop environments" requirement |
| CGO_ENABLED=1 | Required by gotk4; no pure-Go GTK4 binding exists | No alternative: GTK4's C API must be called via CGo |
