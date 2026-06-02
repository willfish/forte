package logx

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mu       sync.Mutex
	logger   = slog.New(slog.NewTextHandler(io.Discard, nil))
	logFile  *os.File
	logPath  string
	level    = slog.LevelWarn
	stderrOn = true
)

// Configure sets the global log level and writers.
// Logs are appended to ~/.config/forte/forte.log (alongside crash.log).
// When stderrOn, the same records are also written to stderr if the level allows.
func Configure(levelName string) error {
	mu.Lock()
	defer mu.Unlock()

	level = ParseLevel(levelName)

	path, err := logFilePath()
	if err != nil {
		return err
	}
	logPath = path

	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	var writers []io.Writer
	if level < slog.LevelError+4 {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		logFile = f
		writers = append(writers, f)
	}

	if stderrOn && level < slog.LevelError+4 {
		writers = append(writers, os.Stderr)
	}

	var w = io.Discard
	if len(writers) == 1 {
		w = writers[0]
	} else if len(writers) > 1 {
		w = io.MultiWriter(writers...)
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	logger = slog.New(handler)
	slog.SetDefault(logger)
	log.SetOutput(&legacyWriter{logger: logger})
	log.SetFlags(0)

	return nil
}

// Logger returns the configured application logger.
func Logger() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	return logger
}

// SlogLevel returns the current minimum slog level.
func SlogLevel() slog.Level {
	mu.Lock()
	defer mu.Unlock()
	return level
}

// LevelString returns the canonical name of the active level.
func LevelString() string {
	return LevelName(SlogLevel())
}

// Path returns the log file path (empty until Configure succeeds).
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

func logFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "forte", "forte.log"), nil
}

type legacyWriter struct {
	logger *slog.Logger
}

func (w *legacyWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}
	w.logger.Warn(msg)
	return len(p), nil
}
