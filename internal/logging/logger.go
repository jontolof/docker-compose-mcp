package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
}

type Logger struct {
	level     Level
	output    io.Writer
	structured bool
	component string
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component,omitempty"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func NewLogger(component string, level Level, structured bool) *Logger {
	return &Logger{
		level:     level,
		output:    os.Stderr,
		structured: structured,
		component: component,
	}
}

func NewFileLogger(component string, level Level, structured bool, filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		level:     level,
		output:    file,
		structured: structured,
		component: component,
	}, nil
}

func (l *Logger) SetLevel(level Level) {
	l.level = level
}

func (l *Logger) SetOutput(output io.Writer) {
	l.output = output
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.log(DebugLevel, msg, fields...)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.log(InfoLevel, msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.log(WarnLevel, msg, fields...)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.log(ErrorLevel, msg, fields...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

func (l *Logger) log(level Level, msg string, fields ...map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     levelNames[level],
		Component: l.component,
		Message:   msg,
	}

	if len(fields) > 0 && fields[0] != nil {
		entry.Fields = fields[0]
	}

	var output string
	if l.structured {
		data, err := json.Marshal(entry)
		if err != nil {
			output = fmt.Sprintf("ERROR: Failed to marshal log entry: %v\n", err)
		} else {
			output = string(data) + "\n"
		}
	} else {
		var fieldsStr string
		if entry.Fields != nil && len(entry.Fields) > 0 {
			var parts []string
			for k, v := range entry.Fields {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			fieldsStr = " [" + strings.Join(parts, " ") + "]"
		}

		componentStr := ""
		if l.component != "" {
			componentStr = fmt.Sprintf("[%s] ", l.component)
		}

		output = fmt.Sprintf("%s [%s] %s%s%s\n", 
			entry.Timestamp, entry.Level, componentStr, entry.Message, fieldsStr)
	}

	l.output.Write([]byte(output))
}

func ParseLevel(levelStr string) Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DebugLevel
	case "INFO":
		return InfoLevel
	case "WARN", "WARNING":
		return WarnLevel
	case "ERROR":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

func GetLogLevel() Level {
	if level := os.Getenv("MCP_LOG_LEVEL"); level != "" {
		return ParseLevel(level)
	}
	return InfoLevel
}

func IsStructuredLogging() bool {
	return strings.ToLower(os.Getenv("MCP_LOG_FORMAT")) == "json"
}

type ContextLogger struct {
	*Logger
	fields map[string]interface{}
}

func (l *Logger) WithFields(fields map[string]interface{}) *ContextLogger {
	return &ContextLogger{
		Logger: l,
		fields: fields,
	}
}

func (cl *ContextLogger) Debug(msg string, fields ...map[string]interface{}) {
	cl.log(DebugLevel, msg, cl.mergeFields(fields...)...)
}

func (cl *ContextLogger) Info(msg string, fields ...map[string]interface{}) {
	cl.log(InfoLevel, msg, cl.mergeFields(fields...)...)
}

func (cl *ContextLogger) Warn(msg string, fields ...map[string]interface{}) {
	cl.log(WarnLevel, msg, cl.mergeFields(fields...)...)
}

func (cl *ContextLogger) Error(msg string, fields ...map[string]interface{}) {
	cl.log(ErrorLevel, msg, cl.mergeFields(fields...)...)
}

func (cl *ContextLogger) mergeFields(fields ...map[string]interface{}) []map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Add context fields first
	for k, v := range cl.fields {
		merged[k] = v
	}
	
	// Override with provided fields
	for _, fieldMap := range fields {
		if fieldMap != nil {
			for k, v := range fieldMap {
				merged[k] = v
			}
		}
	}
	
	return []map[string]interface{}{merged}
}