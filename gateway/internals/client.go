package internals

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
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
