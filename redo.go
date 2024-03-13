package redo

import (
	"context"
	"errors"
	"time"

	"andy.dev/redo/backoff"
)

const (
	DefaultInitialDelay = 1 * time.Second
	DefaultMaxDelay     = 20 * time.Minute
	DefaultMaxTries     = 10
)

type RetryFn interface {
	func() error | func(context.Context) error
}

type RetryFnOut[OUT any] interface {
	func() (OUT, error) | func(context.Context) (OUT, error)
}

type RetryFnIn[IN any] interface {
	func(IN) error | func(context.Context, IN) error
}

type RetryFnIO[IN, OUT any] interface {
	func(IN) (OUT, error) | func(context.Context, IN) (OUT, error)
}

// Fn is a retrier for functions with the signatures of:
//
//	func() error
//
// The error returned will be the ultimate error returned after all retries are
// complete or nil, in the case of a successful run. For more information on how
// functions will be retried and values returned, see the package documentation.
func Fn(ctx context.Context,
	fn func() error,
	options ...Option,
) error {
	return FnCtx(ctx, func(context.Context) error {
		return fn()
	}, options...)
}

// FnOut is a retrier for functions with the signature of:
//
//	func() (OUT, error)
//
// Where OUT is a return value of any type.
//
// The function will be retried following the rules described in the package
// documentation, and will return the values of the first successful run or the
// final unsuccessful run.
func FnOut[OUT any](
	ctx context.Context,
	fn func() (OUT, error),
	options ...Option,
) (OUT, error) {
	return FnOutCtx(ctx, func(context.Context) (OUT, error) {
		return fn()
	}, options...)
}

// FnIn is a retrier for functions with the signature of:
//
//	func(IN) error
//
// Where IN is an input argument fnArg of any type.
//
// Note: fn is passed by value, separately from fnArg:
//
//	FnIn(ctx, fnToRetry, <argument>) - CORRECT
//	FnIn(ctx, fnToRetry(<argument)) - INCORRECT
func FnIn[IN any](
	ctx context.Context,
	fn func(IN) error,
	fnArg IN,
	options ...Option,
) error {
	return FnInCtx(ctx, func(_ context.Context, arg IN) error {
		return (fn(arg))
	}, fnArg, options...)
}

// FnInRefr is a retrier for functions with the signature of:
//
//	func(IN) error
//
// Where IN is an input argument of any type. The initial value for this
// argument is passed using the fnArg argument and will be refreshed using
// refreshFn for subsequent retries, if needed.
func FnInRefr[IN any](
	ctx context.Context,
	fn func(IN) error,
	refreshFn RefreshFn[IN],
	fnArg IN,
	options ...Option,
) error {
	return FnInCtxRefr(ctx, func(_ context.Context, arg IN) error {
		return fn(arg)
	}, fnArg, refreshFn, options...)
}

// FnIO is a retrier for functions with the signature of:
//
//	func(IN)(OUT, ERROR)
//
// Where IN is an input argument fnArg of any type and OUT is a return value of
// any type.
//
// The function will be retried following the rules described in the package
// documentation, and will return the values of the first successful run or the
// final unsuccessful run. It is a combination of [FnIn] and [FnOut].
func FnIO[IN, OUT any](
	ctx context.Context,
	fn func(IN) (OUT, error),
	fnArg IN,
	options ...Option,
) (OUT, error) {
	return FnIOCtx(ctx, func(_ context.Context, arg IN) (OUT, error) {
		return fn(arg)
	}, fnArg, options...)
}

// FnIORefr is a retrier for functions with the signature of:
//
//	func(IN)(OUT, ERROR)
//
// Where IN is an input argument fnArg of any type and OUT is a return value of
// any type.The initial input value for fn is passed using the fnArg argument
// and will be refreshed using refreshFn for subsequent retries, if needed. It
// is a combination of [FnInRefr] and [FnOut].
func FnIORefr[IN, OUT any](
	ctx context.Context,
	fn func(IN) (OUT, error),
	fnArg IN,
	refreshFn RefreshFn[IN],
	options ...Option,
) (OUT, error) {
	return FnIOCtxRefr(ctx, func(_ context.Context, arg IN) (OUT, error) {
		return fn(arg)
	}, fnArg, refreshFn, options...)
}

// FnCtx is a retrier for functions with the following signature:
//
//	func(context.Context) error
//
// The error returned will be the ultimate error returned after all retries are
// complete or nil, in the case of a successful run. For more information on how
// functions will be retried and values returned, see the package documentation.
func FnCtx(
	ctx context.Context,
	fn func(context.Context) error,
	options ...Option,
) error {
	opts := &opts{}
	for _, o := range options {
		o(opts)
	}
	applyDefaults(opts)
	backoff := backoff.New(opts.initialDelay, opts.maxDelay, opts.firstFast)
	t := time.NewTimer(DefaultMaxDelay)
	t.Stop()
	try := 0
	var lastErr error
	for {
		// prefetch the next delay so that the user can see it in the stats.
		delay := backoff()
		status := Status{
			TryNumber: try + 1,
			MaxTries:  opts.maxTries,
			Err:       lastErr,
			NextDelay: delay,
		}
		rctx := context.WithValue(ctx, retryCtxKey, status)
		lastErr = fn(rctx)
		if lastErr == nil {
			return nil
		}
		status.Err = lastErr
		if opts.eachFn != nil {
			opts.eachFn(status)
		}
		try++
		switch {
		case errors.Is(lastErr, context.Canceled):
			return context.Cause(ctx)
		case Halted(lastErr):
			return lastErr
		case opts.haltFn != nil && opts.haltFn(lastErr):
			return Halt(lastErr)
		case opts.maxTries > 0 && try == opts.maxTries:
			return errExhausted(lastErr)
		}
		t.Reset(delay)
		select {
		case <-ctx.Done():
			if !t.Stop() {
				<-t.C
			}
			return context.Cause(ctx)
		case <-t.C:
			continue
		}
	}
}

// FnOutCtx is a retrier for functions with the signature of:
//
//	func(context.Context) (OUT, error)
//
// Where OUT is a return value of any type.
//
// The function will be retried following the rules described in the package
// documentation, and will return the values of the first successful run or the
// final unsuccessful run.
func FnOutCtx[OUT any](
	ctx context.Context,
	fn func(context.Context) (OUT, error),
	options ...Option,
) (OUT, error) {
	var (
		zero  OUT
		val   OUT
		fnErr error
	)
	err := FnCtx(ctx, func(ctx context.Context) error {
		val, fnErr = fn(ctx)
		return fnErr
	}, options...)
	if err != nil {
		return zero, err
	}
	return val, nil
}

// FnInCtx is a retrier for functions with the signature of:
//
//	func(context.Context, IN) error
//
// Where IN is an input argument fnArg of any type.
//
// Note: fn is passed by value, separately from fnArg:
//
//	FnInCtx(ctx, fnToRetry, <arg>) - CORRECT
//	FnInCtx(ctx, fnToRetry(arg)) - INCORRECT
func FnInCtx[IN any](
	ctx context.Context,
	fn func(context.Context, IN) error,
	fnArg IN,
	options ...Option,
) error {
	return FnCtx(ctx, func(ictx context.Context) error {
		return fn(ictx, fnArg)
	}, options...)
}

// FnInCtxRefr is a retrier for functions with the signature of:
//
//	func(context.Context, IN) error
//
// Where IN is an input argument of any type. The initial value for this
// argument is passed using the fnArg argument and will be refreshed using
// refreshFn for subsequent retries, if needed.
func FnInCtxRefr[IN any](
	ctx context.Context,
	fn func(context.Context, IN) error,
	fnArg IN,
	refreshFn RefreshFn[IN],
	options ...Option,
) error {
	return FnCtx(ctx, func(ictx context.Context) error {
		err := fn(ictx, fnArg)
		if err != nil {
			if refreshFn != nil {
				nArg, refreshErr := refreshFn()
				if refreshErr != nil {
					return errRefresh(refreshErr, err)
				}
				fnArg = nArg
			}
		}
		return err
	}, options...)
}

// FnIO is a retrier for functions with the signature of:
//
//	func(context.Context, IN)(OUT, ERROR)
//
// Where IN is an input argument fnArg of any type and OUT is a return value of
// any type.
//
// The function will be retried following the rules described in the package
// documentation, and will return the values of the first successful run or the
// final unsuccessful run. It is a combination of [FnInCtx] and [FnOutCtx].
func FnIOCtx[IN, OUT any](
	ctx context.Context,
	fn func(context.Context, IN) (OUT, error),
	fnArg IN,
	options ...Option,
) (OUT, error) {
	var (
		zero  OUT
		val   OUT
		fnErr error
	)
	err := FnInCtx(ctx, func(ictx context.Context, arg IN) error {
		val, fnErr = fn(ictx, arg)
		return fnErr
	}, fnArg, options...)
	if err != nil {
		return zero, err
	}
	return val, nil
}

// FnIOCtxRefr is a retrier for functions with the signature of:
//
//	func(context.Context, IN)(OUT, ERROR)
//
// Where IN is an input argument fnArg of any type and OUT is a return value of
// any type.The initial input value for fn is passed using the fnArg argument
// and will be refreshed using refreshFn for subsequent retries, if needed. It
// is a combination of [FnInCtxRefr] and [FnOutCtx].
func FnIOCtxRefr[IN, OUT any](
	ctx context.Context,
	fn func(context.Context, IN) (OUT, error),
	fnArg IN,
	refreshFn RefreshFn[IN],
	options ...Option,
) (OUT, error) {
	var (
		zero  OUT
		val   OUT
		fnErr error
	)
	err := FnInCtxRefr(ctx, func(ictx context.Context, arg IN) error {
		val, fnErr = fn(ictx, arg)
		return fnErr
	}, fnArg, refreshFn, options...)
	if err != nil {
		return zero, err
	}
	return val, nil
}

// RefreshFn is a function that can be passed to any of the -Refresh retriers to
// recreate or reset the input argument to the function between retries. If this
// function returns an error, it will be wrapped in a [*RefreshError] value,
// along with the underlying error that triggered the retry.
type RefreshFn[T any] func() (T, error)

// Halted returns true if the retry was manually halted by the user by returning.
// an error wrapped with [Halt]
func Halted(e error) bool {
	_, ok := e.(*haltErr)
	return ok
}

// Halt allows you to return a halting error from within the retry loop itself,
// as an alternative to using [HaltFn]. Simply:
//
//	return retry.Halt(err)
//
// To stop the retry run immediately.
func Halt(e error) *haltErr {
	return &haltErr{e}
}
