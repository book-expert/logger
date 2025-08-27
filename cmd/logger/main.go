// Logger CLI - standalone logging service
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"logger"
)

func main() {
	var (
		logDir   = flag.String("dir", "./logs", "Log directory")
		filename = flag.String("file", "", "Log filename (required)")
		level    = flag.String("level", "info", "Log level (info, warn, error, success, fatal, panic, system)")
		message  = flag.String("message", "", "Log message (required)")
		help     = flag.Bool("help", false, "Show help")
		daemon   = flag.Bool("daemon", false, "Run as daemon service (accept log messages on stdin)")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *daemon {
		runDaemon(*logDir)
		return
	}

	if *filename == "" {
		fmt.Fprintf(os.Stderr, "Error: -file is required\n")
		showHelp()
		os.Exit(1)
	}

	if *message == "" {
		fmt.Fprintf(os.Stderr, "Error: -message is required\n")
		showHelp()
		os.Exit(1)
	}

	// Create logger
	log, err := logger.New(*logDir, *filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Close() // Ignore error in defer
	}()

	// Log the message
	switch *level {
	case "info":
		log.Info(*message)
	case "warn":
		log.Warn(*message)
	case "error":
		log.Error(*message)
	case "success":
		log.Success(*message)
	case "fatal":
		log.Fatal(*message)
	case "panic":
		log.Panic(*message)
	case "system":
		log.System(*message)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown log level '%s'\n", *level)
		os.Exit(1)
	}
}

func runDaemon(logDir string) {
	// Generate timestamped filename
	filename := fmt.Sprintf("daemon-%s.log", time.Now().Format("20060102-150405"))

	log, err := logger.New(logDir, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating daemon logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Close() // Ignore error in defer
	}()

	log.System("Logger daemon started, reading from stdin...")
	fmt.Printf("Logger daemon started: %s/%s\n", logDir, filename)
	fmt.Println("Send log messages in format: LEVEL:MESSAGE")
	fmt.Println("Example: INFO:Application started")
	fmt.Println("Press Ctrl+C to stop")

	// Read from stdin and log messages
	var line string
	for {
		_, err := fmt.Scanln(&line)
		if err != nil {
			break
		}

		// Parse level:message format
		level, message := parseLogLine(line)

		switch level {
		case "INFO":
			log.Info(message)
		case "WARN":
			log.Warn(message)
		case "ERROR":
			log.Error(message)
		case "SUCCESS":
			log.Success(message)
		case "FATAL":
			log.Fatal(message)
		case "PANIC":
			log.Panic(message)
		case "SYSTEM":
			log.System(message)
		default:
			log.Info(line) // Default to info if no level specified
		}
	}

	log.System("Logger daemon stopped")
}

func parseLogLine(line string) (string, string) {
	for i, char := range line {
		if char == ':' && i > 0 {
			return line[:i], line[i+1:]
		}
	}
	return "INFO", line
}

func showHelp() {
	fmt.Println(`Logger - Standalone logging service

Usage: logger [options]

Options:
  -dir PATH        Log directory (default: ./logs)
  -file NAME       Log filename (required for single message mode)
  -level LEVEL     Log level: info, warn, error, success, fatal, panic, system (default: info)
  -message TEXT    Log message (required for single message mode)
  -daemon          Run as daemon service, reading log messages from stdin
  -help            Show this help message

Single Message Mode:
  logger -file app.log -level error -message "Database connection failed"
  logger -dir /var/log -file service.log -message "Service started"

Daemon Mode:
  logger -daemon -dir /var/log
  # Then send messages via stdin in format: LEVEL:MESSAGE
  # Example: echo "ERROR:Database timeout" | logger -daemon

Log Levels:
  info     - General information
  warn     - Warning messages
  error    - Error conditions
  success  - Success/completion messages
  fatal    - Fatal system errors
  panic    - Panic conditions
  system   - System-level events

Exit codes:
  0  Success
  1  Error (invalid arguments, file creation failed, etc.)`)
}
