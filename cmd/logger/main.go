// Logger CLI - standalone logging service
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

	"logger"
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
  logger -daemon -dir /var/log
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
	config := parseFlags()
	if config.help {
		showHelp()

		return nil
	}

	if config.daemon {
		return runDaemon(config.logDir)
	}

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

func parseFlags() config {
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
	loggerInstance, err := logger.New(logDir, filename)
	if err != nil {
		return nil, fmt.Errorf(errorCreatingLogger, err)
	}

	return loggerInstance, nil
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
	loggerInstance.System(daemonStoppedMsg)

	return nil
}

func generateDaemonFilename() string {
	return fmt.Sprintf(daemonLogFilenameFmt, time.Now().Format(daemonTimestampFmt))
}

func startDaemon(loggerInstance *logger.Logger, logDir, filename string) {
	loggerInstance.System(daemonStartedMsg)
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
		loggerInstance.Error(daemonStdinErrorFmt, err)
	}
}

func processLogLine(loggerInstance *logger.Logger, line string) {
	if line == "" {
		return
	}

	level, message := parseLogLine(line)
	logMessageInDaemon(loggerInstance, level, message)
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
		handler = func(l *logger.Logger, msg string) { l.Info(msg) } // Default to INFO
	}

	handler(loggerInstance, message)
}

func parseLogLine(line string) (level, message string) {
	level, message, found := strings.Cut(line, ":")
	if !found {
		return logLevelINFO, line
	}

	return level, message
}

func showHelp() {
	log.Println(helpText)
}
