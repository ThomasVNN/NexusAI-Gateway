package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/cache"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	buckets  map[string]*bucket
	mu       sync.RWMutex
	rate     int // tokens per interval
	interval time.Duration
}

// bucket holds tokens for a single client
type bucket struct {
	tokens    int
	lastCheck time.Time
}

type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int, burstSize int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     requestsPerMinute / 60, // per second
		interval: time.Minute,
	}

	// Cleanup old buckets periodically
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string, burst int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &bucket{
			tokens:    burst - 1,
			lastCheck: time.Now(),
		}
		return true
	}

	now := time.Now()
	elapsed := now.Sub(b.lastCheck)
	b.tokens += int(elapsed.Seconds()) * rl.rate
	if b.tokens > burst {
		b.tokens = burst
	}
	b.lastCheck = now

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// Middleware returns an HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use client IP as key
		key := r.RemoteAddr
		burst := 10 // default burst

		if !rl.Allow(key, burst) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanup removes old buckets
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.Sub(b.lastCheck) > time.Hour {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// CacheMiddleware creates a caching middleware
func CacheMiddleware(c *cache.TTLCache, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests
		if r.Method != "GET" {
			next.ServeHTTP(w, r)
			return
		}

		cacheKey := r.URL.Path + "?" + r.URL.RawQuery
		if val, ok := c.Get(cacheKey); ok {
			// Return cached response
			if response, ok := val.(string); ok {
				w.Header().Set("X-Cache", "HIT")
				w.Write([]byte(response))
				return
			}
		}

		// Wrap response writer to capture response
		recorder := &responseRecorder{ResponseWriter: w, body: []byte{}}

		next.ServeHTTP(recorder, r)

		// Cache the response
		if recorder.statusCode == 200 {
			c.SetWithTTL(cacheKey, string(recorder.body), 5*time.Minute)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
