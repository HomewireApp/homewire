package logger

import (
	"fmt"
	"os"

	"golang.org/x/exp/slog"
)

var logger *slog.Logger

func Init() {
	if logger != nil {
		return
	}

	opts := slog.HandlerOptions{
		Level: getDesiredLogLevel(),
	}

	h := slog.NewTextHandler(os.Stdout, &opts)
	logger = slog.New(h)
}

func Debug(msg string, args ...interface{}) {
	if logger != nil {
		logger.Debug(fmt.Sprintf(msg, args...))
	}
}

func Info(msg string, args ...interface{}) {
	if logger != nil {
		logger.Info(fmt.Sprintf(msg, args...))
	}
}

func Warn(msg string, args ...interface{}) {
	if logger != nil {
		logger.Warn(fmt.Sprintf(msg, args...))
	}
}

func Error(msg string, args ...interface{}) {
	if logger != nil {
		logger.Error(fmt.Sprintf(msg, args...))
	}
}

func getDesiredLogLevel() slog.Level {
	envLevel := os.Getenv("LOG_LEVEL")
	switch envLevel {
	case "ERROR":
		return slog.LevelError
	case "WARN":
		return slog.LevelWarn
	case "INFO":
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

type LoggerBridge struct{}

func NewBridge() *LoggerBridge {
	return &LoggerBridge{}
}

func (l *LoggerBridge) Print(message string) {
	Debug(message)
}

func (l *LoggerBridge) Trace(message string) {
	Debug(message)
}

func (l *LoggerBridge) Debug(message string) {
	Debug(message)
}

func (l *LoggerBridge) Info(message string) {
	Info(message)
}

func (l *LoggerBridge) Warning(message string) {
	Warn(message)
}

func (l *LoggerBridge) Error(message string) {
	Error(message)
}

func (l *LoggerBridge) Fatal(message string) {
	Error(message)
}
