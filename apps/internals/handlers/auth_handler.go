package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
	"github.com/chann44/ikyk/pkg/utils"
)

type AuthHandler struct {
	storage *redis.Client
	log     *logger.Logger
}

func NewAuthHandler(storage *redis.Client, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		storage: storage,
		log:     log,
	}
}

func (ah *AuthHandler) CreateAuthConfig(w http.ResponseWriter, r *http.Request) {
	var config types.AuthConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		utils.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := utils.ValidatePath(config.Path); err != nil {
		utils.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	key := fmt.Sprintf("auth:path:%s", config.Path)

	headersJSON, _ := json.Marshal(config.Headers)
	apiKeysJSON, _ := json.Marshal(config.APIKeys)

	err := ah.storage.HSet(ctx, key,
		"service_name", config.ServiceName,
		"path", config.Path,
		"type", config.Type,
		"enabled", fmt.Sprintf("%v", config.Enabled),
		"headers", string(headersJSON),
		"api_keys", string(apiKeysJSON),
	).Err()

	if err != nil {
		utils.ErrorResponse(w, "Failed to create auth config", http.StatusInternalServerError)
		return
	}

	ah.log.Info("auth config created", "path", config.Path)
	utils.SuccessResponse(w, "Auth config created successfully", config)
}

func (ah *AuthHandler) GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	ctx := r.Context()

	key := fmt.Sprintf("auth:path:%s", path)
	data, err := ah.storage.HGetAll(ctx, key).Result()

	if err != nil || len(data) == 0 {
		utils.ErrorResponse(w, "Auth config not found", http.StatusNotFound)
		return
	}

	config := types.AuthConfig{
		ServiceName: data["service_name"],
		Path:        data["path"],
		Type:        data["type"],
		Enabled:     data["enabled"] == "true",
	}

	if data["headers"] != "" {
		json.Unmarshal([]byte(data["headers"]), &config.Headers)
	}
	if data["api_keys"] != "" {
		json.Unmarshal([]byte(data["api_keys"]), &config.APIKeys)
	}

	utils.JSONResponse(w, config, http.StatusOK)
}

func (ah *AuthHandler) UpdateAuthConfig(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")

	var config types.AuthConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		utils.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config.Path = path
	ctx := r.Context()

	// Check if exists
	key := fmt.Sprintf("auth:path:%s", path)
	exists := ah.storage.Exists(ctx, key)
	if exists.Val() == 0 {
		utils.ErrorResponse(w, "Auth config not found", http.StatusNotFound)
		return
	}

	headersJSON, _ := json.Marshal(config.Headers)
	apiKeysJSON, _ := json.Marshal(config.APIKeys)

	ah.storage.HSet(ctx, key,
		"service_name", config.ServiceName,
		"path", config.Path,
		"type", config.Type,
		"enabled", fmt.Sprintf("%v", config.Enabled),
		"headers", string(headersJSON),
		"api_keys", string(apiKeysJSON),
	)

	ah.log.Info("auth config updated", "path", path)
	utils.SuccessResponse(w, "Auth config updated successfully", config)
}

func (ah *AuthHandler) DeleteAuthConfig(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	ctx := r.Context()

	key := fmt.Sprintf("auth:path:%s", path)
	result := ah.storage.Del(ctx, key)

	if result.Val() == 0 {
		utils.ErrorResponse(w, "Auth config not found", http.StatusNotFound)
		return
	}

	ah.log.Info("auth config deleted", "path", path)
	utils.SuccessResponse(w, "Auth config deleted successfully", nil)
}
