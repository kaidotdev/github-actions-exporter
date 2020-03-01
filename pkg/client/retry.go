package client

import (
	"math/rand"
	"time"
)

type RetryStrategy interface {
	Sleep() (time.Duration, bool)
}

type NoRetry struct{}

func (nr *NoRetry) Sleep() (time.Duration, bool) {
	return 0, false
}

type Entropy func(int64) int64

type ExponentialBackOff struct {
	Base       time.Duration
	RetryCount uint
	Entropy    Entropy
	n          uint
}

func (eb *ExponentialBackOff) Sleep() (time.Duration, bool) {
	entropy := eb.getEntropy()
	if eb.n >= eb.RetryCount {
		return 0, false
	}

	d := time.Duration(entropy((1 << eb.n) * int64(eb.Base)))
	eb.n++
	return d, true
}

func (eb *ExponentialBackOff) getEntropy() Entropy {
	if eb.Entropy == nil {
		return rand.Int63n
	}
	return eb.Entropy
}
