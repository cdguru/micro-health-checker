package scheduler

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/christiandente/micro-health-checker/internal/config"
	appmetrics "github.com/christiandente/micro-health-checker/internal/metrics"
	"github.com/christiandente/micro-health-checker/internal/storage"
)

func TestSchedulerRunsAndPersistsCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := storage.Open(ctx, config.StorageConfig{Type: "sqlite", SQLite: config.SQLiteConfig{Path: ":memory:"}})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	s := New(ctx, store, appmetrics.New(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer s.Stop()
	check := config.CheckConfig{
		ID: "tcp-test", Name: "TCP Test", Type: "tcp",
		Interval: config.Duration{Duration: time.Hour}, Timeout: config.Duration{Duration: time.Second},
		TCP: &config.TCPConfig{Address: listener.Addr().String()},
	}
	if err := s.Apply([]config.CheckConfig{check}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		result, ok := s.Get("tcp-test")
		if ok && result.Status == "ok" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduled check did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, err := s.RunNow(context.Background(), "tcp-test"); err != nil {
		t.Fatal(err)
	}
	history, err := store.History(context.Background(), "tcp-test", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) < 2 {
		t.Fatalf("history length = %d, want at least 2", len(history))
	}
}
