package config

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	path     string
	logger   *slog.Logger
	debounce time.Duration
}

func NewWatcher(path string, logger *slog.Logger) *Watcher {
	return &Watcher{path: filepath.Clean(path), logger: logger, debounce: 300 * time.Millisecond}
}

func (w *Watcher) Run(ctx context.Context, onValid func(Config) error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create config watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(filepath.Dir(w.path)); err != nil {
		return fmt.Errorf("watch config directory: %w", err)
	}

	var timer *time.Timer
	var timerC <-chan time.Time
	reload := func() {
		cfg, err := Load(w.path)
		if err != nil {
			w.logger.Error("configuration reload rejected; keeping last known good configuration", "error", err)
			return
		}
		if err := onValid(cfg); err != nil {
			w.logger.Error("configuration reload could not be applied; keeping last known good configuration", "error", err)
			return
		}
		w.logger.Info("configuration reloaded", "checks", len(cfg.Checks))
	}

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return nil
		case err := <-watcher.Errors:
			if err != nil {
				w.logger.Warn("configuration watcher error", "error", err)
			}
		case event, ok := <-watcher.Events:
			if !ok {
				return errorsClosed("configuration watcher closed")
			}
			if filepath.Clean(event.Name) != w.path {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(w.debounce)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(w.debounce)
			}
			timerC = timer.C
		case <-timerC:
			timerC = nil
			reload()
		}
	}
}

type watcherClosedError string

func (e watcherClosedError) Error() string { return string(e) }

func errorsClosed(message string) error { return watcherClosedError(message) }
