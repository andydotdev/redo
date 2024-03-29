package redo

import (
	"errors"
	"time"
)

// Option represents an optional retry setting.
type Option func(o *opts)

// WithPolicy applies a the settings in a [Policy] to a run, allowing you to reuse
// a set of options for multiple functions.
func WithPolicy(p Policy) Option {
	return func(o *opts) {
		o.initialDelay = p.InitialDelay
		o.maxDelay = p.MaxDelay
		o.maxTries = p.MaxTries
		o.firstFast = p.FirstFast
		o.haltFn = p.Halt
		o.eachFn = p.Each
		o.noCause = p.NoCtxCause
	}
}

// InitialDelay sets the initial median delay of the first retry, and will
// serve to scale the rest of the run. If this is <= 0, it will default to
// DefaultInitialDelay (1 * time.Second)
func InitialDelay(duration time.Duration) Option {
	return func(o *opts) {
		o.initialDelay = duration
	}
}

// MaxDelay will cap the exponential delay to a maximum value. If this is <=
// 0, it will default to DefaultMaxDelay (20 * time.Minutes) or
// InitialDelay, whichever is greater.
func MaxDelay(duration time.Duration) Option {
	return func(o *opts) {
		o.maxDelay = duration
	}
}

// MaxTries is the number of tries to attempt. A negative value will retry
// until explicitly cancelled via context or a call to [Halt]. If unset, it
// will default to DefaultMaxTries (10)
func MaxTries(tries int) Option {
	return func(o *opts) {
		o.maxTries = tries
	}
}

// FirstFast defines whether or not the first retry should be made
// immediately. Defaults to false.
func FirstFast(firstRetryImmediate bool) Option {
	return func(o *opts) {
		o.firstFast = firstRetryImmediate
	}
}

// HaltFn allows you to set a function to use for identifying fatal errors.
// It will be called for each error returned from the target function. If it
// returns true, the retry loop will terminate immediately. Defaults to nil.
//
// Note: this will not affect the processing of [context.Canceled] and
// [context.DeadlineExceeded], which will always halt the retry loop.
func HaltFn(haltFn func(error) bool) Option {
	return func(o *opts) {
		o.haltFn = haltFn
	}
}

// HaltErrors is a shortcut to writing a [HaltFn] of the form
//
//	func(e error) bool {
//	    return errors.Is(e, Err1) || errors.Is(e, Err2) /* ... */
//	}
//
// Note: [context.Canceled] and [context.DeadlineExceeded], are already
// handled specially, so adding them using HaltErrors is a no-op.
func HaltErrors(errs ...error) Option {
	return func(o *opts) {
		o.haltFn = func(e error) bool {
			for i := range errs {
				if errors.Is(e, errs[i]) {
					return true
				}
			}
			return false
		}
	}
}

// Each allows you to set a function to be called directly after each failed
// retry. It is passed a [Status] value that you can use for logging or
// reporting. Defaults to nil, which will take no action.
func Each(eachFn func(Status)) Option {
	return func(o *opts) {
		o.eachFn = eachFn
	}
}

// CtxCause will enable or disable automatic context cancellation cause
// extraction.
// If enabled, redo will call [context.Cause] on all values of
// [context.Canceled] and [context.DeadlineExceeded] to get the underlying
// error, if it is set. Defaults to true, which enables this behavior
func CtxCause(enabled bool) Option {
	return func(o *opts) {
		o.noCause = !enabled
	}
}

func applyDefaults(ro *opts) {
	if ro.initialDelay <= 0 {
		ro.initialDelay = DefaultInitialDelay
	}
	if ro.maxDelay <= 0 {
		if ro.initialDelay > DefaultMaxDelay {
			ro.maxDelay = ro.initialDelay
		} else {
			ro.maxDelay = DefaultMaxDelay
		}
	}
	if ro.maxTries == 0 {
		ro.maxTries = DefaultMaxTries
	}
}

type opts struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	maxTries     int
	firstFast    bool
	eachFn       func(Status)
	haltFn       func(error) bool
	noCause      bool
}
