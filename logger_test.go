package logger_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"logger"
)

const (
	testLogFile        = "test.log"
	newLoggerError     = "New logger: %v"
	emptyLogFile       = "empty.log"
	readLogFileErr     = "read log file: %v"
	formatLogFile      = "format.log"
	longLogFile        = "long.log"
	errorClosingLogger = "Error closing logger: %v"
	testLogPattern     = "../test.log"
)

func TestLogger_WritesToStdoutAndFile(t *testing.T) {
	t.Parallel()
	logDir := t.TempDir()

	loggerInstance, err := logger.New(logDir, testLogFile)
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	defer func() {
		err := loggerInstance.Close()
		if err != nil {
			t.Logf(errorClosingLogger, err)
		}
	}()

	testLoggerAllLevels(t, loggerInstance)
	verifyLogFileContents(t, logDir, testLogFile)
}

func testLoggerAllLevels(t *testing.T, loggerInstance *logger.Logger) {
	t.Helper()
	loggerInstance.Info("hello %s", "world")
	loggerInstance.Warn("warn %d", 42)
	loggerInstance.Error("err %v", 1)
	loggerInstance.Success("ok")
	loggerInstance.Fatal("system failure: %s", "disk full")
	loggerInstance.Panic("panic condition: %v", "nil pointer")
	loggerInstance.System("system event: %s", "startup complete")
}

func verifyLogFileContents(t *testing.T, logDir, filename string) {
	t.Helper()

	path := filepath.Join(logDir, filename)

	// #nosec G304 - Path is from t.TempDir() which is safe
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}

	defer func() {
		err := file.Close()
		if err != nil {
			t.Logf("Error closing file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	joined := strings.Join(lines, "\n")

	expectedMessages := []string{
		"[INFO] hello world",
		"[WARN] warn 42",
		"[ERROR] err 1",
		"[SUCCESS] ok",
		"[FATAL] system failure: disk full",
		"[PANIC] panic condition: nil pointer",
		"[SYSTEM] system event: startup complete",
	}
	for _, want := range expectedMessages {
		if !strings.Contains(joined, want) {
			t.Errorf("log file missing %q; got:\n%s", want, joined)
		}
	}
}

func TestLogger_CloseIdempotent(t *testing.T) {
	t.Parallel()
	logDir := t.TempDir()

	loggerInstance, err := logger.New(logDir, "test2.log")
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	err = loggerInstance.Close()
	if err != nil {
		t.Fatalf("first close: %v", err)
	}
	// Second close should be safe
	err = loggerInstance.Close()
	if err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestLogger_ValidatePath(t *testing.T) {
	t.Parallel()
	runValidatePathTest(t, "/tmp/logs", "valid path", false)
	runValidatePathTest(t, "", "empty path", true)
	runValidatePathTest(t, "/tmp/../etc", "path traversal dots", true)
	runValidatePathTest(t, "~/logs", "path traversal tilde", true)
	runValidatePathTest(t, "logs", "relative path", false)
}

func runValidatePathTest(t *testing.T, path, name string, wantErr bool) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()

		err := logger.ValidatePath(path)
		if (err != nil) != wantErr {
			t.Errorf("validatePath() error = %v, wantErr %v", err, wantErr)
		}
	})
}

func TestLogger_ValidateFilename(t *testing.T) {
	t.Parallel()
	runValidateFilenameTest(t, testLogFile, "valid filename", false)
	runValidateFilenameTest(t, "", "empty filename", true)
	runValidateFilenameTest(t, "dir/test.log", "filename with slash", true)
	runValidateFilenameTest(t, "dir\\test.log", "filename with backslash", true)
	runValidateFilenameTest(t, testLogPattern, "filename with dots", true)
	runValidateFilenameTest(t, "~test.log", "filename with tilde", true)
}

func runValidateFilenameTest(
	t *testing.T, filename, name string, wantErr bool,
) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()

		err := logger.ValidateFilename(filename)
		if (err != nil) != wantErr {
			t.Errorf("validateFilename() error = %v, wantErr %v", err, wantErr)
		}
	})
}

func TestLogger_InvalidLogDir(t *testing.T) {
	t.Parallel()

	_, err := logger.New("../invalid", testLogFile)
	if err == nil {
		t.Error("expected error for invalid log directory")
	}

	if !strings.Contains(err.Error(), "invalid log directory") {
		t.Errorf("expected 'invalid log directory' in error, got: %v", err)
	}
}

func TestLogger_InvalidFilename(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	_, err := logger.New(tempDir, "../test.log")
	if err == nil {
		t.Error("expected error for invalid filename")
	}

	if !strings.Contains(err.Error(), "invalid filename") {
		t.Errorf("expected 'invalid filename' in error, got: %v", err)
	}
}

func TestLogger_CreateLogDirIfNotExists(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	newLogDir := filepath.Join(tempDir, "new", "log", "dir")

	loggerInstance := createTestLogger(t, newLogDir, testLogFile)
	defer closeTestLogger(t, loggerInstance)

	verifyDirectoryCreated(t, newLogDir)
	verifyLogFileCreated(t, newLogDir, testLogFile)
}

func createTestLogger(t *testing.T, logDir, filename string) *logger.Logger {
	t.Helper()

	loggerInstance, err := logger.New(logDir, filename)
	if err != nil {
		t.Fatalf("New logger with new directory: %v", err)
	}

	return loggerInstance
}

func closeTestLogger(t *testing.T, loggerInstance *logger.Logger) {
	t.Helper()

	err := loggerInstance.Close()
	if err != nil {
		t.Logf(errorClosingLogger, err)
	}
}

func verifyDirectoryCreated(t *testing.T, dir string) {
	t.Helper()

	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Error("log directory was not created")
	}
}

func verifyLogFileCreated(t *testing.T, logDir, filename string) {
	t.Helper()

	logPath := filepath.Join(logDir, filename)

	_, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		t.Error("log file was not created")
	}
}

func TestLogger_EmptyMessage(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	loggerInstance, err := logger.New(tempDir, emptyLogFile)
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	defer func() {
		err := loggerInstance.Close()
		if err != nil {
			t.Logf(errorClosingLogger, err)
		}
	}()

	// Test empty format string
	loggerInstance.Info("", "some", "args")

	// Verify it logged something (should show "(empty message)")
	logPath := filepath.Join(tempDir, emptyLogFile)

	// #nosec G304 - Path is from t.TempDir() which is safe
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf(readLogFileErr, err)
	}

	if !strings.Contains(string(content), "(empty message)") {
		t.Errorf("expected '(empty message)', got: %s", string(content))
	}
}

func TestLogger_FormatMismatch(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	loggerInstance, err := logger.New(tempDir, formatLogFile)
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	defer func() {
		err := loggerInstance.Close()
		if err != nil {
			t.Logf(errorClosingLogger, err)
		}
	}()

	// Test format with % but no args - should not panic
	loggerInstance.Info("100% complete")

	// Test format mismatch - this might cause issues but should be handled
	loggerInstance.Warn("value: %d %s", 42) // Missing second arg

	// Should not crash, and file should exist
	logPath := filepath.Join(tempDir, formatLogFile)

	_, err = os.Stat(logPath)
	if os.IsNotExist(err) {
		t.Error("log file should exist even with format errors")
	}
}

func TestLogger_LongMessage(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	loggerInstance, err := logger.New(tempDir, longLogFile)
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	defer func() {
		err := loggerInstance.Close()
		if err != nil {
			t.Logf(errorClosingLogger, err)
		}
	}()

	// Create a very long message
	longMsg := strings.Repeat("A", 5000) // Longer than maxLogMessageLength
	loggerInstance.Info("Long message: %s", longMsg)

	// Verify it was truncated
	logPath := filepath.Join(tempDir, longLogFile)

	// #nosec G304 - Path is from t.TempDir() which is safe
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf(readLogFileErr, err)
	}

	if !strings.Contains(string(content), "[TRUNCATED]") {
		t.Errorf("expected truncation marker, got length: %d", len(content))
	}
}

func TestLogger_LogAfterClose(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	loggerInstance, err := logger.New(tempDir, "closed.log")
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	// Close the logger
	err = loggerInstance.Close()
	if err != nil {
		t.Fatalf("close logger: %v", err)
	}

	// Try to log after closing - should not panic
	loggerInstance.Info("This should go to stderr")
	loggerInstance.Error("This should also go to stderr")
	// Should not crash the program
}
