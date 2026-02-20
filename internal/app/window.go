package app

import (
	"codeberg.org/daniel-ciaglia/goop/internal/engine"
	"codeberg.org/daniel-ciaglia/goop/internal/logging"
	"codeberg.org/daniel-ciaglia/goop/internal/scripts"
	"codeberg.org/daniel-ciaglia/goop/internal/ui"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// ApplicationWindow is the main window of goop.
type ApplicationWindow struct {
	Win      *gtk.ApplicationWindow
	editor   *ui.Editor
	picker   *ui.ScriptPicker
	status   *ui.StatusBar
	revealer *gtk.Revealer
	LogPath  string
	prefs    AppPreferences
}

// NewApplicationWindow builds the complete UI hierarchy and wires keyboard
// shortcuts.
func NewApplicationWindow(
	app *gtk.Application,
	lib scripts.Library,
	exec engine.Executor,
	logPath string,
	prefs AppPreferences,
) *ApplicationWindow {
	w := &ApplicationWindow{LogPath: logPath, prefs: prefs}

	// ── Core widgets ─────────────────────────────────────────────────────────
	w.editor = ui.NewEditor()
	w.editor.ApplyScheme(resolveActiveScheme(prefs))
	w.status = ui.NewStatusBar()

	// ── Script picker revealer (overlay panel) ────────────────────────────────
	w.revealer = gtk.NewRevealer()
	w.revealer.SetTransitionType(gtk.RevealerTransitionTypeSlideLeft)
	w.revealer.SetTransitionDuration(150)
	w.revealer.SetHAlign(gtk.AlignEnd)
	w.revealer.SetVExpand(true)

	w.picker = ui.NewScriptPicker(lib, exec, w.editor, w.status, logPath, w.HideScriptPicker)

	pickerFrame := gtk.NewFrame("")
	pickerFrame.SetChild(w.picker.Box)
	pickerFrame.AddCSSClass("picker-frame")
	w.revealer.SetChild(pickerFrame)

	// ── Overlay: editor as base + picker panel as overlay ────────────────────
	overlay := gtk.NewOverlay()
	overlay.SetChild(w.editor.View)
	overlay.AddOverlay(w.revealer)
	overlay.SetVExpand(true)

	// ── Root layout ──────────────────────────────────────────────────────────
	root := gtk.NewBox(gtk.OrientationVertical, 0)
	root.Append(overlay)
	root.Append(w.status.Box)

	// ── Application window ───────────────────────────────────────────────────
	w.Win = gtk.NewApplicationWindow(app)
	w.Win.SetTitle("goop")
	w.Win.SetIconName("goop")
	w.Win.SetDefaultSize(1000, 700)
	w.Win.SetChild(root)

	// ── Header bar ───────────────────────────────────────────────────────────
	header := gtk.NewHeaderBar()
	header.SetShowTitleButtons(true)

	scriptsBtn := gtk.NewButton()
	scriptsBtn.SetIconName("system-search-symbolic")
	scriptsBtn.SetTooltipText("Scripts (Ctrl+/)")
	scriptsBtn.AddCSSClass("flat")
	scriptsBtn.ConnectClicked(func() { w.ToggleScriptPicker() })
	header.PackEnd(scriptsBtn)

	settingsBtn := gtk.NewButton()
	settingsBtn.SetIconName("preferences-system-symbolic")
	settingsBtn.SetTooltipText("Preferences")
	settingsBtn.AddCSSClass("flat")
	settingsBtn.ConnectClicked(func() {
		ShowSettingsDialog(w.Win, w.prefs, func(newPrefs AppPreferences) {
			w.prefs = newPrefs
			applyPreferences(newPrefs)
			w.editor.ApplyScheme(resolveActiveScheme(newPrefs))
			if err := SavePreferences(newPrefs); err != nil {
				logging.Log(logging.WARN, "", "preferences: "+err.Error())
			}
		})
	})
	header.PackEnd(settingsBtn)

	w.Win.SetTitlebar(header)

	// ── Watch system dark/light mode ─────────────────────────────────────────
	// Always register the watch; only act when the preference is enabled.
	if gtkSettings := gtk.SettingsGetDefault(); gtkSettings != nil {
		gtkSettings.NotifyProperty("gtk-application-prefer-dark-theme", func() {
			if w.prefs.EditorSchemeFollowSystem {
				w.editor.ApplyScheme(resolveActiveScheme(w.prefs))
			}
		})
	}

	// ── Keyboard shortcuts ────────────────────────────────────────────────────
	w.registerActions()
	w.setupKeyboard()

	return w
}

// ShowScriptPicker reveals the picker panel and focuses the search entry.
func (w *ApplicationWindow) ShowScriptPicker() {
	w.picker.Reset()
	w.revealer.SetRevealChild(true)
	w.picker.Focus()
}

// HideScriptPicker hides the picker panel and returns focus to the editor.
func (w *ApplicationWindow) HideScriptPicker() {
	w.revealer.SetRevealChild(false)
	w.editor.View.GrabFocus()
}

// ToggleScriptPicker shows or hides the picker panel.
func (w *ApplicationWindow) ToggleScriptPicker() {
	if w.revealer.RevealChild() {
		w.HideScriptPicker()
	} else {
		w.ShowScriptPicker()
	}
}

// registerActions adds named window actions so that application-level
// accelerators (set in app.go via SetAccelsForAction) can trigger them.
func (w *ApplicationWindow) registerActions() {
	toggleAction := gio.NewSimpleAction("toggle-picker", nil)
	toggleAction.ConnectActivate(func(_ *glib.Variant) {
		w.ToggleScriptPicker()
	})
	w.Win.AddAction(toggleAction)
}

func (w *ApplicationWindow) setupKeyboard() {
	ctrl := gtk.NewEventControllerKey()
	ctrl.SetPropagationPhase(gtk.PhaseCapture)
	ctrl.ConnectKeyPressed(func(keyval, keycode uint, state gdk.ModifierType) bool {
		ctrlMask := gdk.ControlMask
		if keyval == gdk.KEY_Escape && w.revealer.RevealChild() {
			w.HideScriptPicker()
			return true
		}
		if state&ctrlMask != 0 && (keyval == 'z' || keyval == 'Z') {
			w.editor.Undo()
			return true
		}
		return false
	})
	w.Win.AddController(ctrl)
}
