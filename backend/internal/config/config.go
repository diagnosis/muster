package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the muster server.
type Config struct {
	Database DatabaseConfig
}

// DatabaseConfig holds connection settings and pool tuning for Postgres.
type DatabaseConfig struct {
	DSN               string
	MinConns          int32
	MaxConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
}

// Load reads configuration from the environment, applying defaults for
// optional values. It returns an error if a required variable is missing
// or any present value fails to parse.
func Load() (*Config, error) {
	dsn := getEnv("DATABASE_URL", "")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	var err error
	var minConns, maxConns int32
	var maxConnLifeTime, maxConnIdleTime, healthCheckPeriod, connectTimeout time.Duration

	minConns, err = getEnvInt32("DB_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}
	maxConns, err = getEnvInt32("DB_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	maxConnLifeTime, err = getEnvDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour)
	if err != nil {
		return nil, err
	}
	maxConnIdleTime, err = getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute)
	if err != nil {
		return nil, err
	}
	healthCheckPeriod, err = getEnvDuration("DB_HEALTH_CHECK_PERIOD", 1*time.Minute)
	if err != nil {
		return nil, err
	}
	connectTimeout, err = getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &Config{Database: DatabaseConfig{
		DSN:               dsn,
		MinConns:          minConns,
		MaxConns:          maxConns,
		MaxConnLifetime:   maxConnLifeTime,
		MaxConnIdleTime:   maxConnIdleTime,
		HealthCheckPeriod: healthCheckPeriod,
		ConnectTimeout:    connectTimeout,
	}}, nil
}

func getEnv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getEnvInt32(key string, def int32) (int32, error) {
	val := os.Getenv(key)
	if val == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid value %q: %w", key, val, err)
	}
	return int32(n), nil
}
func getEnvDuration(key string, def time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return def, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid value %q: %w", key, val, err)
	}
	return d, nil
}
