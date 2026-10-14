<div align="center">

<img src="assets/logo.png" width="200" alt="ikyk">

# ikyk

**A Go API gateway with rate limiting, auth, caching, and observability baked in.**

[![Go](https://img.shields.io/badge/go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Redis](https://img.shields.io/badge/cache-Redis-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/deploy-Docker%20Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![Prometheus](https://img.shields.io/badge/metrics-Prometheus-E6522C?logo=prometheus&logoColor=white)](https://prometheus.io/)
[![License](https://img.shields.io/badge/license-MIT-21bb4a.svg)](#license)

*Route traffic. Break circuits. Watch it all in Grafana.*

</div>

## Features

- **Rate limiting** — token bucket per client, backed by Redis
- **Circuit breaker** — trips after repeated failures, recovers automatically
- **Auth** — request authentication via the Redis-backed auth manager
- **Response caching** — Redis cache with TTL
- **Service registry** — services register themselves, a health checker evicts the dead ones
- **Observability** — Prometheus metrics, Loki logs, provisioned Grafana dashboard

## Quick Start

Bring up the whole stack — gateway, management API, Redis, Prometheus, Loki, and Grafana:

```bash
docker compose up --build
```

Or run the services directly in development:

```bash
go run gateway/cmd/cli/main.go
go run apps/cmd/main.go
```

## Monorepo Layout

| Path        | Purpose                                          |
| ----------- | ------------------------------------------------ |
| `gateway/`  | The gateway service — routing, limits, metrics   |
| `apps/`     | Management API for services and dashboard        |
| `pkg/`      | Shared logger, types, and utils                  |

## Configuration

| Variable         | Default    | Purpose                          |
| ---------------- | ---------- | -------------------------------- |
| `PORT`           | `8080`     | HTTP port to listen on           |
| `ENVIRONMENT`    | `dev`      | `dev` or `production` (logging)  |
| `REDIS_HOST`     | `127.0.0.1`| Redis host                       |
| `REDIS_PORT`     | `6379`     | Redis port                       |
| `REDIS_PASSWORD` | —          | Redis password, if required      |
| `LOKI_URL`       | —          | Loki endpoint for log shipping   |

## Observability

Every component is wired up by `docker compose`:

| Service         | Port | Notes                          |
| --------------- | ---- | ------------------------------ |
| Gateway         | 8080 | `GET /gateway/health`, `/gateway/status`, `/metrics` |
| Management API  | 9090 | Service and API management     |
| Grafana         | 3001 | `admin` / `admin`, pre-provisioned dashboard |
| Prometheus      | 9091 | Scrapes gateway metrics        |
| Loki            | 3100 | Receives structured logs       |

## License

[MIT](#license) © 2026 [Chann](https://github.com/chann44)
