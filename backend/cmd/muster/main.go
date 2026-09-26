package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diagnosis/go-toolkit/v3/logger"
	"github.com/diagnosis/go-toolkit/v3/mailer"
	"github.com/diagnosis/go-toolkit/v3/secure"
	"github.com/diagnosis/muster/internal/api"
	"github.com/diagnosis/muster/internal/authtoken"
	"github.com/diagnosis/muster/internal/config"
	"github.com/diagnosis/muster/internal/events"
	"github.com/diagnosis/muster/internal/hiker"
	"github.com/diagnosis/muster/internal/message"
	"github.com/diagnosis/muster/internal/notification"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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
	tokenStore := postgres.NewAuthTokenStore(pool)
	outingsStore := postgres.NewOutingStore(pool)
	notificationStore := postgres.NewNotificationStore(pool)
	messageStore := postgres.NewMessageStore(pool)
	tokenService := authtoken.NewService(tokenStore)

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
	var m mailer.Mailer
	if cfg.Resend.APIKey == "" {
		logger.Warn(ctx, "mail disabled: RESEND_API_KEY not set")
		m = noopMailer{}
	} else {
		m = mailer.NewResendMailer(cfg.Resend.APIKey, cfg.Resend.EmailFrom)
	}
	hikerServiceConfig := hiker.ServiceConfig{
		Store:     hikerStore,
		Token:     tokenService,
		Mail:      m,
		JWT:       signer,
		BaseURL:   cfg.App.BaseURL,
		VerifyTTL: 24 * time.Hour,
	}
	hikers := hiker.NewService(hikerServiceConfig)

	dispatcher := notification.NewDispatcher(notificationStore, m, cfg.App.DispatcherInterval, cfg.App.BaseURL)
	hub := events.NewHub()

	messages := message.NewService(messageStore, hub)
	outings := outing.NewService(outingsStore, notificationStore, hub)
	srv := api.NewServer(cfg, hikers, signer, outings, notificationStore, hub, messages)

	go dispatcher.Run(ctx)
	server := &http.Server{
		Addr:              cfg.App.Host + ":" + cfg.App.Port,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(ctx, "muster listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("http server: %w", err)
		}
	}()
	select {
	case err := <-serverErrors:
		return err

	case <-ctx.Done():
		logger.Info(context.Background(), "muster shutting down gracefully...")

		shutDownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelShutdown()
		hub.CloseAll()
		if err := server.Shutdown(shutDownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	logger.Info(context.Background(), "muster stopped cleanly")
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

type noopMailer struct{}

func (noopMailer) Send(context.Context, []string, string, string) error { return nil }

var _ mailer.Mailer = noopMailer{}
