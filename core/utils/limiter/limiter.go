package limiter

import (
	"sync"

	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    *sync.Mutex
	limit rate.Limit
	burst int
}

// NewIPRateLimiter new IP rate limiter
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips:   make(map[string]*rate.Limiter),
		mu:    &sync.Mutex{},
		limit: r,
		burst: b,
	}
}

// GetLimiter get limiter
func (r *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	r.mu.Lock()

	limiter, exists := r.ips[ip]
	if !exists {
		r.mu.Unlock()
		return r.addIP(ip)
	}
	r.mu.Unlock()

	return limiter
}

// addIP add IP
func (r *IPRateLimiter) addIP(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter := rate.NewLimiter(r.limit, r.burst)
	r.ips[ip] = limiter

	return limiter
}
