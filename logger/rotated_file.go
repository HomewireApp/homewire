package logger

import (
	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/gookit/slog/rotatefile"
	"github.com/ztrue/tracerr"
)

type RotatePeriod int

const (
	RotateEverySecond    RotatePeriod = 1
	RotateEveryMinute    RotatePeriod = 60 * RotateEverySecond
	RotateEvery5Minutes  RotatePeriod = 10 * RotateEveryMinute
	RotateEvery10Minutes RotatePeriod = 10 * RotateEveryMinute
	RotateEvery15Minutes RotatePeriod = 15 * RotateEveryMinute
	RotateEvery30Minutes RotatePeriod = 30 * RotateEveryMinute
	RotateEveryHour      RotatePeriod = 60 * RotateEveryMinute
	RotateEveryDay       RotatePeriod = 24 * RotateEveryHour
	RotateEveryWeek      RotatePeriod = 7 * RotateEveryDay
	RotateEveryMonth     RotatePeriod = 30 * RotateEveryDay
)

type RotatedFileLoggerConfig struct {
	Level        LogLevel
	LogPath      string
	BufferSize   int
	MaxSize      uint64
	RotatePeriod RotatePeriod
	err          error
}

func DefaultRotatedFileConfig() *RotatedFileLoggerConfig {
	fileCfg := DefaultFileConfig()
	return &RotatedFileLoggerConfig{
		err:          fileCfg.err,
		Level:        defaultLevel,
		LogPath:      fileCfg.LogPath,
		BufferSize:   fileCfg.BufferSize,
		RotatePeriod: RotateEveryWeek,
		MaxSize:      64 * 1024 * 1024,
	}
}

func (r *RotatedFileLoggerConfig) WithLevel(level LogLevel) *RotatedFileLoggerConfig {
	r.Level = level
	return r
}

func (r *RotatedFileLoggerConfig) WithLogPath(logPath string) *RotatedFileLoggerConfig {
	r.LogPath = logPath
	return r
}

func (r *RotatedFileLoggerConfig) WithBufferSize(bufferSize int) *RotatedFileLoggerConfig {
	r.BufferSize = bufferSize
	return r
}

func (r *RotatedFileLoggerConfig) WithMaxSize(maxSize uint64) *RotatedFileLoggerConfig {
	r.MaxSize = maxSize
	return r
}

func (r *RotatedFileLoggerConfig) WithRotatePeriod(rotatePeriod RotatePeriod) *RotatedFileLoggerConfig {
	r.RotatePeriod = rotatePeriod
	return r
}

func (r *RotatedFileLoggerConfig) CreateLogger(name string) (*slog.Logger, error) {
	if r.err != nil {
		return nil, tracerr.Wrap(r.err)
	}

	handler, err := handler.NewRotateFileHandler(r.LogPath, rotatefile.RotateTime(r.RotatePeriod), handler.WithLogLevels(r.Level.intoSlogLevels()), handler.WithBuffSize(r.BufferSize))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	l := slog.NewWithName(name)
	l.PushHandler(handler)
	return l, nil
}
