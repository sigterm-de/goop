# Research: Editor Syntax Highlighting

**Feature**: `002-syntax-highlight` | **Date**: 2026-02-21

## 1. GtkSourceView Language Manager API

**Decision**: Use `gtksource.LanguageManagerGetDefault().Language(id)` to resolve a language by ID, then `buffer.SetLanguage(lang)` to apply it. Pass `nil` to `SetLanguage` to clear highlighting.

**Rationale**: The default `LanguageManager` loads all system-installed language definitions (installed via `gtksourceview5-data` / `libgtksourceview-5-dev`). The ID strings are stable across GtkSourceView 5.x releases. Setting `nil` is the documented way to disable highlighting on a buffer.

**GtkSourceView language IDs confirmed for supported formats**:

| Format | GtkSourceView ID | Notes |
|--------|-----------------|-------|
| JSON   | `"json"`        | Available in all GtkSourceView 5 installations |
| XML    | `"xml"`         | Available in all GtkSourceView 5 installations |
| HTML   | `"html"`        | Available in all GtkSourceView 5 installations |
| YAML   | `"yaml"`        | Available in all GtkSourceView 5 installations |
| SQL    | `"sql"`         | Available in all GtkSourceView 5 installations |
| Markdown | `"markdown"` | Available in all GtkSourceView 5 installations |
| CSV    | `"csv"`         | **Not universally available** — omitted from auto-detection (see below) |

**Human-readable name retrieval**: Call `lang.Name()` on the returned `*gtksource.Language` to get the display name (e.g. "JSON", "XML", "YAML"). This avoids hardcoding display names and respects any system localization.

**Alternatives considered**: Bundling custom language definition files (.lang) — rejected because system-provided definitions are kept current and there is no need for custom syntax.

---

## 2. Auto-Detection Scope

**Decision**: Auto-detection covers JSON, XML, HTML, and YAML in this feature. SQL and Markdown are supported for rendering (FR-001) but excluded from auto-detection.

**Rationale**:
- **JSON**: Unambiguous heuristic (starts with `{` or `[` after trimming) + stdlib `encoding/json` validation. Zero false-positive risk.
- **XML**: Unambiguous heuristic (`<?xml` or root element pattern) + stdlib `encoding/xml` validation. Zero false-positive risk.
- **HTML**: Reliable heuristic (`<!DOCTYPE html` or `<html`) + structural check. Negligible false-positive risk.
- **YAML**: Heuristic (`---` document separator or key: value pattern at line start) + `gopkg.in/yaml.v3` parse (already in `go.mod`). Low false-positive risk for well-structured YAML.
- **SQL**: No reliable heuristic without a full parser; plain text with `SELECT` is too common. **Excluded from auto-detection** to preserve the zero false-positive requirement (SC-002).
- **Markdown**: No definitive structural marker; plain text with `#` headings has high false-positive risk. **Excluded from auto-detection** for the same reason.
- **CSV**: `gtksourceview5-data` does not ship a CSV language definition on all target platforms. **Excluded from rendering support** in this implementation to avoid silent failures.

**FR-001 resolution**: The editor MUST support rendering JSON, XML, HTML, YAML, SQL, and Markdown (6 of the 7 listed formats; CSV removed from scope). Auto-detection is provided for JSON, XML, HTML, and YAML.

**Alternatives considered**: Using `file/magic` or `go-mimetype` libraries for content-type detection — rejected to avoid new external dependencies (Constitution Principle V).

---

## 3. Two-Tier Detection Algorithm

**Decision**: For each candidate format, apply a fast structural heuristic first; if it passes, run a validation parse. Only if both pass is the format declared.

**Tier 1 — Structural heuristics** (O(1) or O(prefix) string operations):

| Format | Heuristic |
|--------|-----------|
| JSON   | `content[0] == '{'` or `content[0] == '['` after `strings.TrimSpace` |
| XML    | `strings.HasPrefix(content, "<?xml")` or `strings.HasPrefix(content, "<")` followed by a letter (element start) |
| HTML   | `strings.Contains(lower prefix, "<!doctype html")` or `strings.Contains(lower prefix, "<html")` |
| YAML   | `strings.HasPrefix(content, "---")` or presence of `key: value` pattern at line start (`regexp` or manual scan of first line) |

**Tier 2 — Validation** (stdlib or already-in-go.mod parsers):

| Format | Validator |
|--------|-----------|
| JSON   | `json.Valid([]byte(content))` |
| XML    | `xml.NewDecoder(strings.NewReader(content)).Decode(&struct{}{})` — returns nil error for well-formed XML |
| HTML   | Structural check: balanced root tag or `<!DOCTYPE html>` — full parse not required; heuristic sufficient as HTML is already gated by `<!doctype html`/`<html` |
| YAML   | `yaml.Unmarshal([]byte(content), &interface{}{})` using `gopkg.in/yaml.v3` |

**Performance**: For 1 MB of content, `json.Valid` takes ~5–10 ms. `yaml.Unmarshal` on structured YAML is similar. Total detection pass is well within the 50 ms budget. If content exceeds a configurable byte limit (implementation suggestion: 4 MB), detection is skipped.

**Detection ordering**: JSON → HTML → XML → YAML. HTML is checked before XML because HTML can look like malformed XML. The first format that passes both tiers wins; subsequent formats are not evaluated.

---

## 4. Status Bar Split — Layout Strategy

**Decision**: Add a second `gtk.Label` to the existing `StatusBar.Box` (horizontal box). The existing left label retains `HExpand(true)`. The new right label has no horizontal expansion and is right-aligned. A small left margin (e.g. 8px) separates the zones visually.

**Rationale**: The `gtk.Box` with `OrientationHorizontal` naturally pushes a non-expanding right label to the right edge when the left label expands. No new layout widget is needed. This is the same pattern used in VS Code, GNOME Builder, and similar GTK applications.

**Empty state**: When no language is active, the right label is set to `""`. GTK renders an empty label with zero width, so no visual gap is created. The layout adjusts dynamically.

**Alternatives considered**: Using a `gtk.Stack` or `gtk.Grid` — rejected as unnecessarily complex for two labels.

---

## 5. Wiring Architecture

**Decision**: Add a `postScript func()` callback parameter to `ui.NewScriptPicker()`. The `ApplicationWindow` provides a closure that reads from the editor, runs detection (if the preference is enabled), and updates the editor and status bar accordingly.

**Rationale**: This keeps `ScriptPicker` unaware of detection logic (single responsibility, Constitution Principle V). The `ApplicationWindow` already has access to `prefs`, `editor`, and `status`, making it the natural orchestration point. The closure pattern is already used for `onHide`.

**Empty-buffer clear**: The `Editor` constructor connects a `buffer.ConnectChanged` signal. When the buffer length drops to zero, `ClearLanguage()` is called on the buffer and the window is notified via a second callback `onEditorEmpty func()`. Alternatively, the window can set up this signal directly after construction. The latter is simpler and avoids a callback chain.

**Alternatives considered**: Embedding detection logic directly in `ScriptPicker.applyResult()` — rejected because it couples UI rendering to feature logic and makes testing harder.

---

## 6. Preference Storage

**Decision**: Add `SyntaxAutoDetect bool` (JSON key `"syntax_auto_detect"`) to `AppPreferences` with a default value of `true` in `defaultPreferences()`.

**Rationale**: Consistent with all existing preference fields. The JSON key follows the existing snake_case convention. The default `true` matches the intended out-of-the-box experience.

**Alternatives considered**: A separate config file — rejected; one JSON file for all preferences is simpler and already established.
