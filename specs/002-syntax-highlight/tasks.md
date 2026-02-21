# Tasks: Editor Syntax Highlighting

**Input**: Design documents from `/specs/002-syntax-highlight/`
**Prerequisites**: plan.md ✅ spec.md ✅ research.md ✅ data-model.md ✅ quickstart.md ✅

**Tests**: Included — the project constitution (Principle II) mandates test-first development. Test tasks appear before their corresponding implementation tasks in each phase.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. See [data-model.md](data-model.md) for entity details and [quickstart.md](quickstart.md) for code sketches.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to ([US1], [US2], [US3])
- All paths are relative to the repository root

---

## Phase 1: Setup

No project-level setup is required. All dependencies (`gopkg.in/yaml.v3`, `libdb.so/gotk4-sourceview`) are already in `go.mod`. No new packages or build configuration changes are needed.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared infrastructure that ALL user stories depend on. No story work can begin until this phase is complete.

**⚠️ CRITICAL**: US1 wiring, US2 preference check, and US3 clear path all depend on these.

- [x] T001 Add `SyntaxAutoDetect bool` field (JSON key `"syntax_auto_detect"`, default `true`) to `AppPreferences` struct and `defaultPreferences()` in `internal/app/preferences.go`
- [x] T002 [P] Add `syntaxLabel *gtk.Label` field to `StatusBar` struct; append right-aligned syntax label to the horizontal box in `NewStatusBar()`; add `SetSyntaxLanguage(name string)` and `ClearSyntaxLanguage()` methods in `internal/ui/statusbar.go`
- [x] T003 [P] Add `.statusbar-syntax` CSS rule to `assets/style.css` — right-aligned, muted/dimmed text colour appropriate for a secondary indicator, no background change

**Checkpoint**: Foundation complete — T001, T002, T003 done. User story implementation can begin.

---

## Phase 3: User Story 1 — Highlighting After Script Run (Priority: P1) 🎯 MVP

**Goal**: After a successful script execution, the editor automatically detects the output format and applies syntax highlighting; the detected language appears in the status bar syntax zone.

**Independent Test**: Run the bundled `FormatJSON.js` script with JSON input; verify the editor shows coloured JSON tokens and the status bar right zone shows "JSON". Run a plain-text script; verify no highlighting is applied and syntax zone is empty.

### Tests for User Story 1 (TDD — write FIRST, verify they FAIL before implementing)

- [x] T004 [P] [US1] Write table-driven unit tests for `Detect()` in `internal/ui/detect_test.go` — cover: JSON object, JSON array, invalid JSON (no highlight), HTML with DOCTYPE, HTML with `<html`, XML with declaration, XML element, YAML with `---`, YAML key-value, plain text (no match), empty string (no match), whitespace-only (no match), content > 4 MB (no match)
- [x] T005 [P] [US1] Write integration test in `tests/integration/syntax_detection_test.go` — call `ui.Detect()` for each detectable format and assert correct `langID`; assert `("", "")` for plain text and empty input

### Implementation for User Story 1

- [x] T006 [US1] Implement `Detect(content string) (langID, langName string)` in `internal/ui/detect.go` — two-tier detection (heuristic + stdlib validation) for JSON, HTML, XML, YAML in that order; return `("", "")` for no match or content > 4 MB (makes T004 and T005 pass)
- [x] T007 [P] [US1] Add `SetLanguage(langID string)` and `ClearLanguage()` methods to `Editor` in `internal/ui/editor.go` — use `gtksource.LanguageManagerGetDefault().Language(langID)` and `buffer.SetLanguage(lang)`; treat empty `langID` as clear; both methods must be called on the GTK main thread
- [x] T008 [US1] Add `postScript func()` field to `ScriptPicker` struct; add the parameter to `NewScriptPicker()` constructor; call `sp.postScript()` at the end of `applyResult()` after a successful mutation and `ShowSuccess()` call in `internal/ui/scriptpicker.go`
- [x] T009 [US1] Wire detection callback in `NewApplicationWindow()` in `internal/app/window.go` — create `postScript` closure that (1) returns early if `!w.prefs.SyntaxAutoDetect`, (2) calls `ui.Detect(w.editor.GetFullText())`, (3) if match: calls `w.editor.SetLanguage(langID)` and `w.status.SetSyntaxLanguage(langName)`, (4) if no match: calls `w.editor.ClearLanguage()` and `w.status.ClearSyntaxLanguage()`; pass closure as the new final argument to `ui.NewScriptPicker()`

**Checkpoint**: US1 complete and independently testable. Build and run goop, open FormatJSON.js, verify JSON highlighting and "JSON" in the status bar right zone.

---

## Phase 4: User Story 2 — Disabling Auto-Detection (Priority: P2)

**Goal**: A user can toggle automatic syntax detection off in Preferences; when disabled, scripts run normally with no highlighting changes applied.

**Independent Test**: Open Preferences, uncheck "Auto-detect syntax highlighting", close Preferences, run FormatJSON.js — verify editor shows unstyled plain text and status bar syntax zone remains empty.

### Tests for User Story 2 (TDD — write FIRST, verify they FAIL before implementing)

- [x] T010 [US2] Write unit test asserting `defaultPreferences().SyntaxAutoDetect == true` in `internal/app/preferences_test.go` (create file if absent); also assert that JSON-unmarshalling `{"syntax_auto_detect": false}` into `AppPreferences` produces `SyntaxAutoDetect = false`

### Implementation for User Story 2

- [x] T011 [US2] Add `syntaxDetectCheck := gtk.NewCheckButtonWithLabel("Auto-detect syntax highlighting")` to `ShowSettingsDialog()` in `internal/app/settings.go` — initialise with `prefs.SyntaxAutoDetect`; connect `ConnectToggled` to call `applyChanges()`; update `applyChanges()` closure to write `p.SyntaxAutoDetect = syntaxDetectCheck.Active()`; place the widget in the Editor section of the grid using `attachSpan`

**Checkpoint**: US2 complete. Toggling the preference in the UI stops/resumes highlighting. Preference survives restart.

---

## Phase 5: User Story 3 — Highlighting Resets on Editor Clear (Priority: P3)

**Goal**: When the editor buffer is fully emptied (zero characters), any active syntax highlighting and the status bar syntax zone indicator are cleared automatically.

**Independent Test**: Apply FormatJSON.js (verify JSON highlighting active), then select-all and delete all content — verify syntax highlighting disappears and the status bar syntax zone becomes empty.

### Implementation for User Story 3

- [x] T012 [US3] In `NewApplicationWindow()` in `internal/app/window.go`, after constructing `w.editor` and `w.status`, connect `w.editor.View.Buffer().ConnectChanged(func() { if w.editor.GetFullText() == "" { w.editor.ClearLanguage(); w.status.ClearSyntaxLanguage() } })` — this must be wired after the `ScriptPicker` constructor call so that the buffer signal is connected on the main thread

**Checkpoint**: US3 complete. All three user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T013 [P] Run `golangci-lint run ./...` from repo root and fix all reported issues in the files touched by this feature (`detect.go`, `editor.go`, `statusbar.go`, `scriptpicker.go`, `preferences.go`, `settings.go`, `window.go`)
- [x] T014 [P] Run `CGO_ENABLED=1 CC=clang go test -race -coverprofile=coverage.out ./...` and verify coverage ≥ 80% for `internal/ui` and `internal/app` packages; add targeted unit tests if the gate would fail
- [x] T015 Run `CGO_ENABLED=1 CC=clang go build -o goop ./cmd/goop` and manually execute the verification steps in `quickstart.md` — confirm JSON, HTML, XML, and YAML formats highlight correctly after their respective bundled formatting scripts

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 2)**: No dependencies — start immediately; T002 and T003 are parallel
- **US1 (Phase 3)**: Requires Phase 2 complete; T004 and T005 are parallel; T006 and T007 are parallel (after tests fail); T008 after T006+T007; T009 last in this phase
- **US2 (Phase 4)**: Requires T001 (Phase 2); independent of US1 implementation tasks
- **US3 (Phase 5)**: Requires T007 (editor.ClearLanguage) and T002 (status.ClearSyntaxLanguage); can start after Phase 3 T007 is complete
- **Polish (Phase 6)**: Requires all desired stories complete

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2 complete (T001, T002, T003) — no dependency on US2 or US3
- **US2 (P2)**: Depends on T001 only — can be started as soon as Phase 2 T001 is done
- **US3 (P3)**: Depends on T002 (ClearSyntaxLanguage) and T007 (ClearLanguage) — can start after T002 and T007 are complete

### Within Each Phase

- **TDD order**: Test tasks MUST be written first; run them and confirm they FAIL before writing implementation
- **US1 internal order**: T004/T005 (parallel) → T006/T007 (parallel, after tests fail) → T008 → T009
- **US2 internal order**: T010 → T011
- **US3 internal order**: T012 (single task)

---

## Parallel Execution Examples

### Phase 2 (Foundational) — two parallel streams

```
Stream A: T001 (preferences.go)
Stream B: T002 + T003 in parallel (statusbar.go + style.css)
```

### Phase 3 (US1) — parallel test writing

```
Stream A: T004 (detect_test.go)
Stream B: T005 (syntax_detection_test.go)
→ Both fail → then parallel:
Stream A: T006 (detect.go)
Stream B: T007 (editor.go)
→ Both complete → T008 (scriptpicker.go) → T009 (window.go)
```

### Phase 6 (Polish) — parallel checks

```
Stream A: T013 (lint)
Stream B: T014 (tests + coverage)
→ Both pass → T015 (manual validation)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Foundational (T001–T003)
2. Complete Phase 3: User Story 1 (T004–T009)
3. **STOP and VALIDATE**: Run FormatJSON.js in goop; verify JSON highlighting and status bar label
4. Demo/merge if satisfactory; US2 and US3 add polish but are not blocking

### Incremental Delivery

1. Phase 2 → Phase 3 (US1): Core highlighting — **ship-ready MVP**
2. Add Phase 4 (US2): User preference toggle — **ship-ready**
3. Add Phase 5 (US3): Auto-clear on empty — **ship-ready**
4. Phase 6 polish can run in parallel with any story

---

## Notes

- All GTK method calls (`SetLanguage`, `ClearLanguage`, `SetSyntaxLanguage`, `ClearSyntaxLanguage`) must execute on the GTK main thread — the wiring in `window.go` via `glib.IdleAdd` already guarantees this for the `postScript` callback path
- `Detect()` in `detect.go` is pure Go with no GTK dependency — all unit tests can run headless without a display
- See `quickstart.md` for exact code sketches for each implementation task
- No new external Go module dependencies are introduced
