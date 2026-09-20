package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cdguru/micro-health-checker/internal/config"
	appmetrics "github.com/cdguru/micro-health-checker/internal/metrics"
	"github.com/cdguru/micro-health-checker/internal/scheduler"
	"github.com/cdguru/micro-health-checker/internal/storage"
)

func TestLiveHealthEndpointReturnsTargetStatus(t *testing.T) {
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

	metrics := appmetrics.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sched := scheduler.New(ctx, store, metrics, logger)
	defer sched.Stop()
	check := config.CheckConfig{
		ID: "service", Name: "Service", Type: "tcp",
		Interval: config.Duration{Duration: time.Hour}, Timeout: config.Duration{Duration: time.Second},
		TCP: &config.TCPConfig{Address: listener.Addr().String()},
	}
	if err := sched.Apply([]config.CheckConfig{check}); err != nil {
		t.Fatal(err)
	}
	api := New(Options{Address: ":0", ReadTimeout: time.Second, WriteTimeout: 2 * time.Second}, sched, store, metrics, logger)
	target := httptest.NewServer(api.Handler())
	defer target.Close()

	request, _ := http.NewRequest(http.MethodHead, target.URL+"/health/service", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("healthy status = %d, want 200", response.StatusCode)
	}

	listener.Close()
	response, err = http.Get(target.URL + "/health/service")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unhealthy status = %d, want 503", response.StatusCode)
	}
}
