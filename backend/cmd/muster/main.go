package main

import (
	"context"
	"fmt"
	"os"

	"github.com/diagnosis/go-toolkit/v2/logger"
	"github.com/diagnosis/muster/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		logger.Error(context.Background(), "muster failed to start", "err", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger.Init(cfg.App.Env)

	ctx := context.Background()
	pool, err := openPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	logger.Info(ctx, "muster connected")
	return nil
}

func openPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	dbConfig := cfg.Database
	pgxConfig, err := pgxpool.ParseConfig(dbConfig.DSN)
	if err != nil {
		return nil, err
	}
	pgxConfig.MinConns = dbConfig.MinConns
	pgxConfig.MaxConns = dbConfig.MaxConns
	pgxConfig.MaxConnIdleTime = dbConfig.MaxConnIdleTime
	pgxConfig.HealthCheckPeriod = dbConfig.HealthCheckPeriod
	pgxConfig.MaxConnLifetime = dbConfig.MaxConnLifetime
	pgxConfig.ConnConfig.ConnectTimeout = dbConfig.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)

	return pool, err
}
