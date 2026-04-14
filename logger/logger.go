// Package logger provides a structured debug logger for a0hero.
// When debug mode is enabled (via --debug flag), all operations write
// JSON lines to logs/ in the format specified by AGENTS.md:
//
//	{ts, level, tenant, module, action, target, status, error}
//
// When debug mode is off, logging is silent.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// L is the global logger. Nil when debug mode is off.
var L *slog.Logger
var mu sync.Mutex
var logDir string
var logFile *os.File

// Setup initializes the debug logger. If debug is false, all log calls are no-ops.
// Logs are written to logDir/<date>.log as JSON lines.
func Setup(debug bool, dir string) error {
	mu.Lock()
	defer mu.Unlock()

	if !debug {
		L = slog.New(slog.NewTextHandler(io.Discard, nil))
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

	logFile = f
	L = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	L.Info("logger initialized", "log_dir", logDir, "log_file", path)
	return nil
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		L.Info("logger shutting down")
		logFile.Close()
		logFile = nil
	}
}

// LogPath returns the current log file path, or empty string if debug is off.
func LogPath() string {
	mu.Lock()
	defer mu.Unlock()
	if logFile == nil {
		return ""
	}
	return logFile.Name()
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
func With(args ...any) *slog.Logger {
	if L == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return L.With(args...)
}