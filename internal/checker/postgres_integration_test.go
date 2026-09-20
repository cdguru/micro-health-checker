package checker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cdguru/micro-health-checker/internal/config"
)

func TestPostgresCheckerIntegration(t *testing.T) {
	dsn := os.Getenv("MHC_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("MHC_TEST_POSTGRES_DSN is not set")
	}
	instance, err := newPostgres(&config.PostgresConfig{DSN: dsn, Query: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := instance.Check(ctx); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}
