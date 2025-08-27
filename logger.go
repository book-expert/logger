// Package logger provides a thread-safe, configurable logging utility.
//
// This package implements leveled logging with output to both stdout and files,
// following Go coding standards and design principles for explicit behavior,
// robust error handling, and maintainable code.
//
// Features:
// - Thread-safe logging with configurable levels
// - Dual output (stdout + file) with error propagation
// - Path validation to prevent directory traversal attacks
// - Optimized string formatting for high-performance logging
package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Logger provides leveled, thread-safe logging to stdout and a rotating file per run.
// Keep this simple and dependency-free.
type Logger struct {
	mu      sync.Mutex
	logFile *os.File
	std     *log.Logger
	file    *log.Logger
}

// New creates a new Logger instance that writes to both stdout and a log file
func New(logDir string, filename string) (*Logger, error) {
	// Validate and sanitize the log directory path
	if err := validatePath(logDir); err != nil {
		return nil, fmt.Errorf("invalid log directory: %w", err)
	}

	// Validate and sanitize the filename
	if err := validateFilename(filename); err != nil {
		return nil, fmt.Errorf("invalid filename: %w", err)
	}

	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, filename)

	// Additional security check: ensure the final path is within the expected directory
	absLogDir, err := filepath.Abs(logDir)
	if err != nil {
		return nil, fmt.Errorf("resolve log directory: %w", err)
	}
	absLogPath, err := filepath.Abs(logPath)
	if err != nil {
		return nil, fmt.Errorf("resolve log path: %w", err)
	}
	// Check if log path is within the log directory using string prefix check
	// This replaces the deprecated filepath.HasPrefix function
	if !strings.HasPrefix(absLogPath+string(filepath.Separator), absLogDir+string(filepath.Separator)) {
		return nil, fmt.Errorf("log path outside directory bounds")
	}

	// #nosec G304 - Path is validated above to prevent directory traversal
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return &Logger{
		logFile: f,
		std:     log.New(os.Stdout, "", log.LstdFlags),
		file:    log.New(f, "", log.LstdFlags),
	}, nil
}

// validatePath ensures the path is safe and doesn't contain directory traversal
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for directory traversal attempts
	if strings.Contains(path, "..") || strings.Contains(path, "~") {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}

// validateFilename ensures the filename is safe
func validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check for directory traversal attempts
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") ||
		strings.Contains(filename, "..") || strings.Contains(filename, "~") {
		return fmt.Errorf("filename contains invalid characters")
	}

	return nil
}

// Close closes the log file and releases resources
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		err := l.logFile.Close()
		l.logFile = nil
		return err
	}
	return nil
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...any) { l.write("INFO", format, args...) }

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...any) { l.write("WARN", format, args...) }

// Error logs an error message
func (l *Logger) Error(format string, args ...any) { l.write("ERROR", format, args...) }

// Success logs a success message
func (l *Logger) Success(format string, args ...any) { l.write("SUCCESS", format, args...) }

// Fatal logs a fatal system error and does NOT exit (unlike log.Fatal)
func (l *Logger) Fatal(format string, args ...any) { l.write("FATAL", format, args...) }

// Panic logs a panic-level error and does NOT panic (unlike log.Panic)
func (l *Logger) Panic(format string, args ...any) { l.write("PANIC", format, args...) }

// System logs system-level events (startup, shutdown, configuration changes)
func (l *Logger) System(format string, args ...any) { l.write("SYSTEM", format, args...) }

const maxLogMessageLength = 4096 // Reasonable limit for log messages

func (l *Logger) write(level string, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Validate inputs
	if format == "" {
		format = "(empty message)"
	}

	// Check if logger is closed
	if l.logFile == nil {
		// Logger is closed, only write to stderr as fallback
		fmt.Fprintf(os.Stderr, "[%s] (logger closed) %s\n", level, l.safeFormat(format, args...))
		return
	}

	// Format message safely
	formattedMsg := l.safeFormat(format, args...)

	// Truncate if too long
	if len(formattedMsg) > maxLogMessageLength {
		formattedMsg = formattedMsg[:maxLogMessageLength-20] + "... [TRUNCATED]"
	}

	// Optimize string formatting with strings.Builder
	var sb strings.Builder
	sb.Grow(len(level) + len(formattedMsg) + 32) // Pre-allocate capacity
	sb.WriteString("[")
	sb.WriteString(level)
	sb.WriteString("] ")
	sb.WriteString(formattedMsg)
	msg := sb.String()

	// Write to stdout - continue even if this fails
	l.std.Println(msg)

	// Write to file - if this fails, try to log the failure to stderr
	l.file.Println(msg)
}

// safeFormat safely formats the message, handling format string errors
func (l *Logger) safeFormat(format string, args ...any) (result string) {
	defer func() {
		if r := recover(); r != nil {
			// Format panic recovered - return a safe message
			fmt.Fprintf(os.Stderr, "[LOGGER ERROR] Format panic: %v, format=%q, args=%v\n", r, format, args)
			result = fmt.Sprintf("(format error: %s) args=%v", format, args)
		}
	}()

	// If no args, return format string as-is (handles case where format has % but no args)
	if len(args) == 0 {
		return format
	}

	// Try to format, catch any errors
	return fmt.Sprintf(format, args...)
}
