// Package main provides a command-line utility for logging.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/book-expert/logger"
)

// Constants for command-line flags, usage text, and log messages.
const (
	defaultLogLevel      = "info"
	defaultLogDir        = "./logs"
	flagNameDir          = "dir"
	flagNameFile         = "file"
	flagNameLevel        = "level"
	flagNameMessage      = "message"
	flagNameHelp         = "help"
	flagNameDaemon       = "daemon"
	usageDir             = "Log directory"
	usageFile            = "Log filename (required)"
	usageLevel           = "Log level (info, warn, error, success, fatal, panic, system)"
	usageMessage         = "Log message (required)"
	usageHelp            = "Show help"
	usageDaemon          = "Run as daemon service (accept log messages on stdin)"
	logLevelINFO         = "INFO"
	errorFormat          = "error: %v\n"
	errorClosingLogger   = "error closing logger: %v"
	errorCreatingLogger  = "error creating logger: %w"
	errorFmtUnknownLevel = "%w: '%s'"
	daemonLogFilenameFmt = "daemon-%s.log"
	daemonTimestampFmt   = "20060102-150405"
	daemonStartedMsg     = "Logger daemon started, reading from stdin..."
	daemonStartedInfoFmt = "Logger daemon started: %s/%s\n"
	daemonUsageMsg       = "Send log messages in format: LEVEL:MESSAGE"
	daemonExampleMsg     = "Example: INFO:Application started"
	daemonStopMsg        = "Press Ctrl+C to stop"
	daemonStoppedMsg     = "Logger daemon stopped"
	daemonStdinErrorFmt  = "error reading from stdin: %v"
	logLineSplitCount    = 2
	// Error messages.
	errFileRequiredMsg    = "-file is required"
	errMessageRequiredMsg = "-message is required"
	errUnknownLogLevelMsg = "unknown log level"

	helpText = `Logger - Standalone logging service

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
  logger -daemon -dir /var/var/log
  # Then send messages via stdin in format: LEVEL:MESSAGE
  # Example: echo "ERROR:Database connection timeout" | \
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
  1  Error (invalid arguments, file creation failed, etc.)`
)

var (
	ErrFileRequired    = errors.New(errFileRequiredMsg)
	ErrMessageRequired = errors.New(errMessageRequiredMsg)
	ErrUnknownLogLevel = errors.New(errUnknownLogLevelMsg)
)

func main() {
	err := run()
	if err != nil {
		log.Printf(errorFormat, err)
		os.Exit(1)
	}
}

func run() error {
	// parseFlags parses command-line arguments into a config struct.
	config := parseFlags()
	// If the help flag is set, show the help message and exit.
	if config.help {
		showHelp()

		return nil
	}

	// If the daemon flag is set, run the logger in daemon mode.
	if config.daemon {
		return runDaemon(config.logDir)
	}

	// Otherwise, run the logger in single message mode.
	return runSingleMessage(&config)
}

type config struct {
	logDir   string
	filename string
	level    string
	message  string
	help     bool
	daemon   bool
}

func showHelp() {
	log.Println(helpText)
}

func parseFlags() config {
	// parseFlags parses command-line arguments into a config struct. This function
	// is responsible for defining and parsing all the command line flags that the
	// application accepts.
	var cfg config
	flag.StringVar(&cfg.logDir, flagNameDir, defaultLogDir, usageDir)
	flag.StringVar(&cfg.filename, flagNameFile, "", usageFile)
	flag.StringVar(&cfg.level, flagNameLevel, defaultLogLevel, usageLevel)
	flag.StringVar(&cfg.message, flagNameMessage, "", usageMessage)
	flag.BoolVar(&cfg.help, flagNameHelp, false, usageHelp)
	flag.BoolVar(&cfg.daemon, flagNameDaemon, false, usageDaemon)
	flag.Parse()

	return cfg
}

func runSingleMessage(cfg *config) error {
	// runSingleMessage runs the logger in single message mode. This function is
	// responsible for validating the arguments, creating the logger, and logging
	// the message.
	err := validateArgs(cfg.filename, cfg.message)
	if err != nil {
		showHelp()

		return err
	}

	loggerInstance, err := createLogger(cfg.logDir, cfg.filename)
	if err != nil {
		return err
	}
	defer closeLogger(loggerInstance)

	return logMessage(loggerInstance, cfg.level, cfg.message)
}

func createLogger(logDir, filename string) (*logger.Logger, error) {
	// createLogger creates a new logger instance. This function is responsible for
	// creating a new logger with the specified log directory and filename.
	loggerInstance, err := logger.New(logDir, filename)
	if err != nil {
		return nil, fmt.Errorf(errorCreatingLogger, err)
	}

	return loggerInstance, nil
}

func closeLogger(loggerInstance *logger.Logger) {
	// closeLogger closes the logger instance. This function is responsible for
	// closing the logger and handling any errors that may occur.
	err := loggerInstance.Close()
	if err != nil {
		log.Printf(errorClosingLogger, err)
	}
}

func validateArgs(filename, message string) error {
	// validateArgs validates the command-line arguments. This function is
	// responsible for ensuring that the required arguments are provided.
	if filename == "" {
		return ErrFileRequired
	}

	if message == "" {
		return ErrMessageRequired
	}

	return nil
}

func getLogLevelHandlers() map[string]func(*logger.Logger, string) {
	return map[string]func(*logger.Logger, string){
		logLevelINFO: func(l *logger.Logger, msg string) { l.Infof(msg) },
		"WARN":       func(l *logger.Logger, msg string) { l.Warnf(msg) },
		"ERROR":      func(l *logger.Logger, msg string) { l.Errorf(msg) },
		"SUCCESS":    func(l *logger.Logger, msg string) { l.Successf(msg) },
		"FATAL":      func(l *logger.Logger, msg string) { l.Fatalf(msg) },
		"PANIC":      func(l *logger.Logger, msg string) { l.Panicf(msg) },
		"SYSTEM":     func(l *logger.Logger, msg string) { l.Systemf(msg) },
	}
}

func getLevelHandlers() map[string]func(*logger.Logger, string) {
	// getLevelHandlers returns a map of log level handlers. This function is
	// responsible for mapping log level strings to their corresponding logger
	// functions.
	return getLogLevelHandlers()
}

func logMessage(loggerInstance *logger.Logger, level, message string) error {
	handlers := getLevelHandlers()

	handler, exists := handlers[level]
	if !exists {
		return fmt.Errorf(errorFmtUnknownLevel, ErrUnknownLogLevel, level)
	}

	handler(loggerInstance, message)

	return nil
}

func runDaemon(logDir string) error {
	filename := generateDaemonFilename()

	loggerInstance, err := createLogger(logDir, filename)
	if err != nil {
		return err
	}
	defer closeLogger(loggerInstance)

	startDaemon(loggerInstance, logDir, filename)
	processDaemonInput(loggerInstance)
	loggerInstance.Systemf(daemonStoppedMsg)

	return nil
}

func generateDaemonFilename() string {
	return fmt.Sprintf(daemonLogFilenameFmt, time.Now().Format(daemonTimestampFmt))
}
func startDaemon(loggerInstance *logger.Logger, logDir, filename string) {
	loggerInstance.Systemf(daemonStartedMsg)
	log.Printf(daemonStartedInfoFmt, logDir, filename)
	log.Println(daemonUsageMsg)
	log.Println(daemonExampleMsg)
	log.Println(daemonStopMsg)
}
func processDaemonInput(loggerInstance *logger.Logger) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		processLogLine(loggerInstance, scanner.Text())
	}

	err := scanner.Err()
	if err != nil {
		loggerInstance.Errorf(daemonStdinErrorFmt, err)
	}
}
func processLogLine(loggerInstance *logger.Logger, line string) {
	if line == "" {
		return
	}

	level, message := parseLogLine(line)

	err := logMessage(loggerInstance, level, message)
	if err != nil {
		loggerInstance.Errorf("error logging message from daemon: %v", err)
	}
}

func parseLogLine(line string) (string, string) {
	parts := strings.SplitN(line, ":", logLineSplitCount)
	if len(parts) != logLineSplitCount {
		return logLevelINFO, line // Default to INFO if format is incorrect
	}

	return strings.ToUpper(parts[0]), parts[1]
}

