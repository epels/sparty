package jobqueue

import (
	"context"
	"errors"
	"testing"
)

func TestClose(t *testing.T) {
	mem := NewMemory()
	if err := mem.Close(); err != nil {
		t.Errorf("Got %T (%s), expected nil", err, err)
	}
	err := mem.Consume(context.Background(), func(url string) {
		t.Error("Unexpected call to fn")
	})
	if !errors.Is(err, ErrChannelClosed) {
		t.Errorf("Got %T (%s), expected ErrChannelClosed", err, err)
	}
}

func TestConsume(t *testing.T) {
	mem := NewMemory()
	for _, url := range []string{"foo", "bar", "baz"} {
		if err := mem.Put(url); err != nil {
			t.Errorf("Got %T (%s), expected nil", err, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	var count int
	fn := func(url string) {
		count++
		switch count {
		case 1:
			if url != "foo" {
				t.Errorf("Got %q, expected foo", url)
			}
		case 2:
			if url != "bar" {
				t.Errorf("Got %q, expected bar", url)
			}
		case 3:
			if url != "baz" {
				t.Errorf("Got %q, expected baz", url)
			}
			cancel()
		default:
			cancel()
		}
	}
	if err := mem.Consume(ctx, fn); !errors.Is(err, context.Canceled) {
		t.Errorf("Got %T (%s), expected context.Canceled", err, err)
	}
}

func TestPut(t *testing.T) {
	mem := NewMemory()
	if err := mem.Put("foo"); err != nil {
		t.Errorf("Got %T (%s), expected nil", err, err)
	}
	if err := mem.Put("bar"); err != nil {
		t.Errorf("Got %T (%s), expected nil", err, err)
	}
	if err := mem.Put("baz"); err != nil {
		t.Errorf("Got %T (%s), expected nil", err, err)
	}
}
