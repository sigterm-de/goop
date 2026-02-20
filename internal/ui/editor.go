package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	gtksource "libdb.so/gotk4-sourceview/pkg/gtksource/v5"
)

// Editor wraps a GtkSourceView and its buffer, providing text access and a
// single-level undo capability.
type Editor struct {
	View   *gtksource.View
	buffer *gtksource.Buffer

	undoSnapshot string
	hasSnapshot  bool
}

// NewEditor creates and initialises an Editor widget.
func NewEditor() *Editor {
	buf := gtksource.NewBuffer(nil)

	view := gtksource.NewViewWithBuffer(buf)
	view.SetWrapMode(gtk.WrapWord)
	view.SetMonospace(true)
	view.SetShowLineNumbers(false)
	view.SetAutoIndent(true)
	view.SetTabWidth(4)
	view.AddCSSClass("goop-editor")

	// Visual comfort — generous padding so text doesn't crowd the window edges.
	view.SetTopMargin(20)
	view.SetBottomMargin(20)
	view.SetLeftMargin(24)
	view.SetRightMargin(24)

	return &Editor{View: view, buffer: buf}
}

// GetFullText returns the complete text currently in the buffer.
func (e *Editor) GetFullText() string {
	start := e.buffer.StartIter()
	end := e.buffer.EndIter()
	return e.buffer.Text(start, end, true)
}

// SetFullText replaces the entire buffer content.
func (e *Editor) SetFullText(text string) {
	e.buffer.SetText(text)
}

// GetSelectedText returns the currently selected text, or the full text if
// nothing is selected.
func (e *Editor) GetSelectedText() string {
	start, end, ok := e.buffer.SelectionBounds()
	if !ok {
		return e.GetFullText()
	}
	return e.buffer.Text(start, end, true)
}

// GetSelection returns the 0-based character offsets of the current selection.
// When nothing is selected, start == end (cursor position).
func (e *Editor) GetSelection() (start, end int) {
	startIter, endIter, ok := e.buffer.SelectionBounds()
	if !ok {
		ins := e.buffer.IterAtMark(e.buffer.GetInsert())
		off := ins.Offset()
		return off, off
	}
	return startIter.Offset(), endIter.Offset()
}

// ReplaceSelection replaces the current selection with text, or replaces the
// full document if nothing is selected.
func (e *Editor) ReplaceSelection(text string) {
	start, end, ok := e.buffer.SelectionBounds()
	if !ok {
		e.SetFullText(text)
		return
	}
	e.buffer.BeginUserAction()
	e.buffer.Delete(start, end)
	ins := e.buffer.IterAtMark(e.buffer.GetInsert())
	e.buffer.Insert(ins, text)
	e.buffer.EndUserAction()
}

// InsertAtCursor inserts text at the current cursor position, replacing any
// active selection.
func (e *Editor) InsertAtCursor(text string) {
	e.buffer.InsertAtCursor(text)
}

// SaveUndoSnapshot saves the current full text so it can be restored by Undo.
// Call this before applying a transformation.
func (e *Editor) SaveUndoSnapshot() {
	e.undoSnapshot = e.GetFullText()
	e.hasSnapshot = true
}

// Undo restores the text saved by the last SaveUndoSnapshot call.
// Returns true if a snapshot was available and was applied.
func (e *Editor) Undo() bool {
	if !e.hasSnapshot {
		return false
	}
	e.SetFullText(e.undoSnapshot)
	e.hasSnapshot = false
	return true
}

// SetEnabled enables or disables keyboard input on the editor widget.
func (e *Editor) SetEnabled(enabled bool) {
	e.View.SetEditable(enabled)
	e.View.SetCursorVisible(enabled)
}

// ApplyScheme applies a GtkSourceView style scheme by ID. If the scheme is not
// found the buffer's current scheme is left unchanged.
func (e *Editor) ApplyScheme(schemeID string) {
	mgr := gtksource.StyleSchemeManagerGetDefault()
	if scheme := mgr.Scheme(schemeID); scheme != nil {
		e.buffer.SetStyleScheme(scheme)
	}
}
