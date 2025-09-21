package simulation

import (
	"math/rand"
	"time"
)

const FailureProb = 0.20

const (
	MinProc = 100 * time.Millisecond
	MaxProc = 500 * time.Millisecond
)

// ShouldFail возвращает true с вероятностью FailureProb
func ShouldFail(r *rand.Rand) bool {
	if r == nil {
		return false
	}
	p := FailureProb

	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}

	return r.Float64() < p
}

// SimulatedDuration возвращает псевдослучайную длительность обработки
func SimulatedDuration(r *rand.Rand) time.Duration {
	if MaxProc <= MinProc {
		return MinProc
	}
	if r == nil {
		return MinProc
	}
	span := MaxProc - MinProc
	off := time.Duration(r.Int63n(int64(span) + 1))

	return MinProc + off
}
