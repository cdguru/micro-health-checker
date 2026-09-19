package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppliesDefaultsAndExpandsEnvironment(t *testing.T) {
	t.Setenv("TEST_PG_DSN", "postgres://user:secret@db:5432/app")
	path := filepath.Join(t.TempDir(), "config.yml")
	content := `
storage:
  type: sqlite
  retention: 30d
  sqlite:
    path: "` + filepath.Join(t.TempDir(), "mhc.db") + `"
checks:
  - id: postgres-prod
    type: postgres
    postgres:
      dsn: ${TEST_PG_DSN}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.Storage.Retention.Duration, 30*24*time.Hour; got != want {
		t.Fatalf("retention = %s, want %s", got, want)
	}
	if cfg.Checks[0].Name != "postgres-prod" || cfg.Checks[0].Postgres.Query != "SELECT 1" {
		t.Fatalf("defaults not applied: %#v", cfg.Checks[0])
	}
	if cfg.Checks[0].Postgres.DSN != "postgres://user:secret@db:5432/app" {
		t.Fatal("environment variable was not expanded")
	}
}

func TestLoadRejectsMissingEnvironmentVariable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	content := `storage: {type: sqlite, sqlite: {path: ":memory:"}}
checks:
  - id: pg
    type: postgres
    postgres: {dsn: "${DEFINITELY_UNSET_MHC_TEST}"}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() expected an error")
	}
}

func TestValidateRejectsDuplicateIDs(t *testing.T) {
	cfg := Config{
		Storage:   StorageConfig{Type: "sqlite", Retention: Duration{time.Hour}, SQLite: SQLiteConfig{Path: ":memory:"}},
		Scheduler: SchedulerConfig{DefaultInterval: Duration{time.Second}, DefaultTimeout: Duration{time.Second}},
		Checks: []CheckConfig{
			{ID: "same", Type: "tcp", Interval: Duration{time.Second}, Timeout: Duration{time.Second}, TCP: &TCPConfig{Address: "localhost:1"}},
			{ID: "same", Type: "tcp", Interval: Duration{time.Second}, Timeout: Duration{time.Second}, TCP: &TCPConfig{Address: "localhost:2"}},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() expected duplicate ID error")
	}
}
