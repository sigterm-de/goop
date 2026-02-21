package app

import (
	"codeberg.org/sigterm-de/goop/internal/engine"
	"codeberg.org/sigterm-de/goop/internal/logging"
	"codeberg.org/sigterm-de/goop/internal/scripts"
	"codeberg.org/sigterm-de/goop/internal/ui"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// ApplicationWindow is the main window of goop.
type ApplicationWindow struct {
	Win        *gtk.ApplicationWindow
	editor     *ui.Editor
	picker     *ui.ScriptPicker
	status     *ui.StatusBar
	revealer   *gtk.Revealer
	LogPath    string
	prefs      AppPreferences
	app        *gtk.Application
	scriptsBtn *gtk.Button
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
	w := &ApplicationWindow{LogPath: logPath, prefs: prefs, app: app}

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

	w.scriptsBtn = gtk.NewButton()
	w.scriptsBtn.SetIconName("system-search-symbolic")
	w.scriptsBtn.AddCSSClass("flat")
	w.scriptsBtn.ConnectClicked(func() { w.ToggleScriptPicker() })
	header.PackEnd(w.scriptsBtn)

	settingsBtn := gtk.NewButton()
	settingsBtn.SetIconName("preferences-system-symbolic")
	settingsBtn.SetTooltipText("Preferences")
	settingsBtn.AddCSSClass("flat")
	settingsBtn.ConnectClicked(func() {
		// Sync the editor scheme with the current system state before opening,
		// in case the dark/light toggle happened while the app was running but
		// the GtkSettings notification was not delivered (e.g. portal delay).
		if w.prefs.EditorSchemeFollowSystem {
			w.editor.ApplyScheme(resolveActiveScheme(w.prefs))
		}
		ShowSettingsDialog(w.Win, w.prefs, func(newPrefs AppPreferences) {
			if newPrefs.ScriptPickerShortcut != w.prefs.ScriptPickerShortcut {
				w.app.SetAccelsForAction("win.toggle-picker", []string{newPrefs.ScriptPickerShortcut})
			}
			w.prefs = newPrefs
			applyPreferences(newPrefs)
			w.editor.ApplyScheme(resolveActiveScheme(newPrefs))
			w.updateShortcutHints(newPrefs)
			if err := SavePreferences(newPrefs); err != nil {
				logging.Log(logging.WARN, "", "preferences: "+err.Error())
			}
		})
	})
	header.PackEnd(settingsBtn)

	w.Win.SetTitlebar(header)

	// ── Watch system dark/light mode ─────────────────────────────────────────
	applySchemeIfFollowing := func() {
		if w.prefs.EditorSchemeFollowSystem {
			w.editor.ApplyScheme(resolveActiveScheme(w.prefs))
		}
	}

	// GNOME: subscribe directly to org.gnome.desktop.interface color-scheme.
	// This fires reliably when the user or a script changes the GSettings key,
	// regardless of portal or GdkDisplay indirection.
	if gnomeInterfaceSettings != nil {
		gnomeInterfaceSettings.ConnectChanged(func(key string) {
			if key == "color-scheme" {
				applySchemeIfFollowing()
			}
		})
	}

	// KDE / other DEs: GdkDisplay::setting-changed covers theme-name switches
	// (e.g. Breeze → Breeze-Dark) and portal-based dark-mode changes that
	// don't go through org.gnome.desktop.interface.
	if display := gdk.DisplayGetDefault(); display != nil {
		display.ConnectSettingChanged(func(setting string) {
			switch setting {
			case "gtk-application-prefer-dark-theme", "gtk-theme-name":
				// Defer so GtkSettings has already applied the new value.
				glib.IdleAdd(applySchemeIfFollowing)
			}
		})
	}

	// ── Keyboard shortcuts ────────────────────────────────────────────────────
	w.registerActions()
	w.setupKeyboard()

	// Apply shortcut-derived hints (tooltip + status bar) from stored prefs.
	w.updateShortcutHints(prefs)

	return w
}

// accelToLabel converts a GTK accelerator string (e.g. "<Primary>slash") into
// a human-readable label (e.g. "Ctrl+/").
func accelToLabel(accel string) string {
	key, mods, ok := gtk.AcceleratorParse(accel)
	if !ok || key == 0 {
		return accel
	}
	return gtk.AcceleratorGetLabel(key, mods)
}

// updateShortcutHints refreshes the scripts-button tooltip and the status-bar
// idle hint to reflect the currently configured shortcut.
func (w *ApplicationWindow) updateShortcutHints(prefs AppPreferences) {
	label := accelToLabel(prefs.ScriptPickerShortcut)
	w.scriptsBtn.SetTooltipText("Scripts (" + label + ")")
	w.status.SetIdleHint("Press " + label + " for commands")
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
