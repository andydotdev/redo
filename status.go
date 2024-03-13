package redo

import (
	"context"
	"fmt"
	"time"
)

type retryCtxKeyT string

const (
	retryCtxKey retryCtxKeyT = "redo"
)

// GetStatus can be used to retrieve information about the current retry loop
// from within the function being retried, as opposed to setting a callback with
// [Each].
// It will return Status{} if not called in a retry context, so make sure to use
// [Retrying] if your function might be run outside of a retry loop.
func GetStatus(ctx context.Context) Status {
	stats := ctx.Value(retryCtxKey)
	if stats == nil {
		return Status{}
	}
	return stats.(Status)
}

// Status represents the state of the current retry loop.[GetStatus]
type Status struct {
	TryNumber int
	MaxTries  int
	Err       error
	NextDelay time.Duration
}

// String implements fmt.Stringer
func (s Status) String() string {
	if s.MaxTries <= 0 {
		return fmt.Sprintf("attempt %d", s.TryNumber)
	}
	return fmt.Sprintf("attempt %d/%d", s.TryNumber, s.MaxTries)
}

// Format implements fmt.Formatter it supports the %s and %q print verbs. Output
// is flag-dependent:
//
//	%s -  "attempt #"
//	%+s - "attempt # - next in <duration>"
//
// Where '#' is the attempt number as an integer such starting from '1'
// optionally followed by `/#` and the maximum number of tries if
// [MaxTries] is set.
func (s Status) Format(state fmt.State, verb rune) {
	switch verb {
	case 's', 'q':
		str := s.String()
		if state.Flag('+') {
			switch {
			case s.NextDelay < time.Second:
				s.NextDelay = s.NextDelay.Truncate(time.Millisecond)
			case s.NextDelay < time.Minute:
				s.NextDelay = s.NextDelay.Truncate(time.Second)
			case s.NextDelay < time.Hour:
				s.NextDelay = s.NextDelay.Truncate(time.Hour)
			}
			str = fmt.Sprintf("%s - next in %v", str, s.NextDelay)
		}
		if verb == 'q' {
			str = fmt.Sprintf("%q", str)
		}
		fmt.Fprint(state, str)
	}
}

// Next returns a time.Time value representing the approximate time the next
// iteration will occur, assuming it has just failed.
func (s Status) Next() time.Time {
	return time.Now().Add(s.NextDelay)
}
