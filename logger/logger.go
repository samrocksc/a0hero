// Package logger provides a structured debug logger for a0hero.
// When debug mode is enabled (via --debug flag), all operations write
// JSON lines to logs/ in a readable format with key-value pairs.
//
// When debug mode is off, logging is silent.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// L is the global logger. Nil when debug mode is off.
var L *log.Logger
var mu sync.Mutex
var logDir string

// Setup initializes the debug logger. If debug is false, all log calls are no-ops.
// Logs are written to logDir/<date>.log as formatted lines.
func Setup(debug bool, dir string) error {
	mu.Lock()
	defer mu.Unlock()

	if !debug {
		L = log.New(io.Discard)
		return nil
	}

	logDir = dir
	if logDir == "" {
		logDir = "logs"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log directory %s: %w", logDir, err)
	}

	// Rotate by date: logs/2026-04-13.log
	filename := time.Now().Format("2006-01-02") + ".log"
	path := filepath.Join(logDir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", path, err)
	}

	// Create charmbracelet/log logger with pretty formatting
	L = log.New(f)
	L.SetReportTimestamp(true)
	L.SetReportCaller(false)
	L.SetTimeFormat("2006-01-02 15:04:05")
	
	// Use pretty printer for development (easier to read than JSON)
	// For JSON output, use: L.SetFormatter(log.JSONFormatter)
	L.SetLevel(log.DebugLevel)

	L.Info("logger initialized", "log_dir", logDir, "log_file", path)
	return nil
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if L != nil {
		L.Info("logger shutting down")
	}
}

// LogPath returns the current log file path, or empty string if debug is off.
func LogPath() string {
	mu.Lock()
	defer mu.Unlock()
	return logDir
}

// Convenience functions that are no-ops when debug mode is off.

func Debug(msg string, args ...any) {
	if L != nil {
		L.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if L != nil {
		L.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if L != nil {
		L.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if L != nil {
		L.Error(msg, args...)
	}
}

// With returns a logger pre-loaded with key-value pairs.
// Useful for adding tenant/module context.
func With(args ...any) *log.Logger {
	if L == nil {
		return log.New(os.Stderr)
	}
	return L.With(args...)
}
