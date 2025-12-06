package internals

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chann44/ikyk/pkg/logger"
)

type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half-open"
)

type CircuitBreaker struct {
	storage          *RedisClient
	log              *logger.Logger
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	mu               sync.RWMutex
}

func NewCircuitBreaker(storage *RedisClient, log *logger.Logger, failThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		storage:          storage,
		log:              log,
		failureThreshold: failThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

func (cb *CircuitBreaker) AllowRequest(serviceName string) bool {
	ctx := context.Background()
	state := cb.getState(ctx, serviceName)

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		lastFailure := cb.getLastFailureTime(ctx, serviceName)
		if time.Since(lastFailure) > cb.timeout {
			cb.setState(ctx, serviceName, StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}

	return true
}

func (cb *CircuitBreaker) RecordSuccess(serviceName string) {
	ctx := context.Background()
	state := cb.getState(ctx, serviceName)

	if state == StateHalfOpen {
		successCount := cb.incrementSuccess(ctx, serviceName)
		if successCount >= cb.successThreshold {
			cb.setState(ctx, serviceName, StateClosed)
			cb.resetCounters(ctx, serviceName)
			cb.log.Info("circuit breaker closed", "service", serviceName)
		}
	} else if state == StateClosed {
		cb.resetCounters(ctx, serviceName)
	}
}

func (cb *CircuitBreaker) RecordFailure(serviceName string) {
	ctx := context.Background()
	state := cb.getState(ctx, serviceName)

	if state == StateHalfOpen {
		cb.setState(ctx, serviceName, StateOpen)
		cb.setLastFailureTime(ctx, serviceName)
		cb.log.Warn("circuit breaker reopened", "service", serviceName)
		return
	}

	if state == StateClosed {
		failureCount := cb.incrementFailure(ctx, serviceName)
		if failureCount >= cb.failureThreshold {
			cb.setState(ctx, serviceName, StateOpen)
			cb.setLastFailureTime(ctx, serviceName)
			cb.log.Warn("circuit breaker opened", "service", serviceName, "failures", failureCount)
		}
	}
}

func (cb *CircuitBreaker) getState(ctx context.Context, serviceName string) CircuitState {
	key := fmt.Sprintf("circuit:%s:state", serviceName)
	state, err := cb.storage.Get(ctx, key).Result()
	if err != nil {
		return StateClosed
	}
	return CircuitState(state)
}

func (cb *CircuitBreaker) setState(ctx context.Context, serviceName string, state CircuitState) {
	key := fmt.Sprintf("circuit:%s:state", serviceName)
	cb.storage.Set(ctx, key, string(state), 24*time.Hour)
}

func (cb *CircuitBreaker) incrementFailure(ctx context.Context, serviceName string) int {
	key := fmt.Sprintf("circuit:%s:failures", serviceName)
	count, _ := cb.storage.Incr(ctx, key).Result()
	cb.storage.Expire(ctx, key, 10*time.Minute)
	return int(count)
}

func (cb *CircuitBreaker) incrementSuccess(ctx context.Context, serviceName string) int {
	key := fmt.Sprintf("circuit:%s:successes", serviceName)
	count, _ := cb.storage.Incr(ctx, key).Result()
	cb.storage.Expire(ctx, key, 5*time.Minute)
	return int(count)
}

func (cb *CircuitBreaker) resetCounters(ctx context.Context, serviceName string) {
	cb.storage.Del(ctx,
		fmt.Sprintf("circuit:%s:failures", serviceName),
		fmt.Sprintf("circuit:%s:successes", serviceName))
}

func (cb *CircuitBreaker) setLastFailureTime(ctx context.Context, serviceName string) {
	key := fmt.Sprintf("circuit:%s:last_failure", serviceName)
	cb.storage.Set(ctx, key, time.Now().Unix(), 24*time.Hour)
}

func (cb *CircuitBreaker) getLastFailureTime(ctx context.Context, serviceName string) time.Time {
	key := fmt.Sprintf("circuit:%s:last_failure", serviceName)
	timestamp, err := cb.storage.Get(ctx, key).Int64()
	if err != nil {
		return time.Time{}
	}
	return time.Unix(timestamp, 0)
}
