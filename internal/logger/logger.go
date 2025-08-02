package logger

import (
	"io"
	"log"
	"os"
)

// Logger holds the logging configuration
type Logger struct {
	debugEnabled bool
}

var defaultLogger = &Logger{
	debugEnabled: false,
}

// SetDebugMode enables or disables debug logging globally
func SetDebugMode(enabled bool) {
	defaultLogger.debugEnabled = enabled

	if !enabled {
		// Disable all log output by default
		log.SetOutput(io.Discard)
	} else {
		// Enable log output to stderr when debug is on
		log.SetOutput(os.Stderr)
	}
}

// IsDebugEnabled returns whether debug logging is enabled
func IsDebugEnabled() bool {
	return defaultLogger.debugEnabled
}

// Debug logs a message only if debug mode is enabled
func Debug(format string, args ...interface{}) {
	if defaultLogger.debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an info message only if debug mode is enabled
func Info(format string, args ...interface{}) {
	if defaultLogger.debugEnabled {
		log.Printf("[INFO] "+format, args...)
	}
}

// Error logs an error message only if debug mode is enabled
func Error(format string, args ...interface{}) {
	if defaultLogger.debugEnabled {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Warn logs a warning message only if debug mode is enabled
func Warn(format string, args ...interface{}) {
	if defaultLogger.debugEnabled {
		log.Printf("[WARN] "+format, args...)
	}
}
