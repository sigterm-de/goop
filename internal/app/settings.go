package app

import (
	"fmt"
	"sort"
	"strconv"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	gtksource "libdb.so/gotk4-sourceview/pkg/gtksource/v5"
)

// pangoDescToCSS converts a Pango font description string (e.g. "Monospace 12")
// into a CSS snippet targeting the editor widget.
func pangoDescToCSS(descStr string) string {
	desc := pango.FontDescriptionFromString(descStr)
	if desc == nil {
		return `textview.goop-editor { font-family: "Monospace"; font-size: 12pt; }`
	}
	family := desc.Family()
	if family == "" {
		family = "Monospace"
	}
	// Pango stores size in units of 1/1024 point.
	const pangoScale = 1024
	sizePoints := float64(desc.Size()) / pangoScale
	if sizePoints <= 0 {
		sizePoints = 12
	}
	// Use strconv for correct, allocation-efficient float formatting.
	sizePt := strconv.FormatFloat(sizePoints, 'f', -1, 64)
	return fmt.Sprintf(`textview.goop-editor { font-family: %q; font-size: %spt; }`, family, sizePt)
}

// buildSchemeDropDown constructs a DropDown listing all available GtkSourceView
// style schemes sorted by human-readable name, with currentID pre-selected.
// The returned slice mirrors the dropdown's indices → scheme IDs.
func buildSchemeDropDown(currentID string) (*gtk.DropDown, []string) {
	type entry struct{ id, name string }
	mgr := gtksource.StyleSchemeManagerGetDefault()
	var entries []entry
	for _, id := range mgr.SchemeIDs() {
		name := id
		if s := mgr.Scheme(id); s != nil {
			name = s.Name()
		}
		entries = append(entries, entry{id, name})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	names := make([]string, len(entries))
	ids := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
		ids[i] = e.id
	}

	drop := gtk.NewDropDownFromStrings(names)
	drop.SetHExpand(true)
	for i, id := range ids {
		if id == currentID {
			drop.SetSelected(uint(i))
			break
		}
	}
	return drop, ids
}

// isModifierKey reports whether keyval is a bare modifier key (Ctrl, Shift,
// Alt, Super, …) that should be ignored during shortcut capture.
func isModifierKey(keyval uint) bool {
	switch keyval {
	case gdk.KEY_Control_L, gdk.KEY_Control_R,
		gdk.KEY_Shift_L, gdk.KEY_Shift_R,
		gdk.KEY_Alt_L, gdk.KEY_Alt_R,
		gdk.KEY_Super_L, gdk.KEY_Super_R,
		gdk.KEY_Meta_L, gdk.KEY_Meta_R,
		gdk.KEY_Hyper_L, gdk.KEY_Hyper_R,
		gdk.KEY_ISO_Level3_Shift:
		return true
	}
	return false
}

// monospaceFontFilter returns a Filter that passes only monospace font
// families. Pass nil to FontDialog.SetFilter to clear it.
//
// FontDialog's list model contains PangoFontFace items (one row per face).
// We reach the family via face.Family() and then check IsMonospace.
func monospaceFontFilter() *gtk.Filter {
	cf := gtk.NewCustomFilter(func(item *coreglib.Object) bool {
		face, ok := item.Cast().(*pango.FontFace)
		if !ok {
			return true
		}
		family, ok := face.Family().(*pango.FontFamily)
		if !ok {
			return true
		}
		return family.IsMonospace()
	})
	return &cf.Filter
}

// ShowSettingsDialog opens a modal preferences window transient to parent.
func ShowSettingsDialog(
	parent *gtk.ApplicationWindow,
	prefs AppPreferences,
	onApply func(AppPreferences),
) {
	win := gtk.NewWindow()
	win.SetTitle("Preferences")
	win.SetTransientFor(&parent.Window)
	win.SetModal(true)
	win.SetDefaultSize(440, 0)
	win.SetResizable(false)
	win.SetDestroyWithParent(true)

	// ── Editor ────────────────────────────────────────────────────────────────
	fontDialog := gtk.NewFontDialog()
	fontBtn := gtk.NewFontDialogButton(fontDialog)
	fontBtn.SetUseFont(true)
	fontBtn.SetUseSize(true)
	fontBtn.SetHExpand(true)
	if desc := pango.FontDescriptionFromString(prefs.EditorFont); desc != nil {
		fontBtn.SetFontDesc(desc)
	}
	if prefs.FontMonospaceOnly {
		fontDialog.SetFilter(monospaceFontFilter())
	}

	monoCheck := gtk.NewCheckButtonWithLabel("Monospace fonts only")
	monoCheck.SetActive(prefs.FontMonospaceOnly)
	monoCheck.ConnectToggled(func() {
		if monoCheck.Active() {
			fontDialog.SetFilter(monospaceFontFilter())
		} else {
			fontDialog.SetFilter(nil)
		}
	})

	syntaxDetectCheck := gtk.NewCheckButtonWithLabel("Auto-detect syntax highlighting")
	syntaxDetectCheck.SetActive(prefs.SyntaxAutoDetect)
	syntaxDetectCheck.SetTooltipText("Automatically apply syntax highlighting after running a script")

	schemeFollowCheck := gtk.NewCheckButtonWithLabel("Follow system dark/light")
	schemeFollowCheck.SetActive(prefs.EditorSchemeFollowSystem)

	lightDrop, lightIDs := buildSchemeDropDown(prefs.EditorSchemeLight)
	darkDrop, darkIDs := buildSchemeDropDown(prefs.EditorSchemeDark)

	// ── Keyboard shortcut capture ─────────────────────────────────────────────
	currentAccel := prefs.ScriptPickerShortcut
	capturing := false

	shortcutBtn := gtk.NewButton()
	shortcutBtn.SetLabel(accelToLabel(currentAccel))
	shortcutBtn.SetHExpand(true)
	shortcutBtn.SetTooltipText("Click, then press a key combination")

	// ── Apply helper ─────────────────────────────────────────────────────────
	applyChanges := func() {
		p := prefs
		if fd := fontBtn.FontDesc(); fd != nil {
			p.EditorFont = fd.String()
		}
		p.FontMonospaceOnly = monoCheck.Active()
		p.EditorSchemeFollowSystem = schemeFollowCheck.Active()
		if idx := lightDrop.Selected(); int(idx) < len(lightIDs) {
			p.EditorSchemeLight = lightIDs[idx]
		}
		if idx := darkDrop.Selected(); int(idx) < len(darkIDs) {
			p.EditorSchemeDark = darkIDs[idx]
		}
		p.ScriptPickerShortcut = currentAccel
		p.SyntaxAutoDetect = syntaxDetectCheck.Active()
		prefs = p
		onApply(p)
	}

	fontBtn.NotifyProperty("font-desc", func() { applyChanges() })
	monoCheck.ConnectToggled(func() { applyChanges() })
	syntaxDetectCheck.ConnectToggled(func() { applyChanges() })
	schemeFollowCheck.ConnectToggled(func() { applyChanges() })
	lightDrop.NotifyProperty("selected", func() { applyChanges() })
	darkDrop.NotifyProperty("selected", func() { applyChanges() })

	// Key capture: listen on the settings window for keypresses when active.
	keyCtrl := gtk.NewEventControllerKey()
	keyCtrl.SetPropagationPhase(gtk.PhaseCapture)
	keyCtrl.ConnectKeyPressed(func(keyval, _ uint, state gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			if capturing {
				// Cancel shortcut capture and restore previous label.
				shortcutBtn.SetLabel(accelToLabel(currentAccel))
				capturing = false
			} else {
				// Close the dialog.
				win.Close()
			}
			return true
		}
		if !capturing {
			return false
		}
		if isModifierKey(keyval) {
			return false // wait for a non-modifier key
		}
		state &= gtk.AcceleratorGetDefaultModMask()
		accel := gtk.AcceleratorName(keyval, state)
		if accel == "" {
			return false
		}
		currentAccel = accel
		shortcutBtn.SetLabel(gtk.AcceleratorGetLabel(keyval, state))
		capturing = false
		applyChanges()
		return true
	})

	// Cancel capture when the button loses keyboard focus.
	focusCtrl := gtk.NewEventControllerFocus()
	focusCtrl.ConnectLeave(func() {
		if capturing {
			capturing = false
			shortcutBtn.SetLabel(accelToLabel(currentAccel))
		}
	})
	shortcutBtn.AddController(focusCtrl)

	shortcutBtn.ConnectClicked(func() {
		capturing = true
		shortcutBtn.SetLabel("Press a key combination…")
		shortcutBtn.GrabFocus()
	})

	// ── Layout ───────────────────────────────────────────────────────────────
	grid := gtk.NewGrid()
	grid.SetRowSpacing(10)
	grid.SetColumnSpacing(12)
	grid.SetMarginTop(20)
	grid.SetMarginBottom(12)
	grid.SetMarginStart(20)
	grid.SetMarginEnd(20)

	row := 0
	attachLabel := func(text string) {
		lbl := gtk.NewLabel("<b>" + text + "</b>")
		lbl.SetUseMarkup(true)
		lbl.SetXAlign(0)
		if row > 0 {
			lbl.SetMarginTop(6)
		}
		grid.Attach(lbl, 0, row, 2, 1)
		row++
	}
	attachRow := func(label string, widget gtk.Widgetter) {
		lbl := gtk.NewLabel(label)
		lbl.SetXAlign(1)
		grid.Attach(lbl, 0, row, 1, 1)
		grid.Attach(widget, 1, row, 1, 1)
		row++
	}
	attachSpan := func(widget gtk.Widgetter) {
		grid.Attach(widget, 0, row, 2, 1)
		row++
	}
	attachSep := func() {
		sep := gtk.NewSeparator(gtk.OrientationHorizontal)
		sep.SetMarginTop(4)
		sep.SetMarginBottom(4)
		grid.Attach(sep, 0, row, 2, 1)
		row++
	}

	// Register the key capture controller on the settings window so it
	// intercepts key events regardless of which widget has focus.
	win.AddController(keyCtrl)

	attachLabel("Editor")
	attachRow("Font:", fontBtn)
	attachSpan(monoCheck)
	attachSpan(syntaxDetectCheck)
	attachSep()
	attachLabel("Colour scheme")
	attachSpan(schemeFollowCheck)
	attachRow("Light scheme:", lightDrop)
	attachRow("Dark scheme:", darkDrop)
	attachSep()
	attachLabel("Keyboard")
	attachRow("Script picker:", shortcutBtn)

	closeBtn := gtk.NewButtonWithLabel("Close")
	closeBtn.SetHAlign(gtk.AlignEnd)
	closeBtn.SetMarginTop(8)
	closeBtn.SetMarginEnd(20)
	closeBtn.SetMarginBottom(16)
	closeBtn.ConnectClicked(func() { win.Close() })

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(grid)
	box.Append(closeBtn)

	win.SetChild(box)
	win.Present()
}
