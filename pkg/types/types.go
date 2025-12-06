package types

import (
	"net/url"
	"sync"
	"time"
)

type Service struct {
	Name      string
	URL       *url.URL
	Healthy   bool
	Mu        sync.RWMutex
	LastCheck time.Time
}

type ServiceData struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
	Path      string    `json:"path"`
}

// AuthConfig stores authentication configuration for a service
type AuthConfig struct {
	ServiceName string            `json:"service_name"`
	Path        string            `json:"path"`
	Type        string            `json:"type"` // "api_key", "custom_header", "none"
	Headers     map[string]string `json:"headers,omitempty"`
	APIKeys     []string          `json:"api_keys,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// HealthConfig defines health check parameters
type HealthConfig struct {
	ServiceName    string        `json:"service_name"`
	Path           string        `json:"path"`
	Endpoint       string        `json:"endpoint"`        // Default: /health
	Interval       time.Duration `json:"interval"`        // Default: 30s
	Timeout        time.Duration `json:"timeout"`         // Default: 5s
	HealthyCount   int           `json:"healthy_count"`   // Consecutive successes
	UnhealthyCount int           `json:"unhealthy_count"` // Consecutive failures
}

// CacheConfig defines caching rules
type CacheConfig struct {
	Enabled    bool          `json:"enabled"`
	TTL        time.Duration `json:"ttl"`
	Methods    []string      `json:"methods"`     // Default: ["GET"]
	PathPrefix string        `json:"path_prefix"`
}

// RateLimitConfig defines rate limiting rules
type RateLimitConfig struct {
	Enabled        bool   `json:"enabled"`
	RequestsPerMin int    `json:"requests_per_min"`
	BurstSize      int    `json:"burst_size"`
	Path           string `json:"path"`
}

// CircuitBreakerConfig defines circuit breaker parameters
type CircuitBreakerConfig struct {
	Enabled          bool          `json:"enabled"`
	FailureThreshold int           `json:"failure_threshold"` // Default: 5
	SuccessThreshold int           `json:"success_threshold"` // Default: 2
	Timeout          time.Duration `json:"timeout"`           // Default: 60s
}

// CachedResponse stores a cached HTTP response
type CachedResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte      `json:"body"`
	CachedAt   time.Time   `json:"cached_at"`
}
