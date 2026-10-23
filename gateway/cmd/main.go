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

	"github.com/Impervguin/ds-lab2/gateway/internal/adapter/httpclient"
	"github.com/Impervguin/ds-lab2/gateway/internal/auth"
	"github.com/Impervguin/ds-lab2/gateway/internal/handler/health"
	handlerlibrary "github.com/Impervguin/ds-lab2/gateway/internal/handler/library"
	handlerrating "github.com/Impervguin/ds-lab2/gateway/internal/handler/rating"
	handlerreservation "github.com/Impervguin/ds-lab2/gateway/internal/handler/reservation"
	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

const (
	shutdownTimeout           = 10 * time.Second
	circuitBreakerOpenTimeout = 10 * time.Second
)

var retryerConfig = httpclient.RetryerConfig{
	Workers:        2,
	Capacity:       1024,
	BaseDelay:      2 * time.Second,
	MaxDelay:       10 * time.Second,
	AttemptTimeout: 5 * time.Second,
}

func main() {
	if err := logger.Init(logger.FromEnv("gateway")); err != nil {
		fmt.Fprintf(os.Stderr, "configure logging: %v\n", err)
		os.Exit(1)
	}

	log := logger.Named("bootstrap")
	log.Info("gateway service starting")

	if err := run(log); err != nil {
		log.Error("gateway service stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("gateway service stopped")
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The JWKs keep refreshing until the server has drained.
	keysCtx, stopKeys := context.WithCancel(context.Background())
	defer stopKeys()
	verifier, err := auth.NewVerifier(keysCtx, auth.Config{
		Issuer:        os.Getenv("OIDC_ISSUER"),
		Audience:      os.Getenv("OIDC_AUDIENCE"),
		JWKsURL:       os.Getenv("OIDC_JWKS_URL"),
		UsernameClaim: env("OIDC_USERNAME_CLAIM", auth.DefaultUsernameClaim),
	})
	if err != nil {
		return fmt.Errorf("configure auth: %w", err)
	}

	// The retryer outlives the signal: it stops only after the server has
	// drained, so requests queued by the last handlers are not lost silently.
	retryer := httpclient.NewHttpRetryer(retryerConfig)
	retryCtx, stopRetries := context.WithCancel(context.Background())
	retryDone := make(chan struct{})
	go func() {
		retryer.Run(retryCtx)
		close(retryDone)
	}()
	defer func() {
		stopRetries()
		<-retryDone
	}()

	server := &http.Server{
		Addr:              net.JoinHostPort("", env("HTTP_PORT", "8080")),
		Handler:           newRouter(retryer, verifier),
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

func newRouter(retryer *httpclient.HttpRetryer, verifier *auth.Verifier) chi.Router {
	libraries := httpclient.NewLibraryClient(newTransport(env("LIBRARY_SERVICE_URL", "http://localhost:8060")), retryer)
	reservations := httpclient.NewReservationClient(newTransport(env("RESERVATION_SERVICE_URL", "http://localhost:8070")), retryer)
	ratings := httpclient.NewRatingClient(newTransport(env("RATING_SERVICE_URL", "http://localhost:8050")), retryer)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	health.NewHealthHandler().Register(router)

	router.Group(func(api chi.Router) {
		api.Use(verifier.Middleware)

		handlerlibrary.NewLibraryHandler(usecase.NewLibraryUseCase(libraries)).Register(api)
		handlerrating.NewRatingHandler(usecase.NewRatingUseCase(ratings)).Register(api)
		handlerreservation.NewReservationHandler(
			usecase.NewReservationUseCase(reservations, libraries, ratings),
		).Register(api)
	})

	return router
}

// newTransport gives every upstream its own circuit breaker, so one failing
// service does not cut off the others.
func newTransport(baseURL string) httpclient.HttpDoer {
	return httpclient.NewCircuitBreaker(httpclient.NewBaseClient(baseURL), circuitBreakerOpenTimeout)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
