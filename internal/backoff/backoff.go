package backoff

import (
	"math"
	"math/rand"
	"time"
)

// Delay вычисляет задержку (backoff) перед следующей попыткой
func Delay(attempt int, base, cap time.Duration, rnd *rand.Rand) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if base <= 0 {
		base = 1
	}
	if cap < base {
		cap = base
	}

	d := safeExpDoubling(base, attempt, cap)

	if rnd == nil {
		return 0
	}

	max := float64(d)
	x := rnd.Float64() * max

	if x >= max {
		return d
	}
	return time.Duration(x)
}

// safeExpDoubling возвращает min(cap, base*2^attempt), не допуская переполнения
func safeExpDoubling(base time.Duration, attempt int, cap time.Duration) time.Duration {
	if base >= cap {
		return cap
	}
	cur := base

	for i := 0; i < attempt; i++ {
		if cur > cap/2 {
			return cap
		}
		if cur > math.MaxInt64/2 {
			return cap
		}
		cur *= 2
		if cur >= cap {
			return cap
		}
	}

	return cur
}
