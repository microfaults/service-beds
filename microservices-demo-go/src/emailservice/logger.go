package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel string

const (
	DEBUG   LogLevel = "DEBUG"
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
)

// LogEntry represents a structured JSON log entry
type LogEntry struct {
	Timestamp float64  `json:"timestamp"`
	Severity  string   `json:"severity"`
	Name      string   `json:"name"`
	Message   string   `json:"message"`
}

// JSONLogger provides structured JSON logging similar to Python's jsonlogger
type JSONLogger struct {
	name   string
	level  LogLevel
	writer io.Writer
}

// GetJSONLogger creates a new JSON logger that writes to stdout
// This is equivalent to the Python getJSONLogger function
func GetJSONLogger(name string) *JSONLogger {
	return &JSONLogger{
		name:   name,
		level:  INFO,
		writer: os.Stdout,
	}
}

// log writes a structured JSON log entry
func (l *JSONLogger) log(level LogLevel, message string) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: float64(time.Now().UnixNano()) / 1e9,
		Severity:  strings.ToUpper(string(level)),
		Name:      l.name,
		Message:   message,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		return
	}

	l.writer.Write(append(jsonBytes, '\n'))
}

// shouldLog checks if the message level meets the logger's threshold
func (l *JSONLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		DEBUG:   10,
		INFO:    20,
		WARNING: 30,
		ERROR:   40,
	}
	return levels[level] >= levels[l.level]
}

// SetLevel sets the minimum log level
func (l *JSONLogger) SetLevel(level LogLevel) {
	l.level = level
}

// Debug logs a debug message
func (l *JSONLogger) Debug(message string) {
	l.log(DEBUG, message)
}

// Info logs an info message
func (l *JSONLogger) Info(message string) {
	l.log(INFO, message)
}

// Warning logs a warning message
func (l *JSONLogger) Warning(message string) {
	l.log(WARNING, message)
}

// Error logs an error message
func (l *JSONLogger) Error(message string) {
	l.log(ERROR, message)
}