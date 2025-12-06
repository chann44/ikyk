package internals

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
)

type Gateway struct {
	registry       *Registery
	log            *logger.Logger
	metrics        *MetricsCollector
	cache          *CacheManager
	circuitBreaker *CircuitBreaker
}

func NewGateway(log *logger.Logger, registry *Registery, metrics *MetricsCollector, cache *CacheManager, cb *CircuitBreaker) *Gateway {
	return &Gateway{
		registry:       registry,
		log:            log,
		metrics:        metrics,
		cache:          cache,
		circuitBreaker: cb,
	}
}

func (g *Gateway) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path
	ctx := r.Context()

	// Find matching service path (longest prefix match)
	servicePath, err := g.findServicePath(ctx, path)
	if err != nil {
		g.log.Error("path not found: %v", err)
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	// Check cache for GET requests
	if r.Method == "GET" {
		if cached := g.cache.Get(r); cached != nil {
			g.metrics.RecordCacheHit(servicePath)
			g.serveCachedResponse(w, cached)
			return
		}
	}

	// Get next healthy service (round-robin)
	service, err := g.registry.GetNextService(ctx, servicePath)
	if err != nil {
		g.log.Error("no healthy service found: %v", err)
		g.metrics.RecordError(servicePath, "no_healthy_service")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Check circuit breaker
	if !g.circuitBreaker.AllowRequest(service.Name) {
		g.log.Warn("circuit breaker open for service: %s", service.Name)
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	// Strip service prefix from path
	targetPath := g.stripPrefix(path, servicePath)

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(service.URL)
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Cache successful GET responses
		if r.Method == "GET" && resp.StatusCode == http.StatusOK {
			g.cache.Set(r, resp)
		}

		// Record metrics
		duration := time.Since(start)
		g.metrics.RecordRequest(service.Name, r.Method, resp.StatusCode, duration)

		// Update circuit breaker
		if resp.StatusCode >= 500 {
			g.circuitBreaker.RecordFailure(service.Name)
		} else {
			g.circuitBreaker.RecordSuccess(service.Name)
		}

		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		g.log.Error("proxy error: %v", err)
		g.metrics.RecordError(service.Name, "proxy_error")
		g.circuitBreaker.RecordFailure(service.Name)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Modify request
	r.URL.Path = targetPath
	r.URL.Host = service.URL.Host
	r.URL.Scheme = service.URL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)
	r.Host = service.URL.Host

	g.log.Info("proxying request",
		"service", service.Name,
		"path", path,
		"target", targetPath)

	proxy.ServeHTTP(w, r)
}

func (g *Gateway) findServicePath(ctx context.Context, requestPath string) (string, error) {
	paths, err := g.registry.ListAllPaths(ctx)
	if err != nil {
		return "", err
	}

	// Longest prefix match
	var matched string
	for _, path := range paths {
		if len(requestPath) >= len(path) && requestPath[:len(path)] == path {
			if len(path) > len(matched) {
				matched = path
			}
		}
	}

	if matched == "" {
		return "", ErrPathNotFound
	}

	return matched, nil
}

func (g *Gateway) stripPrefix(fullPath, prefix string) string {
	if len(fullPath) > len(prefix) {
		return fullPath[len(prefix):]
	}
	return "/"
}

func (g *Gateway) serveCachedResponse(w http.ResponseWriter, cached *types.CachedResponse) {
	for k, v := range cached.Headers {
		w.Header()[k] = v
	}
	w.Header().Set("X-Cache", "HIT")
	w.WriteHeader(cached.StatusCode)
	w.Write(cached.Body)
}
