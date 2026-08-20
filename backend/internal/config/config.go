package config

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the muster server.
type Config struct {
	App         *AppConfig
	Database    *DatabaseConfig
	JWT         *JWTConfig
	RateLimiter *RateLimiterConfig
}

// AppConfig holds app configurations. platform, env, port etc.
type AppConfig struct {
	Env            string
	Port           string
	Host           string
	CORSOrigins    []string
	CookieSameSite http.SameSite
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

// JWTConfig holds secrets and expirations
type JWTConfig struct {
	AccessSecret       string
	RefreshSecret      string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
	Audience           string
}

// RateLimiterConfig holds the per-IP request throttle: RPS is the
// sustained rate, Burst the short-term allowance. Env-overridable for
// e2e runs; defaults match production shape.
type RateLimiterConfig struct {
	RPS   int32
	Burst int32
}

// Load reads configuration from the environment, applying defaults for
// optional values. It returns an error if a required variable is missing
// or any present value fails to parse.
func Load() (*Config, error) {

	appConfig, err := loadAppConfig()
	if err != nil {
		return nil, err
	}

	databaseConfig, err := loadDatabaseConfig()
	if err != nil {
		return nil, err
	}

	jwtConfig, err := loadJWTConfig()
	if err != nil {
		return nil, err
	}

	rateLimiterConfig, err := loadRateLimitConfig()
	if err != nil {
		return nil, err
	}
	return &Config{App: appConfig, Database: databaseConfig, JWT: jwtConfig, RateLimiter: rateLimiterConfig}, nil
}

func loadAppConfig() (*AppConfig, error) {
	env := getEnv("APP_ENV", "dev")
	port := getEnv("APP_PORT", "8088")
	host := getEnv("APP_HOST", "")

	corsOrigins := []string{}
	corsRaw := getEnv("CORS_ORIGINS", "")
	if corsRaw != "" {
		for origin := range strings.SplitSeq(corsRaw, ",") {
			corsOrigins = append(corsOrigins, strings.TrimSpace(origin))
		}
	}

	cookieSameSite, err := parseSameSite("COOKIE_SAMESITE", "lax")
	if err != nil {
		return nil, err
	}
	if cookieSameSite == http.SameSiteNoneMode && env == "dev" {
		return nil, fmt.Errorf("COOKIE_SAMESITE=none requires non-dev APP_ENV (Secure cookies)")
	}

	return &AppConfig{
		Env:            env,
		Port:           port,
		Host:           host,
		CORSOrigins:    corsOrigins,
		CookieSameSite: cookieSameSite,
	}, nil
}

// loadDatabaseConfig loads database env variables
func loadDatabaseConfig() (*DatabaseConfig, error) {
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
	connectTimeout, err = getEnvDuration("DB_CONN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &DatabaseConfig{
		DSN:               dsn,
		MinConns:          minConns,
		MaxConns:          maxConns,
		MaxConnLifetime:   maxConnLifeTime,
		MaxConnIdleTime:   maxConnIdleTime,
		HealthCheckPeriod: healthCheckPeriod,
		ConnectTimeout:    connectTimeout,
	}, nil
}

func loadJWTConfig() (*JWTConfig, error) {
	accessSecret := getEnv("JWT_ACCESS_SECRET", "")
	if accessSecret == "" {
		return nil, fmt.Errorf("access secret cannot be empty or nil")
	}
	refreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	if refreshSecret == "" {
		return nil, fmt.Errorf("refresh secret cannot be empty or nil")
	}
	if accessSecret == refreshSecret {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must differ")
	}
	accessTokenExpiry, err := getEnvDuration("JWT_ACCESS_TOKEN_EXPIRY", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshTokenExpiry, err := getEnvDuration("JWT_REFRESH_TOKEN_EXPIRY", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	issuer := getEnv("JWT_ISSUER", "muster")
	audience := getEnv("JWT_AUDIENCE", "muster-api")
	return &JWTConfig{
		AccessSecret:       accessSecret,
		RefreshSecret:      refreshSecret,
		AccessTokenExpiry:  accessTokenExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
		Issuer:             issuer,
		Audience:           audience,
	}, nil
}

func loadRateLimitConfig() (*RateLimiterConfig, error) {
	rps, err := getEnvInt32("RATE_LIMIT_RPS", 10)
	if err != nil {
		return nil, err
	}
	burst, err := getEnvInt32("RATE_LIMIT_BURST", 20)
	if err != nil {
		return nil, err
	}
	return &RateLimiterConfig{
		RPS:   rps,
		Burst: burst,
	}, nil
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
func parseSameSite(key, def string) (http.SameSite, error) {
	val := strings.ToLower(strings.TrimSpace(getEnv(key, def)))
	switch val {
	case "lax":
		return http.SameSiteLaxMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	default:
		return 0, fmt.Errorf("%s: invalid value %q (want lax, none, or strict)", key, val)
	}
}
