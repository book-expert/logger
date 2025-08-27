package logger

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger_WritesToStdoutAndFile(t *testing.T) {
	logDir := t.TempDir()
	logger, err := New(logDir, "test.log")
	if err != nil {
		t.Fatalf("New logger: %v", err)
	}
	defer func() {
		_ = logger.Close() // Ignore errors in test cleanup
	}()

	logger.Info("hello %s", "world")
	logger.Warn("warn %d", 42)
	logger.Error("err %v", 1)
	logger.Success("ok")
	logger.Fatal("system failure: %s", "disk full")
	logger.Panic("panic condition: %v", "nil pointer")
	logger.System("system event: %s", "startup complete")

	// Verify file content contains the messages and levels
	path := filepath.Join(logDir, "test.log")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}
	defer func() {
		_ = f.Close() // Ignore errors in test cleanup
	}()

	s := bufio.NewScanner(f)
	var lines []string
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"[INFO] hello world", "[WARN] warn 42", "[ERROR] err 1", "[SUCCESS] ok", "[FATAL] system failure: disk full", "[PANIC] panic condition: nil pointer", "[SYSTEM] system event: startup complete"} {
		if !strings.Contains(joined, want) {
			t.Errorf("log file missing %q; got:\n%s", want, joined)
		}
	}
}

func TestLogger_CloseIdempotent(t *testing.T) {
	logDir := t.TempDir()
	logger, err := New(logDir, "test2.log")
	if err != nil {
		t.Fatalf("New logger: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	// Second close should be safe
	if err := logger.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestLogger_ValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "/tmp/logs", false},
		{"empty path", "", true},
		{"path traversal dots", "/tmp/../etc", true},
		{"path traversal tilde", "~/logs", true},
		{"relative path", "logs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogger_ValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"valid filename", "test.log", false},
		{"empty filename", "", true},
		{"filename with slash", "dir/test.log", true},
		{"filename with backslash", "dir\\test.log", true},
		{"filename with dots", "../test.log", true},
		{"filename with tilde", "~test.log", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilename() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogger_InvalidLogDir(t *testing.T) {
	_, err := New("../invalid", "test.log")
	if err == nil {
		t.Error("expected error for invalid log directory")
	}
	if !strings.Contains(err.Error(), "invalid log directory") {
		t.Errorf("expected 'invalid log directory' in error, got: %v", err)
	}
}

func TestLogger_InvalidFilename(t *testing.T) {
	tempDir := t.TempDir()
	_, err := New(tempDir, "../test.log")
	if err == nil {
		t.Error("expected error for invalid filename")
	}
	if !strings.Contains(err.Error(), "invalid filename") {
		t.Errorf("expected 'invalid filename' in error, got: %v", err)
	}
}

func TestLogger_CreateLogDirIfNotExists(t *testing.T) {
	tempDir := t.TempDir()
	newLogDir := filepath.Join(tempDir, "new", "log", "dir")

	logger, err := New(newLogDir, "test.log")
	if err != nil {
		t.Fatalf("New logger with new directory: %v", err)
	}
	defer func() {
		_ = logger.Close()
	}()

	// Verify directory was created
	if _, err := os.Stat(newLogDir); os.IsNotExist(err) {
		t.Error("log directory was not created")
	}

	// Verify log file was created
	logPath := filepath.Join(newLogDir, "test.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file was not created")
	}
}

func TestLogger_EmptyMessage(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := New(tempDir, "empty.log")
	if err != nil {
		t.Fatalf("New logger: %v", err)
	}
	defer func() {
		_ = logger.Close()
	}()

	// Test empty format string
	logger.Info("", "some", "args")

	// Verify it logged something (should show "(empty message)")
	logPath := filepath.Join(tempDir, "empty.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "(empty message)") {
		t.Errorf("expected '(empty message)', got: %s", string(content))
	}
}

func TestLogger_FormatMismatch(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := New(tempDir, "format.log")
	if err != nil {
		t.Fatalf("New logger: %v", err)
	}
	defer func() {
		_ = logger.Close()
	}()

	// Test format with % but no args - should not panic
	logger.Info("100% complete")

	// Test format mismatch - this might cause issues but should be handled
	logger.Warn("value: %d %s", 42) // Missing second arg

	// Should not crash, and file should exist
	logPath := filepath.Join(tempDir, "format.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file should exist even with format errors")
	}
}

func TestLogger_LongMessage(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := New(tempDir, "long.log")
	if err != nil {
		t.Fatalf("New logger: %v", err)
	}
	defer func() {
		_ = logger.Close()
	}()

	// Create a very long message
	longMsg := strings.Repeat("A", 5000) // Longer than maxLogMessageLength
	logger.Info("Long message: %s", longMsg)

	// Verify it was truncated
	logPath := filepath.Join(tempDir, "long.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "[TRUNCATED]") {
		t.Errorf("expected truncation marker, got length: %d", len(string(content)))
	}
}

func TestLogger_LogAfterClose(t *testing.T) {
	tempDir := t.TempDir()
	logger, err := New(tempDir, "closed.log")
	if err != nil {
		t.Fatalf("New logger: %v", err)
	}

	// Close the logger
	if err := logger.Close(); err != nil {
		t.Fatalf("close logger: %v", err)
	}

	// Try to log after closing - should not panic
	logger.Info("This should go to stderr")
	logger.Error("This should also go to stderr")

	// Should not crash the program
}
