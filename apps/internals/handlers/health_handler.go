package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
	"github.com/chann44/ikyk/pkg/utils"
)

type HealthHandler struct {
	storage *redis.Client
	log     *logger.Logger
}

func NewHealthHandler(storage *redis.Client, log *logger.Logger) *HealthHandler {
	return &HealthHandler{
		storage: storage,
		log:     log,
	}
}

func (hh *HealthHandler) ListServiceHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all paths
	paths, err := hh.storage.SMembers(ctx, "registry:paths").Result()
	if err != nil {
		utils.ErrorResponse(w, "Failed to list service health", http.StatusInternalServerError)
		return
	}

	result := make(map[string][]types.ServiceData)

	for _, path := range paths {
		services := hh.getServicesForPath(ctx, path)
		result[path] = services
	}

	utils.JSONResponse(w, result, http.StatusOK)
}

func (hh *HealthHandler) GetServiceHealth(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	ctx := r.Context()

	// Find service across all paths
	paths, _ := hh.storage.SMembers(ctx, "registry:paths").Result()

	for _, path := range paths {
		serviceKey := redisKey("registry:path", path, "service", name)
		fields, err := hh.storage.HGetAll(ctx, serviceKey).Result()

		if err == nil && len(fields) > 0 {
			healthy, _ := strconv.ParseBool(fields["healthy"])
			lastCheck, _ := time.Parse(time.RFC3339, fields["last_check"])

			service := types.ServiceData{
				Name:      fields["name"],
				URL:       fields["url"],
				Healthy:   healthy,
				LastCheck: lastCheck,
				Path:      path,
			}

			utils.JSONResponse(w, service, http.StatusOK)
			return
		}
	}

	utils.ErrorResponse(w, "Service not found", http.StatusNotFound)
}

func (hh *HealthHandler) getServicesForPath(ctx context.Context, path string) []types.ServiceData {
	servicesKey := redisKey("registry:path", path, "services")
	serviceNames, _ := hh.storage.SMembers(ctx, servicesKey).Result()

	services := []types.ServiceData{}
	for _, name := range serviceNames {
		serviceKey := redisKey("registry:path", path, "service", name)
		fields, _ := hh.storage.HGetAll(ctx, serviceKey).Result()

		if len(fields) > 0 {
			healthy, _ := strconv.ParseBool(fields["healthy"])
			lastCheck, _ := time.Parse(time.RFC3339, fields["last_check"])

			services = append(services, types.ServiceData{
				Name:      fields["name"],
				URL:       fields["url"],
				Healthy:   healthy,
				LastCheck: lastCheck,
				Path:      path,
			})
		}
	}

	return services
}
