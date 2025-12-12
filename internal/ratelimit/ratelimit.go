package ratelimit

import (
	"crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
	"sync"
	"time"
)

// cryptoSeed generates a cryptographically secure random seed for math/rand.
// This ensures unique seeds even when multiple RateLimiters are created simultaneously.
func cryptoSeed() int64 {
	var seed int64
	err := binary.Read(rand.Reader, binary.BigEndian, &seed)
	if err != nil {
		// Fallback to time-based seed if crypto/rand fails (extremely rare)
		return time.Now().UnixNano()
	}
	return seed
}

// RateLimiter implements a token bucket rate limiter with sampling support.
// When the rate limit is exceeded, it samples logs based on the configured ratio.
type RateLimiter struct {
	maxLogsPerSecond float64
	samplingRatio    float64
	tokens           float64
	lastRefill       time.Time
	mu               sync.Mutex
	droppedCount     int
	sampledCount     int
	rng              *mathrand.Rand
}

// NewRateLimiter creates a new rate limiter with the specified configuration.
// maxLogsPerSecond: maximum logs allowed per second (0 to disable)
// samplingRatio: ratio of logs to keep when over limit (0.0-1.0, e.g., 0.1 = keep 1 in 10)
func NewRateLimiter(maxLogsPerSecond int, samplingRatio float64) *RateLimiter {
	if maxLogsPerSecond <= 0 {
		return nil // Rate limiting disabled
	}

	if samplingRatio < 0 {
		samplingRatio = 0
	}
	if samplingRatio > 1 {
		samplingRatio = 1
	}

	return &RateLimiter{
		maxLogsPerSecond: float64(maxLogsPerSecond),
		samplingRatio:    samplingRatio,
		tokens:           float64(maxLogsPerSecond),
		lastRefill:       time.Now(),
		rng:              mathrand.New(mathrand.NewSource(cryptoSeed())),
	}
}

// Allow checks if a log should be allowed based on rate limiting and sampling.
// Returns:
// - allowed: true if the log should be processed
// - sampled: true if the log was allowed due to sampling (not regular rate limit)
func (rl *RateLimiter) Allow() (allowed bool, sampled bool) {
	if rl == nil {
		return true, false // Rate limiting disabled
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.maxLogsPerSecond

	// Cap tokens at max rate
	if rl.tokens > rl.maxLogsPerSecond {
		rl.tokens = rl.maxLogsPerSecond
	}
	rl.lastRefill = now

	// If we have tokens, allow the log
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true, false
	}

	// No tokens available - check if we should sample
	if rl.samplingRatio > 0 && rl.rng.Float64() < rl.samplingRatio {
		rl.sampledCount++
		return true, true
	}

	// Drop the log
	rl.droppedCount++
	return false, false
}

// GetAndResetStats returns the number of dropped and sampled logs since last call,
// then resets the counters.
func (rl *RateLimiter) GetAndResetStats() (dropped int, sampled int) {
	if rl == nil {
		return 0, 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	dropped = rl.droppedCount
	sampled = rl.sampledCount
	rl.droppedCount = 0
	rl.sampledCount = 0

	return dropped, sampled
}
