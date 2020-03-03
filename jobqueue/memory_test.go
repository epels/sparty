package jobqueue

import (
	"context"
	"errors"
	"testing"
)

func TestConsume(t *testing.T) {
	mem := NewMemory()
	for _, uri := range []string{"foo", "bar", "baz"} {
		if err := mem.Put(uri); err != nil {
			t.Errorf("Got %T (%s), expected nil", err, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	var count int
	fn := func(uri string) {
		count++
		switch count {
		case 1:
			if uri != "foo" {
				t.Errorf("Got %q, expected foo", uri)
			}
		case 2:
			if uri != "bar" {
				t.Errorf("Got %q, expected bar", uri)
			}
		case 3:
			if uri != "baz" {
				t.Errorf("Got %q, expected baz", uri)
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
