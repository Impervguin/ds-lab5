package httpclient

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"
)

var (
	errQueueClosed = errors.New("retry queue is closed")
	errQueueFull   = errors.New("retry queue is full")
)

type delayQueue struct {
	capacity int

	mutex  sync.Mutex
	items  delayHeap
	closed bool

	wake chan struct{}
	out  chan work
}

func newDelayQueue(capacity int) *delayQueue {
	return &delayQueue{
		capacity: capacity,
		wake:     make(chan struct{}, 1),
		out:      make(chan work),
	}
}

func (q *delayQueue) enqueue(w work, delay time.Duration) error {
	q.mutex.Lock()
	if q.closed {
		q.mutex.Unlock()
		return errQueueClosed
	}
	if len(q.items) >= q.capacity {
		q.mutex.Unlock()
		return errQueueFull
	}
	heap.Push(&q.items, delayed{work: w, due: time.Now().Add(delay)})
	q.mutex.Unlock()

	select {
	case q.wake <- struct{}{}:
	default: 
	}
	return nil
}

func (q *delayQueue) works() <-chan work {
	return q.out
}

// run dispatches due works until ctx is cancelled, then closes the works
// channel and reports how many works were left undone.
func (q *delayQueue) run(ctx context.Context) int {
	defer close(q.out)

	timer := time.NewTimer(0)
	timer.Stop()
	defer timer.Stop()

	for {
		due, wait, ok := q.next()
		if ok {
			select {
			case q.out <- due:
				continue
			case <-ctx.Done():
				return q.shut() + 1
			}
		}

		var fired <-chan time.Time
		if wait > 0 {
			timer.Reset(wait)
			fired = timer.C
		}

		select {
		case <-ctx.Done():
			return q.shut()
		case <-q.wake:
		case <-fired:
		}
		timer.Stop()
	}
}


func (q *delayQueue) next() (work, time.Duration, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.items) == 0 {
		return work{}, 0, false
	}
	if wait := time.Until(q.items[0].due); wait > 0 {
		return work{}, wait, false
	}
	return heap.Pop(&q.items).(delayed).work, 0, true
}

func (q *delayQueue) shut() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.closed = true
	left := len(q.items)
	q.items = nil
	return left
}

type delayed struct {
	work work
	due  time.Time
}

type delayHeap []delayed

func (h delayHeap) Len() int           { return len(h) }
func (h delayHeap) Less(i, j int) bool { return h[i].due.Before(h[j].due) }
func (h delayHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *delayHeap) Push(x any)        { *h = append(*h, x.(delayed)) }

func (h *delayHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}
