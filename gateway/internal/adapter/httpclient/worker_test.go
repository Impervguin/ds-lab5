package httpclient

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/gateway/internal/usecase"
)

var (
	errRefused  = &HttpClientError{netErr: errors.New("connection refused")}
	errRejected = func() error {
		status := http.StatusConflict
		return &HttpClientError{StatusCode: &status, wrapped: errors.New("conflict")}
	}()
)

// scriptedDoer answers with the given errors in turn, then with success.
type scriptedDoer struct {
	mutex   sync.Mutex
	answers []error
	calls   chan call
}

func newScriptedDoer(answers ...error) *scriptedDoer {
	return &scriptedDoer{answers: answers, calls: make(chan call, 16)}
}

func (d *scriptedDoer) do(_ context.Context, c call) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.calls <- c
	if len(d.answers) == 0 {
		return nil
	}
	answer := d.answers[0]
	d.answers = d.answers[1:]
	return answer
}

func (d *scriptedDoer) waitCall(t *testing.T) call {
	t.Helper()
	select {
	case c := <-d.calls:
		return c
	case <-time.After(time.Second):
		t.Fatal("the doer was not called")
		return call{}
	}
}

func (d *scriptedDoer) assertNoCall(t *testing.T, within time.Duration) {
	t.Helper()
	select {
	case c := <-d.calls:
		t.Fatalf("unexpected call %s %s", c.method, c.path)
	case <-time.After(within):
	}
}

func startRetryer(t *testing.T) *HttpRetryer {
	t.Helper()

	retryer := NewHttpRetryer(RetryerConfig{
		Workers:        1,
		Capacity:       8,
		BaseDelay:      10 * time.Millisecond,
		MaxDelay:       20 * time.Millisecond,
		AttemptTimeout: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		retryer.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return retryer
}

func TestDelayQueueHandsOutWorksInDueOrder(t *testing.T) {
	queue := newDelayQueue(8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go queue.run(ctx)

	require.NoError(t, queue.enqueue(work{call: call{path: "/late"}}, 40*time.Millisecond))
	require.NoError(t, queue.enqueue(work{call: call{path: "/early"}}, 10*time.Millisecond))

	assert.Equal(t, "/early", (<-queue.works()).call.path)
	assert.Equal(t, "/late", (<-queue.works()).call.path)
}

func TestDelayQueueRefusesWorksWhenFullOrClosed(t *testing.T) {
	queue := newDelayQueue(1)

	require.NoError(t, queue.enqueue(work{}, time.Hour))
	assert.ErrorIs(t, queue.enqueue(work{}, time.Hour), errQueueFull)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Equal(t, 1, queue.run(ctx))
	assert.ErrorIs(t, queue.enqueue(work{}, 0), errQueueClosed)
}

func TestDoOrQueueReturnsTheResultWhenTheUpstreamAnswers(t *testing.T) {
	doer := newScriptedDoer()

	err := doOrQueue(context.Background(), doer, startRetryer(t), true, call{path: "/ok"})

	require.NoError(t, err)
	doer.waitCall(t)
	doer.assertNoCall(t, 50*time.Millisecond)
}

func TestDoOrQueueRetriesUntilTheUpstreamAnswers(t *testing.T) {
	doer := newScriptedDoer(errRefused, errRefused)

	err := doOrQueue(context.Background(), doer, startRetryer(t), true, call{path: "/return"})

	assert.ErrorIs(t, err, usecase.ErrQueued)
	for range 3 {
		assert.Equal(t, "/return", doer.waitCall(t).path)
	}
	doer.assertNoCall(t, 50*time.Millisecond)
}

func TestDoOrQueueDoesNotQueueWithoutPermission(t *testing.T) {
	doer := newScriptedDoer(errRefused)

	err := doOrQueue(context.Background(), doer, startRetryer(t), false, call{path: "/return"})

	assert.ErrorIs(t, err, errRefused)
	doer.waitCall(t)
	doer.assertNoCall(t, 50*time.Millisecond)
}

func TestDoOrQueueDoesNotQueueARejectedRequest(t *testing.T) {
	doer := newScriptedDoer(errRejected)

	err := doOrQueue(context.Background(), doer, startRetryer(t), true, call{path: "/return"})

	assert.ErrorIs(t, err, errRejected)
	doer.waitCall(t)
	doer.assertNoCall(t, 50*time.Millisecond)
}

func TestRetryerGivesUpOnARejectedRetry(t *testing.T) {
	doer := newScriptedDoer(errRefused, errRejected)

	err := doOrQueue(context.Background(), doer, startRetryer(t), true, call{path: "/return"})

	assert.ErrorIs(t, err, usecase.ErrQueued)
	doer.waitCall(t)
	doer.waitCall(t)
	doer.assertNoCall(t, 50*time.Millisecond)
}

func TestRetryerDelayGrowsUpToTheLimit(t *testing.T) {
	retryer := NewHttpRetryer(RetryerConfig{BaseDelay: 2 * time.Second, MaxDelay: 10 * time.Second})

	assert.Equal(t, 2*time.Second, retryer.delay(0))
	assert.Equal(t, 4*time.Second, retryer.delay(1))
	assert.Equal(t, 8*time.Second, retryer.delay(2))
	assert.Equal(t, 10*time.Second, retryer.delay(3))
	assert.Equal(t, 10*time.Second, retryer.delay(30))
}
