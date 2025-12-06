package internals

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chann44/ikyk/logger"
	types "github.com/chann44/ikyk/types"
)

const RegistryDB = 0
const RegistryPassowrd = "123456"

type ServiceData struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
}

var (
	ErrPathNotFound         = errors.New("path not registered")
	ErrServiceNotFound      = errors.New("service not found for path")
	ErrNoServicesForPath    = errors.New("no services registered for path")
	ErrAllServicesUnhealthy = errors.New("all services unhealthy for path")
	ErrInvalidPath          = errors.New("invalid path format")
	ErrInvalidService       = errors.New("invalid service data")
)

type Registery struct {
	log     *logger.Logger
	storage *RedisClient
}

func NewRegistery(log *logger.Logger) *Registery {
	storage, err := NewRedisClient("127.0.0.1:6379", RegistryPassowrd, RegistryDB)
	if err != nil {
		log.Error("failed to create redis client: %v", err)
	}
	return &Registery{
		log:     log,
		storage: storage,
	}
}

func redisKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func serviceToData(service *types.Service) *ServiceData {
	return &ServiceData{
		Name:      service.Name,
		URL:       service.URL.String(),
		Healthy:   service.Healthy,
		LastCheck: service.LastCheck,
	}
}

func dataToService(data *ServiceData) (*types.Service, error) {
	parsedURL, err := url.Parse(data.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid service URL: %w", err)
	}

	return &types.Service{
		Name:      data.Name,
		URL:       parsedURL,
		Healthy:   data.Healthy,
		LastCheck: data.LastCheck,
	}, nil
}

func (r *Registery) AddService(ctx context.Context, path string, service *types.Service) error {
	if !strings.HasPrefix(path, "/") {
		return ErrInvalidPath
	}
	if service == nil || service.URL == nil {
		return ErrInvalidService
	}

	serviceData := serviceToData(service)

	pipe := r.storage.Pipeline()
	pipe.SAdd(ctx, "registry:paths", path)
	pipe.SAdd(ctx, redisKey("registry:path", path, "services"), service.Name)
	pipe.HSet(ctx, redisKey("registry:path", path, "service", service.Name),
		"name", serviceData.Name,
		"url", serviceData.URL,
		"healthy", strconv.FormatBool(serviceData.Healthy),
		"last_check", serviceData.LastCheck.Format(time.RFC3339))

	countKey := redisKey("registry:path", path, "index")
	exists := r.storage.Exists(ctx, countKey)
	if exists.Val() == 0 {
		pipe.Set(ctx, countKey, 0, 0)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add service: %w", err)
	}

	r.log.Info("service %s added to path %s", service.Name, path)
	return nil
}

func (r *Registery) RemoveService(ctx context.Context, path, serviceName string) error {
	servicesKey := redisKey("registry:path", path, "services")
	serviceKey := redisKey("registry:path", path, "service", serviceName)

	pipe := r.storage.Pipeline()
	pipe.SRem(ctx, servicesKey, serviceName)
	pipe.Del(ctx, serviceKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove service: %w", err)
	}

	count := r.storage.SCard(ctx, servicesKey)
	if count.Val() == 0 {
		pipe := r.storage.Pipeline()
		pipe.SRem(ctx, "registry:paths", path)
		pipe.Del(ctx, redisKey("registry:path", path, "index"))
		_, err := pipe.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to cleanup path: %w", err)
		}
		r.log.Info("path %s removed (no services left)", path)
	}

	r.log.Info("service %s removed from path %s", serviceName, path)
	return nil
}

func (r *Registery) GetServices(ctx context.Context, path string) ([]*types.Service, error) {
	servicesKey := redisKey("registry:path", path, "services")
	serviceNames, err := r.storage.SMembers(ctx, servicesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get service names: %w", err)
	}

	if len(serviceNames) == 0 {
		return []*types.Service{}, nil
	}

	services := make([]*types.Service, 0, len(serviceNames))
	for _, name := range serviceNames {
		serviceKey := redisKey("registry:path", path, "service", name)
		fields, err := r.storage.HGetAll(ctx, serviceKey).Result()
		if err != nil {
			r.log.Error("failed to get service %s: %v", name, err)
			continue
		}

		healthy, _ := strconv.ParseBool(fields["healthy"])
		lastCheck, _ := time.Parse(time.RFC3339, fields["last_check"])

		serviceData := &ServiceData{
			Name:      fields["name"],
			URL:       fields["url"],
			Healthy:   healthy,
			LastCheck: lastCheck,
		}

		service, err := dataToService(serviceData)
		if err != nil {
			r.log.Error("failed to parse service %s: %v", name, err)
			continue
		}

		services = append(services, service)
	}

	return services, nil
}

func (r *Registery) GetNextService(ctx context.Context, path string) (*types.Service, error) {
	servicesKey := redisKey("registry:path", path, "services")
	serviceNames, err := r.storage.SMembers(ctx, servicesKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get service names: %w", err)
	}

	if len(serviceNames) == 0 {
		return nil, ErrNoServicesForPath
	}

	indexKey := redisKey("registry:path", path, "index")
	maxAttempts := len(serviceNames) + 1

	for attempts := 0; attempts < maxAttempts; attempts++ {
		index, err := r.storage.Incr(ctx, indexKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to increment index: %w", err)
		}

		position := int(index-1) % len(serviceNames)
		serviceName := serviceNames[position]

		serviceKey := redisKey("registry:path", path, "service", serviceName)
		fields, err := r.storage.HGetAll(ctx, serviceKey).Result()
		if err != nil {
			r.log.Error("failed to get service %s: %v", serviceName, err)
			continue
		}

		healthy, _ := strconv.ParseBool(fields["healthy"])
		if !healthy {
			continue
		}

		lastCheck, _ := time.Parse(time.RFC3339, fields["last_check"])
		serviceData := &ServiceData{
			Name:      fields["name"],
			URL:       fields["url"],
			Healthy:   healthy,
			LastCheck: lastCheck,
		}

		service, err := dataToService(serviceData)
		if err != nil {
			r.log.Error("failed to parse service %s: %v", serviceName, err)
			continue
		}

		return service, nil
	}

	return nil, ErrAllServicesUnhealthy
}

func (r *Registery) ListAllPaths(ctx context.Context) ([]string, error) {
	paths, err := r.storage.SMembers(ctx, "registry:paths").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list paths: %w", err)
	}

	sort.Strings(paths)
	return paths, nil
}

func (r *Registery) UpdateServiceHealth(ctx context.Context, path, serviceName string, healthy bool, lastCheck time.Time) error {
	serviceKey := redisKey("registry:path", path, "service", serviceName)

	exists := r.storage.Exists(ctx, serviceKey)
	if exists.Val() == 0 {
		return ErrServiceNotFound
	}

	err := r.storage.HSet(ctx, serviceKey,
		"healthy", strconv.FormatBool(healthy),
		"last_check", lastCheck.Format(time.RFC3339)).Err()

	if err != nil {
		return fmt.Errorf("failed to update service health: %w", err)
	}

	r.log.Info("service %s health updated: healthy=%v", serviceName, healthy)
	return nil
}
