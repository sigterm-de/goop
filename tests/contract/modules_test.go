// Package contract contains acceptance tests for @boop/yaml and @boop/plist modules.
// These tests map to TC modules contract cases in contracts/engine.md.
package contract_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"codeberg.org/daniel-ciaglia/goop/internal/engine"
)

func moduleInput(src string) engine.ExecutionInput {
	return engine.ExecutionInput{
		ScriptSource:   src,
		ScriptName:     "module-test",
		FullText:       "",
		SelectionText:  "",
		SelectionStart: 0,
		SelectionEnd:   0,
		Timeout:        5 * time.Second,
	}
}

// TestYAMLParseThenStringify verifies the @boop/yaml module round-trip.
func TestYAMLParseThenStringify(t *testing.T) {
	inp := moduleInput(`
var yaml = require('@boop/yaml');
function main(state) {
    var obj = yaml.parse('name: Alice\nage: 30');
    state.fullText = yaml.stringify(obj);
}`)
	result := newExec().Execute(context.Background(), inp)
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	out := result.NewFullText
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected output to contain 'Alice', got: %q", out)
	}
	if !strings.Contains(out, "30") {
		t.Errorf("expected output to contain '30', got: %q", out)
	}
}

// TestPlistParseThenStringify verifies the @boop/plist module round-trip.
func TestPlistParseThenStringify(t *testing.T) {
	plistXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>foo</key>
    <string>bar</string>
</dict>
</plist>`

	inp := engine.ExecutionInput{
		ScriptSource: `
var plist = require('@boop/plist');
function main(state) {
    var obj = plist.parse(state.fullText);
    state.fullText = JSON.stringify(obj);
}`,
		ScriptName:     "plist-test",
		FullText:       plistXML,
		SelectionText:  plistXML,
		SelectionStart: 0,
		SelectionEnd:   len(plistXML),
		Timeout:        5 * time.Second,
	}

	result := newExec().Execute(context.Background(), inp)
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	out := result.NewFullText
	if !strings.Contains(out, "bar") {
		t.Errorf("expected output to contain 'bar', got: %q", out)
	}
}
