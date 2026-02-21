package app

import (
	"fmt"
	"os"
	"strings"

	"codeberg.org/sigterm-de/goop/assets"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// fontCSSProvider is updated whenever the user changes the editor font.
var fontCSSProvider *gtk.CSSProvider

// gnomeInterfaceSettings holds a GSettings handle for
// "org.gnome.desktop.interface", or nil when that schema is not present
// (non-GNOME desktops).  Initialised once by loadCSS.
var gnomeInterfaceSettings *gio.Settings

// loadCSS installs the embedded stylesheet and initialises the font provider.
func loadCSS(prefs AppPreferences) {
	display := gdk.DisplayGetDefault()
	if display == nil {
		fmt.Fprintln(os.Stderr, "goop: warning: no display available, skipping CSS")
		return
	}

	// Application stylesheet.
	appProvider := gtk.NewCSSProvider()
	appProvider.LoadFromString(string(assets.StyleCSS()))
	gtk.StyleContextAddProviderForDisplay(
		display,
		appProvider,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
	)

	// Font override provider (higher priority so it wins over the stylesheet).
	fontCSSProvider = gtk.NewCSSProvider()
	gtk.StyleContextAddProviderForDisplay(
		display,
		fontCSSProvider,
		uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)+1,
	)

	applyEditorFont(prefs.EditorFont)
	initGnomeInterfaceSettings()
}

// initGnomeInterfaceSettings creates a GSettings handle for
// "org.gnome.desktop.interface" when the schema is available (GNOME desktops).
func initGnomeInterfaceSettings() {
	src := gio.SettingsSchemaSourceGetDefault()
	if src == nil {
		return
	}
	if src.Lookup("org.gnome.desktop.interface", true) == nil {
		return
	}
	gnomeInterfaceSettings = gio.NewSettings("org.gnome.desktop.interface")
}

// applyEditorFont updates the font CSS provider. Must be called on the GTK
// main thread.
func applyEditorFont(fontDesc string) {
	if fontCSSProvider == nil {
		return
	}
	fontCSSProvider.LoadFromString(pangoDescToCSS(fontDesc))
}

// applyPreferences applies all live preference values to the GTK state.
func applyPreferences(prefs AppPreferences) {
	applyEditorFont(prefs.EditorFont)
}

// isSystemDark reports whether the system currently prefers dark mode.
//
// Three sources are checked in priority order:
//  1. org.gnome.desktop.interface color-scheme (GNOME, most direct)
//  2. gtk-application-prefer-dark-theme        (GTK/portal fallback)
//  3. gtk-theme-name contains "dark"           (KDE and others)
func isSystemDark() bool {
	if gnomeInterfaceSettings != nil {
		return gnomeInterfaceSettings.String("color-scheme") == "prefer-dark"
	}
	settings := gtk.SettingsGetDefault()
	if settings == nil {
		return false
	}
	if v, ok := settings.ObjectProperty("gtk-application-prefer-dark-theme").(bool); ok && v {
		return true
	}
	if name, ok := settings.ObjectProperty("gtk-theme-name").(string); ok {
		return strings.Contains(strings.ToLower(name), "dark")
	}
	return false
}

// resolveActiveScheme returns the editor scheme ID that should be active right
// now, honouring both the follow-system preference and the current GTK dark
// mode state.
func resolveActiveScheme(prefs AppPreferences) string {
	if prefs.EditorSchemeFollowSystem && isSystemDark() {
		return prefs.EditorSchemeDark
	}
	return prefs.EditorSchemeLight
}
