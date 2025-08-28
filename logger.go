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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	pathTraversalDots       = ".."
	loggerErrorFormatString = "[LOGGER ERROR] Format panic: %v, " +
		"format=%q, args=%v\n"
)

// Predefined errors for better error handling.
var (
	ErrLogPathOutsideBounds = errors.New(
		"log path outside directory bounds",
	)
	ErrPathCannotBeEmpty        = errors.New("path cannot be empty")
	ErrPathContainsInvalidChars = errors.New("path contains invalid characters")
	ErrFilenameCannotBeEmpty    = errors.New("filename cannot be empty")
	ErrFilenameContainsInvalid  = errors.New(
		"filename contains invalid characters",
	)
)

// Logger provides leveled, thread-safe logging to stdout and a rotating file
// per run.
// Keep this simple and dependency-free.
type Logger struct {
	mu      sync.Mutex
	logFile *os.File
	std     *log.Logger
	file    *log.Logger
}

// New creates a new Logger instance that writes to both stdout and a log file.
func New(logDir, filename string) (*Logger, error) {
	err := validateInputs(logDir, filename)
	if err != nil {
		return nil, err
	}

	logPath, err := setupLogDirectory(logDir, filename)
	if err != nil {
		return nil, err
	}

	err = validateLogPath(logDir, logPath)
	if err != nil {
		return nil, err
	}

	f, err := openLogFile(logPath)
	if err != nil {
		return nil, err
	}

	return createLoggerInstance(f), nil
}

func validateInputs(logDir, filename string) error {
	err := ValidatePath(logDir)
	if err != nil {
		return fmt.Errorf("invalid log directory: %w", err)
	}

	err = ValidateFilename(filename)
	if err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}

	return nil
}

func setupLogDirectory(logDir, filename string) (string, error) {
	const logDirPerm = 0o750

	err := os.MkdirAll(logDir, logDirPerm)
	if err != nil {
		return "", fmt.Errorf("create log dir: %w", err)
	}

	return filepath.Join(logDir, filename), nil
}

func validateLogPath(logDir, logPath string) error {
	absLogDir, err := filepath.Abs(logDir)
	if err != nil {
		return fmt.Errorf("resolve log directory: %w", err)
	}

	absLogPath, err := filepath.Abs(logPath)
	if err != nil {
		return fmt.Errorf("resolve log path: %w", err)
	}

	if !strings.HasPrefix(
		absLogPath+string(filepath.Separator),
		absLogDir+string(filepath.Separator),
	) {
		return ErrLogPathOutsideBounds
	}

	return nil
}

func openLogFile(logPath string) (*os.File, error) {
	const logFilePerm = 0o600
	// #nosec G304 - Path is validated above to prevent directory traversal
	logFile, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		logFilePerm,
	)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return logFile, nil
}

func createLoggerInstance(f *os.File) *Logger {
	return &Logger{
		mu:      sync.Mutex{},
		logFile: f,
		std:     log.New(os.Stdout, "", log.LstdFlags),
		file:    log.New(f, "", log.LstdFlags),
	}
}

// ValidatePath ensures the path is safe and doesn't contain directory
// traversal.
func ValidatePath(path string) error {
	if path == "" {
		return ErrPathCannotBeEmpty
	}

	if containsInvalidPathChars(path) {
		return ErrPathContainsInvalidChars
	}

	return nil
}

// ValidateFilename ensures the filename is safe.
func ValidateFilename(filename string) error {
	if filename == "" {
		return ErrFilenameCannotBeEmpty
	}

	if containsInvalidFilenameChars(filename) {
		return ErrFilenameContainsInvalid
	}

	return nil
}

func containsInvalidPathChars(path string) bool {
	return strings.Contains(path, pathTraversalDots) ||
		strings.Contains(path, "~")
}

func containsInvalidFilenameChars(filename string) bool {
	invalidChars := []string{"/", "\\", pathTraversalDots, "~"}
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return true
		}
	}

	return false
}

// Close closes the log file and releases resources.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		err := l.logFile.Close()
		l.logFile = nil

		return fmt.Errorf("close log file: %w", err)
	}

	return nil
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...any) {
	l.write("INFO", format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...any) {
	l.write("WARN", format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...any) {
	l.write("ERROR", format, args...)
}

// Success logs a success message.
func (l *Logger) Success(format string, args ...any) {
	l.write("SUCCESS", format, args...)
}

// Fatal logs a fatal system error and does NOT exit (unlike log.Fatal).
func (l *Logger) Fatal(format string, args ...any) {
	l.write("FATAL", format, args...)
}

// Panic logs a panic-level error and does NOT panic (unlike log.Panic).
func (l *Logger) Panic(format string, args ...any) {
	l.write("PANIC", format, args...)
}

// System logs system-level events (startup, shutdown, configuration changes).
func (l *Logger) System(format string, args ...any) {
	l.write("SYSTEM", format, args...)
}

const maxLogMessageLength = 4096 // Reasonable limit for log messages

func (l *Logger) write(level, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	format = l.validateFormat(format)

	if l.logFile == nil {
		l.writeToStderrFallback(level, format, args...)

		return
	}

	msg := l.prepareMessage(level, format, args...)
	if msg != "" {
		l.outputMessage(msg)
	}
}

func (l *Logger) validateFormat(format string) string {
	if format == "" {
		return "(empty message)"
	}

	return format
}

func (l *Logger) prepareMessage(level, format string, args ...any) string {
	formattedMsg := l.safeFormat(format, args...)
	if len(formattedMsg) > maxLogMessageLength {
		formattedMsg = formattedMsg[:maxLogMessageLength-20] + "... [TRUNCATED]"
	}

	return l.formatLogMessage(level, formattedMsg)
}

func (l *Logger) outputMessage(msg string) {
	l.std.Println(msg)
	l.file.Println(msg)
}

func (l *Logger) writeToStderrFallback(level, format string, args ...any) {
	// Logger is closed, only write to stderr as fallback
	_, err := fmt.Fprintf(
		os.Stderr,
		"[%s] (logger closed) %s\n",
		level,
		l.safeFormat(format, args...),
	)
	_ = err // Error ignored - cannot log safely
}

func (l *Logger) formatLogMessage(level, formattedMsg string) string {
	var stringBuilder strings.Builder

	const extraCapacity = 32

	// Pre-allocate capacity
	stringBuilder.Grow(len(level) + len(formattedMsg) + extraCapacity)

	var err error

	_, err = stringBuilder.WriteString("[")
	if err != nil {
		return "" // Cannot recover from string builder error
	}

	_, err = stringBuilder.WriteString(level)
	if err != nil {
		return ""
	}

	_, err = stringBuilder.WriteString("] ")
	if err != nil {
		return ""
	}

	_, err = stringBuilder.WriteString(formattedMsg)
	if err != nil {
		return ""
	}

	return stringBuilder.String()
}

// safeFormat safely formats the message, handling format string errors.
func (l *Logger) safeFormat(format string, args ...any) string {
	defer func() {
		if r := recover(); r != nil {
			// Format panic recovered - return a safe message
			_, err := fmt.Fprintf(
				os.Stderr,
				loggerErrorFormatString,
				r,
				format,
				args,
			)
			_ = err // Error ignored - cannot log safely
		}
	}()

	// If no args, return format string as-is (handles case where format has %
	// but no args)
	if len(args) == 0 {
		return format
	}

	// Try to format, catch any errors
	result := fmt.Sprintf(format, args...)

	if r := recover(); r != nil {
		// Format panic recovered - return a safe message
		_, err := fmt.Fprintf(
			os.Stderr,
			loggerErrorFormatString,
			r,
			format,
			args,
		)
		_ = err // Error ignored - cannot log safely

		return fmt.Sprintf("(format error: %s) args=%v", format, args)
	}

	return result
}
