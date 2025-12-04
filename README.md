# IKYK API Gateway

API Gateway for Me

## Todo

### Core Protection
- [ ] Rate limiting (token bucket)
- [ ] Circuit breaker
- [ ] JWT authentication
- [ ] API key validation

### Performance
- [ ] Connection pooling
- [ ] Response caching (Redis)
- [ ] Request timeouts
- [ ] Retry logic

### Observability
- [ ] Prometheus metrics
- [ ] Distributed tracing
- [ ] Better logging

### Nice to Have
- [ ] Request transformation
- [ ] CORS handling
- [ ] Admin dashboard

## Quick Start

```bash
go run cmd/gateway/main.go
```

## Environment Variables

```bash
PORT=8080
DATABASE_URL=postgresql://user:pass@localhost:5432/db
REDIS_URL=redis://localhost:6379

USERS_SERVICE_URL=http://localhost:8081
IDENTITY_SERVICE_URL=http://localhost:8082
ONBOARDING_SERVICE_URL=http://localhost:8083
TRANSACTION_SERVICE_URL=http://localhost:8084
ADMIN_SERVICE_URL=http://localhost:8085
LEADERBOARD_SERVICE_URL=http://localhost:8086
REWARDS_SERVICE_URL=http://localhost:8087
```

## Status

Check gateway health: `GET /gateway/status`
