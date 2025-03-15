package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the log level
type LogLevel int

const (
	// DebugLevel is for debug messages
	DebugLevel LogLevel = iota
	// InfoLevel is for info messages
	InfoLevel
	// WarnLevel is for warning messages
	WarnLevel
	// ErrorLevel is for error messages
	ErrorLevel
)

// Logger represents a logger
type Logger struct {
	level  LogLevel
	prefix string
	logger *log.Logger
}

// NewLogger creates a new logger
func NewLogger(level LogLevel, prefix string) *Logger {
	return &Logger{
		level:  level,
		prefix: prefix,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	if l.level <= DebugLevel {
		l.log("DEBUG", msg)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	if l.level <= InfoLevel {
		l.log("INFO", msg)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	if l.level <= WarnLevel {
		l.log("WARN", msg)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	if l.level <= ErrorLevel {
		l.log("ERROR", msg)
	}
}

// log logs a message with the given level
func (l *Logger) log(level, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	l.logger.Printf("[%s] [%s] [%s] %s", timestamp, level, l.prefix, msg)
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel gets the log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// SetPrefix sets the log prefix
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// GetPrefix gets the log prefix
func (l *Logger) GetPrefix() string {
	return l.prefix
}

// LogLevelFromString converts a string to a log level
func LogLevelFromString(level string) LogLevel {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return fmt.Sprintf("LogLevel(%d)", l)
	}
}
