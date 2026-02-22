// Package contract — additional tests targeting coverage gaps.
package contract_test

import (
	"context"
	"strings"
	"testing"
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

// TestEvalIsPoisoned verifies that eval() is not available to scripts.
func TestEvalIsPoisoned(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"x",
		`function main(state) { eval("state.text = 'injected'"); }`,
	))
	if result.Success {
		t.Fatal("expected failure: eval should be undefined")
	}
}

// TestFunctionConstructorIsAvailable verifies that new Function() works for
// scripts that need it (e.g. Boop-compatible libraries like node-forge).
// Function() only creates closures in global scope (no local variable access),
// so it is intentionally NOT poisoned.
func TestFunctionConstructorIsAvailable(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"x",
		`function main(state) { var f = new Function("return '42'"); state.text = f(); }`,
	))
	if !result.Success {
		t.Fatalf("expected success: Function constructor should be available, got: %s", result.ErrorMessage)
	}
	if result.NewText != "42" {
		t.Fatalf("expected '42', got %q", result.NewText)
	}
}
