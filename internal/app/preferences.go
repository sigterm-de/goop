package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// AppPreferences holds persistent user-configurable settings.
type AppPreferences struct {
	EditorFont        string `json:"editor_font"`         // Pango font description, e.g. "Monospace 12"
	FontMonospaceOnly bool   `json:"font_monospace_only"` // restrict font picker to monospace families

	// Editor colour scheme.  When EditorSchemeFollowSystem is true the
	// application automatically switches between the light and dark scheme
	// according to the GTK dark-theme preference.
	EditorSchemeFollowSystem bool   `json:"editor_scheme_follow_system"`
	EditorSchemeLight        string `json:"editor_scheme_light"`
	EditorSchemeDark         string `json:"editor_scheme_dark"`

	// ScriptPickerShortcut is the GTK accelerator string used to toggle the
	// script picker panel (e.g. "<Primary>slash").
	ScriptPickerShortcut string `json:"script_picker_shortcut"`

	// SyntaxAutoDetect controls whether the editor automatically detects and
	// applies syntax highlighting after each successful script execution.
	SyntaxAutoDetect bool `json:"syntax_auto_detect"`
}

func defaultPreferences() AppPreferences {
	return AppPreferences{
		EditorFont:               "Monospace 12",
		FontMonospaceOnly:        false,
		EditorSchemeFollowSystem: true,
		EditorSchemeLight:        "classic",
		EditorSchemeDark:         "oblivion",
		ScriptPickerShortcut:     "<Primary>slash",
		SyntaxAutoDetect:         true,
	}
}

func preferencesFilePath() (string, error) {
	return xdg.ConfigFile(filepath.Join(appName, "preferences.json"))
}

// LoadPreferences loads preferences from disk, returning defaults on any error.
func LoadPreferences() AppPreferences {
	path, err := preferencesFilePath()
	if err != nil {
		return defaultPreferences()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultPreferences()
	}
	prefs := defaultPreferences()
	if err := json.Unmarshal(data, &prefs); err != nil {
		return defaultPreferences()
	}
	sanitizePreferences(&prefs)
	return prefs
}

// sanitizePreferences replaces any field that would cause silent misbehaviour
// with its default value. Called after JSON unmarshal so that a hand-edited
// preferences file cannot leave the application in a broken state.
func sanitizePreferences(p *AppPreferences) {
	def := defaultPreferences()
	// An empty or unparseable shortcut makes the picker unreachable by keyboard.
	if p.ScriptPickerShortcut == "" {
		p.ScriptPickerShortcut = def.ScriptPickerShortcut
	}
	// Validate that GTK can actually parse the stored accelerator string.
	if _, _, ok := gtk.AcceleratorParse(p.ScriptPickerShortcut); !ok {
		p.ScriptPickerShortcut = def.ScriptPickerShortcut
	}
}

// SavePreferences writes preferences to disk.
func SavePreferences(prefs AppPreferences) error {
	path, err := preferencesFilePath()
	if err != nil {
		return fmt.Errorf("preferences: resolve path: %w", err)
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("preferences: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("preferences: write: %w", err)
	}
	return nil
}
