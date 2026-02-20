---

description: "Task list for goop — Linux Text Transformation Tool"
---

# Tasks: goop — Linux Text Transformation Tool

**Input**: Design documents from `/specs/001-boop-core/`
**Prerequisites**: plan.md ✅ spec.md ✅ research.md ✅ data-model.md ✅ contracts/ ✅

**Tests**: Included per Constitution Principle II (Test-First — NON-NEGOTIABLE).
Contract tests from `contracts/engine.md` (TC-E-01→TC-E-10) and
`contracts/library.md` (TC-L-01→TC-L-10) MUST be written and verified to FAIL
before the corresponding implementation tasks are started.

**Organization**: Tasks are grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story?] Description with file path`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- Entry point: `cmd/goop/`
- Application logic: `internal/{app,ui,engine,scripts,logging}/`
- Embedded assets: `assets/scripts/`
- Contract tests: `tests/contract/`
- Integration tests: `tests/integration/`

---

## Phase 1: Setup

**Purpose**: Repository initialization, directory structure, project scaffolding, and
asset acquisition.

- [X] T001 Initialize Go module at `go.mod` with module path `codeberg.org/daniel-ciaglia/goop` and add all required dependencies: `github.com/diamondburned/gotk4`, `github.com/diamondburned/gotk4-sourceview`, `github.com/dop251/goja`, `github.com/dop251/goja_nodejs`, `github.com/sahilm/fuzzy`, `github.com/adrg/xdg`, `gopkg.in/yaml.v3`, `howett.net/plist`
- [X] T002 Create full directory structure per plan.md: `cmd/goop/`, `internal/app/`, `internal/ui/`, `internal/engine/`, `internal/scripts/`, `internal/logging/`, `assets/scripts/`, `tests/contract/`, `tests/integration/`, `.desktop/` — add a `.gitkeep` in each empty directory
- [X] T003 [P] Create `.golangci.yml` with lint rules: enable `errcheck`, `staticcheck`, `gosimple`, `govet`, `gofmt`, `goimports`, `misspell`, `unused`; set `run.timeout=5m`; exclude test files from `errcheck`
- [X] T004 [P] Copy all 65 Boop script files from `github.com/IvanMathy/Boop/Boop/Scripts/` into `assets/scripts/` (download or clone upstream repo and copy the `.js` files); verify each file starts with `/**!` header

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared types, interfaces, and infrastructure that MUST be complete before
any user story implementation can begin.

⚠️ **CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 Implement `internal/logging/logging.go`: define `LogLevel` int enum (`INFO=0`, `WARN=1`, `ERROR=2`), `LogEntry` struct (`Timestamp time.Time`, `Level LogLevel`, `ScriptName string`, `Message string`), `InitLogger(appName string) (logPath string, err error)` using `github.com/adrg/xdg` to open `~/.config/goop/goop.log` in `os.O_APPEND|os.O_CREATE|os.O_WRONLY` mode; `Log(level LogLevel, scriptName, message string)` writing ISO timestamp + level + fields; return the resolved log path for display in UI error messages
- [X] T006 Implement types in `internal/scripts/metadata.go`: `ScriptSource` int enum (`BuiltIn ScriptSource = iota`, `UserProvided`); `Selection` struct (`Start, End int`); `Script` struct with fields `Name`, `Description`, `Icon`, `Tags []string`, `Bias float64`, `Source ScriptSource`, `FilePath`, `Content string`
- [X] T007 Implement `ParseHeader(content string) (Script, error)` in `internal/scripts/metadata.go`: detect `/**!` prefix (return error if absent), scan lines for `@name`, `@description`, `@icon`, `@tags` (comma-split + trim), `@bias` (parse float64, default 0.0), ignore unknown `@`-keys, return error if `@name` or `@description` empty after trim; store full `content` in `Script.Content`
- [X] T008 [P] Implement `internal/engine/state.go`: `ScriptState` struct with json tags (`FullText`, `Text`, `Selection Selection`) plus unexported tracking fields (`fullTextMutated`, `textMutated`, `errorPosted bool`, `errorMessage string`, `insertText string`, `insertPending bool`); exported methods `Insert(text string)` (sets `insertText` + `insertPending=true`) and `PostError(msg string)` (sets `errorPosted=true`, `errorMessage=msg`); `Snapshot() string` returning `FullText` pre-execution copy; `Result() ExecutionResult` applying write-semantics priority table from `contracts/engine.md`
- [X] T009 [P] Implement types in `internal/engine/engine.go`: `MutationKind` int enum (`MutationNone`, `MutationReplaceDoc`, `MutationReplaceSelect`, `MutationInsertAtCursor`); `ExecutionInput` struct (`ScriptSource`, `ScriptName`, `FullText`, `SelectionText`, `SelectionStart`, `SelectionEnd int`, `Timeout time.Duration`); `ExecutionResult` struct (`Success bool`, `MutationKind`, `NewFullText`, `NewText`, `InsertText`, `ErrorMessage`, `ScriptName string`, `TimedOut bool`); `Executor` interface with `Execute(ctx context.Context, input ExecutionInput) ExecutionResult`
- [X] T010 Implement `internal/app/config.go`: `UserConfiguration` struct (`ScriptsDir`, `LogFilePath string`, `ScriptTimeout time.Duration`); `NewUserConfiguration() UserConfiguration` resolving paths via `github.com/adrg/xdg` (`xdg.ConfigHome + "/goop/scripts"` and `xdg.ConfigFile("goop/goop.log")`); call `os.MkdirAll(ScriptsDir, 0755)` to auto-create if absent; set `ScriptTimeout = 5 * time.Second`

**Checkpoint**: Foundation ready — shared types and infrastructure complete, user story phases can begin.

---

## Phase 3: User Story 1 — Apply a Text Transformation (Priority: P1) 🎯 MVP

**Goal**: User opens goop, pastes text, selects a script from the list, and sees the
transformed text replace their input immediately. Errors preserve original text.

**Independent Test**: Launch the app, paste `{"a":1}`, click "JSON Prettify", verify
editor shows `{\n    "a": 1\n}`. Apply an invalid JSON string to "JSON Prettify",
verify the original text is unchanged and an error message appears.

### Contract Tests for User Story 1 ⚠️

> **Write these tests FIRST — verify they FAIL before beginning T013**

- [X] T011 [P] [US1] Write all 10 engine contract test cases (TC-E-01 through TC-E-10) in `tests/contract/script_api_test.go` per `contracts/engine.md`: cover state mutations, postError rollback, unhandled exceptions, timeout, @boop/yaml round-trip, prohibited globals, `insert()`, no-mutation, `btoa`/`atob` — each as a table-driven sub-test; verify tests compile but FAIL before engine implementation
- [X] T012 [P] [US1] Write @boop/plist and @boop/yaml contract tests in `tests/contract/modules_test.go`: `TestYAMLParseThenStringify` (parse `"name: Alice\nage: 30"` → object, stringify → verify contains key/value), `TestPlistParseThenStringify` (parse XML plist → object → stringify → valid plist); verify tests FAIL before modules implementation

### Implementation for User Story 1

- [X] T013 [US1] Implement `@boop/yaml` module factory in `internal/engine/modules.go`: Go struct with `Parse(s string) (interface{}, error)` using `gopkg.in/yaml.v3` and `Stringify(v interface{}) (string, error)`; register on goja VM via `require.NewRegistry` WithLoader for path `"@boop/yaml"`; expose `parse` and `stringify` as JS-callable functions on the module exports object
- [X] T014 [P] [US1] Implement `@boop/plist` module factory in `internal/engine/modules.go`: Go struct with `Parse(s string) (interface{}, error)` and `Stringify(v interface{}) (string, error)` using `howett.net/plist`; register for path `"@boop/plist"`; expose `parse`, `stringify`, `parseBinary` on module exports
- [X] T015 [US1] Implement `Execute(ctx context.Context, input ExecutionInput) ExecutionResult` in `internal/engine/engine.go`: (1) `vm := goja.New()`, (2) `vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))`, (3) construct `ScriptState` from input and `vm.Set("state", vm.ToValue(&state))`, (4) poison globals: `require`, `process`, `global`, `Buffer`, `setTimeout`, `setInterval`, `fetch`, (5) register `@boop/` require registry, (6) register `btoa`/`atob` globals, (7) `goja.Compile(...)` for syntax-check before timer, (8) `time.AfterFunc(input.Timeout, func(){ vm.Interrupt(errTimeout) })`, (9) `vm.RunProgram(prog)`, (10) classify error types (`*goja.InterruptedError`, `*goja.Exception`), (11) return `state.Result()` on success; recover from panic and return error result
- [X] T016 [US1] Register `btoa` and `atob` globals in `internal/engine/engine.go`: implement as Go closures calling `base64.StdEncoding.EncodeToString` / `DecodeString` on `[]byte(input)`; set on VM via `vm.Set("btoa", ...)` and `vm.Set("atob", ...)` before running any script
- [X] T017 [US1] Implement `internal/scripts/loader.go`: add `//go:embed assets/scripts` directive; implement `Loader` interface `Load(userScriptsDir string) (LoadResult, error)`: iterate embedded FS files, call `ParseHeader`, append to `LoadResult.Scripts` with `Source=BuiltIn`; log and collect skipped files; do NOT error on individual bad files
- [X] T018 [US1] Implement `internal/scripts/library.go`: `ScriptLibrary` struct holding `[]Script`; `NewLibrary(result LoadResult) *ScriptLibrary`; `All() []Script` returning copy sorted by `Bias` asc then `Source` (BuiltIn before UserProvided) then `Name` asc (case-insensitive); `Len() int`; stub `Search(query string) []Script` returning `All()` for all queries (fuzzy implemented in US2)
- [X] T019 [US1] Implement `internal/ui/editor.go`: wrap `gtksource.View` + `gtksource.Buffer`; expose `GetFullText() string`, `SetFullText(text string)`, `GetSelectedText() string`, `GetSelection() (start, end int)`, `ReplaceSelection(text string)`, `InsertAtCursor(text string)`; implement single-level undo buffer: `SaveUndoSnapshot()` (called before each transform), `Undo() bool` (restores snapshot if available, returns true if undo was performed); `SetEnabled(bool)` to disable/enable editing during script execution
- [X] T020 [US1] Implement `internal/ui/statusbar.go`: `gtk.Label` inside a `gtk.Box`; `ShowError(message, logPath string)` sets label text to `message + " (log: " + logPath + ")"` and makes bar visible; `ShowSuccess(scriptName string)` shows brief success message; `Clear()` hides the bar; initially hidden
- [X] T021 [US1] Implement script list in `internal/ui/scriptpicker.go`: `gtk.ListView` backed by `gtk.StringList`; `SetScripts(scripts []Script)` populates list; each row shows script Name, Description, and a "user" badge for `UserProvided` scripts (FR-013); `OnScriptSelected(func(Script))` callback; no search entry yet (added in US2)
- [X] T022 [US1] Implement execution integration in `internal/ui/scriptpicker.go`: when a script is selected, call `editor.SaveUndoSnapshot()`, disable editor + script list via `SetEnabled(false)`, show spinner in status bar, launch goroutine calling `engine.Execute()`, use `glib.IdleAdd()` to marshal result back to main thread: apply mutation via editor methods, show error via status bar if `!result.Success`, re-enable editor + script list
- [X] T023 [US1] Implement `internal/app/window.go`: create `gtk.ApplicationWindow`; layout: `gtk.Overlay` with `editor.View` as base child and `gtk.Revealer` (containing script picker panel) as overlay; `gtk.Box` vertical root packing overlay + status bar; `SetDefaultSize(1000, 700)`; `ShowScriptPicker()` / `HideScriptPicker()` toggle the revealer; wire Ctrl+/ keyboard shortcut to toggle picker; expose `LogPath string` for status bar error messages
- [X] T024 [US1] Implement `internal/app/app.go`: `gtk.NewApplication("org.codeberg.daniel_ciaglia.go_boop", 0)`; `ConnectActivate`: call `NewUserConfiguration()`, initialize logging (capture `logPath`), create `Loader.Load(config.ScriptsDir)`, log any skipped scripts, create `NewLibrary()`, create `NewApplicationWindow(app, library, engine, logPath)`; `Run(app *gtk.Application) int` returning `app.Run(os.Args)`
- [X] T025 [US1] Implement `cmd/goop/main.go`: `func main()` calls `app.Run()` and `os.Exit(returnCode)`; declare `var (version, commit, date string)` for ldflags injection; print version info if `--version` flag passed

**Checkpoint**: US1 complete — user can open the app, paste text, click any bundled script,
see the result, undo the last transformation, and see errors with original text preserved.

---

## Phase 4: User Story 2 — Search and Browse the Script Library (Priority: P2)

**Goal**: User types a partial script name in the search field and the list filters in
real time to matching scripts. Clearing the search restores the full list.

**Independent Test**: Launch app, type "base64" in search field — only Base64 Encode
and Base64 Decode appear. Clear field — all 65+ scripts appear again.

### Integration Tests for User Story 2 ⚠️

> **Write these tests FIRST — verify they FAIL before beginning T027**

- [X] T026 [P] [US2] Write fuzzy-search integration tests in `tests/integration/library_test.go` covering TC-L-05, TC-L-06, TC-L-07 from `contracts/library.md`: empty query returns all scripts, "base64" query returns scripts with "Base64" in name, unmatched query returns empty (non-nil) slice; verify tests FAIL before Search implementation

### Implementation for User Story 2

- [X] T027 [US2] Implement `Search(query string) []Script` in `internal/scripts/library.go` using `github.com/sahilm/fuzzy`: implement `fuzzy.Source` interface on `scriptSource []Script` (`String(i int)` returns `s[i].Name`, `Len()` returns `len(s)`); call `fuzzy.FindFrom(query, scriptSource)` and map `Match.Index` back to original scripts in score-descending order; return empty non-nil slice when no matches; return `All()` when query is empty
- [X] T028 [US2] Add real-time search to `internal/ui/scriptpicker.go`: add `gtk.SearchEntry` above the script list; `ConnectChanged` callback calls `library.Search(entry.Text())` and calls `SetScripts(filtered)` to update the list model; when query cleared restore full list; when results empty show a "No matching scripts" label in place of the list

**Checkpoint**: US2 complete — script library is fully searchable; users can locate any
script by typing partial name.

---

## Phase 5: User Story 3 — Add a Custom Transformation Script (Priority: P3)

**Goal**: User places a valid `.js` file in `~/.config/goop/scripts/`, restarts the
app, and the new script appears in the picker list with a visual "user" badge.

**Independent Test**: Place a test script in the user scripts directory, relaunch the
app, verify the script appears in the list with the user badge, apply it, verify output.

### Integration Tests for User Story 3 ⚠️

> **Write these tests FIRST — verify they FAIL before beginning T030**

- [X] T029 [P] [US3] Write user-script loading integration tests in `tests/integration/library_test.go` covering TC-L-02, TC-L-03, TC-L-04, TC-L-08 from `contracts/library.md`: valid user script loaded + labelled UserProvided, invalid files skipped without error, missing dir returns no error, name collision loads both scripts; verify tests FAIL before user-script loading is implemented

### Implementation for User Story 3

- [X] T030 [US3] Implement user script loading in `internal/scripts/loader.go`: in `Load()`, after embedded scripts, read `*.js` files from `userScriptsDir` using `os.ReadDir`; for each file call `ParseHeader`, label `Source=UserProvided`, set `FilePath` to absolute path; on parse error call `logging.Log(WARN, filename, reason)` and append filename to `LoadResult.SkippedFiles` rather than returning error; handle non-existent dir gracefully (log info, continue)
- [X] T031 [US3] Implement visual badge for user-provided scripts in `internal/ui/scriptpicker.go`: in list row factory, check `script.Source == UserProvided`; if true append a `gtk.Label` with text "user" and a distinct CSS class (`user-script-badge`) using `gtk.NewLabel("user")` with `AddCSSClass("user-script-badge")`; add CSS provider to display the badge as a small coloured pill

**Checkpoint**: US3 complete — users can add custom scripts at runtime by placing files in
the XDG config directory and restarting the app.

---

## Phase 6: User Story 4 — Install and Run Without Setup (Priority: P4)

**Goal**: User downloads the binary, marks it executable, runs it — fully functional
with no additional install steps. Desktop integration via `.desktop` file.

**Independent Test**: Copy binary to a clean Arch/Manjaro VM with gtk4 and gtksourceview5
installed but no Go toolchain; run binary and verify app opens with all built-in scripts.

### Integration Tests for User Story 4 ⚠️

> **Write this test FIRST — verify it FAIL before beginning T033**

- [X] T032 [P] [US4] Write smoke-test in `tests/integration/transform_test.go`: use `go build` subprocess to produce the binary, then run it with `--version` flag (no display needed), assert exit code 0 and version string present; serves as build-pipeline regression gate

### Implementation for User Story 4

- [X] T033 [US4] Create `.desktop/goop.desktop`: XDG desktop entry with `[Desktop Entry]`, `Name=goop`, `Comment=Text transformation tool`, `Exec=goop`, `Icon=goop`, `Type=Application`, `Categories=Utility;TextTools;`, `Keywords=text;transform;boop;encode;decode;`; validate with `desktop-file-validate` if available
- [X] T034 [US4] Create `.goreleaser.yml`: goreleaser v2 config (`version: 2`); two build IDs (`goop-amd64`, `goop-arm64`) each with `CGO_ENABLED=1`, `goos: [linux]`, respective `goarch`, ldflags `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`; archives with `format: zip`, `name_template: "goop_{{.Version}}_{{.Os}}_{{.Arch}}"`, files `[LICENSE, README.md, SYSTEM_REQUIREMENTS.md]`; checksum `sha256`; Forgejo release section with `gitea: {owner: daniel-ciaglia, name: goop}`; changelog groups for feat/fix/perf, excluding chore/docs/ci/test
- [X] T035 [P] [US4] Create `.semrelrc.yml`: go-semantic-release config with `provider-plugins: [{name: provider-gitea, version: ">=1.0.0"}]`; plugins `commit-analyzer` and `changelog`; `condition-default` requiring `branch: main`; changelog groups for Features (feat), Bug Fixes (fix), Performance (perf)
- [X] T036 [P] [US4] Create `.woodpecker.yml`: CI pipeline with steps: `lint` (golangci-lint, push events), `test` (install libgtk-4-dev + libgtksourceview-5-dev + pkg-config via apt, run `go test -race -coverprofile=coverage.out ./...`, assert coverage ≥80%), `semantic-release` (ghcr.io/go-semantic-release image, push to main only, uses `CODEBERG_TOKEN` secret), `goreleaser-amd64` (tag events, `platform: linux/amd64`, `--ids goop-amd64`), `goreleaser-arm64` (tag events, `platform: linux/arm64`, `--ids goop-arm64`); set `GITEA_API=https://codeberg.org/api/v1`

**Checkpoint**: US4 complete — binary is distributable as a zip archive, CI produces
semantic releases automatically, and the app integrates with Linux desktop launchers.

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Hardening, documentation, and final validation across all user stories.

- [X] T037 [P] Write end-to-end transform integration tests in `tests/integration/transform_test.go`: `TestURLEncode` (input `"hello world"` → `"hello%20world"`), `TestJSONPrettify` (input `{"a":1}` → prettified), `TestBase64Encode` (known input → known output); load scripts from embedded FS, construct engine, verify round-trip results match expected
- [X] T038 Write TC-L-01 integration test in `tests/integration/library_test.go`: call `Loader.Load("")` (no user scripts dir) and assert `LoadResult.BuiltInCount >= 60`, all scripts have non-empty Name and Description, zero SkippedFiles
- [X] T039 [P] Create `SYSTEM_REQUIREMENTS.md`: GTK4 and GtkSourceView install instructions for Arch/Manjaro (`sudo pacman -S gtk4 gtksourceview5`), Debian/Ubuntu 24.04 (`sudo apt install libgtk-4-1 libgtksourceview-5-0`), Fedora 39+ (`sudo dnf install gtk4 gtksourceview5`); note that the embedded Boop scripts require no external files
- [X] T040 [P] Update `README.md` with: project description, system requirements section (link to SYSTEM_REQUIREMENTS.md), quick build instructions (`CGO_ENABLED=1 go build ./cmd/goop`), user guide (paste text, pick script, undo, add custom scripts), custom script authoring reference (header format + state API quick-ref)
- [X] T041 [P] Run `golangci-lint run ./...` and resolve all warnings in all packages; ensure `gofmt -l .` produces no output; run `go vet ./...` and resolve any issues
- [X] T042 [P] Run `go test -race -coverprofile=coverage.out ./...` and `go tool cover -func=coverage.out`; identify any package below 80% coverage and add targeted unit tests in `*_test.go` files alongside the package to reach the coverage gate

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Foundational completion — no dependency on US2/US3/US4
- **US2 (Phase 4)**: Depends on US1 completion (Search builds on Library from US1)
- **US3 (Phase 5)**: Depends on Foundational completion; can run in parallel with US2
- **US4 (Phase 6)**: Depends on Foundational completion; can run in parallel with US2/US3
- **Polish (Phase N)**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational — no dependency on other user stories
- **US2 (P2)**: Depends on US1 (the library's `All()` and `ScriptLibrary` struct are needed)
- **US3 (P3)**: Depends on Foundational only (loader is extended, not rebuilt)
- **US4 (P4)**: Depends on Foundational only (release pipeline is independent of UI)

### Within Each User Story

- Contract/integration tests MUST be written and verified to FAIL before implementation
- Engine modules (T013, T014) before Executor (T015)
- State types (T008) before Executor (T015)
- Script loader (T017) before library (T018)
- Library (T018) before script picker (T021)
- Editor (T019), status bar (T020), script picker (T021) can be built in parallel
- Script picker execution (T022) after editor (T019) + status bar (T020) + picker (T021)
- Window (T023) after editor + picker + status bar
- App (T024) after window
- Main entry point (T025) after app

### Parallel Opportunities

**In Setup**: T003 (.golangci.yml) and T004 (fetch scripts) run in parallel.

**In Foundational**: T008 (state.go types) and T009 (engine.go types) run in parallel.
T003 and T004 complete before T007 starts.

**In US1**:
- T011 (engine contract tests) and T012 (modules contract tests) run in parallel
- T013 (@boop/yaml) and T014 (@boop/plist) run in parallel
- T019 (editor), T020 (statusbar) run in parallel after T008 completes
- T021 (script list) runs in parallel with T019 and T020

**Across phases (after Foundational)**:
- US3 (T029→T031) can run in parallel with US2 (T026→T028)
- US4 (T032→T036) can run in parallel with US2 and US3

---

## Parallel Example: User Story 1

```bash
# Step 1: Write contract tests together (both fail before implementation):
Task: "Write engine contract tests in tests/contract/script_api_test.go"  [T011]
Task: "Write modules contract tests in tests/contract/modules_test.go"     [T012]

# Step 2: Implement modules in parallel:
Task: "Implement @boop/yaml in internal/engine/modules.go"                 [T013]
Task: "Implement @boop/plist in internal/engine/modules.go"                [T014]

# Step 3: After T008 (state types) complete, build UI widgets in parallel:
Task: "Implement editor widget in internal/ui/editor.go"                   [T019]
Task: "Implement status bar in internal/ui/statusbar.go"                   [T020]
Task: "Implement script list in internal/ui/scriptpicker.go"               [T021]

# Step 4: Wire execution flow (after T019 + T020 + T021 + T015):
Task: "Implement execution integration in internal/ui/scriptpicker.go"     [T022]
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all user stories)
3. Write contract tests T011, T012 → verify they FAIL
4. Complete Phase 3: User Story 1 (engine → modules → loader → library → UI → app → main)
5. **STOP and VALIDATE**: Run app manually, verify all P1 acceptance scenarios pass
6. Run `go test -race ./tests/contract/...` — all 10 engine + 2 module contract tests PASS

### Incremental Delivery

1. Setup + Foundational → shared infrastructure ready
2. User Story 1 → functional text transformation tool (MVP)
3. User Story 2 → fuzzy search added on top of existing library
4. User Story 3 → custom script support added to existing loader
5. User Story 4 → release pipeline + desktop integration
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers, after Foundational:

- Developer A: User Story 1 (core engine + UI)
- Developer B: User Story 3 (custom scripts) — depends only on Foundational
- Developer C: User Story 4 (release pipeline) — depends only on Foundational
- Once US1 complete: Developer A moves to US2 (fuzzy search)

---

## Notes

- `[P]` tasks = different files, no blocking dependencies on incomplete tasks
- `[Story]` label maps each task to a specific user story for traceability
- Constitution Principle II (Test-First) is NON-NEGOTIABLE: contract tests MUST fail before implementation begins
- All GTK API calls MUST be on the main thread; use `glib.IdleAdd()` from goroutines (engine runs in goroutine, result marshalled back via IdleAdd)
- `CGO_ENABLED=1` required for all build and test commands
- Commit after each task or logical group using conventional commit format (`feat:`, `fix:`, etc.)
- Stop at any checkpoint to validate the story independently before proceeding
- Avoid: vague tasks, same-file conflicts within a parallel group, cross-story dependencies that break independence
