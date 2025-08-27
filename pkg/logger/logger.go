package logger

import (
	"log"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = INFO
	debugLogger  *log.Logger
	infoLogger   *log.Logger
	warnLogger   *log.Logger
	errorLogger  *log.Logger
)

func init() {
	// Initialize loggers with different prefixes
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
	infoLogger = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	warnLogger = log.New(os.Stdout, "[WARN] ", log.LstdFlags)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
}

// SetLogLevel sets the global log level
func SetLogLevel(level LogLevel) {
	currentLevel = level
}

// SetLogLevelFromString sets the global log level from a string
func SetLogLevelFromString(levelStr string) {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARN":
		currentLevel = WARN
	case "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
}

// GetLogLevel returns the current log level
func GetLogLevel() LogLevel {
	return currentLevel
}

// Debug logs a debug message if debug level is enabled
func Debug(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		debugLogger.Printf(format, v...)
	}
}

// Info logs an info message if info level is enabled
func Info(format string, v ...interface{}) {
	if currentLevel <= INFO {
		infoLogger.Printf(format, v...)
	}
}

// Warn logs a warning message if warn level is enabled
func Warn(format string, v ...interface{}) {
	if currentLevel <= WARN {
		warnLogger.Printf(format, v...)
	}
}

// Error logs an error message if error level is enabled
func Error(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		errorLogger.Printf(format, v...)
	}
}

// Debugf is an alias for Debug for consistency
func Debugf(format string, v ...interface{}) {
	Debug(format, v...)
}

// Infof is an alias for Info for consistency
func Infof(format string, v ...interface{}) {
	Info(format, v...)
}

// Warnf is an alias for Warn for consistency
func Warnf(format string, v ...interface{}) {
	Warn(format, v...)
}

// Errorf is an alias for Error for consistency
func Errorf(format string, v ...interface{}) {
	Error(format, v...)
}
