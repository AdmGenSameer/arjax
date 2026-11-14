package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelError
)

// Logger provides structured logging for operations
type Logger struct {
	level  LogLevel
	output io.Writer
}

var globalLogger *Logger

func InitLogger(verbose bool) {
	logDir := os.ExpandEnv("$HOME/.arjax/logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "provider.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	
	var writer io.Writer = os.Stderr
	if err == nil {
		if verbose {
			writer = io.MultiWriter(os.Stderr, logFile)
		} else {
			writer = logFile
		}
	}

	level := LevelInfo
	if verbose {
		level = LevelDebug
	}

	globalLogger = &Logger{
		level:  level,
		output: writer,
	}
}

func LogDebug(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.level <= LevelDebug {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.output, "[DEBUG] %s - %s\n", time.Now().Format(time.RFC3339), msg)
	}
}

func LogInfo(format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.level <= LevelInfo {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.output, "[INFO] %s - %s\n", time.Now().Format(time.RFC3339), msg)
	}
}

func LogError(format string, args ...interface{}) {
	if globalLogger != nil {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(globalLogger.output, "[ERROR] %s - %s\n", time.Now().Format(time.RFC3339), msg)
	}
}
