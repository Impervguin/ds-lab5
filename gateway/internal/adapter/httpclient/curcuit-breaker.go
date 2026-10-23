package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

type circuitBreakerState int

const (
	circuitBreakerClosed circuitBreakerState = iota
	circuitBreakerOpen
	circuitBreakerHalfOpen
)

const failureThreshold = 3

type CircuitBreaker struct {
	next        HttpDoer
	openTimeout time.Duration

	mutex    sync.Mutex
	state    circuitBreakerState
	failures int
	openedAt time.Time
}

var _ HttpDoer = (*CircuitBreaker)(nil)

func NewCircuitBreaker(next HttpDoer, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{next: next, openTimeout: openTimeout}
}

func (c *CircuitBreaker) do(ctx context.Context, call call) error {
	if call.method != http.MethodGet {
		return c.next.do(ctx, call)
	}

	if !c.allow() {
		return usecase.ErrServiceUnavailable
	}

	err := c.next.do(ctx, call)

	switch {
	case ctx.Err() != nil:
		// The caller gave up, the upstream is not to blame.
		c.neutral()
		return err
	case isFailure(err):
		c.failure()
		return fmt.Errorf("%w: %w", usecase.ErrServiceUnavailable, err)
	default:
		c.success()
		return err
	}
}

func (c *CircuitBreaker) allow() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch c.state {
	case circuitBreakerClosed:
		return true
	case circuitBreakerOpen:
		if time.Since(c.openedAt) < c.openTimeout {
			return false
		}
		c.state = circuitBreakerHalfOpen
		return true
	default: // half-open: a probe is already in flight
		return false
	}
}

func (c *CircuitBreaker) failure() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch c.state {
	case circuitBreakerClosed:
		c.failures++
		if c.failures >= failureThreshold {
			c.open()
		}
	case circuitBreakerHalfOpen:
		c.open()
	}
	// Already open: a late failure from a request started before the trip.
}

func (c *CircuitBreaker) success() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.state == circuitBreakerOpen {
		return
	}
	c.state = circuitBreakerClosed
	c.failures = 0
}

func (c *CircuitBreaker) neutral() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.state == circuitBreakerHalfOpen {
		// The probe was cancelled; go back to open, the timeout has already
		// elapsed, so the next request becomes the new probe.
		c.state = circuitBreakerOpen
	}
}

// open must be called with the mutex held.
func (c *CircuitBreaker) open() {
	c.state = circuitBreakerOpen
	c.openedAt = time.Now()
	c.failures = 0
}

func isFailure(err error) bool {
	if err == nil {
		return false
	}

	var httpErr *HttpClientError
	if errors.As(err, &httpErr) {
		return httpErr.netErr != nil || httpErr.StatusCode != nil && *httpErr.StatusCode >= http.StatusInternalServerError
	}
	// Unknown error
	return true
}
