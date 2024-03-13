package backoff

import (
	"math"
	"math/rand"
	"time"
)

const (
	smoothing = 4.0
	maxintf   = float64(math.MaxInt64) - 1
)

type Iterator func() time.Duration

func New(initialMedian time.Duration, maxDelay time.Duration, firstFast bool) Iterator {
	if maxDelay < 0 {
		panic("maxDelay must not be negative")
	}
	initial := float64(initialMedian)
	maxDf := float64(maxDelay)
	var (
		prev float64
		i    int
	)
	return func() time.Duration {
		if i == 0 && firstFast {
			i++
			return 0
		}
		t := float64(i) + rand.Float64()
		i++
		next := math.Pow(2, t) * math.Tanh(math.Sqrt(smoothing*t))
		out := (next - prev) * initial
		switch {
		case maxDelay > 0 && out > maxDf:
			return maxDelay
		case out > maxintf:
			// maxintf serves as a backstop against float64->int64 overflow
			return time.Duration(math.MaxInt64)
		default:
			prev = next
			return time.Duration(out)
		}
	}
}
