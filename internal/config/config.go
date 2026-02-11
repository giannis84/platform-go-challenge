package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config.yaml"

// Config holds the application configuration.
type Config struct {
	APIPort    string `yaml:"api_port"`
	HealthPort string `yaml:"health_port"`

	// HTTP server timeouts (optional, defaults apply in server.go)
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`

	// JWT signing secret (env var only). When empty, only unsigned tokens
	// (alg=none) are accepted, this is suitable for local development.
	JWTSecret string `yaml:"-"`

	// Database configuration (env vars only — secrets must not live in config.yaml)
	DBHost     string `yaml:"-"`
	DBPort     string `yaml:"-"`
	DBUser     string `yaml:"-"`
	DBPassword string `yaml:"-"`
	DBName     string `yaml:"-"`
}

// Load reads configuration with the following precedence (highest wins):
//  1. Environment variables (API_PORT, HEALTH_PORT)
//  2. YAML config file (path from CONFIG_PATH env var, or "config.yaml")
//
// Database settings are loaded exclusively from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file %s: %w", path, err)
		}
	}

	if v := os.Getenv("API_PORT"); v != "" {
		cfg.APIPort = v
	}
	if v := os.Getenv("HEALTH_PORT"); v != "" {
		cfg.HealthPort = v
	}

	if cfg.APIPort == "" {
		return nil, fmt.Errorf("api_port is required (set via config file or API_PORT env var)")
	}
	if cfg.HealthPort == "" {
		return nil, fmt.Errorf("health_port is required (set via config file or HEALTH_PORT env var)")
	}

	// Database configuration from environment variables
	cfg.DBHost = os.Getenv("POSTGRES_HOST")
	cfg.DBPort = os.Getenv("POSTGRES_PORT")
	cfg.DBUser = os.Getenv("POSTGRES_USER")
	cfg.DBPassword = os.Getenv("POSTGRES_PASSWORD")
	cfg.DBName = os.Getenv("POSTGRES_DB")

	// JWT secret (optional — when empty, only unsigned tokens are accepted)
	cfg.JWTSecret = os.Getenv("JWT_SECRET")

	// HTTP server timeouts (optional — defaults apply in server.go if zero)
	if v := os.Getenv("READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ReadTimeout = d
		}
	}
	if v := os.Getenv("WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.WriteTimeout = d
		}
	}
	if v := os.Getenv("IDLE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.IdleTimeout = d
		}
	}

	if cfg.DBHost == "" {
		return nil, fmt.Errorf("POSTGRES_HOST env var is required")
	}
	if cfg.DBPort == "" {
		return nil, fmt.Errorf("POSTGRES_PORT env var is required")
	}
	if cfg.DBUser == "" {
		return nil, fmt.Errorf("POSTGRES_USER env var is required")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD env var is required")
	}
	if cfg.DBName == "" {
		return nil, fmt.Errorf("POSTGRES_DB env var is required")
	}

	return cfg, nil
}

// PostgresConnString returns a PostgreSQL connection string.
func (c *Config) PostgresConnString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

// APIAddr returns the listen address for the API server.
func (c *Config) APIAddr() string {
	return ":" + c.APIPort
}

// HealthAddr returns the listen address for the health check server.
func (c *Config) HealthAddr() string {
	return ":" + c.HealthPort
}
