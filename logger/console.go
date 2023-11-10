package logger

import (
	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
)

type ConsoleConfig struct {
	Level       LogLevel
	EnableColor bool
}

func DefaultConsoleConfig() *ConsoleConfig {
	return &ConsoleConfig{
		Level:       defaultLevel,
		EnableColor: true,
	}
}

func (c *ConsoleConfig) WithLevel(level LogLevel) *ConsoleConfig {
	c.Level = level
	return c
}

func (c *ConsoleConfig) WithEnableColor(enableColor bool) *ConsoleConfig {
	c.EnableColor = enableColor
	return c
}

func (c *ConsoleConfig) CreateLogger(name string) Logger {
	handler := handler.NewConsoleHandler(c.Level.intoSlogLevels())

	handler.TextFormatter().EnableColor = c.EnableColor

	l := slog.NewWithHandlers(handler)
	l.SetName(name)
	return &slogLogger{
		logger: l,
	}
}
