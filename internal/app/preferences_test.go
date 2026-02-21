package app

import (
	"encoding/json"
	"testing"
)

func TestDefaultPreferencesSyntaxAutoDetect(t *testing.T) {
	prefs := defaultPreferences()
	if !prefs.SyntaxAutoDetect {
		t.Error("defaultPreferences().SyntaxAutoDetect = false; want true")
	}
}

func TestPreferencesUnmarshalSyntaxAutoDetect(t *testing.T) {
	cases := []struct {
		name string
		json string
		want bool
	}{
		{"explicit false", `{"syntax_auto_detect": false}`, false},
		{"explicit true", `{"syntax_auto_detect": true}`, true},
		// Omitted key should fall back to Go zero value (false), but in
		// practice LoadPreferences seeds from defaultPreferences() first.
		{"omitted key zero value", `{}`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var prefs AppPreferences
			if err := json.Unmarshal([]byte(tc.json), &prefs); err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}
			if prefs.SyntaxAutoDetect != tc.want {
				t.Errorf("SyntaxAutoDetect = %v; want %v", prefs.SyntaxAutoDetect, tc.want)
			}
		})
	}
}
