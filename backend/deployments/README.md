# Architecture Platform Deployment

## Overview

Complete microservices deployment configuration for the Architecture Platform.

## Services Architecture

```
                    ┌──────────────────────────────────────────────────────────────┐
                    │                        Kong API Gateway                      │
                    │                    (Port 8000/8443/8001)                     │
                    └──────────────┬───────────────────────────────────────────────┘
                                   │
           ┌───────────────────────┼───────────────────────┐
           │                       │                       │
           ▼                       ▼                       ▼
┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│  Java Services   │   │   Go Services    │   │   Infrastructure  │
│                  │   │                  │   │                  │
│ • user-service   │   │ • collaboration  │   │ • PostgreSQL     │
│ • project-svc    │   │ • geometry-svc   │   │ • Redis          │
│ • property-svc   │   │ • script-svc     │   │ • Kafka          │
│ • version-svc    │   │ • file-svc       │   │ • Elasticsearch  │
│ • search-svc     │   │ • notification   │   │ • MinIO          │
│                  │   │ • analytics-svc  │   │ • Temporal       │
└──────────────────┘   └──────────────────┘   └──────────────────┘
```

## Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 8GB+ RAM available

### Environment Setup

1. Copy environment file:
```bash
cp .env.example .env
# Edit .env with your configuration
```

2. Generate JWT secret:
```bash
openssl rand -base64 32
```

### Deployment Commands

```bash
# Start all services
docker-compose -f docker-compose.full.yml up -d

# Start specific service
docker-compose -f docker-compose.full.yml up -d user-service

# View logs
docker-compose -f docker-compose.full.yml logs -f user-service

# Stop all services
docker-compose -f docker-compose.full.yml down

# Stop and remove volumes (WARNING: data loss)
docker-compose -f docker-compose.full.yml down -v
```

## Service Endpoints

| Service | Internal Port | Kong Path | Health |
|---------|---------------|-----------|--------|
| User Service | 8081 | /api/v1/users | /actuator/health |
| Project Service | 8082 | /api/v1/projects | /actuator/health |
| Property Service | 8083 | /api/v1/properties | /actuator/health |
| Version Service | 8084 | /api/v1/versions | /actuator/health |
| Search Service | 8089 | /api/v1/search | /actuator/health |
| Collaboration | 8081 | /api/v1/collaboration | /health |
| Geometry Service | 8082 | /api/v1/geometry | /health |
| Script Service | 8085 | /api/v1/scripts | /health |
| File Service | 8086 | /api/v1/files | /health |
| Notification | 8087 | /api/v1/notifications | /health |
| Analytics Service | 8090 | /api/v1/analytics | /health |

## Infrastructure Access

| Service | URL | Default Credentials |
|---------|-----|---------------------|
| Kong Admin | http://localhost:8001 | - |
| Kong Proxy | http://localhost:8000 | - |
| PostgreSQL | localhost:5432 | postgres/postgres |
| Redis | localhost:6379 | - |
| Kafka | localhost:9092 | - |
| MinIO Console | http://localhost:9001 | minioadmin/minioadmin |
| Elasticsearch | http://localhost:9200 | - |
| Temporal UI | http://localhost:8233 | - |
| Prometheus | http://localhost:9090 | - |
| Grafana | http://localhost:3000 | admin/admin |

## Monitoring

### Prometheus Metrics

All services expose metrics at:
- Java services: `/actuator/prometheus`
- Go services: `/metrics`

### Grafana Dashboards

Default dashboards available for:
- JVM metrics (Java services)
- Go runtime metrics
- Database metrics
- Infrastructure overview

## Security

### Kong Plugins Enabled

1. **JWT Authentication** - Token validation at gateway
2. **Rate Limiting** - Per-service limits via Redis
3. **CORS** - Cross-origin request handling
4. **Request/Response Transformer** - Headers manipulation

### Security Headers

All responses include:
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- X-XSS-Protection: 1; mode=block
- Strict-Transport-Security: max-age=31536000

## Troubleshooting

### Service Won't Start

```bash
# Check service logs
docker-compose -f docker-compose.full.yml logs user-service

# Check dependency health
docker-compose -f docker-compose.full.yml ps
```

### Database Connection Issues

```bash
# Test PostgreSQL connection
docker exec -it archplatform-postgres psql -U postgres -d archplatform
```

### Kong Configuration Reload

```bash
# Validate configuration
docker exec archplatform-kong kong config parse /kong/declarative/kong.yml

# Restart Kong
docker-compose -f docker-compose.full.yml restart kong
```

## Production Deployment

### Checklist

- [ ] Change default passwords
- [ ] Enable TLS/SSL certificates
- [ ] Configure external load balancer
- [ ] Set up log aggregation (ELK/Loki)
- [ ] Configure backup strategy
- [ ] Enable distributed tracing
- [ ] Set up alerting rules
- [ ] Review rate limits

### Scaling

```yaml
# Example: Scale project service
deploy:
  replicas: 3
  resources:
    limits:
      cpus: '2'
      memory: 2G
```
