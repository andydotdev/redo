package redo

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
			str = fmt.Sprintf("%s - next in %v", str, shortNext(s.NextDelay))
		}
		if verb == 'q' {
			str = fmt.Sprintf("%q", str)
		}
		fmt.Fprint(state, str)
	}
}

// LogValue implements [slog.LogValuer], allowing the retry status to be logged as a [slog.GroupValue]
func (s Status) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("try", s.TryNumber),
		slog.Int("max_tries", s.MaxTries),
		slog.Duration("next", shortNext(s.NextDelay)),
		slog.String("last_error", s.Err.Error()),
	)
}

// Next returns a time.Time value representing the approximate time the next
// iteration will occur, assuming it has just failed.
func (s Status) Next() time.Time {
	return time.Now().Add(s.NextDelay)
}

func shortNext(d time.Duration) time.Duration {
	switch {
	case d < time.Second:
		return d.Truncate(time.Millisecond)
	case d < time.Minute:
		return d.Truncate(time.Second)
	case d < time.Hour:
		d.Truncate(time.Minute)
	}
	// Otherwise truncate the number of hours to two decimal places.
	return time.Duration(math.Round(d.Hours()*100) / 100)
}
