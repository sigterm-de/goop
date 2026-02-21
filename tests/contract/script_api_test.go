// Package contract contains acceptance tests derived directly from
// contracts/engine.md. Each test case maps to a TC-E-xx test case in that document.
// These tests MUST be written before the engine implementation and verified to FAIL.
package contract_test

import (
	"context"
	"testing"
	"time"

	"codeberg.org/sigterm-de/goop/internal/engine"
)

// newExec returns the production Executor under test.
func newExec() engine.Executor {
	return engine.NewExecutor()
}

func input(fullText, selText string, selStart, selEnd int, src string) engine.ExecutionInput {
	return engine.ExecutionInput{
		ScriptSource:   src,
		ScriptName:     "test-script",
		FullText:       fullText,
		SelectionText:  selText,
		SelectionStart: selStart,
		SelectionEnd:   selEnd,
		Timeout:        5 * time.Second,
	}
}

func noSelInput(fullText, src string) engine.ExecutionInput {
	return input(fullText, fullText, 0, len(fullText), src)
}

// TC-E-01: Simple text transformation via state.text
func TestTC_E01_SimpleTextTransformation(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"hello",
		`function main(state) { state.text = state.text.toUpperCase(); }`,
	))
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	if result.MutationKind != engine.MutationReplaceSelect {
		t.Fatalf("expected MutationReplaceSelect, got %d", result.MutationKind)
	}
	if result.NewText != "HELLO" {
		t.Fatalf("expected NewText=HELLO, got %q", result.NewText)
	}
}

// TC-E-02: FullText mutation
func TestTC_E02_FullTextMutation(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"abc",
		`function main(state) { state.fullText = "XYZ"; }`,
	))
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	if result.MutationKind != engine.MutationReplaceDoc {
		t.Fatalf("expected MutationReplaceDoc, got %d", result.MutationKind)
	}
	if result.NewFullText != "XYZ" {
		t.Fatalf("expected NewFullText=XYZ, got %q", result.NewFullText)
	}
}

// TC-E-03: postError discards mutations
func TestTC_E03_PostErrorDisardsMutations(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"hello",
		`function main(state) { state.fullText = "REPLACED"; state.postError("oops"); }`,
	))
	if result.Success {
		t.Fatal("expected failure after postError")
	}
	if result.ErrorMessage != "oops" {
		t.Fatalf("expected error message 'oops', got %q", result.ErrorMessage)
	}
	if result.MutationKind != engine.MutationNone {
		t.Fatalf("expected MutationNone after postError, got %d", result.MutationKind)
	}
}

// TC-E-04: Unhandled exception discards mutations
func TestTC_E04_UnhandledExceptionDisardsMutations(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"hello",
		`function main(state) { state.fullText = "REPLACED"; throw new Error("boom"); }`,
	))
	if result.Success {
		t.Fatal("expected failure after unhandled exception")
	}
	if result.MutationKind != engine.MutationNone {
		t.Fatalf("expected MutationNone, got %d", result.MutationKind)
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TC-E-05: Timeout
func TestTC_E05_Timeout(t *testing.T) {
	inp := noSelInput("x", `function main(state) { while(true) {} }`)
	inp.Timeout = 200 * time.Millisecond

	result := newExec().Execute(context.Background(), inp)
	if result.Success {
		t.Fatal("expected failure on timeout")
	}
	if !result.TimedOut {
		t.Fatal("expected TimedOut=true")
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected non-empty timeout error message")
	}
}

// TC-E-06: @boop/yaml round-trip
func TestTC_E06_YAMLRoundTrip(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"name: Alice\nage: 30",
		`
var yaml = require('@boop/yaml');
function main(state) {
    var obj = yaml.parse(state.fullText);
    state.fullText = JSON.stringify(obj);
}`,
	))
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	if result.MutationKind != engine.MutationReplaceDoc {
		t.Fatalf("expected MutationReplaceDoc, got %d", result.MutationKind)
	}
	// Result must contain both fields
	out := result.NewFullText
	if out == "" {
		t.Fatal("expected non-empty NewFullText")
	}
}

// TC-E-07: Prohibited global access — require('fs')
func TestTC_E07_ProhibitedGlobalFS(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"x",
		`function main(state) { var x = require('fs'); }`,
	))
	if result.Success {
		t.Fatal("expected failure when require('fs') is called")
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TC-E-08: insert() at cursor
func TestTC_E08_InsertAtCursor(t *testing.T) {
	result := newExec().Execute(context.Background(), input("", "", 0, 0,
		`function main(state) { state.insert("HELLO"); }`,
	))
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	if result.MutationKind != engine.MutationInsertAtCursor {
		t.Fatalf("expected MutationInsertAtCursor, got %d", result.MutationKind)
	}
	if result.InsertText != "HELLO" {
		t.Fatalf("expected InsertText=HELLO, got %q", result.InsertText)
	}
}

// TC-E-09: No mutation (read-only script)
func TestTC_E09_NoMutation(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"hello",
		`function main(state) { var x = state.fullText.length; }`,
	))
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	if result.MutationKind != engine.MutationNone {
		t.Fatalf("expected MutationNone, got %d", result.MutationKind)
	}
}

// TC-E-10: btoa/atob globals
func TestTC_E10_BtoaAtob(t *testing.T) {
	result := newExec().Execute(context.Background(), noSelInput(
		"hello",
		`function main(state) { state.text = btoa(state.text); }`,
	))
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.ErrorMessage)
	}
	if result.NewText != "aGVsbG8=" {
		t.Fatalf("expected aGVsbG8=, got %q", result.NewText)
	}
}
