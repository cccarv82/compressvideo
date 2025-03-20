package util

import (
	"fmt"
	"os"
	"time"
)

// LogLevel defines the level of logging
type LogLevel int

const (
	// LogLevelError only logs errors
	LogLevelError LogLevel = iota
	// LogLevelInfo logs general information plus errors
	LogLevelInfo
	// LogLevelDebug logs debug information plus info and errors
	LogLevelDebug
)

// Logger handles logging and user feedback
type Logger struct {
	Level   LogLevel
	Verbose bool
}

// NewLogger creates a new Logger with the specified log level and verbosity
func NewLogger(verbose bool) *Logger {
	level := LogLevelInfo
	if verbose {
		level = LogLevelDebug
	}
	return &Logger{
		Level:   level,
		Verbose: verbose,
	}
}

// Info logs information messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] INFO: %s\n", timestamp, fmt.Sprintf(format, args...))
	}
}

// Debug logs debug messages
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.Level >= LogLevelDebug {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] DEBUG: %s\n", timestamp, fmt.Sprintf(format, args...))
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] ERROR: %s\n", timestamp, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] FATAL: %s\n", timestamp, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Success logs a success message
func (l *Logger) Success(format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] SUCCESS: %s\n", timestamp, fmt.Sprintf(format, args...))
} 