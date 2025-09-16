package forge_connect

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// consoleLog show service logs
func consoleLog(logType, fmtMessage string, args ...interface{}) {
	showMessage := fmt.Sprintf(fmtMessage, args...)
	logTime := time.Now().Format(time.RFC3339)
	fmt.Println(fmt.Sprintf("%s[%s] \033[0;33m \033[0m %s", logTime, logType, showMessage))
}

// LogLevel defines the severity levels for logging.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// LoggerConfig holds configuration for the logger.
type LoggerConfig struct {
	Level      LogLevel
	Format     string // "json" or "text"
	Output     *os.File
	CallerInfo bool
}

// Logger is the main logging structure.
type Logger struct {
	config LoggerConfig
}

// NewLogger creates a new Logger instance with the given configuration.
func NewLogger(config LoggerConfig) *Logger {
	return &Logger{config: config}
}

// Log prints a log message based on the configured format and level.
func (l *Logger) Log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.config.Level {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	logData := map[string]interface{}{
		"timestamp": timestamp,
		"level":     levelToString(level),
		"message":   message,
	}

	if l.config.CallerInfo {
		logData["caller"] = getCallerInfo()
	}

	for k, v := range fields {
		logData[k] = v
	}

	switch l.config.Format {
	case "json":
		jsonData, err := json.Marshal(logData)
		if err != nil {
			log.Printf("Failed to marshal log data: %v", err)
			return
		}
		fmt.Println(l.config.Output, string(jsonData))
	case "text":
		fmt.Println(l.config.Output, fmt.Sprintf("[%s] %s: %s", timestamp, levelToString(level), message))
		if l.config.CallerInfo {
			fmt.Println(l.config.Output, fmt.Sprintf(" (caller: %s)", getCallerInfo()))
		}
		fmt.Println(l.config.Output)
	}
}

// levelToString converts LogLevel to a string representation.
func levelToString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// getCallerInfo retrieves the caller's file and line number.
func getCallerInfo() string {
	// This is a simplified version; you may need to adjust based on your needs.
	return "<caller-info>"
}
