package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/diagnosis/go-toolkit/v2/logger"
	"github.com/diagnosis/go-toolkit/v2/secure"
	"github.com/diagnosis/muster/internal/api"
	"github.com/diagnosis/muster/internal/config"
	"github.com/diagnosis/muster/internal/hiker"
	"github.com/diagnosis/muster/internal/outing"
	"github.com/diagnosis/muster/internal/postgres"
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

	if err = pool.Ping(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	logger.Info(ctx, "muster connected")

	hikerStore := postgres.NewHikerStore(pool)
	outingsStore := postgres.NewOutingStore(pool)

	signer, err := secure.NewJWTSigner(secure.JWTConfig{
		AccessSecret:       cfg.JWT.AccessSecret,
		RefreshSecret:      cfg.JWT.RefreshSecret,
		AccessTokenExpiry:  cfg.JWT.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.JWT.RefreshTokenExpiry,
		Issuer:             cfg.JWT.Issuer,
		Audience:           cfg.JWT.Audience,
		Leeway:             0,
	})
	if err != nil {
		return fmt.Errorf("signer err: %w", err)
	}

	hikers := hiker.NewService(hikerStore, signer)
	outings := outing.NewService(outingsStore)
	srv := api.NewServer(cfg, hikers, signer, outings)

	logger.Info(ctx, "muster listening", "port", cfg.App.Port)
	return (&http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}).ListenAndServe()
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
