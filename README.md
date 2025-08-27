# Logger

A robust, thread-safe logging package with comprehensive input validation and system failure handling.

## Features

- **Thread-Safe**: Concurrent logging with mutex protection
- **Dual Output**: Writes to both stdout and file simultaneously
- **Input Validation**: Handles empty messages, format errors, and invalid inputs gracefully
- **System Failure Resilience**: Continues logging even after system failures
- **Security**: Path validation prevents directory traversal attacks
- **Performance**: Optimized string building and pre-allocated capacity
- **Comprehensive Logging Levels**: Info, Warn, Error, Success, Fatal, Panic, System

## Installation

```bash
go get logger
```

## Usage

### Basic Logging

```go
package main

import (
    "logger"
)

func main() {
    // Create logger with directory and filename
    log, err := logger.New("./logs", "app.log")
    if err != nil {
        panic(err)
    }
    defer log.Close()
    
    log.Info("Application starting up")
    log.Warn("Low disk space: %d%% remaining", 15)
    log.Error("Failed to connect to database: %v", err)
    log.Success("Database connection established")
}
```

### System and Critical Logging

```go
// System events
log.System("Server startup complete")
log.System("Configuration reloaded")

// Critical errors that don't exit the program
log.Fatal("Critical system failure: %s", "disk full")
log.Panic("Memory corruption detected")
```

### Robust Error Handling

The logger handles various edge cases gracefully:

```go
// Empty messages
log.Info("") // Logs: "[INFO] (empty message)"

// Format string mismatches
log.Info("Progress: %d%%") // Logs: "[INFO] Progress: %d%%" (no panic)
log.Warn("Values: %s %d", "test") // Logs: "[WARN] Values: test %!d(MISSING)"

// Very long messages (automatically truncated)
longMessage := strings.Repeat("data", 2000)
log.Info("Large payload: %s", longMessage) // Automatically truncated with "[TRUNCATED]"

// Logging after logger is closed (graceful fallback to stderr)
log.Close()
log.Error("This still works") // Goes to stderr with "(logger closed)" prefix
```

## API Reference

### Constructor

#### `New(logDir string, filename string) (*Logger, error)`
Creates a new Logger instance.
- `logDir`: Directory for log files (created if doesn't exist)  
- `filename`: Name of log file
- Returns: Logger instance or error if invalid paths/permissions

### Logging Methods

#### `Info(format string, args ...any)`
Logs informational messages

#### `Warn(format string, args ...any)`
Logs warning messages

#### `Error(format string, args ...any)`
Logs error messages

#### `Success(format string, args ...any)`
Logs success/completion messages

#### `Fatal(format string, args ...any)`
Logs fatal system errors (does NOT exit unlike log.Fatal)

#### `Panic(format string, args ...any)`
Logs panic-level errors (does NOT panic unlike log.Panic)

#### `System(format string, args ...any)`
Logs system-level events (startup, shutdown, config changes)

### Resource Management

#### `Close() error`
Closes the log file and releases resources. Safe to call multiple times.

## Security Features

- **Path Validation**: Prevents directory traversal attacks (`../`, `~`)
- **Filename Sanitization**: Blocks unsafe filename patterns
- **Input Sanitization**: Handles malicious format strings safely

## Error Resilience

- **Format String Protection**: Recovers from format panics gracefully
- **Message Length Limits**: Automatically truncates messages over 4KB
- **Closed Logger Handling**: Continues logging to stderr if file logger is closed
- **Thread Safety**: All operations are mutex-protected

## Testing

Run comprehensive tests including edge cases:

```bash
go test -v
```

Tests cover:
- ✅ Basic logging functionality
- ✅ Path and filename validation
- ✅ Empty message handling
- ✅ Format string mismatches
- ✅ Long message truncation  
- ✅ Logging after close
- ✅ Concurrent access safety
- ✅ Directory creation
- ✅ Security validation

## Performance

- Pre-allocated string builders
- Efficient mutex usage
- Minimal memory allocations
- 4KB message size limit prevents excessive memory usage

## Requirements

- Go 1.25+
- Standard library only (no external dependencies)

## Thread Safety

All logging operations are thread-safe and can be called concurrently from multiple goroutines.

## License

This project follows the same license as the parent projects it serves.