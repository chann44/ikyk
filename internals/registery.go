package internals

import (
	"fmt"
	"log"
	"net/http"
	"time"

	types "github.com/chann44/ikyk/types"
)

type Registery struct {
	Services map[string][]*types.Service
}

func NewRegistery() *Registery {
	return &Registery{}
}

func (r *Registery) AddService(service *types.Service) {
	r.Services = append(r.Services, service)
}

func (r *Registery) GetHealthyService(path string) *types.Service {
	for _, service := range r.Services {
		service.Mu.RLock()
		healthy := service.Healthy
		service.Mu.RUnlock()

		if healthy {
			return service
		}
	}
	return nil
}

func (r *Registery) HealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		for _, service := range r.Services {
			go r.checkServiceHealth(service)
		}
	}
}
func (r *Registery) checkServiceHealth(service *types.Service) {
	healthURL := service.URL.String() + "/health"
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(healthURL)

	service.Mu.Lock()
	defer service.Mu.Unlock()

	if err != nil || resp.StatusCode != http.StatusOK {
		service.Healthy = false
		log.Error(fmt.Sprintf("Service %s is unhealthy", service.Name))
	} else {
		service.Healthy = true
		log.Info(fmt.Sprintf("Service %s is healthy", service.Name))
	}

	if resp != nil {
		resp.Body.Close()
	}

	service.LastCheck = time.Now()
}
