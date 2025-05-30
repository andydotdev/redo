package redo

import "time"

// Policy allows you to predefine all of the options for a retry run ahead of
// time and set them using [WithPolicy]
type Policy struct {
	// Initial median delay.
	// Default: (1 * time.Second)
	InitialDelay time.Duration
	// Maximum delay allowed.
	// Default: (20*time.Minutes >= InitialDelay)
	MaxDelay time.Duration
	// Maximum number of tries to attempt.
	// Default: 10
	MaxTries int
	// Whether to retry the first time immdiaitely.
	// Default: false
	FirstFast bool
	// ErrorHandler allows you provide one or more functions to check for errors -- see [ErrorHandler]
	ErrorHandler ErrorHandlerFn
	// Each allows you to run a function directly after each failure -- see [Each]
	Each func(Status)
	// NoCtxCause disables automatic extraction of context cause -- see [CtxCause]
	NoCtxCause bool
}
