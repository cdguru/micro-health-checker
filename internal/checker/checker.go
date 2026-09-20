package checker

import (
	"context"
	"fmt"

	"github.com/cdguru/micro-health-checker/internal/config"
)

type Checker interface {
	Check(context.Context) error
	Close() error
}

type Registry struct{}

func NewRegistry() Registry { return Registry{} }

func (Registry) Build(cfg config.CheckConfig) (Checker, error) {
	switch cfg.Type {
	case "tcp":
		return newTCP(cfg.TCP), nil
	case "http":
		return newHTTP(cfg.HTTP), nil
	case "postgres":
		return newPostgres(cfg.Postgres)
	default:
		return nil, fmt.Errorf("unsupported check type %q", cfg.Type)
	}
}
