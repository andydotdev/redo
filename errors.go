package redo

import (
	"errors"
	"fmt"
)

// Exhausted returns true if the error is the final result after all tries.
func Exhausted(e error) bool {
	_, ok := e.(*exhaustedErr)
	return ok
}

type exhaustedErr struct {
	err error
}

func (ee *exhaustedErr) Error() string {
	return ee.err.Error()
}

func (ee *exhaustedErr) Unwrap() error {
	return ee.err
}

func errExhausted(e error) *exhaustedErr {
	return &exhaustedErr{e}
}

type haltErr struct {
	err error
}

func (he *haltErr) Error() string {
	return he.err.Error()
}

func (he *haltErr) Unwrap() error {
	return he.err
}

// RefreshError will be returned if a [RefreshFn] returns an error. The
// underlying error that caused the retry will be combined with this error using
// [errors.Join].
// If you would like to inspect just the original error, you can use [errors.As]
// to get the *RefreshError value and call the [RetryErr] Method.
type RefreshError struct {
	err      error
	retryErr error
}

// Error implements the error interface.
func (re *RefreshError) Error() string {
	return fmt.Sprintf("%s\n%s", re.err, re.retryErr)
}

// Unwrap allows a *RefreshError to work with [errors.Is] and [errors.As]
func (re *RefreshError) Unwrap() []error {
	return []error{re.err, re.retryErr}
}

// RetryErr returns the error that caused the function to retry before the
// RefreshFn failed.
func (re *RefreshError) RetryErr() error {
	return re.retryErr
}

// errRefresh is a helper to create a *RefreshError
func errRefresh(refreshErr, retryErr error) *RefreshError {
	return &RefreshError{
		err:      errors.Join(refreshErr, retryErr),
		retryErr: retryErr,
	}
}
