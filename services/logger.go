package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func (ll LogLevel) String() string {
	switch ll {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Function  string                 `json:"function,omitempty"`
}

// Logger provides structured logging with levels
type Logger struct {
	level      LogLevel
	output     io.Writer
	jsonFormat bool
	mu         sync.Mutex

	// Context fields (added to all log entries)
	contextFields map[string]interface{}
}

var (
	globalLogger     *Logger
	globalLoggerOnce sync.Once
)

// InitLogger initializes the global logger
func InitLogger(level LogLevel, jsonFormat bool, output io.Writer) {
	globalLoggerOnce.Do(func() {
		if output == nil {
			output = os.Stdout
		}

		globalLogger = &Logger{
			level:         level,
			output:        output,
			jsonFormat:    jsonFormat,
			contextFields: make(map[string]interface{}),
		}

		fmt.Printf("✅ Logger initialized (level: %s, json: %v)\n", level, jsonFormat)
	})
}

// GetLogger returns the global logger
func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default settings if not already initialized
		InitLogger(LogLevelInfo, false, os.Stdout)
	}
	return globalLogger
}

// WithField adds a field to a new logger instance
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:         l.level,
		output:        l.output,
		jsonFormat:    l.jsonFormat,
		contextFields: make(map[string]interface{}),
	}

	// Copy existing context fields
	for k, v := range l.contextFields {
		newLogger.contextFields[k] = v
	}

	// Add new field
	newLogger.contextFields[key] = value

	return newLogger
}

// WithFields adds multiple fields to a new logger instance
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:         l.level,
		output:        l.output,
		jsonFormat:    l.jsonFormat,
		contextFields: make(map[string]interface{}),
	}

	// Copy existing context fields
	for k, v := range l.contextFields {
		newLogger.contextFields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.contextFields[k] = v
	}

	return newLogger
}

// log writes a log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Get caller information
	pc, file, line, _ := runtime.Caller(2)
	fn := runtime.FuncForPC(pc)
	funcName := ""
	if fn != nil {
		funcName = fn.Name()
	}

	// Extract just the filename
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	// Merge context fields and provided fields
	allFields := make(map[string]interface{})
	for k, v := range l.contextFields {
		allFields[k] = v
	}
	for k, v := range fields {
		allFields[k] = v
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Fields:    allFields,
		File:      file,
		Line:      line,
		Function:  funcName,
	}

	if l.jsonFormat {
		l.writeJSON(entry)
	} else {
		l.writeText(entry)
	}
}

// writeJSON writes a log entry in JSON format
func (l *Logger) writeJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.output, "ERROR: Failed to marshal log entry: %v\n", err)
		return
	}

	fmt.Fprintln(l.output, string(data))
}

// writeText writes a log entry in human-readable format
func (l *Logger) writeText(entry LogEntry) {
	var sb strings.Builder

	// Timestamp and level
	sb.WriteString(fmt.Sprintf("[%s] %5s ", entry.Timestamp, entry.Level))

	// Message
	sb.WriteString(entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		sb.WriteString(" | ")
		first := true
		for k, v := range entry.Fields {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
	}

	// Location (only for errors and above)
	if entry.Level == "ERROR" || entry.Level == "FATAL" {
		sb.WriteString(fmt.Sprintf(" (%s:%d)", entry.File, entry.Line))
	}

	sb.WriteString("\n")
	fmt.Fprint(l.output, sb.String())
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message, nil)
}

// DebugFields logs a debug message with fields
func (l *Logger) DebugFields(message string, fields map[string]interface{}) {
	l.log(LogLevelDebug, message, fields)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, nil)
}

// InfoFields logs an info message with fields
func (l *Logger) InfoFields(message string, fields map[string]interface{}) {
	l.log(LogLevelInfo, message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, nil)
}

// WarnFields logs a warning message with fields
func (l *Logger) WarnFields(message string, fields map[string]interface{}) {
	l.log(LogLevelWarn, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(LogLevelError, message, nil)
}

// ErrorFields logs an error message with fields
func (l *Logger) ErrorFields(message string, fields map[string]interface{}) {
	l.log(LogLevelError, message, fields)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string) {
	l.log(LogLevelFatal, message, nil)
	os.Exit(1)
}

// FatalFields logs a fatal message with fields and exits
func (l *Logger) FatalFields(message string, fields map[string]interface{}) {
	l.log(LogLevelFatal, message, fields)
	os.Exit(1)
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.level
}

// Convenience functions for global logger

// Debug logs a debug message to the global logger
func LogDebug(message string) {
	GetLogger().Debug(message)
}

// Info logs an info message to the global logger
func LogInfo(message string) {
	GetLogger().Info(message)
}

// Warn logs a warning message to the global logger
func LogWarn(message string) {
	GetLogger().Warn(message)
}

// LogError logs an error message to the global logger
func LogError(message string) {
	GetLogger().Error(message)
}

// LogFatal logs a fatal message to the global logger and exits
func LogFatal(message string) {
	GetLogger().Fatal(message)
}

// WithField returns a logger with a context field
func WithField(key string, value interface{}) *Logger {
	return GetLogger().WithField(key, value)
}

// WithFields returns a logger with context fields
func WithFields(fields map[string]interface{}) *Logger {
	return GetLogger().WithFields(fields)
}

// ParseLogLevel parses a log level from a string
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}
