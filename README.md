# Logger

A robust, thread-safe logging system with comprehensive input validation and CLI interface. This standalone logger can be used both as a Go library and as a command-line binary.

## Architecture

This project provides both:
- **Library API** (`logger.go`): Thread-safe logging package for Go applications
- **CLI Binary** (`cmd/logger/main.go`): Standalone executable for shell scripts and external tools

## Features

- **Thread-Safe**: Concurrent logging with mutex protection
- **Dual Output**: Writes to both stdout and file simultaneously  
- **Input Validation**: Handles empty messages, format errors, and invalid inputs gracefully
- **System Failure Resilience**: Continues logging even after system failures
- **Security**: Path validation prevents directory traversal attacks
- **Performance**: Optimized string building and pre-allocated capacity
- **Comprehensive Logging Levels**: Info, Warn, Error, Success, Fatal, Panic, System
- **CLI Interface**: Single message and daemon mode support
- **Wrapper Compatibility**: Designed to work with existing internal logging APIs

## Installation

### As Binary
```bash
# Build to ~/bin (default target)
make build

# Or build manually
cd cmd/logger && go build -o ~/bin/logger .
```

### As Go Module
```bash
go get logger
```

## Usage

### Command Line Interface

#### Single Message Mode
```bash
# Basic usage
~/bin/logger -dir ./logs -file app.log -level info -message "Application started"

# Different log levels
~/bin/logger -dir ./logs -file app.log -level warn -message "Low disk space: 85% full"
~/bin/logger -dir ./logs -file app.log -level error -message "Database connection failed"
~/bin/logger -dir ./logs -file app.log -level success -message "Deployment completed"
```

#### Daemon Mode (stdin)
```bash
# Start daemon mode
~/bin/logger -dir ./logs -file app.log -daemon

# Then send messages in LEVEL:MESSAGE format
echo "info:Server starting up" | ~/bin/logger -dir ./logs -file app.log -daemon
echo "error:Connection timeout" | ~/bin/logger -dir ./logs -file app.log -daemon
```

### Library API

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

### Wrapper Integration

The logger is designed to work with wrapper functions that maintain existing APIs:

```go
// Example wrapper that calls the binary
func (l *Logger) Info(format string, args ...any) {
    message := fmt.Sprintf(format, args...)
    cmd := exec.Command(os.ExpandEnv("$HOME/bin/logger"),
        "-dir", l.logDir,
        "-file", l.filename, 
        "-level", "info",
        "-message", message)
    _ = cmd.Run() // Run in background
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

## Project Structure

```
~/Dev/logger/
├── logger.go              # Core logging library
├── cmd/logger/
│   └── main.go            # CLI binary implementation  
├── logger_test.go         # Comprehensive test suite
├── Makefile              # Build automation (targets ~/bin)
├── go.mod                # Go module definition
├── project.toml          # Project configuration
└── README.md             # This documentation
```

## Development Workflow

```bash
# Format, lint, test, and build in proper sequence
make all

# Individual steps
make format     # Format code with gofmt
make lint       # Run comprehensive linting (go vet, staticcheck, gosec)
make test       # Run test suite with coverage
make build      # Build binary to ~/bin/logger
```

## Requirements

- Go 1.21+
- Standard library only (no external dependencies)
- Unix-like environment (for ~/bin path)

## Thread Safety

All logging operations are thread-safe and can be called concurrently from multiple goroutines.

## Integration Examples

### book_expert Integration
The logger is used in book_expert through wrapper functions that maintain the original internal API while calling the standalone binary underneath.

### Shell Script Integration
```bash
#!/bin/bash
LOG_DIR="./logs"
LOG_FILE="script.log"

# Function to log from shell scripts
log_info() {
    ~/bin/logger -dir "$LOG_DIR" -file "$LOG_FILE" -level info -message "$1"
}

log_error() {
    ~/bin/logger -dir "$LOG_DIR" -file "$LOG_FILE" -level error -message "$1"
}

# Usage
log_info "Script started"
if ! some_command; then
    log_error "Command failed with exit code $?"
fi
log_info "Script completed"
```

### External Process Integration
The binary design allows any language or system to use the logger:

```python
import subprocess
import sys

def log(level, message, log_dir="./logs", log_file="app.log"):
    try:
        subprocess.run([
            f"{os.path.expanduser('~/bin/logger')}",
            "-dir", log_dir,
            "-file", log_file, 
            "-level", level,
            "-message", message
        ], check=True)
    except subprocess.CalledProcessError as e:
        print(f"Logging failed: {e}", file=sys.stderr)

# Usage
log("info", "Python application started")
log("error", "Database connection failed")
```

## Design Philosophy

This logger follows key design principles:
- **No Mocks**: Real implementations only, no fake/mock objects
- **Security First**: Comprehensive input validation and path sanitization
- **Wrapper Compatibility**: Maintains existing APIs while leveraging standalone architecture  
- **Unix Philosophy**: Does one thing well, integrates cleanly with other tools
- **Defensive Programming**: Graceful handling of edge cases and failures

## License

This project follows the same license as the parent projects it serves.