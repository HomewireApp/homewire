package logger

import (
	"os"
	"path"

	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/ztrue/tracerr"
)

type FileConfig struct {
	Level      LogLevel
	LogPath    string
	BufferSize int
	err        error
}

func DefaultFileConfig() *FileConfig {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return &FileConfig{err: tracerr.Wrap(err)}
	}

	logPath := path.Join(userHome, ".homewire", "homewire.log")
	return &FileConfig{
		Level:      defaultLevel,
		LogPath:    logPath,
		BufferSize: 32 * 1024,
	}
}

func (f *FileConfig) WithLevel(level LogLevel) *FileConfig {
	f.Level = level
	return f
}

func (f *FileConfig) WithLogPath(logPath string) *FileConfig {
	f.LogPath = logPath
	return f
}

func (f *FileConfig) WithBufferSize(bufferSize int) *FileConfig {
	f.BufferSize = bufferSize
	return f
}

func (f *FileConfig) CreateLogger(name string) (Logger, error) {
	if f.err != nil {
		return nil, tracerr.Wrap(f.err)
	}

	handler, err := handler.NewBuffFileHandler(f.LogPath, f.BufferSize, handler.WithLogLevels(f.Level.intoSlogLevels()))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	l := slog.NewWithHandlers(handler)
	l.SetName(name)
	return &slogLogger{logger: l}, nil
}
