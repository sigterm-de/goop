// Package contract — additional tests targeting coverage gaps.
package contract_test

import (
	"context"
	"strings"
	"testing"

	"codeberg.org/daniel-ciaglia/goop/internal/engine"
)

// TestAtobDecodeSuccess verifies atob() decodes a valid base64 string.
func TestAtobDecodeSuccess(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"",
		`function main(state) { state.text = atob("aGVsbG8="); }`,
	))
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.ErrorMessage)
	}
	if result.NewText != "hello" {
		t.Fatalf("expected 'hello', got %q", result.NewText)
	}
}

// TestAtobDecodeError verifies atob() propagates an error for invalid input.
func TestAtobDecodeError(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"",
		`function main(state) { state.text = atob("!!!not-base64!!!"); }`,
	))
	if result.Success {
		t.Fatal("expected failure for invalid base64")
	}
}

// TestConsoleLog verifies console.log() does not crash the script.
func TestConsoleLog(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"hi",
		`function main(state) { console.log("hello", "world"); state.text = "ok"; }`,
	))
	if !result.Success {
		t.Fatalf("console.log should not fail: %s", result.ErrorMessage)
	}
	if result.NewText != "ok" {
		t.Fatalf("expected 'ok', got %q", result.NewText)
	}
}

// TestPlistStringify verifies plist.stringify produces XML output.
func TestPlistStringify(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"",
		`
var plist = require('@boop/plist');
function main(state) {
    state.fullText = plist.stringify({key: "value"});
}`,
	))
	if !result.Success {
		t.Fatalf("plist.stringify failed: %s", result.ErrorMessage)
	}
	if !strings.Contains(result.NewFullText, "<?xml") {
		t.Errorf("expected XML output, got: %q", result.NewFullText)
	}
}

// TestPlistParseBinary verifies plist.parseBinary on XML plist input.
func TestPlistParseBinary(t *testing.T) {
	plistXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>foo</key><string>bar</string></dict></plist>`
	inp := engine.ExecutionInput{
		ScriptSource: `
var plist = require('@boop/plist');
function main(state) {
    var obj = plist.parseBinary(state.fullText);
    state.fullText = JSON.stringify(obj);
}`,
		ScriptName:    "plist-binary-test",
		FullText:      plistXML,
		SelectionText: plistXML,
	}
	result := newExec().Execute(context.Background(), inp)
	if !result.Success {
		t.Fatalf("plist.parseBinary failed: %s", result.ErrorMessage)
	}
	if !strings.Contains(result.NewFullText, "bar") {
		t.Errorf("expected 'bar' in output, got: %q", result.NewFullText)
	}
}

// TestYAMLNestedArrayNormalise verifies normaliseYAML handles nested arrays.
func TestYAMLNestedArrayNormalise(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"",
		`
var yaml = require('@boop/yaml');
function main(state) {
    var obj = yaml.parse("items:\n  - a\n  - b\n  - c");
    state.fullText = JSON.stringify(obj);
}`,
	))
	if !result.Success {
		t.Fatalf("YAML array normalise failed: %s", result.ErrorMessage)
	}
	if !strings.Contains(result.NewFullText, "a") {
		t.Errorf("expected array items in output, got: %q", result.NewFullText)
	}
}

// TestYAMLStringifyError verifies yaml.stringify propagates errors.
func TestYAMLStringifyWithMap(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"",
		`
var yaml = require('@boop/yaml');
function main(state) {
    var obj = {name: "Alice", nested: {x: 1}};
    state.fullText = yaml.stringify(obj);
}`,
	))
	if !result.Success {
		t.Fatalf("yaml.stringify failed: %s", result.ErrorMessage)
	}
	if !strings.Contains(result.NewFullText, "Alice") {
		t.Errorf("expected 'Alice' in output, got: %q", result.NewFullText)
	}
}

// TestSyntaxError verifies a script with syntax error fails gracefully.
func TestSyntaxError(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"x",
		`function main(state) { this is not valid JS`,
	))
	if result.Success {
		t.Fatal("expected failure for syntax error")
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TestMissingMainFunction verifies scripts without main() fail gracefully.
func TestMissingMainFunction(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"x",
		`var x = 42;`,
	))
	if result.Success {
		t.Fatal("expected failure for missing main()")
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected non-empty error message for missing main()")
	}
}

// TestRequireUnknownModuleFails verifies non-@boop/ require fails.
func TestRequireUnknownModuleFails(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"x",
		`function main(state) { require('lodash'); }`,
	))
	if result.Success {
		t.Fatal("expected failure for unknown module")
	}
}
