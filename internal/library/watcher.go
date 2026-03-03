package library

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors directories for filesystem changes and updates the library.
type Watcher struct {
	scanner *Scanner
	fsw     *fsnotify.Watcher
	dirs    []string

	mu     sync.Mutex
	paused bool
}

// NewWatcher creates a filesystem watcher backed by the given scanner.
func NewWatcher(scanner *Scanner) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{scanner: scanner, fsw: fsw}, nil
}

// Watch starts watching the given directories for changes.
// It blocks until the context is cancelled.
func (w *Watcher) Watch(ctx context.Context, dirs []string) error {
	w.dirs = dirs

	for _, dir := range dirs {
		if err := w.addRecursive(dir); err != nil {
			return err
		}
	}

	return w.loop(ctx)
}

// Pause stops processing events until Resume is called.
func (w *Watcher) Pause() {
	w.mu.Lock()
	w.paused = true
	w.mu.Unlock()
}

// Resume resumes processing events after a Pause.
func (w *Watcher) Resume() {
	w.mu.Lock()
	w.paused = false
	w.mu.Unlock()
}

// Close releases filesystem watcher resources.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}

func (w *Watcher) isPaused() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.paused
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if d.IsDir() {
			return w.fsw.Add(path)
		}
		return nil
	})
}

func (w *Watcher) loop(ctx context.Context) error {
	// Debounce: collect events and process them after a quiet period.
	const debounce = 100 * time.Millisecond
	timer := time.NewTimer(debounce)
	timer.Stop()

	pending := make(map[string]fsnotify.Op)

	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case event, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			if w.isPaused() {
				continue
			}
			pending[event.Name] = event.Op
			timer.Reset(debounce)

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			slog.Warn("watcher error", "err", err)

		case <-timer.C:
			w.processPending(ctx, pending)
			pending = make(map[string]fsnotify.Op)
		}
	}
}

func (w *Watcher) processPending(ctx context.Context, events map[string]fsnotify.Op) {
	for path, op := range events {
		if ctx.Err() != nil {
			return
		}

		switch {
		case op.Has(fsnotify.Remove) || op.Has(fsnotify.Rename):
			w.handleRemove(path)

		case op.Has(fsnotify.Create):
			w.handleCreate(ctx, path)

		case op.Has(fsnotify.Write):
			if isAudioFile(path) {
				w.handleUpdate(ctx, path)
			}
		}
	}
}

func (w *Watcher) handleCreate(ctx context.Context, path string) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	if fi.IsDir() {
		// Watch new subdirectory and scan it.
		if err := w.addRecursive(path); err != nil {
			slog.Warn("watch new dir", "path", path, "err", err)
		}
		if err := w.scanner.Scan(ctx, []string{path}, nil); err != nil {
			slog.Warn("scan new dir", "path", path, "err", err)
		}
		return
	}

	if isAudioFile(path) {
		if err := w.scanner.Scan(ctx, w.dirs, nil); err != nil {
			slog.Warn("rescan after create", "path", path, "err", err)
		}
	}
}

func (w *Watcher) handleRemove(path string) {
	if isAudioFile(path) {
		// Remove track and its FTS entry.
		_, err := w.scanner.db.Exec("DELETE FROM fts_tracks WHERE rowid IN (SELECT id FROM tracks WHERE file_path = ?)", path)
		if err != nil {
			slog.Warn("delete fts", "path", path, "err", err)
		}
		_, err = w.scanner.db.Exec("DELETE FROM tracks WHERE file_path = ?", path)
		if err != nil {
			slog.Warn("delete track", "path", path, "err", err)
		}
	}
}

func (w *Watcher) handleUpdate(ctx context.Context, path string) {
	// Re-scan the file to pick up tag changes.
	if err := w.scanner.Scan(ctx, w.dirs, nil); err != nil {
		slog.Warn("rescan after update", "path", path, "err", err)
	}
}
