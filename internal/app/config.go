package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

const appName = "goop"

// UserConfiguration holds runtime paths and settings resolved from the XDG
// Base Directory specification.
type UserConfiguration struct {
	ScriptsDir    string        // ~/.config/goop/scripts/
	LogFilePath   string        // ~/.config/goop/goop.log
	ScriptTimeout time.Duration // Hard JS execution timeout
}

// NewUserConfiguration resolves XDG paths, creates the scripts directory if
// absent, and returns a ready-to-use configuration.
func NewUserConfiguration() (UserConfiguration, error) {
	scriptsDir := filepath.Join(xdg.ConfigHome, appName, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return UserConfiguration{}, fmt.Errorf("config: create scripts dir: %w", err)
	}

	logFilePath, err := xdg.ConfigFile(filepath.Join(appName, appName+".log"))
	if err != nil {
		return UserConfiguration{}, fmt.Errorf("config: resolve log path: %w", err)
	}

	return UserConfiguration{
		ScriptsDir:    scriptsDir,
		LogFilePath:   logFilePath,
		ScriptTimeout: 5 * time.Second,
	}, nil
}
