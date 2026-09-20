package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cdguru/micro-health-checker/internal/config"
	appmetrics "github.com/cdguru/micro-health-checker/internal/metrics"
	"github.com/cdguru/micro-health-checker/internal/scheduler"
	"github.com/cdguru/micro-health-checker/internal/server"
	"github.com/cdguru/micro-health-checker/internal/storage"
	"github.com/cdguru/micro-health-checker/internal/version"
)

func main() {
	configPath := flag.String("config", envOrDefault("MHC_CONFIG", "/etc/micro-health-checker/config.yml"), "path to YAML configuration")
	showVersion := flag.Bool("version", false, "print version and exit")
	healthcheckURL := flag.String("healthcheck-url", "", "perform one HTTP readiness check and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("micro-health-checker %s (commit=%s, built=%s)\n", version.Version, version.Commit, version.BuildDate)
		return
	}
	if *healthcheckURL != "" {
		if err := runHealthcheck(*healthcheckURL); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logger := newLogger()
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("could not load configuration", "error", err)
		os.Exit(1)
	}
	if err := config.EnsureSQLiteDirectory(cfg); err != nil {
		logger.Error("could not prepare storage", "error", err)
		os.Exit(1)
	}

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	store, err := storage.Open(rootCtx, cfg.Storage)
	if err != nil {
		logger.Error("could not initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close() //nolint:errcheck // process shutdown

	metrics := appmetrics.New()
	sched := scheduler.New(rootCtx, store, metrics, logger)
	if err := sched.Apply(cfg.Checks); err != nil {
		logger.Error("could not initialize checks", "error", err)
		os.Exit(1)
	}
	defer sched.Stop()

	var configMu sync.RWMutex
	currentConfig := cfg
	applyConfig := func(next config.Config) error {
		configMu.RLock()
		previous := currentConfig
		configMu.RUnlock()
		if immutableChanged(previous, next) {
			logger.Warn("server or storage configuration changed; those settings require a restart")
		}
		if err := sched.Apply(next.Checks); err != nil {
			metrics.ObserveReload(false)
			return err
		}
		configMu.Lock()
		currentConfig.Checks = next.Checks
		currentConfig.Scheduler = next.Scheduler
		configMu.Unlock()
		metrics.ObserveReload(true)
		return nil
	}
	reload := func() error {
		next, err := config.Load(*configPath)
		if err != nil {
			metrics.ObserveReload(false)
			return err
		}
		return applyConfig(next)
	}

	api := server.New(server.Options{
		Address:              cfg.Server.Address,
		ReadTimeout:          cfg.Server.ReadTimeout.Duration,
		WriteTimeout:         cfg.Server.WriteTimeout.Duration,
		EnableReloadEndpoint: cfg.Server.EnableReloadEndpoint,
		Reload:               reload,
	}, sched, store, metrics, logger)

	errCh := make(chan error, 2)
	go func() {
		logger.Info("micro-health-checker started", "version", version.Version, "address", cfg.Server.Address, "checks", len(cfg.Checks), "storage", cfg.Storage.Type)
		if err := api.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("HTTP server: %w", err)
		}
	}()
	go func() {
		watcher := config.NewWatcher(*configPath, logger)
		if err := watcher.Run(rootCtx, applyConfig); err != nil {
			errCh <- err
		}
	}()
	go retentionLoop(rootCtx, store, cfg.Storage.Retention.Duration, logger)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		logger.Error("service stopped unexpectedly", "error", err)
	}

	rootCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := api.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func retentionLoop(ctx context.Context, store storage.Store, retention time.Duration, logger *slog.Logger) {
	prune := func() {
		pruneCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		removed, err := store.Prune(pruneCtx, time.Now().Add(-retention))
		if err != nil {
			logger.Error("history retention failed", "error", err)
			return
		}
		if removed > 0 {
			logger.Info("expired history removed", "rows", removed)
		}
	}
	prune()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prune()
		}
	}
}

func immutableChanged(previous, next config.Config) bool {
	return previous.Server.Address != next.Server.Address ||
		previous.Server.ReadTimeout != next.Server.ReadTimeout ||
		previous.Server.WriteTimeout != next.Server.WriteTimeout ||
		previous.Storage.Type != next.Storage.Type ||
		previous.Storage.SQLite.Path != next.Storage.SQLite.Path ||
		previous.Storage.Postgres.DSN != next.Storage.Postgres.DSN
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("MHC_LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func runHealthcheck(url string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned HTTP %d", resp.StatusCode)
	}
	return nil
}
