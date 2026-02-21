package integration_test

import (
	"testing"

	"codeberg.org/sigterm-de/goop/internal/ui"
)

// TestSyntaxDetectionFormats verifies that Detect correctly identifies all
// auto-detectable formats and returns empty strings for plain or unknown content.
func TestSyntaxDetectionFormats(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantLangID   string
		wantLangName string
	}{
		{
			name:         "FormatJSON script output",
			input:        "{\n  \"formatted\": true,\n  \"value\": 42\n}",
			wantLangID:   "json",
			wantLangName: "JSON",
		},
		{
			name:         "FormatXML script output",
			input:        "<?xml version=\"1.0\"?>\n<root>\n  <child>hello</child>\n</root>",
			wantLangID:   "xml",
			wantLangName: "XML",
		},
		{
			name:         "HTML output",
			input:        "<!DOCTYPE html>\n<html>\n<body><p>Hello</p></body>\n</html>",
			wantLangID:   "html",
			wantLangName: "HTML",
		},
		{
			name:         "JSONtoYAML script output",
			input:        "---\nkey: value\nnested:\n  a: 1\n  b: 2\n",
			wantLangID:   "yaml",
			wantLangName: "YAML",
		},
		{
			name:         "plain text — no highlight",
			input:        "Hello, world! This is plain text.",
			wantLangID:   "",
			wantLangName: "",
		},
		{
			name:         "empty editor — no highlight",
			input:        "",
			wantLangID:   "",
			wantLangName: "",
		},
		{
			name:         "invalid JSON — no highlight",
			input:        `{this is not valid json}`,
			wantLangID:   "",
			wantLangName: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotName := ui.Detect(tc.input)
			if gotID != tc.wantLangID {
				t.Errorf("Detect() langID = %q; want %q (input: %.60q)",
					gotID, tc.wantLangID, tc.input)
			}
			if gotName != tc.wantLangName {
				t.Errorf("Detect() langName = %q; want %q (input: %.60q)",
					gotName, tc.wantLangName, tc.input)
			}
		})
	}
}
