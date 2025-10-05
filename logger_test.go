// Package logger is a cmdline utility and a library for logging
package logger_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/book-expert/logger"
)

const (
	testLogFile                = "test.log"
	newLoggerError             = "New logger: %v"
	emptyLogFile               = "empty.log"
	readLogFileErr             = "read log file: %v"
	formatLogFile              = "format.log"
	longLogFile                = "long.log"
	errorClosingLogger         = "Error closing logger: %v"
	testLogPattern             = "../test.log"
	pathTraversalDotsTest      = "/tmp/../etc"
	invalidDirTest             = "../invalid"
	infoLogFormat              = "hello %s"
	infoLogArg                 = "world"
	warnLogFormat              = "warn %d"
	errorLogFormat             = "err %v"
	successLogMsg              = "ok"
	fatalLogFormat             = "system failure: %s"
	fatalLogArg                = "disk full"
	panicLogFormat             = "panic condition: %v"
	panicLogArg                = "nil pointer"
	systemLogFormat            = "system event: %s"
	systemLogArg               = "startup complete"
	logFileMissingFmt          = "log file missing %q; got:\n%s"
	closeIdempotentFile        = "test2.log"
	firstCloseErrFmt           = "first close: %v"
	secondCloseErrFmt          = "second close should not error: %v"
	validPath                  = "/tmp/logs"
	validPathName              = "valid path"
	emptyPathName              = "empty path"
	pathTraversalDotsName      = "path traversal dots"
	pathTraversalTildeName     = "path traversal tilde"
	pathWithTilde              = "~/logs"
	relativePath               = "logs"
	relativePathName           = "relative path"
	validatePathErrFmt         = "validatePath() error = %v, wantErr %v"
	validFilenameName          = "valid filename"
	emptyFilenameName          = "empty filename"
	filenameWithSlash          = "dir/test.log"
	filenameWithSlashName      = "filename with slash"
	filenameWithBackslash      = "dir\\test.log"
	filenameWithBackslashName  = "filename with backslash"
	filenameWithDotsName       = "filename with dots"
	filenameWithTilde          = "~test.log"
	filenameWithTildeName      = "filename with tilde"
	validateFilenameErrFmt     = "validateFilename() error = %v, wantErr %v"
	expectedErrForInvalidDir   = "expected error for invalid log directory"
	invalidLogDirMsg           = "invalid log directory"
	expectedErrMsgFmt          = "expected '%s' in error, got: %v"
	expectedErrForInvalidFile  = "expected error for invalid filename"
	invalidFilenameMsg         = "invalid filename"
	newLogDirPart1             = "new"
	newLogDirPart2             = "log"
	newLogDirPart3             = "dir"
	newLoggerWithDirErrFmt     = "New logger with new directory: %v"
	logDirNotCreatedMsg        = "log directory was not created"
	logFileNotCreatedMsg       = "log file was not created"
	emptyMsgArg1               = "some"
	emptyMsgArg2               = "args"
	expectedEmptyMsgContent    = "(empty message)"
	expectedEmptyMsgFmt        = "expected '%s', got: %s"
	formatMismatchMsg          = "100% complete"
	formatMismatchWarnMsg      = "value: %d %s"
	logFileExistsMsg           = "log file should exist even with format errors"
	longMsgFormat              = "Long message: %s"
	expectedTruncationMarker   = "[TRUNCATED]"
	truncationErrFmt           = "expected truncation marker, got length: %d"
	closedLogFile              = "closed.log"
	closeLoggerErrFmt          = "close logger: %v"
	logAfterCloseInfoMsg       = "This should go to stderr"
	logAfterCloseErrMsg        = "This should also go to stderr"
	setupTestLoggerErrFmt      = "setupTestLogger: failed to create logger: %v"
	setupTestLoggerCloseErrFmt = "setupTestLogger: failed to close logger: %v"
)

// setupTestLogger is a helper to create and automatically clean up a logger for tests.
func setupTestLogger(
	t *testing.T,
	filename string,
) (loggerInstance *logger.Logger, logPath string) {
	t.Helper()

	tempDir := t.TempDir()

	loggerInstance, err := logger.New(tempDir, filename)
	if err != nil {
		t.Fatalf(setupTestLoggerErrFmt, err)
	}

	t.Cleanup(func() {
		err := loggerInstance.Close()
		if err != nil {
			t.Logf(setupTestLoggerCloseErrFmt, err)
		}
	})

	logPath = filepath.Join(tempDir, filename)

	return loggerInstance, logPath
}

func TestLogger_WritesToStdoutAndFile(t *testing.T) {
	t.Parallel()

	loggerInstance, logPath := setupTestLogger(t, testLogFile)
	testLoggerAllLevels(t, loggerInstance)
	verifyLogFileContents(t, filepath.Dir(logPath), filepath.Base(logPath))
}

func testLoggerAllLevels(t *testing.T, loggerInstance *logger.Logger) {
	t.Helper()
	loggerInstance.Infof(infoLogFormat, infoLogArg)
	loggerInstance.Warnf(warnLogFormat, 42)
	loggerInstance.Errorf(errorLogFormat, 1)
	loggerInstance.Successf(successLogMsg)
	loggerInstance.Fatalf(fatalLogFormat, fatalLogArg)
	loggerInstance.Panicf(panicLogFormat, panicLogArg)
	loggerInstance.Systemf(systemLogFormat, systemLogArg)
}

func verifyLogFileContents(t *testing.T, logDir, filename string) {
	t.Helper()

	path := filepath.Join(logDir, filename)
	// #nosec G304
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(readLogFileErr, err)
	}

	contentStr := string(content)

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
		if !strings.Contains(contentStr, want) {
			t.Errorf(logFileMissingFmt, want, contentStr)
		}
	}
}

func TestLogger_CloseIdempotent(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()

	loggerInstance, err := logger.New(logDir, closeIdempotentFile)
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	err = loggerInstance.Close()
	if err != nil {
		t.Fatalf(firstCloseErrFmt, err)
	}

	err = loggerInstance.Close()
	if err != nil {
		t.Fatalf(secondCloseErrFmt, err)
	}
}

func TestLogger_ValidatePath(t *testing.T) {
	t.Parallel()
	runValidatePathTest(t, validPath, validPathName, false)
	runValidatePathTest(t, "", emptyPathName, true)
	runValidatePathTest(t, pathTraversalDotsTest, pathTraversalDotsName, true)
	runValidatePathTest(t, pathWithTilde, pathTraversalTildeName, true)
	runValidatePathTest(t, relativePath, relativePathName, false)
}

func runValidatePathTest(t *testing.T, path, name string, wantErr bool) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()

		err := logger.ValidatePath(path)
		if (err != nil) != wantErr {
			t.Errorf(validatePathErrFmt, err, wantErr)
		}
	})
}

func TestLogger_ValidateFilename(t *testing.T) {
	t.Parallel()
	runValidateFilenameTest(t, testLogFile, validFilenameName, false)
	runValidateFilenameTest(t, "", emptyFilenameName, true)
	runValidateFilenameTest(t, filenameWithSlash, filenameWithSlashName, true)
	runValidateFilenameTest(t, filenameWithBackslash, filenameWithBackslashName, true)
	runValidateFilenameTest(t, testLogPattern, filenameWithDotsName, true)
	runValidateFilenameTest(t, filenameWithTilde, filenameWithTildeName, true)
}

func runValidateFilenameTest(t *testing.T, filename, name string, wantErr bool) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Parallel()

		err := logger.ValidateFilename(filename)
		if (err != nil) != wantErr {
			t.Errorf(validateFilenameErrFmt, err, wantErr)
		}
	})
}

func TestLogger_InvalidLogDir(t *testing.T) {
	t.Parallel()

	_, err := logger.New(invalidDirTest, testLogFile)
	if err == nil {
		t.Error(expectedErrForInvalidDir)
	}

	if !strings.Contains(err.Error(), invalidLogDirMsg) {
		t.Errorf(expectedErrMsgFmt, invalidLogDirMsg, err)
	}
}

func TestLogger_InvalidFilename(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	_, err := logger.New(tempDir, testLogPattern)
	if err == nil {
		t.Error(expectedErrForInvalidFile)
	}

	if !strings.Contains(err.Error(), invalidFilenameMsg) {
		t.Errorf(expectedErrMsgFmt, invalidFilenameMsg, err)
	}
}

func TestLogger_CreateLogDirIfNotExists(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	newLogDir := filepath.Join(
		tempDir,
		newLogDirPart1,
		newLogDirPart2,
		newLogDirPart3,
	)

	loggerInstance := createTestLogger(t, newLogDir, testLogFile)
	defer closeTestLogger(t, loggerInstance)

	verifyDirectoryCreated(t, newLogDir)
	verifyLogFileCreated(t, newLogDir, testLogFile)
}

func createTestLogger(t *testing.T, logDir, filename string) *logger.Logger {
	t.Helper()

	loggerInstance, err := logger.New(logDir, filename)
	if err != nil {
		t.Fatalf(newLoggerWithDirErrFmt, err)
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
		t.Error(logDirNotCreatedMsg)
	}
}

func verifyLogFileCreated(t *testing.T, logDir, filename string) {
	t.Helper()

	logPath := filepath.Join(logDir, filename)

	_, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		t.Error(logFileNotCreatedMsg)
	}
}

func TestLogger_EmptyMessage(t *testing.T) {
	t.Parallel()

	loggerInstance, logPath := setupTestLogger(t, emptyLogFile)
	loggerInstance.Infof("", emptyMsgArg1, emptyMsgArg2)
	// #nosec G304
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf(readLogFileErr, err)
	}

	if !strings.Contains(string(content), expectedEmptyMsgContent) {
		t.Errorf(expectedEmptyMsgFmt, expectedEmptyMsgContent, string(content))
	}
}

func TestLogger_FormatMismatch(t *testing.T) {
	t.Parallel()

	loggerInstance, logPath := setupTestLogger(t, formatLogFile)
	loggerInstance.Infof(formatMismatchMsg)
	loggerInstance.Warnf(formatMismatchWarnMsg, 42)

	_, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		t.Error(logFileExistsMsg)
	}
}

func TestLogger_LongMessage(t *testing.T) {
	t.Parallel()

	loggerInstance, logPath := setupTestLogger(t, longLogFile)
	longMsg := strings.Repeat("A", 5000)
	loggerInstance.Infof(longMsgFormat, longMsg)
	// #nosec G304
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf(readLogFileErr, err)
	}

	if !strings.Contains(string(content), expectedTruncationMarker) {
		t.Errorf(truncationErrFmt, len(content))
	}
}

func TestLogger_LogAfterClose(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	loggerInstance, err := logger.New(tempDir, closedLogFile)
	if err != nil {
		t.Fatalf(newLoggerError, err)
	}

	err = loggerInstance.Close()
	if err != nil {
		t.Fatalf(closeLoggerErrFmt, err)
	}

	loggerInstance.Infof(logAfterCloseInfoMsg)
	loggerInstance.Errorf(logAfterCloseErrMsg)
}
