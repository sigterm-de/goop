package app

import (
	"fmt"
	"os"

	"codeberg.org/daniel-ciaglia/goop/assets"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// fontCSSProvider is updated whenever the user changes the editor font.
var fontCSSProvider *gtk.CSSProvider

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

// isSystemDark reports whether GTK currently prefers dark mode.
func isSystemDark() bool {
	settings := gtk.SettingsGetDefault()
	if settings == nil {
		return false
	}
	v, ok := settings.ObjectProperty("gtk-application-prefer-dark-theme").(bool)
	return ok && v
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
