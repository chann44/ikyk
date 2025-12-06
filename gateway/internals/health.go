package internals

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
)

type HealthChecker struct {
	registry *Registery
	log      *logger.Logger
	client   *http.Client
	mu       sync.RWMutex
	stopCh   chan struct{}
}

func NewHealthChecker(registry *Registery, log *logger.Logger) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		log:      log,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	hc.log.Info("health checker started")

	for {
		select {
		case <-ticker.C:
			hc.checkAllServices(ctx)
		case <-hc.stopCh:
			hc.log.Info("health checker stopped")
			return
		case <-ctx.Done():
			return
		}
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

func (hc *HealthChecker) checkAllServices(ctx context.Context) {
	paths, err := hc.registry.ListAllPaths(ctx)
	if err != nil {
		hc.log.Error("failed to list paths: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			hc.checkServicesForPath(ctx, p)
		}(path)
	}
	wg.Wait()
}

func (hc *HealthChecker) checkServicesForPath(ctx context.Context, path string) {
	services, err := hc.registry.GetServices(ctx, path)
	if err != nil {
		hc.log.Error("failed to get services for path %s: %v", path, err)
		return
	}

	for _, service := range services {
		healthy := hc.checkServiceHealth(service)

		// Update registry
		err := hc.registry.UpdateServiceHealth(ctx, path, service.Name, healthy, time.Now())
		if err != nil {
			hc.log.Error("failed to update health for %s: %v", service.Name, err)
		}

		if !healthy {
			hc.log.Warn("service unhealthy", "service", service.Name, "path", path)
		}
	}
}

func (hc *HealthChecker) checkServiceHealth(service *types.Service) bool {
	// Default health endpoint
	healthURL := service.URL.String() + "/health"

	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return false
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
