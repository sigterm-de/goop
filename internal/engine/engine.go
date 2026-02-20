package engine

import (
	"context"
	"time"
)

// MutationKind describes which mutation (if any) a script applied.
type MutationKind int

const (
	MutationNone           MutationKind = iota // Script made no changes
	MutationReplaceDoc                         // state.fullText was written
	MutationReplaceSelect                      // state.text was written
	MutationInsertAtCursor                     // state.insert() was called
)

// ExecutionInput carries everything the engine needs to run a single script.
type ExecutionInput struct {
	ScriptSource   string        // Full JS source text of the script
	ScriptName     string        // Display name (for error messages and log entries)
	FullText       string        // Current full editor content
	SelectionText  string        // Selected text (equals FullText if no selection)
	SelectionStart int           // 0-based character offset of selection start
	SelectionEnd   int           // 0-based character offset of selection end
	Timeout        time.Duration // Hard execution timeout (typically 5 s)
}

// ExecutionResult is the structured outcome returned by Execute.
// MutationKind and the New* fields are only valid when Success == true.
type ExecutionResult struct {
	Success      bool
	MutationKind MutationKind
	NewFullText  string // Valid when MutationKind == MutationReplaceDoc
	NewText      string // Valid when MutationKind == MutationReplaceSelect
	InsertText   string // Valid when MutationKind == MutationInsertAtCursor
	ErrorMessage string // Human-readable; valid when Success == false
	InfoMessage  string // Set when the script called postInfo(); shown in status bar
	ScriptName   string
	TimedOut     bool
}

// Executor runs a single JavaScript script against a given input.
// Implementations MUST be safe to call from any goroutine.
// Each call creates a fresh JS runtime — no state persists between calls.
type Executor interface {
	Execute(ctx context.Context, input ExecutionInput) ExecutionResult
}
