package logger

import (
	"os"
)

// Init initializes the global logger with default settings
func Init() {
	// Check if LOG_LEVEL environment variable is set
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		SetLogLevelFromString(logLevel)
	} else {
		// Default to INFO level
		SetLogLevel(INFO)
	}
}

// InitWithLevel initializes the global logger with a specific level
func InitWithLevel(level LogLevel) {
	SetLogLevel(level)
}

// InitWithString initializes the global logger with a string level
func InitWithString(levelStr string) {
	SetLogLevelFromString(levelStr)
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	return currentLevel <= DEBUG
}

// IsInfoEnabled returns true if info logging is enabled
func IsInfoEnabled() bool {
	return currentLevel <= INFO
}

// IsWarnEnabled returns true if warn logging is enabled
func IsWarnEnabled() bool {
	return currentLevel <= WARN
}

// IsErrorEnabled returns true if error logging is enabled
func IsErrorEnabled() bool {
	return currentLevel <= ERROR
}
