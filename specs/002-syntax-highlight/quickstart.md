# Developer Quickstart: Editor Syntax Highlighting

**Feature**: `002-syntax-highlight` | **Date**: 2026-02-21

## Prerequisites

```bash
# Same as the rest of the project — CGO_ENABLED=1 and clang are required
CGO_ENABLED=1 CC=clang go build -o goop ./cmd/goop

# Run all tests (requires display for GTK tests; headless uses Xvfb or skips GTK tests)
CGO_ENABLED=1 CC=clang go test -race ./...

# Run only the pure-Go detection tests (no display required)
go test -race ./internal/ui/... -run TestDetect
```

---

## Implementation Order (TDD — tests first)

### Step 1 — Write detection tests (`internal/ui/detect_test.go`)

Create the test file **before** `detect.go`. All test cases must fail initially.

```go
// internal/ui/detect_test.go
package ui_test

import "testing"

func TestDetect(t *testing.T) {
    cases := []struct {
        name         string
        input        string
        wantLangID   string
        wantLangName string
    }{
        // JSON
        {"json object", `{"key": "value"}`, "json", "JSON"},
        {"json array", `[1, 2, 3]`, "json", "JSON"},
        {"json invalid", `{not json}`, "", ""},
        // HTML
        {"html doctype", `<!DOCTYPE html><html></html>`, "html", "HTML"},
        {"html tag", `<html lang="en"></html>`, "html", "HTML"},
        // XML
        {"xml declaration", `<?xml version="1.0"?><root/>`, "xml", "XML"},
        {"xml element", `<root><child/></root>`, "xml", "XML"},
        // YAML
        {"yaml document", "---\nkey: value\n", "yaml", "YAML"},
        {"yaml mapping", "key: value\n", "yaml", "YAML"},
        // No match
        {"plain text", "hello world", "", ""},
        {"empty string", "", "", ""},
        {"whitespace only", "   \n", "", ""},
        // Ambiguous (JSON/YAML — JSON wins as more specific)
        {"bare json number", `42`, "", ""},  // no heuristic match for bare scalar
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            gotID, gotName := Detect(tc.input)
            if gotID != tc.wantLangID || gotName != tc.wantLangName {
                t.Errorf("Detect(%q) = (%q, %q); want (%q, %q)",
                    tc.input, gotID, gotName, tc.wantLangID, tc.wantLangName)
            }
        })
    }
}
```

### Step 2 — Implement detection (`internal/ui/detect.go`)

```go
// internal/ui/detect.go
package ui

import (
    "encoding/json"
    "encoding/xml"
    "strings"

    "gopkg.in/yaml.v3"
)

// maxDetectBytes is the content size limit beyond which detection is skipped.
const maxDetectBytes = 4 * 1024 * 1024 // 4 MB

// Detect returns the GtkSourceView language ID and display name for the
// given content using a two-tier heuristic + validation approach.
// Returns ("", "") if no format can be identified with high confidence.
func Detect(content string) (langID, langName string) {
    content = strings.TrimSpace(content)
    if content == "" || len(content) > maxDetectBytes {
        return "", ""
    }

    // Detection order: HTML before XML (HTML can look like broken XML).
    switch {
    case isHTML(content):
        return "html", "HTML"
    case isJSON(content):
        return "json", "JSON"
    case isXML(content):
        return "xml", "XML"
    case isYAML(content):
        return "yaml", "YAML"
    }
    return "", ""
}

func isJSON(s string) bool {
    if len(s) == 0 {
        return false
    }
    if s[0] != '{' && s[0] != '[' {
        return false
    }
    return json.Valid([]byte(s))
}

func isHTML(s string) bool {
    lower := strings.ToLower(s[:min(512, len(s))])
    if !strings.Contains(lower, "<!doctype html") && !strings.Contains(lower, "<html") {
        return false
    }
    // Validate: must parse as XML-ish structure (lenient check).
    // HTML5 is not strict XML, so we accept if heuristic passes.
    return true
}

func isXML(s string) bool {
    if !strings.HasPrefix(s, "<?xml") && (len(s) == 0 || s[0] != '<') {
        return false
    }
    // Validate using stdlib XML decoder.
    d := xml.NewDecoder(strings.NewReader(s))
    for {
        _, err := d.Token()
        if err != nil {
            break
        }
    }
    return d.InputOffset() > 0
}

func isYAML(s string) bool {
    if !strings.HasPrefix(s, "---") {
        // Check for key: value on first non-empty line.
        first := firstNonEmptyLine(s)
        if !strings.Contains(first, ": ") {
            return false
        }
    }
    var v interface{}
    return yaml.Unmarshal([]byte(s), &v) == nil && v != nil
}

func firstNonEmptyLine(s string) string {
    for _, line := range strings.SplitN(s, "\n", 10) {
        if trimmed := strings.TrimSpace(line); trimmed != "" {
            return trimmed
        }
    }
    return ""
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Step 3 — Extend Editor (`internal/ui/editor.go`)

Add after the existing `ApplyScheme` method:

```go
// SetLanguage enables GtkSourceView syntax highlighting for the given
// GtkSourceView language ID (e.g. "json", "xml"). If the language is not
// found on the system, the buffer's current highlighting is left unchanged.
// Must be called on the GTK main thread.
func (e *Editor) SetLanguage(langID string) {
    if langID == "" {
        e.ClearLanguage()
        return
    }
    mgr := gtksource.LanguageManagerGetDefault()
    if lang := mgr.Language(langID); lang != nil {
        e.buffer.SetLanguage(lang)
    }
}

// ClearLanguage removes all syntax highlighting from the editor buffer.
// Safe to call when no language is active. Must be called on the GTK main thread.
func (e *Editor) ClearLanguage() {
    e.buffer.SetLanguage(nil)
}
```

### Step 4 — Split StatusBar (`internal/ui/statusbar.go`)

Modify `NewStatusBar` to add a right-aligned syntax label:

```go
func NewStatusBar() *StatusBar {
    // existing notification label (unchanged)
    label := gtk.NewLabel(defaultIdleText)
    label.SetXAlign(0)
    label.SetEllipsize(pango.EllipsizeEnd)
    label.SetHExpand(true)

    // new syntax zone label
    syntaxLabel := gtk.NewLabel("")
    syntaxLabel.SetXAlign(1)
    syntaxLabel.SetMarginStart(8)
    syntaxLabel.AddCSSClass("statusbar-syntax")

    box := gtk.NewBox(gtk.OrientationHorizontal, 0)
    box.AddCSSClass("statusbar")
    box.AddCSSClass("statusbar-idle")
    box.SetMarginTop(4)
    box.SetMarginBottom(4)
    box.SetMarginStart(12)
    box.SetMarginEnd(12)
    box.Append(label)
    box.Append(syntaxLabel)  // right-aligned, pushed by expanding left label

    return &StatusBar{Box: box, label: label, syntaxLabel: syntaxLabel,
        idleText: defaultIdleText, isIdle: true}
}

// SetSyntaxLanguage shows the detected language name in the syntax zone.
func (s *StatusBar) SetSyntaxLanguage(name string) {
    s.syntaxLabel.SetText(name)
}

// ClearSyntaxLanguage removes any language indicator from the syntax zone.
func (s *StatusBar) ClearSyntaxLanguage() {
    s.syntaxLabel.SetText("")
}
```

Add `syntaxLabel *gtk.Label` to the `StatusBar` struct.

### Step 5 — Wire ScriptPicker callback (`internal/ui/scriptpicker.go`)

Add `postScript func()` parameter to `NewScriptPicker` and call it in `applyResult`:

```go
// NewScriptPicker signature change:
func NewScriptPicker(
    lib scripts.Library,
    exec engine.Executor,
    editor *Editor,
    status *StatusBar,
    logPath string,
    onHide func(),
    postScript func(),   // NEW: called after every successful script execution
) *ScriptPicker

// In applyResult, after the mutation switch:
func (sp *ScriptPicker) applyResult(result engine.ExecutionResult) {
    // ... existing success/error handling ...
    if result.Success {
        // existing mutation application + ShowSuccess call
        if sp.postScript != nil {
            sp.postScript()
        }
    }
}
```

### Step 6 — Add preference field (`internal/app/preferences.go`)

```go
type AppPreferences struct {
    // ... existing fields ...
    SyntaxAutoDetect bool `json:"syntax_auto_detect"` // auto-detect content language after script run
}

func defaultPreferences() AppPreferences {
    return AppPreferences{
        // ... existing defaults ...
        SyntaxAutoDetect: true,
    }
}
```

### Step 7 — Add settings toggle (`internal/app/settings.go`)

In `ShowSettingsDialog`, within the Editor section:

```go
syntaxDetectCheck := gtk.NewCheckButtonWithLabel("Auto-detect syntax highlighting")
syntaxDetectCheck.SetActive(prefs.SyntaxAutoDetect)
syntaxDetectCheck.ConnectToggled(func() { applyChanges() })

// In applyChanges():
p.SyntaxAutoDetect = syntaxDetectCheck.Active()

// In layout (after monoCheck):
attachSpan(syntaxDetectCheck)
```

### Step 8 — Wire detection in ApplicationWindow (`internal/app/window.go`)

In `NewApplicationWindow`, update the `NewScriptPicker` call:

```go
postScript := func() {
    if !w.prefs.SyntaxAutoDetect {
        return
    }
    text := w.editor.GetFullText()
    langID, langName := ui.Detect(text)
    if langID != "" {
        w.editor.SetLanguage(langID)
        w.status.SetSyntaxLanguage(langName)
    } else {
        w.editor.ClearLanguage()
        w.status.ClearSyntaxLanguage()
    }
}

w.picker = ui.NewScriptPicker(lib, exec, w.editor, w.status, logPath, w.HideScriptPicker, postScript)
```

Connect the empty-buffer clear in `NewApplicationWindow` after widget construction:

```go
// Clear language highlighting when the editor is emptied.
w.editor.View.Buffer().ConnectChanged(func() {
    if w.editor.GetFullText() == "" {
        w.editor.ClearLanguage()
        w.status.ClearSyntaxLanguage()
    }
})
```

---

## Integration Test

```go
// tests/integration/syntax_detection_test.go
package integration_test

import (
    "testing"
    "codeberg.org/sigterm-de/goop/internal/ui"
)

func TestSyntaxDetectionEndToEnd(t *testing.T) {
    cases := []struct {
        name       string
        scriptOutput string
        wantLangID string
    }{
        {"json format script output", `{"formatted": true}`, "json"},
        {"xml output", `<?xml version="1.0"?><root/>`, "xml"},
        {"html output", `<!DOCTYPE html><html><body></body></html>`, "html"},
        {"yaml output", "---\nkey: value\n", "yaml"},
        {"plain text output", "hello world", ""},
        {"empty output", "", ""},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            gotID, _ := ui.Detect(tc.scriptOutput)
            if gotID != tc.wantLangID {
                t.Errorf("Detect(%q) langID = %q; want %q", tc.scriptOutput, gotID, tc.wantLangID)
            }
        })
    }
}
```

---

## Build & Verify

```bash
# Build
CGO_ENABLED=1 CC=clang go build -o goop ./cmd/goop

# Pure-Go detection tests (no display required)
go test -race ./internal/ui/ -run TestDetect -v

# All tests
CGO_ENABLED=1 CC=clang go test -race -coverprofile=coverage.out ./...

# Lint
golangci-lint run ./...

# Coverage gate (must be ≥ 80%)
go tool cover -func=coverage.out | grep total
```
