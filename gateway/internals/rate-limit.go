package internals

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
)

type RateLimiter struct {
	storage           *RedisClient
	log               *logger.Logger
	requestsPerMinute int
	burstSize         int
}

func NewRateLimiter(storage *RedisClient, log *logger.Logger, rpm, burst int) *RateLimiter {
	return &RateLimiter{
		storage:           storage,
		log:               log,
		requestsPerMinute: rpm,
		burstSize:         burst,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !rl.allowRequest(ctx, r) {
			rl.log.Warn("rate limit exceeded", "ip", r.RemoteAddr, "path", r.URL.Path)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allowRequest(ctx context.Context, r *http.Request) bool {
	key := rl.getKey(r)

	// Get current count
	count, err := rl.storage.Incr(ctx, key).Result()
	if err != nil {
		rl.log.Error("rate limit error: %v", err)
		return true // Fail open
	}

	// Set expiry on first request
	if count == 1 {
		rl.storage.Expire(ctx, key, time.Minute)
	}

	return int(count) <= rl.requestsPerMinute
}

func (rl *RateLimiter) getKey(r *http.Request) string {
	ip := r.RemoteAddr
	path := r.URL.Path
	minute := time.Now().Unix() / 60

	return fmt.Sprintf("ratelimit:%s:%s:%d", ip, path, minute)
}
