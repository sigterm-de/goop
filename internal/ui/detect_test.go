package ui

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantLangID   string
		wantLangName string
	}{
		// JSON: object
		{"json object", `{"key": "value"}`, "json", "JSON"},
		// JSON: array
		{"json array", `[1, 2, 3]`, "json", "JSON"},
		// JSON: pretty-printed
		{"json pretty", "{\n  \"a\": 1\n}", "json", "JSON"},
		// JSON: invalid content starting with brace — must not match
		{"json invalid brace", `{not json}`, "", ""},
		// JSON: bare number — no heuristic match
		{"json bare number", `42`, "", ""},
		// HTML: with DOCTYPE
		{"html doctype", `<!DOCTYPE html><html><body></body></html>`, "html", "HTML"},
		// HTML: lowercase doctype
		{"html doctype lower", `<!doctype html><html></html>`, "html", "HTML"},
		// HTML: opening html tag only
		{"html tag", `<html lang="en"></html>`, "html", "HTML"},
		// XML: with declaration
		{"xml declaration", `<?xml version="1.0"?><root/>`, "xml", "XML"},
		// XML: element only
		{"xml element", `<root><child/></root>`, "xml", "XML"},
		// YAML: document separator
		{"yaml document separator", "---\nkey: value\n", "yaml", "YAML"},
		// YAML: mapping without separator
		{"yaml mapping", "key: value\n", "yaml", "YAML"},
		// YAML: nested
		{"yaml nested", "outer:\n  inner: 42\n", "yaml", "YAML"},
		// No match: plain text
		{"plain text", "hello world", "", ""},
		// No match: empty string
		{"empty string", "", "", ""},
		// No match: whitespace only
		{"whitespace only", "   \n\t  \n", "", ""},
		// No match: SQL keyword — not auto-detected
		{"sql select", "SELECT * FROM users", "", ""},
		// No match: markdown heading — not auto-detected
		{"markdown heading", "# Hello\n\nWorld\n", "", ""},
		// Large content: over 4 MB returns no match
		{"over 4MB", string(make([]byte, 4*1024*1024+1)), "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotName := Detect(tc.input)
			if gotID != tc.wantLangID || gotName != tc.wantLangName {
				t.Errorf("Detect(%q...) = (%q, %q); want (%q, %q)",
					truncate(tc.input, 40), gotID, gotName, tc.wantLangID, tc.wantLangName)
			}
		})
	}
}

// truncate returns s trimmed to at most n runes, with "…" appended if truncated.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
