package internals

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/chann44/ikyk/apps/internals/handlers"
	"github.com/chann44/ikyk/pkg/logger"
)

const RegistryDB = 0
const RegistryPassword = "123456"

func SetupManagementAPI(log *logger.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logger.LoggerMiddleware(log))

	// Get Redis host from environment or use default
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "127.0.0.1"
	}
	redisAddr := fmt.Sprintf("%s:6379", redisHost)

	// Initialize storage
	redisClient, err := NewRedisClient(redisAddr, RegistryPassword, RegistryDB)
	if err != nil {
		log.Error("failed to create redis client: %v", err)
		panic(fmt.Sprintf("Cannot start without Redis connection: %v", err))
	}

	// Initialize handlers with embedded client
	serviceHandler := handlers.NewServiceHandler(redisClient.Client, log)
	authHandler := handlers.NewAuthHandler(redisClient.Client, log)
	metricsHandler := handlers.NewMetricsHandler(redisClient.Client, log)
	healthHandler := handlers.NewHealthHandler(redisClient.Client, log)

	// Service management
	r.Route("/api/services", func(r chi.Router) {
		r.Get("/", serviceHandler.ListServices)
		r.Post("/", serviceHandler.CreateService)
		r.Get("/{name}", serviceHandler.GetService)
		r.Delete("/{name}", serviceHandler.DeleteService)
	})

	// Auth configuration
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/", authHandler.CreateAuthConfig)
		r.Get("/{path}", authHandler.GetAuthConfig)
		r.Put("/{path}", authHandler.UpdateAuthConfig)
		r.Delete("/{path}", authHandler.DeleteAuthConfig)
	})

	// Metrics analytics
	r.Get("/api/metrics/analytics", metricsHandler.GetAnalytics)
	r.Get("/api/metrics/services/{name}", metricsHandler.GetServiceMetrics)

	// Health status
	r.Get("/api/health/services", healthHandler.ListServiceHealth)
	r.Get("/api/health/services/{name}", healthHandler.GetServiceHealth)

	return r
}
