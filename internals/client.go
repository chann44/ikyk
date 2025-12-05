package internals

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/chann44/ikyk/logger"
)

type Gateway struct {
	registry *Registery
	mu       sync.RWMutex
	log      *logger.Logger
}

func NewGateway(logger *logger.Logger, registry *Registery) *Gateway {
	return &Gateway{
		registry: registry,
		log:      logger,
	}
}

func (g *Gateway) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	var servicePath string
	g.mu.RLock()
	for route := range g.routes {
		if len(path) >= len(route) && path[:len(route)] == route {
			servicePath = route
			break
		}
	}
	g.mu.RUnlock()

	if servicePath == "" {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(service.URL)

	// Strip the service prefix from the path
	originalPath := r.URL.Path
	if len(originalPath) > len(servicePath) {
		r.URL.Path = originalPath[len(servicePath):]
	} else {
		r.URL.Path = "/"
	}

	r.URL.Host = service.URL.Host
	r.URL.Scheme = service.URL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = service.URL.Host

	log.Printf("Proxying %s %s to %s (service path: %s)", r.Method, originalPath, service.Name, r.URL.Path)

	proxy.ServeHTTP(w, r)
}

func (g *Gateway) GatewayStatusHandler(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	status := make(map[string][]map[string]interface{})

	for route, services := range g.routes {
		serviceList := make([]map[string]interface{}, 0)
		for _, service := range services {
			service.mu.RLock()
			serviceList = append(serviceList, map[string]interface{}{
				"name":       service.Name,
				"url":        service.URL.String(),
				"healthy":    service.Healthy,
				"last_check": service.lastCheck,
			})
			service.mu.RUnlock()
		}
		status[route] = serviceList
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func SetupGateway() http.Handler {
	gateway := NewGateway()

	usersService, _ := url.Parse(getEnvOrDefault("USERS_SERVICE_URL", "http://localhost:8081"))
	identityService, _ := url.Parse(getEnvOrDefault("IDENTITY_SERVICE_URL", "http://localhost:8082"))
	onboardingService, _ := url.Parse(getEnvOrDefault("ONBOARDING_SERVICE_URL", "http://localhost:8083"))
	transactionService, _ := url.Parse(getEnvOrDefault("TRANSACTION_SERVICE_URL", "http://localhost:8084"))
	adminService, _ := url.Parse(getEnvOrDefault("ADMIN_SERVICE_URL", "http://localhost:8085"))
	leaderboardService, _ := url.Parse(getEnvOrDefault("LEADERBOARD_SERVICE_URL", "http://localhost:8086"))
	rewardsService, _ := url.Parse(getEnvOrDefault("REWARDS_SERVICE_URL", "http://localhost:8087"))

	// Register routes
	gateway.AddRoute("/users", []*Service{
		{Name: "users-service", URL: usersService, Healthy: true},
	})

	gateway.AddRoute("/identity", []*Service{
		{Name: "identity-service", URL: identityService, Healthy: true},
	})

	gateway.AddRoute("/onboarding", []*Service{
		{Name: "onboarding-service", URL: onboardingService, Healthy: true},
	})

	gateway.AddRoute("/transaction", []*Service{
		{Name: "transaction-service", URL: transactionService, Healthy: true},
	})

	gateway.AddRoute("/admin", []*Service{
		{Name: "admin-service", URL: adminService, Healthy: true},
	})

	gateway.AddRoute("/leaderboard", []*Service{
		{Name: "leaderboard-service", URL: leaderboardService, Healthy: true},
	})

	gateway.AddRoute("/rewards", []*Service{
		{Name: "rewards-service", URL: rewardsService, Healthy: true},
	})

	// Start health check in background
	go gateway.HealthCheck(30 * time.Second)

	r := chi.NewRouter()

	r.Use(logger.LoggerMiddleware(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Get("/gateway/status", gateway.GatewayStatusHandler)

	return r

}
