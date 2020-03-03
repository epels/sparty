package jobqueue

import (
	"context"
	"errors"
)

// memory is a dead simple in-memory job queue that is only focused on
// facilitating fast acceptance at the API level. It does not provide any other
// "fancy" features like delays and retries.
type memory struct {
	consumer func(uri string)
	ch       chan string
}

var ErrChannelClosed = errors.New("channel was closed")

func NewMemory() *memory {
	return &memory{ch: make(chan string, 10)}
}

func (m *memory) Close() error {
	close(m.ch)
	return nil
}

// Consume will watch the memory jobqueue for new jobs, and pass them on to fn
// as they become available. Invocation blocks until the context is cancelled:
// then, the context error is returned.
func (m *memory) Consume(ctx context.Context, fn func(uri string)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case uri, ok := <-m.ch:
			if !ok {
				return ErrChannelClosed
			}
			// Block on fn so order is guaranteed and we won't flood the
			// Spotify Web API. This won't impose performance bottlenecks as
			// long as we're not going multi-tenant.
			fn(uri)
		}
	}
}

// Put enqueues a job. At the moment this never returns a non-nil error, but
// return an error nonetheless to ease a potential migration towards a more
// sophisticated jobqueue backend.
func (m *memory) Put(uri string) error {
	m.ch <- uri
	return nil
}
