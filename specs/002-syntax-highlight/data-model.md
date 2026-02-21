# Data Model: Editor Syntax Highlighting

**Feature**: `002-syntax-highlight` | **Date**: 2026-02-21

## Overview

This feature adds no persistent data beyond one new preference field. All syntax highlighting state is ephemeral (in-process only, never written to disk). The entities from the spec map to Go types as described below.

---

## Entity: SyntaxLanguage

**Spec definition**: A named highlighting definition provided by the system's source-view component, identified by a stable language ID string.

**Go representation** (no struct needed — represented as a pair of strings):

| Field | Go Type | Source | Description |
|-------|---------|--------|-------------|
| `langID` | `string` | GtkSourceView language ID | Stable identifier used to look up the language definition (e.g. `"json"`, `"xml"`, `"yaml"`, `"html"`) |
| `langName` | `string` | `gtksource.Language.Name()` | Human-readable display name shown in the status bar syntax zone (e.g. `"JSON"`, `"XML"`, `"YAML"`, `"HTML"`) |

**Valid langID values** (auto-detection scope):

| langID  | langName | Detection method |
|---------|----------|-----------------|
| `"json"` | `"JSON"` | Heuristic: starts with `{` or `[`; Validation: `json.Valid()` |
| `"xml"`  | `"XML"`  | Heuristic: starts with `<?xml` or `<letter`; Validation: `xml.Decode()` succeeds |
| `"html"` | `"HTML"` | Heuristic: contains `<!doctype html` or `<html`; Validation: structural check |
| `"yaml"` | `"YAML"` | Heuristic: starts with `---` or `key: value` pattern; Validation: `yaml.Unmarshal()` succeeds |

**Rendering-only** (supported via `Editor.SetLanguage()` but not auto-detected):

| langID | langName |
|--------|----------|
| `"sql"` | `"SQL"` |
| `"markdown"` | `"Markdown"` |

---

## Entity: DetectionResult

**Spec definition**: The outcome of analysing editor content — either a matched SyntaxLanguage with a confidence level, or a "no match" result.

**Go representation** (return values of `ui.Detect()`):

```go
// Detect returns the GtkSourceView language ID and display name for the
// given content. Returns ("", "") if no format is confidently detected.
func Detect(content string) (langID, langName string)
```

**State transitions**:

```
content → Detect() → ("", "")       → ClearLanguage() + ClearSyntaxLanguage()
content → Detect() → ("json", "JSON") → SetLanguage("json") + SetSyntaxLanguage("JSON")
```

**Constraints**:
- Result is stateless: same input always produces same output (pure function)
- Result is ephemeral: not stored, not logged, discarded after application
- Empty content always returns `("", "")`
- Content exceeding 4 MB returns `("", "")` without analysis (performance guard)

---

## Entity: AppPreferences (extended)

**Modified struct field** added to existing `internal/app/AppPreferences`:

| Field | JSON key | Go Type | Default | Description |
|-------|----------|---------|---------|-------------|
| `SyntaxAutoDetect` | `"syntax_auto_detect"` | `bool` | `true` | When false, detection is skipped entirely after script execution |

**Storage**: XDG config file (`~/.config/goop/preferences.json`), alongside existing fields. Persisted by the existing `SavePreferences()` / `LoadPreferences()` mechanism.

---

## Entity: StatusBar (restructured)

The `StatusBar` widget gains a second label. No new persistent state is added.

**Layout**:

```
┌─────────────────────────────────────────────────────┐
│  [notification zone — left, HExpand]   [syntax zone] │
└─────────────────────────────────────────────────────┘
```

**New fields on StatusBar struct**:

| Field | Type | Description |
|-------|------|-------------|
| `syntaxLabel` | `*gtk.Label` | Right-aligned label showing the active language name; empty string when no language is active |

**New methods on StatusBar**:

| Method | Signature | Description |
|--------|-----------|-------------|
| `SetSyntaxLanguage` | `func(name string)` | Sets the syntax zone text; idempotent |
| `ClearSyntaxLanguage` | `func()` | Clears the syntax zone text (sets to "") |

**Invariants**:
- The notification zone and syntax zone are fully independent; a `ShowError()` call never touches `syntaxLabel`
- `ClearSyntaxLanguage()` is safe to call when no language is active (no-op)

---

## Entity: Editor (extended)

**New methods on Editor struct**:

| Method | Signature | Description |
|--------|-----------|-------------|
| `SetLanguage` | `func(langID string)` | Applies GtkSourceView highlighting for the given language ID; silently does nothing if the language is not found |
| `ClearLanguage` | `func()` | Removes all syntax highlighting from the buffer |

**Constraints**:
- Both methods MUST be called on the GTK main thread
- `SetLanguage("")` is equivalent to `ClearLanguage()` (treat empty string as clear)
- `ClearLanguage()` is safe to call when no language is active

---

## Wiring: Execution Flow

```
User selects script
    └─→ ScriptPicker.runScript() [goroutine]
            └─→ engine.Execute()
                    └─→ glib.IdleAdd (back to main thread)
                            └─→ ScriptPicker.applyResult()
                                    ├─→ editor mutation applied
                                    ├─→ status.ShowSuccess(...)   [notification zone]
                                    └─→ postScript()              [detection callback]
                                            └─→ (in ApplicationWindow closure)
                                                    ├─→ if !prefs.SyntaxAutoDetect → return
                                                    ├─→ text := editor.GetFullText()
                                                    ├─→ langID, langName := ui.Detect(text)
                                                    ├─→ if langID != ""
                                                    │       ├─→ editor.SetLanguage(langID)
                                                    │       └─→ status.SetSyntaxLanguage(langName)
                                                    └─→ else
                                                            ├─→ editor.ClearLanguage()
                                                            └─→ status.ClearSyntaxLanguage()
```

**Empty-buffer clear path**:

```
User clears editor content (all characters deleted)
    └─→ buffer.ConnectChanged (connected in ApplicationWindow after construction)
            └─→ if editor.GetFullText() == ""
                    ├─→ editor.ClearLanguage()
                    └─→ status.ClearSyntaxLanguage()
```
