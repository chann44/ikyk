package internals

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
	"github.com/chann44/ikyk/pkg/types"
)

type CacheManager struct {
	storage *RedisClient
	log     *logger.Logger
	ttl     time.Duration
}

func NewCacheManager(storage *RedisClient, log *logger.Logger, ttl time.Duration) *CacheManager {
	return &CacheManager{
		storage: storage,
		log:     log,
		ttl:     ttl,
	}
}

func (cm *CacheManager) Get(r *http.Request) *types.CachedResponse {
	if r.Method != "GET" {
		return nil
	}

	key := cm.generateKey(r)
	ctx := context.Background()

	data, err := cm.storage.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var cached types.CachedResponse
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		cm.log.Error("failed to unmarshal cached response: %v", err)
		return nil
	}

	return &cached
}

func (cm *CacheManager) Set(r *http.Request, resp *http.Response) {
	if r.Method != "GET" || resp.StatusCode != http.StatusOK {
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Restore body for downstream
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	// Convert http.Header to map[string][]string for JSON
	headers := make(map[string][]string)
	for k, v := range resp.Header {
		headers[k] = v
	}

	cached := types.CachedResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
		CachedAt:   time.Now(),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		cm.log.Error("failed to marshal cached response: %v", err)
		return
	}

	key := cm.generateKey(r)
	ctx := context.Background()

	cm.storage.Set(ctx, key, data, cm.ttl)
}

func (cm *CacheManager) generateKey(r *http.Request) string {
	// Create unique key: method + path + query
	raw := r.Method + ":" + r.URL.Path + ":" + r.URL.RawQuery
	hash := sha256.Sum256([]byte(raw))
	return "cache:response:" + hex.EncodeToString(hash[:])
}
