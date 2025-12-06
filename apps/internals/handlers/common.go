package handlers

import "strings"

// redisKey creates a Redis key from multiple parts
func redisKey(parts ...string) string {
	return strings.Join(parts, ":")
}
