package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cdguru/micro-health-checker/internal/config"
)

func TestSQLiteRecordHistoryAndPrune(t *testing.T) {
	cfg := config.StorageConfig{
		Type:   "sqlite",
		SQLite: config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "history.db")},
	}
	store, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	old := Record{CheckID: "pg", Name: "Postgres", Type: "postgres", Success: false, StartedAt: time.Now().Add(-48 * time.Hour), Duration: 12.4, Error: "down"}
	recent := Record{CheckID: "pg", Name: "Postgres", Type: "postgres", Success: true, StartedAt: time.Now(), Duration: 4.2}
	for _, record := range []Record{old, recent} {
		if err := store.Record(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	history, err := store.History(context.Background(), "pg", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || !history[0].Success {
		t.Fatalf("unexpected history: %#v", history)
	}
	removed, err := store.Prune(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
}
