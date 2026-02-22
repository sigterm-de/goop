package app

import (
	"fmt"
	"os"
	"path/filepath"

	"codeberg.org/sigterm-de/goop/assets"
	"codeberg.org/sigterm-de/goop/internal/engine"
	"codeberg.org/sigterm-de/goop/internal/logging"
	"codeberg.org/sigterm-de/goop/internal/scripts"
	"github.com/adrg/xdg"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Run initialises and runs the GTK application. It returns the exit code that
// main() should pass to os.Exit.
func Run(appVersion string) int {
	app := gtk.NewApplication("org.codeberg.sigterm-de.goop", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app, appVersion) })
	return int(app.Run(os.Args))
}

func activate(app *gtk.Application, appVersion string) {
	// ── Configuration ─────────────────────────────────────────────────────────
	cfg, err := NewUserConfiguration()
	if err != nil {
		showFatalError(app, fmt.Sprintf("Failed to initialise configuration: %v", err))
		return
	}

	// ── Logging ───────────────────────────────────────────────────────────────
	logPath, err := logging.InitLogger(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goop: warning: cannot initialise logger: %v\n", err)
		logPath = ""
	}
	logging.Log(logging.INFO, "", fmt.Sprintf("goop %s starting", appVersion))

	// ── Script loading ────────────────────────────────────────────────────────
	loader := scripts.NewLoader(assets.Scripts())
	result, err := loader.Load(cfg.ScriptsDir)
	if err != nil {
		showFatalError(app, fmt.Sprintf("Failed to load scripts: %v", err))
		return
	}
	for _, skipped := range result.SkippedFiles {
		logging.Log(logging.WARN, skipped, "script was skipped during load")
	}
	logging.Log(logging.INFO, "",
		fmt.Sprintf("loaded %d built-in scripts, %d user scripts (%d skipped)",
			result.BuiltInCount, result.UserCount, len(result.SkippedFiles)))

	lib := scripts.NewLibrary(result)
	exec := engine.NewExecutor()

	// ── Preferences ──────────────────────────────────────────────────────────
	prefs := LoadPreferences()

	// ── CSS + theme ───────────────────────────────────────────────────────────
	loadCSS(prefs)

	// ── App icon ──────────────────────────────────────────────────────────────
	if iconDir := setupAppIcon(); iconDir != "" {
		if display := gdk.DisplayGetDefault(); display != nil {
			gtk.IconThemeGetForDisplay(display).AddSearchPath(iconDir)
		}
	}

	// ── Window ────────────────────────────────────────────────────────────────
	win := NewApplicationWindow(app, lib, exec, logPath, prefs, appVersion)

	app.SetAccelsForAction("win.toggle-picker", []string{prefs.ScriptPickerShortcut})

	win.Win.Present()
}

// setupAppIcon writes the embedded icon to the XDG cache and returns the icon
// theme search-path root (e.g. ~/.cache/goop/icons). Returns "" on failure.
func setupAppIcon() string {
	iconPath, err := xdg.CacheFile("goop/icons/hicolor/256x256/apps/goop.png")
	if err != nil {
		return ""
	}
	if err := os.WriteFile(iconPath, assets.Icon(), 0o644); err != nil {
		return ""
	}
	return filepath.Join(xdg.CacheHome, "goop", "icons")
}

// showFatalError creates a minimal error dialog and quits the application.
// The window has a Close button and responds to Escape so users can dismiss it.
func showFatalError(app *gtk.Application, msg string) {
	fmt.Fprintln(os.Stderr, "goop: fatal:", msg)

	win := gtk.NewApplicationWindow(app)
	win.SetDefaultSize(420, 0)
	win.SetTitle("goop — fatal error")
	win.SetResizable(false)

	label := gtk.NewLabel(msg)
	label.SetWrap(true)
	label.SetMarginTop(16)
	label.SetMarginBottom(8)
	label.SetMarginStart(16)
	label.SetMarginEnd(16)

	closeBtn := gtk.NewButtonWithLabel("Close")
	closeBtn.SetHAlign(gtk.AlignCenter)
	closeBtn.SetMarginBottom(16)
	closeBtn.ConnectClicked(func() { app.Quit() })

	// Close on Escape.
	keyCtrl := gtk.NewEventControllerKey()
	keyCtrl.SetPropagationPhase(gtk.PhaseCapture)
	keyCtrl.ConnectKeyPressed(func(keyval, _ uint, _ gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			app.Quit()
			return true
		}
		return false
	})
	win.AddController(keyCtrl)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(label)
	box.Append(closeBtn)
	win.SetChild(box)
	win.Present()
}
