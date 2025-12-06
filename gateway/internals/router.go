package internals

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/chann44/ikyk/pkg/logger"
)

func SetupGateway(log *logger.Logger) http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logger.LoggerMiddleware(log))

	// Initialize components
	registry := NewRegistery(log)
	metrics := NewMetricsCollector()

	redisClient, err := NewRedisClient("127.0.0.1:6379", RegistryPassowrd, RegistryDB)
	if err != nil {
		log.Error("failed to create redis client: %v", err)
	}

	cache := NewCacheManager(redisClient, log, 5*time.Minute)
	circuitBreaker := NewCircuitBreaker(redisClient, log, 5, 2, 60*time.Second)
	authManager := NewAuthManager(redisClient, log)
	rateLimiter := NewRateLimiter(redisClient, log, 100, 10)

	gateway := NewGateway(log, registry, metrics, cache, circuitBreaker)

	// Start health checker
	healthChecker := NewHealthChecker(registry, log)
	go healthChecker.Start(context.Background())

	// Metrics endpoint
	r.Handle("/metrics", metrics.Handler())

	// Health check endpoint
	r.Get("/gateway/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Proxy all other requests through middleware chain:
	// Metrics -> Rate Limit -> Auth -> Proxy
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		handler := http.HandlerFunc(gateway.ProxyHandler)

		// Apply middleware in reverse order
		handler = authManager.Middleware(handler).(http.HandlerFunc)
		handler = rateLimiter.Middleware(handler).(http.HandlerFunc)
		handler = metrics.Middleware(handler).(http.HandlerFunc)

		handler.ServeHTTP(w, r)
	})

	return r
}
