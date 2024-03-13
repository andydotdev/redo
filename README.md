![RedoGopher](.github/logo.svg)

Package redo is an ergonomic retry library for Go with decorrelated backoff.

[![PkgGoDev](https://pkg.go.dev/badge/andy.dev/redo)](https://pkg.go.dev/andy.dev/redo)

# Ergonomic?

The API is intended to be "ergonomic" in that it attempts to be intuitive to use and easy to integrate into existing code, without a lot of cognitive load.

To this end, it has the following features:
  - Declarative syntax to wrap existing functions.
  - Short, memorable retrier functions.
  - Support for functional options with sensible defaults as well as a `RetryPolicy` type to predeclare a set of options for re-use.

# Supported Function Types
The following function types are supported:

| Function Signature                       | Retry Method(s)          |
|------------------------------------------|--------------------------|
| `func() error`                           | `Fn`                     |
| `func()(OUT, error)`                     | `FnOut`                  |
| `func(IN) error`                         | `FnIn`, `FnInRefr`       |
| `func(IN) (OUT, error)`                  | `FnIO`, `FnIORefr`       |
| `func(context.Context) error`            | `FnCtx`                  |
| `func(context.Context)(OUT, error)`      | `FnOutCtx`               |
| `func(context.Context, IN) error`        | `FnInCtx`, `FnInCtxRefr` |
| `func(context.Context, IN) (OUT, error)` | `FnIOCtx`, `FnIOCtxRefr` |

# Retry Workflow
Functions are retried by invoking them with the appropriate package-level retry method. If the function fails, it will be run again after some delay. This process will continue until one of the following conditions occurs:
  - The function returns successfully with a nil error value.
  - The function exhausts its configured number of retries.
  - The function is halted by a function provided with `HaltOn` or `Halt` is used to
    manually return a fatal error.
  - The context is cancelled.
  - The refresh function, if used, fails, returning a `*RefreshError`.

In the case of context cancellation, context.Cause will be called on the
context to get the underlying error, if set.

## Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"andy.dev/redo"
)

var try = 0

func ReturnsString() (string, error) {
	try++
	if try != 2 {
		return "", fmt.Errorf("simulate an error")
	}
	return "my result", nil
}

func main() {
	policy := redo.Policy{
		InitialDelay: time.Second,
		MaxDelay:     2 * time.Minute,
		MaxTries:     5,
		Each: func(status redo.Status) {
			log.Printf("Returned error: %v (%+s)", status.Err, status)
		},
	}

	str, err := redo.FnOut(context.Background(), ReturnsString, redo.WithPolicy(policy))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Got: %s", str)
}
```
```
Returned error: simulate an error (attempt 1/5 - next in 984ms)
Got: my result
```
## Beta
The API is mostly complete; however semantics are still being experimented with. The number of unique retrier functions is too high, but this is required by the limitations of the current generic type inference algorithms. It might be possible to reduce them in the future (see the section on union interfaces below), but if that doesn't pan out I may consider splitting the package into context and contextless functions or removing one or the other class altogether to keep the API intuitive and uncluttered.

### Union Interfaces
<details>

It would be nice to unify the `-Ctx` versions of retriers with those that don't require a context using a general interface union. Unfortunately,Go's type inference is [not yet able](https://github.com/golang/go/issues/56975) to make sense of the following without explicit type parameters:

```go
type FnOutT[OUT any] interface {
    func(context.Context) (OUT, error) | func() (OUT, error)
}

func FnOut[OUT any, F FnOutT[OUT]](ctx context.Context, fn F) (OUT, error) {
    var fa any = fn
    /* ... */
}

func ToRetry(ctx context.Context) (string, error){
    return "test", nil
}

func main(){
    // The following results in an error: "cannot infer OUT"
    str, err := FnOut(context.Background(), FnOut(ToRetry))

    // This is required instead, which defeats the point:
    str, err := FnOut[string](context.Background(), FnOut(ToRetry))
}
```

It's unclear if this will be supported any time soon, since support for type switching on union interfaces is complex and [ongoing](https://github.com/golang/go/issues/45380).
</details>

### Refresh Functions
<details>
Refresh function signatures are a bit lengthy, and I may need to look into simplifying them.
</details>
