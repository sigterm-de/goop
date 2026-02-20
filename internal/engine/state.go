package engine

// ScriptState is the `state` object passed to every Boop script.
// It is exposed to the JavaScript VM via goja and tracks all mutations
// so that the write-semantics priority table can be applied after main() returns.
//
// Priority (highest first):
//  1. postError() called   → discard all mutations, show error
//  2. state.text written   → replace selection (or full doc if no selection)
//  3. state.fullText written → replace full document
//  4. state.insert() called → insert at cursor
//  5. nothing written      → no change
type ScriptState struct {
	// Exported fields — visible to the JS VM via TagFieldNameMapper("json").
	FullText      string    `json:"fullText"`
	Text          string    `json:"text"`
	SelectionInfo selection `json:"selection"`

	// Unexported tracking fields — not visible to JS.
	originalFullText string
	fullTextMutated  bool
	textMutated      bool
	errorPosted      bool
	errorMessage     string
	infoPosted       bool
	infoMessage      string
	insertText       string
	insertPending    bool
}

type selection struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// NewScriptState constructs a ScriptState from the execution input.
func NewScriptState(input ExecutionInput) *ScriptState {
	return &ScriptState{
		FullText: input.FullText,
		Text:     input.SelectionText,
		SelectionInfo: selection{
			Start: input.SelectionStart,
			End:   input.SelectionEnd,
		},
		originalFullText: input.FullText,
	}
}

// Insert implements state.insert(text) — inserts text at the cursor position.
// Records the intent; actual application is done by the UI layer.
func (s *ScriptState) Insert(text string) {
	s.insertText = text
	s.insertPending = true
}

// PostError implements state.postError(msg) — signals an error.
// All pending mutations are discarded; only the first call's message is kept.
func (s *ScriptState) PostError(msg string) {
	if !s.errorPosted {
		s.errorPosted = true
		s.errorMessage = msg
	}
}

// PostInfo implements state.postInfo(msg) — shows an informational message in
// the status bar without preventing text mutations. Only the first call is kept.
func (s *ScriptState) PostInfo(msg string) {
	if !s.infoPosted {
		s.infoPosted = true
		s.infoMessage = msg
	}
}

// Result applies the write-semantics priority table and returns the
// ExecutionResult that the caller should use to update the editor.
func (s *ScriptState) Result(scriptName string) ExecutionResult {
	base := ExecutionResult{ScriptName: scriptName}

	if s.errorPosted {
		base.Success = false
		base.ErrorMessage = s.errorMessage
		base.MutationKind = MutationNone
		return base
	}

	base.Success = true

	if s.infoPosted {
		base.InfoMessage = s.infoMessage
	}

	switch {
	case s.textMutated:
		base.MutationKind = MutationReplaceSelect
		base.NewText = s.Text

	case s.fullTextMutated:
		base.MutationKind = MutationReplaceDoc
		base.NewFullText = s.FullText

	case s.insertPending:
		base.MutationKind = MutationInsertAtCursor
		base.InsertText = s.insertText

	default:
		base.MutationKind = MutationNone
	}

	return base
}
