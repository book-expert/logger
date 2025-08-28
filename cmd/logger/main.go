// Logger CLI - standalone logging service
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"logger"
)

const (
	defaultLogLevel    = "info"
	logLevelINFO       = "INFO"
	errorFormat        = "Error: %v\n"
	errorClosingLogger = "Error closing logger: %v"
)

var (
	ErrFileRequired    = errors.New("-file is required")
	ErrMessageRequired = errors.New("-message is required")
	ErrUnknownLogLevel = errors.New("unknown log level")
)

func main() {
	config := parseFlags()

	if config.help {
		showHelp()

		return
	}

	if config.daemon {
		runDaemon(config.logDir)

		return
	}

	runSingleMessage(&config)
}

type config struct {
	logDir   string
	filename string
	level    string
	message  string
	help     bool
	daemon   bool
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.logDir, "dir", "./logs", "Log directory")
	flag.StringVar(&cfg.filename, "file", "", "Log filename (required)")
	flag.StringVar(&cfg.level, "level", defaultLogLevel,
		"Log level (info, warn, error, success, fatal, panic, system)")
	flag.StringVar(&cfg.message, "message", "", "Log message (required)")
	flag.BoolVar(&cfg.help, "help", false, "Show help")
	flag.BoolVar(&cfg.daemon, "daemon", false,
		"Run as daemon service (accept log messages on stdin)")
	flag.Parse()

	return cfg
}

func runSingleMessage(cfg *config) {
	err := validateArgs(cfg.filename, cfg.message)
	if err != nil {
		log.Printf(errorFormat, err)
		showHelp()
		os.Exit(1)
	}

	loggerInstance := createLogger(cfg.logDir, cfg.filename)
	defer closeLogger(loggerInstance)

	err = logMessage(loggerInstance, cfg.level, cfg.message)
	if err != nil {
		log.Printf(errorFormat, err)
		os.Exit(1)
	}
}

func createLogger(logDir, filename string) *logger.Logger {
	loggerInstance, err := logger.New(logDir, filename)
	if err != nil {
		log.Printf("Error creating logger: %v\n", err)
		os.Exit(1)
	}

	return loggerInstance
}

func closeLogger(loggerInstance *logger.Logger) {
	err := loggerInstance.Close()
	if err != nil {
		log.Printf(errorClosingLogger, err)
	}
}

func validateArgs(filename, message string) error {
	if filename == "" {
		return ErrFileRequired
	}

	if message == "" {
		return ErrMessageRequired
	}

	return nil
}

func getLevelHandlers() map[string]func(*logger.Logger, string) {
	return map[string]func(*logger.Logger, string){
		"info":    func(l *logger.Logger, msg string) { l.Info(msg) },
		"warn":    func(l *logger.Logger, msg string) { l.Warn(msg) },
		"error":   func(l *logger.Logger, msg string) { l.Error(msg) },
		"success": func(l *logger.Logger, msg string) { l.Success(msg) },
		"fatal":   func(l *logger.Logger, msg string) { l.Fatal(msg) },
		"panic":   func(l *logger.Logger, msg string) { l.Panic(msg) },
		"system":  func(l *logger.Logger, msg string) { l.System(msg) },
	}
}

func logMessage(loggerInstance *logger.Logger, level, message string) error {
	handlers := getLevelHandlers()

	handler, exists := handlers[level]
	if !exists {
		return fmt.Errorf("%w: '%s'", ErrUnknownLogLevel, level)
	}

	handler(loggerInstance, message)

	return nil
}

func runDaemon(logDir string) {
	filename := generateDaemonFilename()

	loggerInstance := createLogger(logDir, filename)
	defer closeLogger(loggerInstance)

	startDaemon(loggerInstance, logDir, filename)
	processDaemonInput(loggerInstance)
	loggerInstance.System("Logger daemon stopped")
}

func generateDaemonFilename() string {
	return fmt.Sprintf(
		"daemon-%s.log",
		time.Now().Format("20060102-150405"),
	)
}

func startDaemon(loggerInstance *logger.Logger, logDir, filename string) {
	loggerInstance.System("Logger daemon started, reading from stdin...")
	log.Printf("Logger daemon started: %s/%s\n", logDir, filename)
	log.Println("Send log messages in format: LEVEL:MESSAGE")
	log.Println("Example: INFO:Application started")
	log.Println("Press Ctrl+C to stop")
}

func processDaemonInput(loggerInstance *logger.Logger) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			level, message := parseLogLine(line)
			logMessageInDaemon(loggerInstance, level, message)
		}
	}

	err := scanner.Err()
	if err != nil {
		loggerInstance.Error("Error reading from stdin: %v", err)
	}
}

func getDaemonLevelHandlers() map[string]func(*logger.Logger, string) {
	return map[string]func(*logger.Logger, string){
		logLevelINFO: func(l *logger.Logger, msg string) { l.Info(msg) },
		"WARN":       func(l *logger.Logger, msg string) { l.Warn(msg) },
		"ERROR":      func(l *logger.Logger, msg string) { l.Error(msg) },
		"SUCCESS":    func(l *logger.Logger, msg string) { l.Success(msg) },
		"FATAL":      func(l *logger.Logger, msg string) { l.Fatal(msg) },
		"PANIC":      func(l *logger.Logger, msg string) { l.Panic(msg) },
		"SYSTEM":     func(l *logger.Logger, msg string) { l.System(msg) },
	}
}

func logMessageInDaemon(loggerInstance *logger.Logger, level, message string) {
	handlers := getDaemonLevelHandlers()

	handler, exists := handlers[level]
	if !exists {
		handler = func(l *logger.Logger, msg string) { l.Info(msg) }
	}

	handler(loggerInstance, message)
}

func parseLogLine(line string) (string, string) {
	for i, char := range line {
		if char == ':' && i > 0 {
			return line[:i], line[i+1:]
		}
	}

	return logLevelINFO, line
}

func showHelp() {
	log.Println(`Logger - Standalone logging service

Usage: logger [options]

Options:
  -dir PATH        Log directory (default: ./logs)
  -file NAME       Log filename (required for single message mode)
  -level LEVEL     Log level: info, warn, error, success, fatal, panic, system
                   (default: info)
  -message TEXT    Log message (required for single message mode)
  -daemon          Run as daemon service, reading log messages from stdin
  -help            Show this help message

Single Message Mode:
  logger -file app.log -level error -message "Database connection failed"
  logger -dir /var/log -file service.log -message "Service started"

Daemon Mode:
  logger -daemon -dir /var/log
  # Then send messages via stdin in format: LEVEL:MESSAGE
  # Example: echo "ERROR:Database connection timeout" | \\
  #   logger -daemon -dir /var/log
  # Or use with pipes: tail -f app.log | logger -daemon -dir /var/log

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
