package httpclient

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type work struct {
	call     call
	doer     HttpDoer
	attempts int
}

type RetryerConfig struct {
	Workers        int
	Capacity       int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	AttemptTimeout time.Duration
}


type HttpRetryer struct {
	queue  *delayQueue
	config RetryerConfig
	log    *slog.Logger
}

func NewHttpRetryer(config RetryerConfig) *HttpRetryer {
	return &HttpRetryer{
		queue:  newDelayQueue(config.Capacity),
		config: config,
		log:    logger.Named("httpclient.retryer"),
	}
}

func (r *HttpRetryer) Run(ctx context.Context) {
	var workers sync.WaitGroup
	for range r.config.Workers {
		workers.Go(func() {
			for w := range r.queue.works() {
				r.attempt(ctx, w)
			}
		})
	}

	left := r.queue.run(ctx)
	workers.Wait()

	if left > 0 {
		r.log.Warn("retry queue stopped with pending requests", slog.Int("dropped", left))
	}
}

func (r *HttpRetryer) enqueue(doer HttpDoer, call call, cause error) error {
	// Nobody is waiting for the response any more.
	call.out = nil

	w := work{call: call, doer: doer}
	if err := r.queue.enqueue(w, r.delay(w.attempts)); err != nil {
		return err
	}

	r.log.Info("request queued",
		slog.String("method", call.method),
		slog.String("path", call.path),
		slog.String("cause", cause.Error()),
	)
	return nil
}

func (r *HttpRetryer) attempt(ctx context.Context, w work) {
	attemptCtx, cancel := context.WithTimeout(ctx, r.config.AttemptTimeout)
	err := w.doer.do(attemptCtx, w.call)
	cancel()

	w.attempts++
	log := r.log.With(
		slog.String("method", w.call.method),
		slog.String("path", w.call.path),
		slog.Int("attempts", w.attempts),
	)

	switch {
	case err == nil:
		log.Info("queued request completed")
	case ctx.Err() != nil:
		log.Warn("queued request dropped on shutdown", slog.String("error", err.Error()))
	case isFailure(err):
		delay := r.delay(w.attempts)
		if queueErr := r.queue.enqueue(w, delay); queueErr != nil {
			log.Error("queued request dropped", slog.String("error", queueErr.Error()))
			return
		}
		log.Warn("queued request failed, will retry",
			slog.Duration("delay", delay),
			slog.String("error", err.Error()),
		)
	default:
		log.Error("queued request rejected, giving up", slog.String("error", err.Error()))
	}
}

func (r *HttpRetryer) delay(attempts int) time.Duration {
	delay := r.config.BaseDelay
	for range attempts {
		delay *= 2
		if delay >= r.config.MaxDelay {
			return r.config.MaxDelay
		}
	}
	return delay
}

// doOrQueue makes the call right away; if the upstream cannot be reached and
// the caller allowed it, the call is handed over to the retryer and
// usecase.ErrQueued is returned instead of a result.
func doOrQueue(ctx context.Context, doer HttpDoer, retryer *HttpRetryer, inQueue bool, call call) error {
	err := doer.do(ctx, call)
	if err == nil || !inQueue || retryer == nil || ctx.Err() != nil || !isFailure(err) {
		return err
	}

	if queueErr := retryer.enqueue(doer, call, err); queueErr != nil {
		retryer.log.ErrorContext(ctx, "request not queued",
			slog.String("method", call.method),
			slog.String("path", call.path),
			slog.String("error", queueErr.Error()),
		)
		return err
	}
	return usecase.ErrQueued
}
