package app

import (
	"sort"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
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
	return `textview.goop-editor { font-family: "` + family + `"; font-size: ` + formatPt(sizePoints) + `pt; }`
}

// formatPt formats a float to at most one decimal place, dropping the decimal
// when it is zero (e.g. 12.0 → "12", 10.5 → "10.5").
func formatPt(v float64) string {
	i := int(v)
	if float64(i) == v {
		return itoa(i)
	}
	frac := int((v-float64(i))*10 + 0.5)
	return itoa(i) + "." + itoa(frac)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
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

	schemeFollowCheck := gtk.NewCheckButtonWithLabel("Follow system dark/light")
	schemeFollowCheck.SetActive(prefs.EditorSchemeFollowSystem)

	lightDrop, lightIDs := buildSchemeDropDown(prefs.EditorSchemeLight)
	darkDrop, darkIDs := buildSchemeDropDown(prefs.EditorSchemeDark)

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
		prefs = p
		onApply(p)
	}

	fontBtn.NotifyProperty("font-desc", func() { applyChanges() })
	monoCheck.ConnectToggled(func() { applyChanges() })
	schemeFollowCheck.ConnectToggled(func() { applyChanges() })
	lightDrop.NotifyProperty("selected", func() { applyChanges() })
	darkDrop.NotifyProperty("selected", func() { applyChanges() })

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

	attachLabel("Editor")
	attachRow("Font:", fontBtn)
	attachSpan(monoCheck)
	attachSep()
	attachLabel("Colour scheme")
	attachSpan(schemeFollowCheck)
	attachRow("Light scheme:", lightDrop)
	attachRow("Dark scheme:", darkDrop)

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
