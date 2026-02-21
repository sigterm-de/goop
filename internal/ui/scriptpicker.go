package ui

import (
	"context"

	"codeberg.org/sigterm-de/goop/internal/engine"
	"codeberg.org/sigterm-de/goop/internal/logging"
	"codeberg.org/sigterm-de/goop/internal/scripts"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

// ScriptPicker combines a search entry and a script list into a panel that
// users interact with to select and run scripts.
type ScriptPicker struct {
	Box     *gtk.Box
	library scripts.Library
	exec    engine.Executor
	editor  *Editor
	status  *StatusBar
	logPath string

	listBox     *gtk.ListBox
	searchEntry *gtk.SearchEntry
	allScripts  []scripts.Script
	onHide      func()
	postScript  func() // called after every successful script execution
}

// NewScriptPicker creates the script picker panel.
// postScript, if non-nil, is called on the GTK main thread after every
// successful script execution — use it to run syntax detection or other
// post-transform work without coupling ScriptPicker to those details.
func NewScriptPicker(
	lib scripts.Library,
	exec engine.Executor,
	editor *Editor,
	status *StatusBar,
	logPath string,
	onHide func(),
	postScript func(),
) *ScriptPicker {
	sp := &ScriptPicker{
		library:    lib,
		exec:       exec,
		editor:     editor,
		status:     status,
		logPath:    logPath,
		allScripts: lib.All(),
		onHide:     onHide,
		postScript: postScript,
	}

	// ── Search entry ─────────────────────────────────────────────────────────
	sp.searchEntry = gtk.NewSearchEntry()
	sp.searchEntry.SetPlaceholderText("Search scripts…")
	sp.searchEntry.SetHExpand(true)
	sp.searchEntry.ConnectSearchChanged(func() {
		query := sp.searchEntry.Text()
		var results []scripts.Script
		if query == "" {
			results = sp.allScripts
		} else {
			results = sp.library.Search(query)
		}
		sp.setScripts(results)
	})

	// Key controller on the search entry: intercept Down (move to list) and
	// Enter (run first/selected script). ConnectNextMatch fires on Ctrl+G,
	// not Down arrow, so we need an explicit controller here.
	searchCtrl := gtk.NewEventControllerKey()
	searchCtrl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		switch keyval {
		case gdk.KEY_Down:
			sp.focusList()
			return true
		case gdk.KEY_Return, gdk.KEY_KP_Enter:
			sp.activateSelected()
			return true
		}
		return false
	})
	sp.searchEntry.AddController(searchCtrl)

	// ── Script list ──────────────────────────────────────────────────────────
	sp.listBox = gtk.NewListBox()
	sp.listBox.SetSelectionMode(gtk.SelectionSingle)
	sp.listBox.AddCSSClass("script-list")
	sp.listBox.ConnectRowActivated(func(row *gtk.ListBoxRow) {
		idx := row.Index()
		if idx < 0 || idx >= len(sp.allScripts) {
			return
		}
		sp.runScript(sp.allScripts[idx])
	})

	// Key controller on the list in PhaseCapture so we intercept Up/Down
	// before any child GtkListBoxRow handles them. We drive selection manually
	// because GTK's native ListBox navigation stalls once focus lands on a
	// specific child row.
	listCtrl := gtk.NewEventControllerKey()
	listCtrl.SetPropagationPhase(gtk.PhaseCapture)
	listCtrl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		switch keyval {
		case gdk.KEY_Up:
			row := sp.listBox.SelectedRow()
			if row == nil || row.Index() == 0 {
				// Back to search entry when at the top.
				sp.searchEntry.GrabFocus()
				return true
			}
			if prev := sp.listBox.RowAtIndex(row.Index() - 1); prev != nil {
				sp.listBox.SelectRow(prev)
				prev.GrabFocus()
			}
			return true
		case gdk.KEY_Down:
			nextIdx := 0
			if row := sp.listBox.SelectedRow(); row != nil {
				nextIdx = row.Index() + 1
			}
			if next := sp.listBox.RowAtIndex(nextIdx); next != nil {
				sp.listBox.SelectRow(next)
				next.GrabFocus()
			}
			return true
		case gdk.KEY_Return, gdk.KEY_KP_Enter:
			sp.activateSelected()
			return true
		case gdk.KEY_Escape:
			if sp.onHide != nil {
				sp.onHide()
			}
			return true
		}
		return false
	})
	sp.listBox.AddController(listCtrl)

	sp.setScripts(sp.allScripts)

	// ── Scrolled window around list ──────────────────────────────────────────
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetHExpand(true)
	scroll.SetChild(sp.listBox)

	// ── Panel layout ─────────────────────────────────────────────────────────
	sp.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	sp.Box.AddCSSClass("script-picker")
	sp.Box.Append(sp.searchEntry)
	sp.Box.Append(scroll)

	return sp
}

// setScripts repopulates the list box with the given scripts.
func (sp *ScriptPicker) setScripts(list []scripts.Script) {
	sp.allScripts = list

	// Remove all existing rows
	for {
		row := sp.listBox.RowAtIndex(0)
		if row == nil {
			break
		}
		sp.listBox.Remove(row)
	}

	if len(list) == 0 {
		noMatch := gtk.NewLabel("No matching scripts")
		noMatch.AddCSSClass("no-results-label")
		sp.listBox.Append(noMatch)
		return
	}

	for _, s := range list {
		sp.listBox.Append(buildScriptRow(s))
	}
}

// buildScriptRow creates a single list row for a script.
func buildScriptRow(s scripts.Script) *gtk.Box {
	nameLabel := gtk.NewLabel(s.Name)
	nameLabel.SetXAlign(0)
	nameLabel.AddCSSClass("script-name")

	descLabel := gtk.NewLabel(s.Description)
	descLabel.SetXAlign(0)
	descLabel.AddCSSClass("script-desc")
	descLabel.SetEllipsize(pango.EllipsizeEnd)

	textBox := gtk.NewBox(gtk.OrientationVertical, 2)
	textBox.SetHExpand(true)
	textBox.Append(nameLabel)
	textBox.Append(descLabel)

	row := gtk.NewBox(gtk.OrientationHorizontal, 8)
	row.SetMarginTop(6)
	row.SetMarginBottom(6)
	row.SetMarginStart(8)
	row.SetMarginEnd(8)

	img := gtk.NewImageFromIconName(boopIconName(s.Icon))
	img.SetPixelSize(16)
	img.AddCSSClass("script-icon")
	row.Append(img)

	row.Append(textBox)

	if s.Source == scripts.UserProvided {
		badge := gtk.NewLabel("user")
		badge.AddCSSClass("user-script-badge")
		row.Append(badge)
	}

	return row
}

// boopIconName maps a Boop script icon name to the nearest GTK symbolic icon
// name. Falls back to a generic text icon for unknown names.
func boopIconName(icon string) string {
	switch icon {
	case "abacus", "counter", "percentage":
		return "accessories-calculator-symbolic"
	case "broom":
		return "edit-clear-symbolic"
	case "camel", "kebab", "snake", "type":
		return "font-x-generic-symbolic"
	case "collapse":
		return "view-more-horizontal-symbolic"
	case "color-wheel":
		return "preferences-color-symbolic"
	case "colosseum", "roman":
		return "trophy-symbolic"
	case "command", "term":
		return "utilities-terminal-symbolic"
	case "dice":
		return "media-playlist-shuffle-symbolic"
	case "elephant", "pineapple":
		return "text-x-generic-symbolic"
	case "filtration":
		return "edit-find-symbolic"
	case "fingerprint":
		return "fingerprint-symbolic"
	case "flask":
		return "applications-science-symbolic"
	case "flip":
		return "object-flip-vertical-symbolic"
	case "HTML":
		return "text-html-symbolic"
	case "identification":
		return "contact-new-symbolic"
	case "link":
		return "insert-link-symbolic"
	case "metamorphose":
		return "emblem-synchronizing-symbolic"
	case "quote":
		return "format-text-blockquote-symbolic"
	case "scissors":
		return "edit-cut-symbolic"
	case "sort-characters", "sort-numbers":
		return "view-sort-ascending-symbolic"
	case "table":
		return "x-office-spreadsheet-symbolic"
	case "translation":
		return "preferences-desktop-locale-symbolic"
	case "watch":
		return "alarm-symbolic"
	case "website":
		return "web-browser-symbolic"
	default:
		return "text-x-generic-symbolic"
	}
}

// Focus moves keyboard focus to the search entry.
func (sp *ScriptPicker) Focus() {
	sp.searchEntry.GrabFocus()
}

// focusList moves focus to the list box and selects row 0.
// Focusing the row directly (rather than the list container) ensures GTK
// auto-scrolls it into view and keeps the PhaseCapture key controller active.
func (sp *ScriptPicker) focusList() {
	row := sp.listBox.RowAtIndex(0)
	if row == nil {
		return
	}
	sp.listBox.SelectRow(row)
	row.GrabFocus()
}

// activateSelected runs the currently selected script, or the first script
// if nothing is selected.
func (sp *ScriptPicker) activateSelected() {
	row := sp.listBox.SelectedRow()
	if row == nil {
		row = sp.listBox.RowAtIndex(0)
	}
	if row == nil {
		return
	}
	idx := row.Index()
	if idx >= 0 && idx < len(sp.allScripts) {
		sp.runScript(sp.allScripts[idx])
	}
}

// Reset clears the search and restores the full script list.
func (sp *ScriptPicker) Reset() {
	sp.searchEntry.SetText("")
	sp.setScripts(sp.library.All())
	sp.allScripts = sp.library.All()
}

// runScript executes the given script against the current editor content.
// The execution runs in a goroutine; results are marshalled back to the GTK
// main thread via glib.IdleAdd.
func (sp *ScriptPicker) runScript(s scripts.Script) {
	if sp.onHide != nil {
		sp.onHide()
	}

	sp.editor.SaveUndoSnapshot()
	sp.editor.SetEnabled(false)

	fullText := sp.editor.GetFullText()
	selText := sp.editor.GetSelectedText()
	selStart, selEnd := sp.editor.GetSelection()

	inp := engine.ExecutionInput{
		ScriptSource:   s.Content,
		ScriptName:     s.Name,
		FullText:       fullText,
		SelectionText:  selText,
		SelectionStart: selStart,
		SelectionEnd:   selEnd,
		Timeout:        5e9, // 5 seconds
	}

	go func() {
		result := sp.exec.Execute(context.Background(), inp)

		glib.IdleAdd(func() {
			sp.editor.SetEnabled(true)
			sp.applyResult(result)
		})
	}()
}

// applyResult applies the execution result to the editor and status bar.
func (sp *ScriptPicker) applyResult(result engine.ExecutionResult) {
	if !result.Success {
		logging.Log(logging.ERROR, result.ScriptName, result.ErrorMessage)
		sp.status.ShowError(result.ErrorMessage, sp.logPath)
		return
	}

	switch result.MutationKind {
	case engine.MutationReplaceDoc:
		sp.editor.SetFullText(result.NewFullText)
	case engine.MutationReplaceSelect:
		sp.editor.ReplaceSelection(result.NewText)
	case engine.MutationInsertAtCursor:
		sp.editor.InsertAtCursor(result.InsertText)
	}

	if result.InfoMessage != "" {
		sp.status.ShowSuccess(result.InfoMessage)
	} else {
		sp.status.ShowSuccess("✓ " + result.ScriptName + " applied")
	}

	if sp.postScript != nil {
		sp.postScript()
	}
}
