package internals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
)

type AuthManager struct {
	storage *RedisClient
	log     *logger.Logger
}

func NewAuthManager(storage *RedisClient, log *logger.Logger) *AuthManager {
	return &AuthManager{
		storage: storage,
		log:     log,
	}
}

// Middleware for authentication
func (am *AuthManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		path := r.URL.Path

		// Get auth config for this path
		authConfig, err := am.getAuthConfig(ctx, path)
		if err != nil || !authConfig.Enabled {
			// No auth required or error fetching config
			next.ServeHTTP(w, r)
			return
		}

		// Check cache first
		cacheKey := am.getCacheKey(r, path)
		if am.isCached(ctx, cacheKey) {
			next.ServeHTTP(w, r)
			return
		}

		// Validate based on auth type
		valid := false
		switch authConfig.Type {
		case "api_key":
			valid = am.validateAPIKey(r, authConfig)
		case "custom_header":
			valid = am.validateCustomHeaders(r, authConfig)
		default:
			valid = true
		}

		if !valid {
			am.log.Warn("authentication failed", "path", path, "type", authConfig.Type)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Cache the validation result
		am.cacheValidation(ctx, cacheKey)

		next.ServeHTTP(w, r)
	})
}

func (am *AuthManager) validateAPIKey(r *http.Request, config *types.AuthConfig) bool {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}

	for _, validKey := range config.APIKeys {
		if apiKey == validKey {
			return true
		}
	}

	return false
}

func (am *AuthManager) validateCustomHeaders(r *http.Request, config *types.AuthConfig) bool {
	for key, expectedValue := range config.Headers {
		actualValue := r.Header.Get(key)
		if actualValue != expectedValue {
			return false
		}
	}
	return true
}

func (am *AuthManager) getCacheKey(r *http.Request, path string) string {
	apiKey := r.Header.Get("X-API-Key")
	hash := sha256.Sum256([]byte(apiKey + path))
	return "auth:cache:" + hex.EncodeToString(hash[:])
}

func (am *AuthManager) isCached(ctx context.Context, key string) bool {
	exists := am.storage.Exists(ctx, key)
	return exists.Val() > 0
}

func (am *AuthManager) cacheValidation(ctx context.Context, key string) {
	am.storage.Set(ctx, key, "1", 5*time.Minute)
}

func (am *AuthManager) getAuthConfig(ctx context.Context, path string) (*types.AuthConfig, error) {
	// Find matching service for this path (longest prefix match)
	key := fmt.Sprintf("auth:path:%s", path)

	data, err := am.storage.HGetAll(ctx, key).Result()
	if err != nil || len(data) == 0 {
		return &types.AuthConfig{Enabled: false}, nil
	}

	config := &types.AuthConfig{}
	if data["enabled"] == "true" {
		config.Enabled = true
		config.Type = data["type"]

		if data["headers"] != "" {
			json.Unmarshal([]byte(data["headers"]), &config.Headers)
		}
		if data["api_keys"] != "" {
			json.Unmarshal([]byte(data["api_keys"]), &config.APIKeys)
		}
	}

	return config, nil
}

// SaveAuthConfig stores auth configuration
func (am *AuthManager) SaveAuthConfig(ctx context.Context, config *types.AuthConfig) error {
	key := fmt.Sprintf("auth:path:%s", config.Path)

	headersJSON, _ := json.Marshal(config.Headers)
	apiKeysJSON, _ := json.Marshal(config.APIKeys)

	return am.storage.HSet(ctx, key,
		"service_name", config.ServiceName,
		"path", config.Path,
		"type", config.Type,
		"enabled", fmt.Sprintf("%v", config.Enabled),
		"headers", string(headersJSON),
		"api_keys", string(apiKeysJSON),
	).Err()
}

// FindAuthConfigForPath finds the auth config for a given path using longest prefix match
func (am *AuthManager) FindAuthConfigForPath(ctx context.Context, requestPath string) (*types.AuthConfig, error) {
	// Get all auth keys
	pattern := "auth:path:*"
	keys, err := am.storage.Keys(ctx, pattern).Result()
	if err != nil {
		return &types.AuthConfig{Enabled: false}, nil
	}

	var longestMatch string
	for _, key := range keys {
		path := strings.TrimPrefix(key, "auth:path:")
		if strings.HasPrefix(requestPath, path) && len(path) > len(longestMatch) {
			longestMatch = path
		}
	}

	if longestMatch == "" {
		return &types.AuthConfig{Enabled: false}, nil
	}

	return am.getAuthConfig(ctx, longestMatch)
}
