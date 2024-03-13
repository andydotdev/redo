/*
Package redo is an ergonomic retry library for Go.

It provides a set of generic retriers for functions of common signatures using a decorrelated soft exponential backoff delay to limit concurrent requests downstream.

# Ergonomic?

The API is intended to be "ergonomic" in that it attempts to be intuitive to use and easy to integrate into existing code, without a lot of cognitive load.

To this end, it has the following features:
  - Declarative syntax to wrap existing code.
  - Short, memorable names for wrapping functions.
  - Support for functional options with sensible defaults as well as a [Policy] type to predeclare a set of options for re-use.

# Supported Function Types

The following function types are supported:

	|           Function Signature           |   Retry Method(s)    |
	|----------------------------------------|----------------------|
	| func() error                           | Fn                   |
	| func()(OUT, error)                     | FnOut                |
	| func(IN) error                         | FnIn, FnInRefr       |
	| func(IN) (OUT, error)                  | FnIO, FnIORefr       |
	| func(context.Context) error            | FnCtx                |
	| func(context.Context)(OUT, error)      | FnOutCtx             |
	| func(context.Context, IN) error        | FnInCtx, FnInCtxRefr |
	| func(context.Context, IN) (OUT, error) | FnIOCtx, FnIOCtxRefr |

# Retry Workflow

Functions are retried by invoking them with the appropriate package-level retry method. If the function fails, it will be run again after some delay. This process will continue until one of the following conditions occurs:
  - The function returns successfully with a nil error value.
  - The function exhausts its configured number of retries.
  - The function is halted by a [HaltFn] or [Halt] is used to manually return a fatal error.
  - The context is cancelled.
  - The refresh function, if used, fails, returning a [*RefreshError].

In the case of context cancellation, context.Cause will be called on the
context to get the underlying error, if set.
*/
package redo
