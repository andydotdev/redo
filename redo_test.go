package redo_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"andy.dev/redo"
)

func TestCtxCancel(t *testing.T) {
	waitOrCancel := func(ctx context.Context) error {
		select {
		case <-time.After(10 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	inner := func(ctx context.Context) error {
		innerTimeoutCtx, cf := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cf()
		return waitOrCancel(innerTimeoutCtx)
	}

	testPolicy := redo.Policy{
		InitialDelay: 10 * time.Microsecond,
		MaxDelay:     10 * time.Millisecond,
		MaxTries:     3,
		FirstFast:    true,
	}

	t.Run("InnerCtxCancelContinues", func(t *testing.T) {
		err := redo.FnCtx(context.Background(), inner, redo.WithPolicy(testPolicy))
		assert(t, errors.Is(err, context.DeadlineExceeded))
		assert(t, redo.Exhausted(err), "should reach MaxTries")
	})

	t.Run("OuterCtxCancelHalts", func(t *testing.T) {
		outerTimeoutCtx, cf := context.WithTimeout(context.Background(), 1)
		defer cf()
		err := redo.FnCtx(outerTimeoutCtx, inner, redo.WithPolicy(testPolicy))
		assert(t, errors.Is(err, context.DeadlineExceeded))
		assert(t, !redo.Exhausted(err), "should not reach MaxTries")
	})
}

func assert(t *testing.T, v bool, a ...any) {
	t.Helper()
	if !v {
		t.Error(append([]any{"assertion failed:"}, a...))
	}
}

func assertf(t *testing.T, v bool, format string, a ...any) {
	t.Helper()
	if !v {
		t.Errorf("assertion failed: %s", fmt.Sprintf(format, a...))
	}
}
