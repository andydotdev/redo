package redo_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"andy.dev/redo"
)

var ErrIDontLike = errors.New("can't recover from this one")

var testTries = 0

func maybeAFatalError() error {
	testTries++
	if testTries == 3 {
		return ErrIDontLike
	}
	return fmt.Errorf("temporary failure")
}

func ExampleErrorHandler() {
	fnToRetry := func(ctx context.Context) error {
		if err := maybeAFatalError(); err != nil {
			fmt.Printf("there was a problem: %v\n", err)
			return err
		}
		return nil
	}

	err := redo.FnCtx(context.Background(), fnToRetry, redo.MaxTries(10), redo.ErrorHandler(redo.HaltIfErrIs(ErrIDontLike)))
	if err != nil {
		fmt.Printf("output: %v\n", err)
	}

	if redo.Halted(err) {
		fmt.Println("didn't even make it to 10 tries")
	}
	// Output:
	// there was a problem: temporary failure
	// there was a problem: temporary failure
	// there was a problem: can't recover from this one
	// output: can't recover from this one
	// didn't even make it to 10 tries
}

func someFunction() error {
	return fmt.Errorf("some error")
}

func ExampleExhausted() {
	fnToRetry := func(ctx context.Context) error {
		if err := someFunction(); err != nil {
			fmt.Printf("there was a problem: %v\n", err)
			return err
		}
		return nil
	}

	err := redo.FnCtx(context.Background(), fnToRetry, redo.MaxTries(2))
	if err != nil {
		fmt.Println(err)
	}

	if redo.Exhausted(err) {
		fmt.Println("looks like that was it")
	}
	// Output:
	// there was a problem: some error
	// there was a problem: some error
	// some error
	// looks like that was it
}

type testLogger struct{}

func (testLogger) Printf(msg string, a ...any) {
	fmt.Printf(msg+"\n", a...)
}

func (testLogger) Println(a ...any) {
	fmt.Println(a...)
}

var log testLogger

func ExampleEach() {
	fnToRetry := func(ctx context.Context) error {
		if err := someFunction(); err != nil {
			return err
		}
		return nil
	}

	eachFn := func(s redo.Status) {
		log.Printf("got error while retrying: %v (%s)", s.Err, s)
	}

	err := redo.FnCtx(context.Background(), fnToRetry, redo.MaxTries(3), redo.Each(eachFn))
	if err != nil {
		log.Println(err)
	}
	// Output:
	// got error while retrying: some error (attempt 1/3)
	// got error while retrying: some error (attempt 2/3)
	// got error while retrying: some error (attempt 3/3)
	// some error
}

func ExampleFnCtx_withCancelledContextCause() {
	ctx, cf := context.WithCancelCause(context.Background())
	go func() {
		time.Sleep(1 * time.Second)
		cf(errors.New("I've changed my mind"))
	}()

	fnToRetry := func(ctx context.Context) error {
		return errors.New("I'll fail forever")
	}

	err := redo.FnCtx(ctx, fnToRetry, redo.MaxTries(10))
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// I've changed my mind
}

func ExampleFnOutCtx() {
	fnToRetry := func(ctx context.Context) (string, error) {
		status := redo.GetStatus(ctx)
		try := status.TryNumber
		val := fmt.Sprintf("value from try %d", try)
		if try < 3 {
			return "", errors.New("not yet")
		}
		return val, nil
	}

	str, err := redo.FnOutCtx(context.Background(), fnToRetry, redo.MaxTries(3))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Got: %s", str)
	// Output:
	// Got: value from try 3
}

func ExampleFnInCtx() {
	fnToRetry := func(ctx context.Context, str string) error {
		try := redo.GetStatus(ctx).TryNumber
		fmt.Printf("try %d with arg: %q\n", try, str)
		if try < 3 {
			return errors.New("not yet")
		}
		return nil
	}

	err := redo.FnInCtx(context.Background(), fnToRetry, "my argument", redo.MaxTries(3))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Success!")
	// Output:
	// try 1 with arg: "my argument"
	// try 2 with arg: "my argument"
	// try 3 with arg: "my argument"
	// Success!
}

var fetchHttpCount = 0

func fetchHttp(_ context.Context, url string) ([]byte, error) {
	fetchHttpCount++
	if fetchHttpCount < 2 {
		return nil, fmt.Errorf("HTTP error fetching %s", url)
	}
	return []byte(`{"status":"success"}`), nil
}

func ExampleFnIOCtx() {
	val, err := redo.FnIOCtx(context.Background(), fetchHttp, "http://my.site.com", redo.MaxTries(3))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s", val)
	// Output:
	// {"status":"success"}
}
