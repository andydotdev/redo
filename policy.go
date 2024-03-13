package redo

import "time"

// Policy allows you to predefine all of the options for a retry run ahead of
// time and set them using [WithPolicy]
type Policy struct {
	// Initial median delay -- see [InitialDelay]
	InitialDelay time.Duration
	// Maximum delay allowed -- see [MaxDelay]
	MaxDelay time.Duration
	// Maximum number of tries to attempt -- see [MaxTries]
	MaxTries int
	// Whether to retry the first time immdiaitely -- see [FirstFast]
	FirstFast bool
	// Halt allows you to set a function to check for fatal errors -- see [Halt]
	Halt func(error) bool
	// Each allows you to run a function directly after each failure -- see [Each]
	Each func(Status)
}
