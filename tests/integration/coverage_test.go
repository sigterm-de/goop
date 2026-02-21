// Package integration — additional tests targeting coverage gaps in
// scripts and logging packages.
package integration_test

import (
	"os"
	"strings"
	"testing"

	"codeberg.org/sigterm-de/goop/assets"
	"codeberg.org/sigterm-de/goop/internal/logging"
	"codeberg.org/sigterm-de/goop/internal/scripts"
)

// TestLibraryLen verifies that ScriptLibrary.Len() returns the count.
func TestLibraryLen(t *testing.T) {
	result, err := scripts.NewLoader(assets.Scripts()).Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	lib := scripts.NewLibrary(result)
	if lib.Len() != len(result.Scripts) {
		t.Errorf("Len() = %d, want %d", lib.Len(), len(result.Scripts))
	}
}

// TestLoggingInitAndLog verifies InitLogger, Log, and Path.
func TestLoggingInitAndLog(t *testing.T) {
	dir := t.TempDir()
	logFile := dir + "/test-app/log.txt"
	// We cannot call InitLogger directly since it uses xdg, but we can verify
	// the package-level Log function handles uninitialised logger gracefully.
	logging.Log(logging.INFO, "test", "message without init")
	logging.Log(logging.WARN, "test", "warn message")
	logging.Log(logging.ERROR, "test", "error message")

	// Test InitLogger explicitly using a temp path override via environment.
	t.Setenv("XDG_CONFIG_HOME", dir)
	_, err := logging.InitLogger("test-app")
	if err != nil {
		t.Fatalf("InitLogger: %v", err)
	}
	p := logging.Path()
	if p == "" {
		t.Error("Path() returned empty string after InitLogger")
	}
	logging.Log(logging.INFO, "test-script", "hello from test")
	if _, statErr := os.Stat(p); statErr != nil {
		t.Errorf("log file not created at %q: %v", p, statErr)
	}
	_ = logFile
}

// TestLoggingLevelString verifies LogLevel.String() returns readable names.
func TestLoggingLevelString(t *testing.T) {
	cases := []struct {
		level logging.LogLevel
		want  string
	}{
		{logging.INFO, "INFO"},
		{logging.WARN, "WARN"},
		{logging.ERROR, "ERROR"},
	}
	for _, tc := range cases {
		got := tc.level.String()
		if !strings.EqualFold(got, tc.want) && got != tc.want {
			t.Errorf("LogLevel(%d).String() = %q, want %q", tc.level, got, tc.want)
		}
	}
}

// TestLoaderSkipsLib verifies that lib/ JS files are not loaded as scripts.
func TestLoaderSkipsLib(t *testing.T) {
	result, err := scripts.NewLoader(assets.Scripts()).Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	for _, s := range result.Scripts {
		if strings.HasPrefix(s.FilePath, "embedded:lib/") {
			t.Errorf("lib file should not be loaded as script: %s", s.FilePath)
		}
	}
}
