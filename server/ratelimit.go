package server

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	count int
	reset time.Time
}

type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	now       func() time.Time
	lastPrune time.Time
}

func NewRateLimiter() *RateLimiter {
	n := time.Now()
	return &RateLimiter{buckets: map[string]*bucket{}, now: time.Now, lastPrune: n}
}

func (rl *RateLimiter) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	n := rl.now()
	if n.Sub(rl.lastPrune) >= window {
		rl.pruneLocked(n)
		rl.lastPrune = n
	}
	b := rl.buckets[key]
	if b == nil || !n.Before(b.reset) {
		b = &bucket{reset: n.Add(window)}
		rl.buckets[key] = b
	}
	if b.count >= limit {
		wait := b.reset.Sub(n).Round(time.Second)
		if wait < time.Second {
			wait = time.Second
		}
		return false, wait
	}
	b.count++
	return true, 0
}

func (rl *RateLimiter) pruneLocked(now time.Time) {
	for k, b := range rl.buckets {
		if !now.Before(b.reset) {
			delete(rl.buckets, k)
		}
	}
}

// connectionKey intentionally strips ephemeral source ports. Behind a Tor onion
// service the backend sees loopback connections with changing ports; including
// those ports creates both a memory leak and a trivial rate-limit bypass.
// In onion deployment this effectively becomes per-endpoint/global for the
// backend, which is safer than per-port accounting until Tor-circuit identity is
// exposed to the application layer.
func connectionKey(r *http.Request, endpoint string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		host = r.RemoteAddr
	}
	return endpoint + ":" + host
}
