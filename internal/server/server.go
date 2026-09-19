package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	appmetrics "github.com/christiandente/micro-health-checker/internal/metrics"
	"github.com/christiandente/micro-health-checker/internal/scheduler"
	"github.com/christiandente/micro-health-checker/internal/storage"
	"github.com/christiandente/micro-health-checker/internal/version"
)

//go:embed web/index.html
var assets embed.FS

type Options struct {
	Address              string
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	EnableReloadEndpoint bool
	Reload               func() error
}

type Server struct {
	http      *http.Server
	scheduler *scheduler.Scheduler
	store     storage.Store
	logger    *slog.Logger
}

func New(opts Options, scheduler *scheduler.Scheduler, store storage.Store, metrics *appmetrics.Metrics, logger *slog.Logger) *Server {
	s := &Server{scheduler: scheduler, store: store, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /api/v1/status", s.status)
	mux.HandleFunc("GET /api/v1/status/{id}", s.checkStatus)
	mux.HandleFunc("GET /api/v1/history/{id}", s.history)
	mux.HandleFunc("GET /health/{id}", s.health)
	mux.HandleFunc("HEAD /health/{id}", s.health)
	mux.HandleFunc("GET /-/healthy", s.healthy)
	mux.HandleFunc("GET /-/ready", s.ready)
	mux.Handle("GET /metrics", metrics.Handler())
	if opts.EnableReloadEndpoint && opts.Reload != nil {
		mux.HandleFunc("POST /-/reload", func(w http.ResponseWriter, _ *http.Request) {
			if err := opts.Reload(); err != nil {
				s.writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
	s.http = &http.Server{
		Addr:              opts.Address,
		Handler:           s.middleware(mux),
		ReadHeaderTimeout: opts.ReadTimeout,
		ReadTimeout:       opts.ReadTimeout,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }

func (s *Server) Handler() http.Handler { return s.http.Handler }

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := assets.ReadFile("web/index.html")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "web interface unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page) //nolint:errcheck // client disconnect
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	checks, summary := s.scheduler.List()
	s.writeJSON(w, http.StatusOK, map[string]any{
		"version": version.Version,
		"summary": summary,
		"checks":  checks,
	})
}

func (s *Server) checkStatus(w http.ResponseWriter, r *http.Request) {
	result, ok := s.scheduler.Get(r.PathValue("id"))
	if !ok {
		s.writeError(w, http.StatusNotFound, "check not found")
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	result, err := s.scheduler.RunNow(r.Context(), r.PathValue("id"))
	if err != nil {
		switch {
		case errors.Is(err, scheduler.ErrNotFound):
			s.writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, scheduler.ErrDisabled):
			s.writeError(w, http.StatusServiceUnavailable, err.Error())
		default:
			s.writeError(w, http.StatusInternalServerError, "check could not be executed")
		}
		return
	}
	status := http.StatusOK
	if !result.Success {
		status = http.StatusServiceUnavailable
	}
	if r.Method == http.MethodHead {
		w.Header().Set("X-MHC-Status", result.Status)
		w.Header().Set("X-MHC-Duration-Ms", strconv.FormatFloat(result.DurationMS, 'f', 3, 64))
		w.WriteHeader(status)
		return
	}
	s.writeJSON(w, status, result)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.scheduler.Get(id); !ok {
		s.writeError(w, http.StatusNotFound, "check not found")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			s.writeError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return
		}
		limit = parsed
	}
	records, err := s.store.History(r.Context(), id, limit)
	if err != nil {
		s.logger.Error("could not query check history", "check_id", id, "error", err)
		s.writeError(w, http.StatusInternalServerError, "history unavailable")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"check_id": id, "history": records})
}

func (s *Server) healthy(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		s.writeError(w, http.StatusServiceUnavailable, "storage unavailable")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("recovered HTTP panic", "error", fmt.Sprint(recovered))
				s.writeError(w, http.StatusInternalServerError, "internal server error")
			}
			s.logger.Debug("HTTP request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		s.logger.Debug("could not write HTTP response", "error", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": strings.TrimSpace(message)})
}
