package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/cdguru/micro-health-checker/internal/checker"
	"github.com/cdguru/micro-health-checker/internal/config"
	appmetrics "github.com/cdguru/micro-health-checker/internal/metrics"
	"github.com/cdguru/micro-health-checker/internal/storage"
)

var (
	ErrNotFound = errors.New("check not found")
	ErrDisabled = errors.New("check is disabled")
)

type Result struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	Status              string     `json:"status"`
	Success             bool       `json:"success"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	DurationMS          float64    `json:"duration_ms"`
	Error               string     `json:"error,omitempty"`
	LastChangedAt       *time.Time `json:"last_changed_at,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
}

type Summary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Error    int `json:"error"`
	Unknown  int `json:"unknown"`
	Disabled int `json:"disabled"`
}

type Scheduler struct {
	rootCtx  context.Context
	registry checker.Registry
	store    storage.Store
	metrics  *appmetrics.Metrics
	logger   *slog.Logger

	applyMu sync.Mutex
	mu      sync.RWMutex
	configs map[string]config.CheckConfig
	states  map[string]Result
	order   []string
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

func New(ctx context.Context, store storage.Store, metrics *appmetrics.Metrics, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		rootCtx:  ctx,
		registry: checker.NewRegistry(),
		store:    store,
		metrics:  metrics,
		logger:   logger,
		configs:  make(map[string]config.CheckConfig),
		states:   make(map[string]Result),
	}
}

func (s *Scheduler) Apply(checks []config.CheckConfig) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	type prepared struct {
		cfg     config.CheckConfig
		checker checker.Checker
	}
	workers := make([]prepared, 0, len(checks))
	for _, cfg := range checks {
		if !cfg.IsEnabled() {
			continue
		}
		instance, err := s.registry.Build(cfg)
		if err != nil {
			for _, worker := range workers {
				_ = worker.checker.Close()
			}
			return fmt.Errorf("initialize check %q: %w", cfg.ID, err)
		}
		workers = append(workers, prepared{cfg: cfg, checker: instance})
	}

	if s.cancel != nil {
		s.cancel()
		s.wg.Wait()
	}

	configs := make(map[string]config.CheckConfig, len(checks))
	states := make(map[string]Result, len(checks))
	order := make([]string, 0, len(checks))
	s.mu.RLock()
	for _, cfg := range checks {
		configs[cfg.ID] = cfg
		order = append(order, cfg.ID)
		if previous, ok := s.states[cfg.ID]; ok && cfg.IsEnabled() {
			previous.Name, previous.Type = cfg.Name, cfg.Type
			states[cfg.ID] = previous
		} else if !cfg.IsEnabled() {
			states[cfg.ID] = Result{ID: cfg.ID, Name: cfg.Name, Type: cfg.Type, Status: "disabled"}
		} else {
			states[cfg.ID] = Result{ID: cfg.ID, Name: cfg.Name, Type: cfg.Type, Status: "unknown"}
		}
	}
	s.mu.RUnlock()

	ctx, cancel := context.WithCancel(s.rootCtx)
	wg := &sync.WaitGroup{}
	s.mu.Lock()
	s.configs, s.states, s.order = configs, states, order
	s.cancel, s.wg = cancel, wg
	s.mu.Unlock()

	for _, worker := range workers {
		wg.Add(1)
		go s.runWorker(ctx, wg, worker.cfg, worker.checker)
	}
	return nil
}

func (s *Scheduler) runWorker(ctx context.Context, wg *sync.WaitGroup, cfg config.CheckConfig, instance checker.Checker) {
	defer wg.Done()
	defer instance.Close() //nolint:errcheck // shutdown path

	s.execute(ctx, cfg, instance)
	ticker := time.NewTicker(cfg.Interval.Duration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.execute(ctx, cfg, instance)
		}
	}
}

func (s *Scheduler) RunNow(ctx context.Context, id string) (Result, error) {
	s.mu.RLock()
	cfg, ok := s.configs[id]
	s.mu.RUnlock()
	if !ok {
		return Result{}, ErrNotFound
	}
	if !cfg.IsEnabled() {
		return Result{}, ErrDisabled
	}
	instance, err := s.registry.Build(cfg)
	if err != nil {
		return Result{}, fmt.Errorf("initialize check: %w", err)
	}
	defer instance.Close() //nolint:errcheck // short-lived checker
	return s.execute(ctx, cfg, instance), nil
}

func (s *Scheduler) execute(parent context.Context, cfg config.CheckConfig, instance checker.Checker) Result {
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout.Duration)
	started := time.Now().UTC()
	err := instance.Check(ctx)
	duration := time.Since(started)
	cancel()

	result := Result{
		ID:         cfg.ID,
		Name:       cfg.Name,
		Type:       cfg.Type,
		Status:     "ok",
		Success:    err == nil,
		StartedAt:  &started,
		DurationMS: float64(duration.Microseconds()) / 1000,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
	}

	s.mu.Lock()
	previous := s.states[cfg.ID]
	if previous.Status != result.Status || previous.LastChangedAt == nil {
		result.LastChangedAt = &started
	} else {
		result.LastChangedAt = previous.LastChangedAt
	}
	if result.Success {
		result.ConsecutiveFailures = 0
	} else {
		result.ConsecutiveFailures = previous.ConsecutiveFailures + 1
	}
	s.states[cfg.ID] = result
	s.mu.Unlock()

	s.metrics.Observe(cfg.ID, cfg.Type, result.Success, duration, started)
	recordCtx, recordCancel := context.WithTimeout(context.Background(), 3*time.Second)
	recordErr := s.store.Record(recordCtx, storage.Record{
		CheckID: cfg.ID, Name: cfg.Name, Type: cfg.Type, Success: result.Success,
		StartedAt: started, Duration: result.DurationMS, Error: result.Error,
	})
	recordCancel()
	if recordErr != nil {
		s.logger.Error("could not persist check result", "check_id", cfg.ID, "error", recordErr)
	}
	if err != nil {
		s.logger.Warn("health check failed", "check_id", cfg.ID, "type", cfg.Type, "duration_ms", result.DurationMS, "error", err)
	} else {
		s.logger.Debug("health check succeeded", "check_id", cfg.ID, "type", cfg.Type, "duration_ms", result.DurationMS)
	}
	return result
}

func (s *Scheduler) List() ([]Result, Summary) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]Result, 0, len(s.order))
	var summary Summary
	for _, id := range s.order {
		result := s.states[id]
		results = append(results, result)
		summary.Total++
		switch result.Status {
		case "ok":
			summary.OK++
		case "error":
			summary.Error++
		case "disabled":
			summary.Disabled++
		default:
			summary.Unknown++
		}
	}
	return results, summary
}

func (s *Scheduler) Get(id string) (Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, ok := s.states[id]
	return result, ok
}

func (s *Scheduler) IDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := append([]string(nil), s.order...)
	sort.Strings(ids)
	return ids
}

func (s *Scheduler) Stop() {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.wg.Wait()
	}
}
