package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
	"github.com/chann44/ikyk/pkg/utils"
)

type ServiceHandler struct {
	storage *redis.Client
	log     *logger.Logger
}

func NewServiceHandler(storage *redis.Client, log *logger.Logger) *ServiceHandler {
	return &ServiceHandler{
		storage: storage,
		log:     log,
	}
}

type CreateServiceRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Path string `json:"path"`
}

func (sh *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate inputs
	if err := utils.ValidateServiceName(req.Name); err != nil {
		utils.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := utils.ValidatePath(req.Path); err != nil {
		utils.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := utils.ValidateURL(req.URL); err != nil {
		utils.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	parsedURL, _ := url.Parse(req.URL)
	ctx := r.Context()

	// Add service to registry
	pipe := sh.storage.Pipeline()
	pipe.SAdd(ctx, "registry:paths", req.Path)
	pipe.SAdd(ctx, redisKey("registry:path", req.Path, "services"), req.Name)
	pipe.HSet(ctx, redisKey("registry:path", req.Path, "service", req.Name),
		"name", req.Name,
		"url", parsedURL.String(),
		"healthy", "true",
		"last_check", time.Now().Format(time.RFC3339))

	countKey := redisKey("registry:path", req.Path, "index")
	exists := sh.storage.Exists(ctx, countKey)
	if exists.Val() == 0 {
		pipe.Set(ctx, countKey, 0, 0)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		utils.ErrorResponse(w, "Failed to create service", http.StatusInternalServerError)
		return
	}

	sh.log.Info("service created", "name", req.Name, "path", req.Path)
	utils.SuccessResponse(w, "Service created successfully", map[string]string{
		"name": req.Name,
		"path": req.Path,
	})
}

func (sh *ServiceHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all paths
	paths, err := sh.storage.SMembers(ctx, "registry:paths").Result()
	if err != nil {
		utils.ErrorResponse(w, "Failed to list services", http.StatusInternalServerError)
		return
	}

	result := make(map[string][]types.ServiceData)

	for _, path := range paths {
		servicesKey := redisKey("registry:path", path, "services")
		serviceNames, _ := sh.storage.SMembers(ctx, servicesKey).Result()

		services := []types.ServiceData{}
		for _, name := range serviceNames {
			serviceKey := redisKey("registry:path", path, "service", name)
			fields, _ := sh.storage.HGetAll(ctx, serviceKey).Result()

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

		result[path] = services
	}

	utils.JSONResponse(w, result, http.StatusOK)
}

func (sh *ServiceHandler) GetService(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	ctx := r.Context()

	// Find service across all paths
	paths, _ := sh.storage.SMembers(ctx, "registry:paths").Result()

	for _, path := range paths {
		serviceKey := redisKey("registry:path", path, "service", name)
		fields, err := sh.storage.HGetAll(ctx, serviceKey).Result()

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

func (sh *ServiceHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	ctx := r.Context()

	// Find and delete service across all paths
	paths, _ := sh.storage.SMembers(ctx, "registry:paths").Result()

	for _, path := range paths {
		servicesKey := redisKey("registry:path", path, "services")
		serviceKey := redisKey("registry:path", path, "service", name)

		pipe := sh.storage.Pipeline()
		pipe.SRem(ctx, servicesKey, name)
		pipe.Del(ctx, serviceKey)
		_, err := pipe.Exec(ctx)

		if err == nil {
			// Check if path has no more services
			count := sh.storage.SCard(ctx, servicesKey)
			if count.Val() == 0 {
				sh.storage.SRem(ctx, "registry:paths", path)
				sh.storage.Del(ctx, redisKey("registry:path", path, "index"))
			}

			sh.log.Info("service deleted", "name", name, "path", path)
			utils.SuccessResponse(w, "Service deleted successfully", nil)
			return
		}
	}

	utils.ErrorResponse(w, "Service not found", http.StatusNotFound)
}
