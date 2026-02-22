// Package contract contains acceptance tests for the @boop/yaml module.
// These tests map to TC modules contract cases in contracts/engine.md.
package contract_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"codeberg.org/sigterm-de/goop/internal/engine"
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
