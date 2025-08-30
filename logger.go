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
	maxLogMessageLength = 4096 // Reasonable limit for log messages
	// logMessageExtraCap is the extra capacity for the log message builder ([level]
	// msg).
	logMessageExtraCap = 3
	logLevelInfo       = "INFO"
	logLevelWarn       = "WARN"
	logLevelError      = "ERROR"
	logLevelSuccess    = "SUCCESS"
	logLevelFatal      = "FATAL"
	logLevelPanic      = "PANIC"
	logLevelSystem     = "SYSTEM"
	emptyMessage       = "(empty message)"
	truncatedSuffix    = "... [TRUNCATED]"
	fallbackFormat     = "[%s] (logger closed) %s\n"
	formatErrorMsg     = "(format error: %s) args=%v"
	logBracketSpace    = "] "

	// Error messages for predefined errors.
	errLogPathOutsideBoundsMsg     = "log path outside directory bounds"
	errPathCannotBeEmptyMsg        = "path cannot be empty"
	errPathContainsInvalidCharsMsg = "path contains invalid characters"
	errFilenameCannotBeEmptyMsg    = "filename cannot be empty"
	errFilenameContainsInvalidMsg  = "filename contains invalid characters"

	// Error format strings.
	errFmtInvalidLogDir   = "invalid log directory: %w"
	errFmtInvalidFilename = "invalid filename: %w"
	errFmtCreateLogDir    = "create log dir: %w"
	errFmtResolveLogDir   = "resolve log directory: %w"
	errFmtResolveLogPath  = "resolve log path: %w"
	errFmtOpenLogFile     = "open log file: %w"
	errFmtCloseLogFile    = "close log file: %w"
)

// Predefined errors for better error handling.
var (
	ErrLogPathOutsideBounds     = errors.New(errLogPathOutsideBoundsMsg)
	ErrPathCannotBeEmpty        = errors.New(errPathCannotBeEmptyMsg)
	ErrPathContainsInvalidChars = errors.New(errPathContainsInvalidCharsMsg)
	ErrFilenameCannotBeEmpty    = errors.New(errFilenameCannotBeEmptyMsg)
	ErrFilenameContainsInvalid  = errors.New(errFilenameContainsInvalidMsg)
)

// Logger provides leveled, thread-safe logging to stdout and a rotating file per run.
type Logger struct {
	logFile *os.File
	std     *log.Logger
	file    *log.Logger
	mu      sync.Mutex
}

// New creates a new Logger instance that writes to both stdout and a log file.
func New(logDir, filename string) (*Logger, error) {
	err := validateInputs(logDir, filename)
	if err != nil {
		return nil, err
	}

	logPath, err := setupAndValidatePath(logDir, filename)
	if err != nil {
		return nil, err
	}

	f, err := openLogFile(logPath)
	if err != nil {
		return nil, err
	}

	return createLoggerInstance(f), nil
}

func setupAndValidatePath(logDir, filename string) (string, error) {
	logPath, err := setupLogDirectory(logDir, filename)
	if err != nil {
		return "", err
	}

	err = validateLogPath(logDir, logPath)
	if err != nil {
		return "", err
	}

	return logPath, nil
}

func validateInputs(logDir, filename string) error {
	err := ValidatePath(logDir)
	if err != nil {
		return fmt.Errorf(errFmtInvalidLogDir, err)
	}

	err = ValidateFilename(filename)
	if err != nil {
		return fmt.Errorf(errFmtInvalidFilename, err)
	}

	return nil
}

func setupLogDirectory(logDir, filename string) (string, error) {
	const logDirPerm = 0o750

	err := os.MkdirAll(logDir, logDirPerm)
	if err != nil {
		return "", fmt.Errorf(errFmtCreateLogDir, err)
	}

	return filepath.Join(logDir, filename), nil
}

func validateLogPath(logDir, logPath string) error {
	absLogDir, err := filepath.Abs(logDir)
	if err != nil {
		return fmt.Errorf(errFmtResolveLogDir, err)
	}

	absLogPath, err := filepath.Abs(logPath)
	if err != nil {
		return fmt.Errorf(errFmtResolveLogPath, err)
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
	// #nosec G304
	logFile, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		logFilePerm,
	)
	if err != nil {
		return nil, fmt.Errorf(errFmtOpenLogFile, err)
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

// ValidatePath ensures the path is safe and doesn't contain directory traversal.
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
	return strings.Contains(path, pathTraversalDots) || strings.Contains(path, "~")
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
		if err != nil {
			return fmt.Errorf(errFmtCloseLogFile, err)
		}
	}

	return nil
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...any) {
	l.write(logLevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...any) {
	l.write(logLevelWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...any) {
	l.write(logLevelError, format, args...)
}

// Success logs a success message.
func (l *Logger) Success(format string, args ...any) {
	l.write(logLevelSuccess, format, args...)
}

// Fatal logs a fatal system error and does NOT exit (unlike log.Fatal).
func (l *Logger) Fatal(format string, args ...any) {
	l.write(logLevelFatal, format, args...)
}

// Panic logs a panic-level error and does NOT panic (unlike log.Panic).
func (l *Logger) Panic(format string, args ...any) {
	l.write(logLevelPanic, format, args...)
}

// System logs system-level events (startup, shutdown, configuration changes).
func (l *Logger) System(format string, args ...any) {
	l.write(logLevelSystem, format, args...)
}

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
		return emptyMessage
	}

	return format
}

func (l *Logger) prepareMessage(level, format string, args ...any) string {
	formattedMsg := l.safeFormat(format, args...)
	if len(formattedMsg) > maxLogMessageLength {
		truncatedLen := maxLogMessageLength - len(truncatedSuffix)

		formattedMsg = formattedMsg[:truncatedLen] + truncatedSuffix
	}

	return l.formatLogMessage(level, formattedMsg)
}

func (l *Logger) outputMessage(msg string) {
	l.std.Println(msg)
	l.file.Println(msg)
}

func (l *Logger) writeToStderrFallback(level, format string, args ...any) {
	// Logger is closed, only write to stderr as fallback.
	_, err := fmt.Fprintf(
		os.Stderr,
		fallbackFormat,
		level,
		l.safeFormat(format, args...),
	)

	_ = err // Error ignored - cannot log safely.
}

func (l *Logger) formatLogMessage(level, formattedMsg string) string {
	var builder strings.Builder
	builder.Grow(len(level) + len(formattedMsg) + logMessageExtraCap)
	builder.WriteString("[")
	builder.WriteString(level)
	builder.WriteString(logBracketSpace)
	builder.WriteString(formattedMsg)

	return builder.String()
}

// safeFormat safely formats the message, handling format string errors.
func (l *Logger) safeFormat(format string, args ...any) (result string) {
	defer func() {
		if r := recover(); r != nil {
			// Format panic recovered - log a safe message to stderr.
			fmt.Fprintf(os.Stderr, loggerErrorFormatString, r, format, args)
			// Return a safe message to be logged to the file.
			result = fmt.Sprintf(formatErrorMsg, format, args)
		}
	}()
	// If no args, return format string as-is to handle cases like "100%".
	if len(args) == 0 {
		return format
	}

	return fmt.Sprintf(format, args...)
}
