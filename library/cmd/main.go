package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/Impervguin/ds-lab2/library/internal/handler/health"
	handlerlibrary "github.com/Impervguin/ds-lab2/library/internal/handler/library"
	"github.com/Impervguin/ds-lab2/library/internal/logger"
	"github.com/Impervguin/ds-lab2/library/internal/migrations"
	"github.com/Impervguin/ds-lab2/library/internal/repository/postgres"
	"github.com/Impervguin/ds-lab2/library/internal/usecase"
)

const (
	shutdownTimeout    = 10 * time.Second
	databaseWaitPeriod = 30 * time.Second
	databaseRetryDelay = time.Second
)

func main() {
	if err := logger.Init(logger.FromEnv("library")); err != nil {
		fmt.Fprintf(os.Stderr, "configure logging: %v\n", err)
		os.Exit(1)
	}

	log := logger.Named("bootstrap")
	log.Info("library service starting")

	if err := run(log); err != nil {
		log.Error("library service stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("library service stopped")
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := openPool(ctx, log)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate(pool); err != nil {
		return err
	}
	log.Info("migrations applied")

	server := &http.Server{
		Addr:              net.JoinHostPort("", env("HTTP_PORT", "8060")),
		Handler:           newRouter(pool),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverFailed := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverFailed <- fmt.Errorf("serve http: %w", err)
		}
		close(serverFailed)
	}()

	select {
	case err := <-serverFailed:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}

func newRouter(pool *pgxpool.Pool) chi.Router {
	libraries := usecase.NewLibraryUseCase(
		postgres.NewLibraryRepository(pool),
		postgres.NewBookRepository(pool),
		postgres.NewLibraryBookRepository(pool),
	)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	health.NewHealthHandler().Register(router)
	handlerlibrary.NewLibraryHandler(libraries).Register(router)

	return router
}

func migrate(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	return migrations.Run(db)
}

func openPool(ctx context.Context, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	deadline := time.Now().Add(databaseWaitPeriod)
	for attempt := 1; ; attempt++ {
		err = pool.Ping(ctx)
		if err == nil {
			log.Info("database is ready")
			return pool, nil
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			pool.Close()
			return nil, fmt.Errorf("connect to database: %w", err)
		}

		log.Warn("database is not ready yet",
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()),
		)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(databaseRetryDelay):
		}
	}
}

func databaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		env("POSTGRES_USER", "program"),
		env("POSTGRES_PASSWORD", "test"),
		net.JoinHostPort(env("POSTGRES_HOST", "localhost"), env("POSTGRES_PORT", "5432")),
		env("POSTGRES_DB", "libraries"),
		env("POSTGRES_SSLMODE", "disable"),
	)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
