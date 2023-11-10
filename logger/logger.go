package logger

import "github.com/gookit/slog"

type LogLevel string

const (
	LevelTrace LogLevel = "TRACE"
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
	LevelFatal LogLevel = "FATAL"
)

var defaultLevel = LevelDebug

type Logger interface {
	Trace(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	Close() error
}

func DefaultLevel() LogLevel {
	return defaultLevel
}

type slogLogger struct {
	logger *slog.Logger
}

func (l *LogLevel) intoSlogLevels() []slog.Level {
	switch *l {
	case LevelFatal:
		return []slog.Level{slog.PanicLevel, slog.FatalLevel}
	case LevelError:
		return []slog.Level{slog.PanicLevel, slog.FatalLevel, slog.ErrorLevel}
	case LevelWarn:
		return []slog.Level{slog.PanicLevel, slog.FatalLevel, slog.ErrorLevel, slog.WarnLevel}
	case LevelInfo:
		return []slog.Level{slog.PanicLevel, slog.FatalLevel, slog.ErrorLevel, slog.WarnLevel, slog.InfoLevel}
	case LevelDebug:
		return []slog.Level{slog.PanicLevel, slog.FatalLevel, slog.ErrorLevel, slog.WarnLevel, slog.InfoLevel, slog.DebugLevel}
	default:
		return slog.AllLevels
	}
}

func (l *slogLogger) Trace(msg string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Tracef(msg, args...)
	}
}

func (l *slogLogger) Debug(msg string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Debugf(msg, args...)
	}
}

func (l *slogLogger) Info(msg string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Infof(msg, args...)
	}
}

func (l *slogLogger) Warn(msg string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Warnf(msg, args...)
	}
}

func (l *slogLogger) Error(msg string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Errorf(msg, args...)
	}
}

func (l *slogLogger) Fatal(msg string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Fatalf(msg, args...)
	}
}

func (l *slogLogger) Close() error {
	if l.logger != nil {
		return l.logger.Close()
	}

	return nil
}
