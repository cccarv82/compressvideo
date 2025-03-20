package util

import (
	"fmt"
	"os"
	"runtime"
	"strings"
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

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[37m"
	colorWhite   = "\033[97m"
	
	colorBold      = "\033[1m"
	colorUnderline = "\033[4m"
)

// Logger handles logging and user feedback
type Logger struct {
	Level       LogLevel
	Verbose     bool
	UseColors   bool
	TimeFormat  string
	ShowLogTime bool
}

// NewLogger creates a new Logger with the specified log level and verbosity
func NewLogger(verbose bool) *Logger {
	level := LogLevelInfo
	if verbose {
		level = LogLevelDebug
	}
	
	// Determine if we should use colors (disable on Windows unless in ANSICON/ConEmu/WSL)
	useColors := true
	if runtime.GOOS == "windows" {
		// Check for common Windows terminals that support ANSI
		_, hasAnsiCon := os.LookupEnv("ANSICON")
		_, hasConEmu := os.LookupEnv("ConEmuANSI")
		_, hasWT := os.LookupEnv("WT_SESSION")
		
		// Disable colors by default on Windows unless in compatible terminal
		useColors = hasAnsiCon || hasConEmu || hasWT
	}
	
	return &Logger{
		Level:       level,
		Verbose:     verbose,
		UseColors:   useColors,
		TimeFormat:  "15:04:05",
		ShowLogTime: true,
	}
}

// formatMessage creates a formatted log message with timestamp and level
func (l *Logger) formatMessage(level, color, message string) string {
	var timePrefix string
	if l.ShowLogTime {
		timestamp := time.Now().Format(l.TimeFormat)
		timePrefix = fmt.Sprintf("[%s] ", timestamp)
	}
	
	if l.UseColors {
		return fmt.Sprintf("%s%s%s%s: %s%s", 
			timePrefix, color, level, colorReset, color, message)
	}
	
	return fmt.Sprintf("%s%s: %s", timePrefix, level, message)
}

// colorize adds color to a string if colors are enabled
func (l *Logger) colorize(color, message string) string {
	if l.UseColors {
		return color + message + colorReset
	}
	return message
}

// Info logs information messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("INFO", colorBlue, message))
	}
}

// Debug logs debug messages
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.Level >= LogLevelDebug {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("DEBUG", colorCyan, message))
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, l.formatMessage("ERROR", colorRed, message))
}

// Warning logs warning messages
func (l *Logger) Warning(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("WARNING", colorYellow, message))
	}
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, l.formatMessage("FATAL", colorRed+colorBold, message))
	os.Exit(1)
}

// Success logs a success message
func (l *Logger) Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(l.formatMessage("SUCCESS", colorGreen, message))
}

// Title displays a title section in the log
func (l *Logger) Title(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		
		// Create a line of equal signs the same length as the message
		lineLength := len(message)
		line := strings.Repeat("=", lineLength)
		
		if l.UseColors {
			fmt.Println(l.colorize(colorYellow+colorBold, line))
			fmt.Println(l.colorize(colorYellow+colorBold, message))
			fmt.Println(l.colorize(colorYellow+colorBold, line))
		} else {
			fmt.Println(line)
			fmt.Println(message)
			fmt.Println(line)
		}
	}
}

// Section displays a section header in the log
func (l *Logger) Section(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		
		if l.UseColors {
			fmt.Println(l.colorize(colorMagenta+colorBold, message))
			fmt.Println(l.colorize(colorMagenta, strings.Repeat("-", len(message))))
		} else {
			fmt.Println(message)
			fmt.Println(strings.Repeat("-", len(message)))
		}
	}
}

// Field logs a labeled field value
func (l *Logger) Field(label, format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		value := fmt.Sprintf(format, args...)
		
		if l.UseColors {
			fmt.Printf("%s: %s\n", 
				l.colorize(colorYellow, label),
				value)
		} else {
			fmt.Printf("%s: %s\n", label, value)
		}
	}
}

// Progress logs a progress message
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("PROGRESS", colorMagenta, message))
	}
}

// SetUseColors enables or disables color output
func (l *Logger) SetUseColors(useColors bool) {
	l.UseColors = useColors
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.Level = level
}

// IsVerbose returns true if verbose mode is enabled
func (l *Logger) IsVerbose() bool {
	return l.Verbose
} 