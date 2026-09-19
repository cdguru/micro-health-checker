package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultAddress   = ":8080"
	defaultInterval  = 30 * time.Second
	defaultTimeout   = 5 * time.Second
	defaultRetention = 30 * 24 * time.Hour
)

type Duration struct{ time.Duration }

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	parsed, err := parseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", node.Value, err)
	}
	d.Duration = parsed
	return nil
}

func parseDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(value, "d"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	return time.ParseDuration(value)
}

func (d Duration) MarshalYAML() (any, error) { return d.String(), nil }

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Checks    []CheckConfig   `yaml:"checks"`
}

type ServerConfig struct {
	Address              string   `yaml:"address"`
	ReadTimeout          Duration `yaml:"read_timeout"`
	WriteTimeout         Duration `yaml:"write_timeout"`
	EnableReloadEndpoint bool     `yaml:"enable_reload_endpoint"`
}

type SchedulerConfig struct {
	DefaultInterval Duration `yaml:"default_interval"`
	DefaultTimeout  Duration `yaml:"default_timeout"`
}

type StorageConfig struct {
	Type      string          `yaml:"type"`
	Retention Duration        `yaml:"retention"`
	SQLite    SQLiteConfig    `yaml:"sqlite"`
	Postgres  PostgresStorage `yaml:"postgres"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type PostgresStorage struct {
	DSN string `yaml:"dsn"`
}

type CheckConfig struct {
	ID       string          `yaml:"id"`
	Name     string          `yaml:"name"`
	Type     string          `yaml:"type"`
	Enabled  *bool           `yaml:"enabled,omitempty"`
	Interval Duration        `yaml:"interval,omitempty"`
	Timeout  Duration        `yaml:"timeout,omitempty"`
	TCP      *TCPConfig      `yaml:"tcp,omitempty"`
	HTTP     *HTTPConfig     `yaml:"http,omitempty"`
	Postgres *PostgresConfig `yaml:"postgres,omitempty"`
}

type TCPConfig struct {
	Address string `yaml:"address"`
}

type HTTPConfig struct {
	URL                string            `yaml:"url"`
	Method             string            `yaml:"method,omitempty"`
	ExpectedStatus     []int             `yaml:"expected_status,omitempty"`
	BodyContains       string            `yaml:"body_contains,omitempty"`
	Headers            map[string]string `yaml:"headers,omitempty"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify,omitempty"`
	FollowRedirects    *bool             `yaml:"follow_redirects,omitempty"`
}

type PostgresConfig struct {
	DSN   string `yaml:"dsn"`
	Query string `yaml:"query,omitempty"`
}

func (c CheckConfig) IsEnabled() bool { return c.Enabled == nil || *c.Enabled }

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	expanded, err := expandEnvironment(string(raw))
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	dec := yaml.NewDecoder(strings.NewReader(expanded))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Address == "" {
		cfg.Server.Address = defaultAddress
	}
	if cfg.Server.ReadTimeout.Duration == 0 {
		cfg.Server.ReadTimeout.Duration = 5 * time.Second
	}
	if cfg.Server.WriteTimeout.Duration == 0 {
		cfg.Server.WriteTimeout.Duration = 10 * time.Second
	}
	if cfg.Scheduler.DefaultInterval.Duration == 0 {
		cfg.Scheduler.DefaultInterval.Duration = defaultInterval
	}
	if cfg.Scheduler.DefaultTimeout.Duration == 0 {
		cfg.Scheduler.DefaultTimeout.Duration = defaultTimeout
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "sqlite"
	}
	cfg.Storage.Type = strings.ToLower(cfg.Storage.Type)
	if cfg.Storage.Retention.Duration == 0 {
		cfg.Storage.Retention.Duration = defaultRetention
	}
	if cfg.Storage.SQLite.Path == "" {
		cfg.Storage.SQLite.Path = "./data/micro-health-checker.db"
	}

	for i := range cfg.Checks {
		check := &cfg.Checks[i]
		check.Type = strings.ToLower(check.Type)
		if check.Name == "" {
			check.Name = check.ID
		}
		if check.Interval.Duration == 0 {
			check.Interval = cfg.Scheduler.DefaultInterval
		}
		if check.Timeout.Duration == 0 {
			check.Timeout = cfg.Scheduler.DefaultTimeout
		}
		if check.HTTP != nil {
			if check.HTTP.Method == "" {
				check.HTTP.Method = "GET"
			}
			check.HTTP.Method = strings.ToUpper(check.HTTP.Method)
		}
		if check.Postgres != nil && strings.TrimSpace(check.Postgres.Query) == "" {
			check.Postgres.Query = "SELECT 1"
		}
	}
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func (c Config) Validate() error {
	if c.Scheduler.DefaultInterval.Duration <= 0 || c.Scheduler.DefaultTimeout.Duration <= 0 {
		return errors.New("scheduler durations must be positive")
	}
	if c.Storage.Retention.Duration <= 0 {
		return errors.New("storage.retention must be positive")
	}
	switch c.Storage.Type {
	case "sqlite":
		if strings.TrimSpace(c.Storage.SQLite.Path) == "" {
			return errors.New("storage.sqlite.path is required")
		}
	case "postgres":
		if strings.TrimSpace(c.Storage.Postgres.DSN) == "" {
			return errors.New("storage.postgres.dsn is required")
		}
	default:
		return fmt.Errorf("unsupported storage type %q", c.Storage.Type)
	}

	seen := make(map[string]struct{}, len(c.Checks))
	for i, check := range c.Checks {
		prefix := fmt.Sprintf("checks[%d]", i)
		if !idPattern.MatchString(check.ID) {
			return fmt.Errorf("%s.id must match %s", prefix, idPattern.String())
		}
		if _, ok := seen[check.ID]; ok {
			return fmt.Errorf("duplicate check id %q", check.ID)
		}
		seen[check.ID] = struct{}{}
		if check.Interval.Duration <= 0 || check.Timeout.Duration <= 0 {
			return fmt.Errorf("%s interval and timeout must be positive", prefix)
		}
		switch check.Type {
		case "tcp":
			if check.TCP == nil || strings.TrimSpace(check.TCP.Address) == "" {
				return fmt.Errorf("%s.tcp.address is required", prefix)
			}
			if _, _, err := net.SplitHostPort(check.TCP.Address); err != nil {
				return fmt.Errorf("%s.tcp.address must be host:port: %w", prefix, err)
			}
		case "http":
			if check.HTTP == nil || strings.TrimSpace(check.HTTP.URL) == "" {
				return fmt.Errorf("%s.http.url is required", prefix)
			}
			parsedURL, err := url.ParseRequestURI(check.HTTP.URL)
			if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
				return fmt.Errorf("%s.http.url must be an absolute HTTP or HTTPS URL", prefix)
			}
			for _, status := range check.HTTP.ExpectedStatus {
				if status < 100 || status > 599 {
					return fmt.Errorf("%s.http.expected_status contains invalid status %d", prefix, status)
				}
			}
		case "postgres":
			if check.Postgres == nil || strings.TrimSpace(check.Postgres.DSN) == "" {
				return fmt.Errorf("%s.postgres.dsn is required", prefix)
			}
		default:
			return fmt.Errorf("%s.type %q is unsupported", prefix, check.Type)
		}
	}
	return nil
}

func EnsureSQLiteDirectory(cfg Config) error {
	if cfg.Storage.Type != "sqlite" || cfg.Storage.SQLite.Path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(cfg.Storage.SQLite.Path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create sqlite directory: %w", err)
	}
	return nil
}

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandEnvironment(input string) (string, error) {
	var missing []string
	result := envPattern.ReplaceAllStringFunc(input, func(match string) string {
		name := envPattern.FindStringSubmatch(match)[1]
		value, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return value
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("required environment variables are unset: %s", strings.Join(missing, ", "))
	}
	return result, nil
}
