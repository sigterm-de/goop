package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
)

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	INFO  LogLevel = 0
	WARN  LogLevel = 1
	ERROR LogLevel = 2
)

func (l LogLevel) String() string {
	switch l {
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry holds a single structured log record.
type LogEntry struct {
	Timestamp  time.Time
	Level      LogLevel
	ScriptName string
	Message    string
}

var (
	mu      sync.Mutex
	logFile *os.File
	logPath string
)

// InitLogger opens (or creates) the log file under the XDG state home
// (~/.local/state/<appName>/<appName>.log) and returns the resolved absolute
// path so the UI can display it in error messages.
//
// XDG_STATE_HOME is the correct location for runtime-generated log files;
// XDG_CONFIG_HOME is reserved for user-editable configuration.
func InitLogger(appName string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	rel := filepath.Join(appName, appName+".log")
	p, err := xdg.StateFile(rel)
	if err != nil {
		return "", fmt.Errorf("logging: resolve state path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", fmt.Errorf("logging: create log dir: %w", err)
	}

	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("logging: open log file: %w", err)
	}

	logFile = f
	logPath = p
	return p, nil
}

// Log writes a structured line to the log file. Safe to call from any goroutine.
// If the logger has not been initialised, the entry is silently dropped.
func Log(level LogLevel, scriptName, message string) {
	mu.Lock()
	defer mu.Unlock()

	if logFile == nil {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s [%s] script=%q %s\n", ts, level, scriptName, message)
	_, _ = logFile.WriteString(line)
}

// Path returns the resolved log file path (empty string if not initialised).
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}
