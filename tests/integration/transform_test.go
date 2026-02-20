// Package integration contains end-to-end transformation tests and build
// smoke tests.
package integration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"codeberg.org/daniel-ciaglia/goop/assets"
	"codeberg.org/daniel-ciaglia/goop/internal/engine"
	"codeberg.org/daniel-ciaglia/goop/internal/scripts"
)

// loadScript returns the Content of the first script with the given name from
// the embedded library, or skips the test if not found.
func loadScript(t *testing.T, name string) string {
	t.Helper()
	result, err := scripts.NewLoader(assets.Scripts()).Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, s := range result.Scripts {
		if s.Name == name {
			return s.Content
		}
	}
	t.Skipf("built-in script %q not found — skipping", name)
	return ""
}

func execScript(t *testing.T, scriptName, scriptContent, fullText string) engine.ExecutionResult {
	t.Helper()
	inp := engine.ExecutionInput{
		ScriptSource:   scriptContent,
		ScriptName:     scriptName,
		FullText:       fullText,
		SelectionText:  fullText,
		SelectionStart: 0,
		SelectionEnd:   len(fullText),
		Timeout:        5 * time.Second,
	}
	return engine.NewExecutor().Execute(context.Background(), inp)
}

// TestURLEncode verifies the URL Encode script.
func TestURLEncode(t *testing.T) {
	content := loadScript(t, "URL Encode")
	result := execScript(t, "URL Encode", content, "hello world")
	if !result.Success {
		t.Fatalf("URL Encode failed: %s", result.ErrorMessage)
	}
	var out string
	switch result.MutationKind {
	case engine.MutationReplaceSelect:
		out = result.NewText
	case engine.MutationReplaceDoc:
		out = result.NewFullText
	default:
		t.Fatalf("unexpected MutationKind: %d", result.MutationKind)
	}
	if !strings.Contains(out, "%20") && !strings.Contains(out, "+") {
		t.Errorf("expected percent-encoded space, got: %q", out)
	}
}

// TestBase64Encode verifies the Base64 Encode script.
func TestBase64Encode(t *testing.T) {
	content := loadScript(t, "Base64 Encode")
	result := execScript(t, "Base64 Encode", content, "hello")
	if !result.Success {
		t.Fatalf("Base64 Encode failed: %s", result.ErrorMessage)
	}
	var out string
	switch result.MutationKind {
	case engine.MutationReplaceSelect:
		out = result.NewText
	case engine.MutationReplaceDoc:
		out = result.NewFullText
	default:
		t.Fatalf("unexpected MutationKind: %d", result.MutationKind)
	}
	// "hello" base64 = "aGVsbG8="
	if !strings.Contains(out, "aGVsbG8=") {
		t.Errorf("expected base64 of 'hello' to contain 'aGVsbG8=', got: %q", out)
	}
}

// TestJSONPrettify verifies the Format JSON script with valid input.
func TestJSONPrettify(t *testing.T) {
	content := loadScript(t, "Format JSON")
	result := execScript(t, "Format JSON", content, `{"a":1}`)
	if !result.Success {
		t.Fatalf("Format JSON failed: %s", result.ErrorMessage)
	}
	var out string
	switch result.MutationKind {
	case engine.MutationReplaceSelect:
		out = result.NewText
	case engine.MutationReplaceDoc:
		out = result.NewFullText
	default:
		t.Fatalf("unexpected MutationKind: %d", result.MutationKind)
	}
	if !strings.Contains(out, "\n") {
		t.Errorf("expected prettified JSON with newlines, got: %q", out)
	}
}
