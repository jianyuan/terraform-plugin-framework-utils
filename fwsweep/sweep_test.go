package fwsweep_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jianyuan/terraform-plugin-framework-utils/fwsweep"
)

type sweepableFunc func(ctx context.Context) error

func (f sweepableFunc) Delete(ctx context.Context) error { return f(ctx) }

func TestSweep_JoinsErrors(t *testing.T) {
	errA := errors.New("a")
	errB := errors.New("b")
	var calls atomic.Int32

	err := fwsweep.Sweep(t.Context(), []fwsweep.Sweepable{
		sweepableFunc(func(context.Context) error { calls.Add(1); return errA }),
		sweepableFunc(func(context.Context) error { calls.Add(1); return nil }),
		sweepableFunc(func(context.Context) error { calls.Add(1); return errB }),
	})

	if calls.Load() != 3 {
		t.Errorf("expected 3 deletes, got %d", calls.Load())
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Errorf("expected joined errors a and b, got %v", err)
	}
}

func TestSweep_NoErrors(t *testing.T) {
	if err := fwsweep.Sweep(t.Context(), nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestSweep_WithConcurrency(t *testing.T) {
	for _, limit := range []int{1, 3} {
		var running, peak atomic.Int32
		sweepables := make([]fwsweep.Sweepable, 10)
		for i := range sweepables {
			sweepables[i] = sweepableFunc(func(context.Context) error {
				n := running.Add(1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				running.Add(-1)
				return nil
			})
		}

		if err := fwsweep.Sweep(t.Context(), sweepables, fwsweep.WithConcurrency(limit)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if peak.Load() > int32(limit) {
			t.Errorf("limit %d: peak concurrency was %d", limit, peak.Load())
		}
	}
}
