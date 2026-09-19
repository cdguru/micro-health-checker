package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry
	up       *prometheus.GaugeVec
	duration *prometheus.GaugeVec
	lastRun  *prometheus.GaugeVec
	total    *prometheus.CounterVec
	reloads  *prometheus.CounterVec
}

func New() *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		up: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "micro_health_checker",
			Name:      "check_up",
			Help:      "Whether the latest health check succeeded (1) or failed (0).",
		}, []string{"check_id", "check_type"}),
		duration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "micro_health_checker",
			Name:      "check_duration_seconds",
			Help:      "Duration of the latest health check in seconds.",
		}, []string{"check_id", "check_type"}),
		lastRun: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "micro_health_checker",
			Name:      "check_last_run_timestamp_seconds",
			Help:      "Unix timestamp of the latest health check.",
		}, []string{"check_id", "check_type"}),
		total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "micro_health_checker",
			Name:      "checks_total",
			Help:      "Total health checks by result.",
		}, []string{"check_id", "check_type", "result"}),
		reloads: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "micro_health_checker",
			Name:      "config_reloads_total",
			Help:      "Configuration reload attempts by result.",
		}, []string{"result"}),
	}
	registry.MustRegister(m.up, m.duration, m.lastRun, m.total, m.reloads)
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	return m
}

func (m *Metrics) Observe(checkID, checkType string, success bool, duration time.Duration, at time.Time) {
	value, result := 0.0, "error"
	if success {
		value, result = 1, "ok"
	}
	m.up.WithLabelValues(checkID, checkType).Set(value)
	m.duration.WithLabelValues(checkID, checkType).Set(duration.Seconds())
	m.lastRun.WithLabelValues(checkID, checkType).Set(float64(at.Unix()))
	m.total.WithLabelValues(checkID, checkType, result).Inc()
}

func (m *Metrics) ObserveReload(success bool) {
	result := "error"
	if success {
		result = "ok"
	}
	m.reloads.WithLabelValues(result).Inc()
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
