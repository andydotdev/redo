package redo

import (
	"fmt"
	"testing"
	"time"
)

func TestShortNext(t *testing.T) {
	test := func(dur, want string) {
		t.Helper()
		dv, err := time.ParseDuration(dur)
		if err != nil {
			t.Fatalf("ivalid duration %q: %v", dur, err)
		}
		got := fmt.Sprintf("%v", shortNext(dv))
		if got != want {
			t.Errorf("want: %s, got %s", want, got)
		}
	}
	test("0.5s", "500ms")
	test("0.4s", "400ms")
	test("1.4s", "1s")
	test("1.90s", "2s")
	test("66.3s", "1m6s")
	test("3661.3s", "1h1m1s")
}
