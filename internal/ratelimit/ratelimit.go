package ratelimit

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter is an in-memory per-IP rate limiter using a sliding window.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]*bucket
	limit   int
	window  time.Duration
}

type bucket struct {
	timestamps []time.Time
}

// NewLimiter returns a limiter that allows up to limit requests per window per IP.
func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{
		entries: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}
}

// Allow reports whether the request from ip is within the rate limit.
// If true, the request is counted. If false, the client should get 429.
func (l *Limiter) Allow(ip string) bool {
	if ip == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	b, ok := l.entries[ip]
	if !ok {
		b = &bucket{}
		l.entries[ip] = b
	}
	// Evict timestamps outside the window
	var kept []time.Time
	for _, t := range b.timestamps {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	b.timestamps = kept
	if len(b.timestamps) >= l.limit {
		return false
	}
	b.timestamps = append(b.timestamps, now)
	return true
}

// ClientIP returns the client IP from the request, for rate limiting.
// It checks X-Forwarded-For (first hop), then X-Real-IP, then RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First IP is the client when behind a trusted proxy (e.g. Render)
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		addr = addr[:i]
	}
	return addr
}

// Middleware returns a handler that returns 429 when the client IP exceeds the limit.
func Middleware(l *Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(ClientIP(r)) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too Many Requests"))
			return
		}
		next(w, r)
	}
}

// MiddlewarePost applies the limiter only for POST requests; GET and others are not counted.
func MiddlewarePost(l *Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && !l.Allow(ClientIP(r)) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too Many Requests"))
			return
		}
		next(w, r)
	}
}
