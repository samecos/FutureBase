# Architecture Platform Backend

[![CI](https://github.com/archplatform/backend/actions/workflows/ci.yml/badge.svg)](https://github.com/archplatform/backend/actions/workflows/ci.yml)
[![CD](https://github.com/archplatform/backend/actions/workflows/cd.yml/badge.svg)](https://github.com/archplatform/backend/actions/workflows/cd.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> Enterprise-grade microservices platform for architectural design collaboration

## 📋 Overview

Architecture Platform is a comprehensive SaaS solution for architectural design teams, providing real-time collaboration, version control, and advanced geometry processing capabilities.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kong API Gateway                          │
│         (JWT Auth, Rate Limiting, CORS, Security Headers)       │
└─────────────────────────────────────────────────────────────────┘
                              │
    ┌─────────────┬───────────┼───────────┬─────────────┐
    ▼             ▼           ▼           ▼             ▼
┌─────────┐  ┌─────────┐ ┌─────────┐ ┌─────────┐  ┌─────────┐
│  Java   │  │   Go    │ │   Go    │ │   Java  │  │   Go    │
│ Services│  │Services │ │Services │ │Services │  │Services │
├─────────┤  ├─────────┤ ├─────────┤ ├─────────┤  ├─────────┤
│•User    │  │•Collab  │ │•Script  │ │•Project │  │•File    │
│•Project │  │•Geometry│ │•Notif   │ │•Property│  │•Notif   │
│•Property│  │•File    │ │•Analytics│ │•Version │  │•Analytics│
│•Version │  │         │ │         │ │•Search  │  │         │
│•Search  │  │         │ │         │ │         │  │         │
└─────────┘  └─────────┘ └─────────┘ └─────────┘  └─────────┘
                              │
    ┌─────────────┬───────────┼───────────┬─────────────┐
    ▼             ▼           ▼           ▼             ▼
┌─────────┐  ┌─────────┐ ┌─────────┐ ┌─────────┐  ┌─────────┐
│PostgreSQL│  │  Redis  │ │  Kafka  │ │  MinIO  │  │Temporal │
│+PostGIS │  │         │ │         │ │(Object) │  │(Workflow│
└─────────┘  └─────────┘ └─────────┘ └─────────┘  └─────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- Make
- JDK 17 (for Java development)
- Go 1.21+ (for Go development)

### Local Development

```bash
# Clone the repository
git clone https://github.com/archplatform/backend.git
cd backend

# Start infrastructure
docker-compose -f deployments/docker/docker-compose.full.yml up -d

# Build all services
make build

# Run tests
make test

# Check service health
make health-check
```

## 📦 Services

### Java Services (Spring Boot)

| Service | Port | Description | Key Features |
|---------|------|-------------|--------------|
| **user-service** | 8081 | User management | JWT auth, MFA (TOTP), RBAC |
| **project-service** | 8082 | Project management | CRUD, member roles, locking |
| **property-service** | 8083 | Property engine | MVEL rules, unit conversion |
| **version-service** | 8084 | Version control | Git-like branching, merge conflicts |
| **search-service** | 8089 | Full-text search | Elasticsearch, aggregations |

### Go Services

| Service | Port | Description | Key Features |
|---------|------|-------------|--------------|
| **collaboration-service** | 8081 | Real-time collaboration | Yjs CRDT, WebSocket |
| **geometry-service** | 8082 | Geometry processing | PostGIS, boolean operations |
| **script-service** | 8085 | Script execution | Python sandbox, gVisor |
| **file-service** | 8086 | File management | MinIO, multipart upload |
| **notification-service** | 8087 | Notifications | WebSocket, email, webhooks |
| **analytics-service** | 8090 | Analytics | Event tracking, ClickHouse |

## 🛠️ Development

### Build

```bash
# Build all services
make build

# Build Java services only
make build-java

# Build Go services only
make build-go

# Build Docker images
make docker-build VERSION=latest
```

### Testing

```bash
# Run all tests
make test

# Test Java services
make test-java

# Test Go services
make test-go
```

### Code Quality

```bash
# Run linters
make lint

# Format Go code
make fmt-go

# Clean build artifacts
make clean
```

## 🚢 Deployment

### Docker Compose (Development)

```bash
# Start all services
make docker-compose-up

# View logs
make docker-compose-logs

# Stop services
make docker-compose-down
```

### Kubernetes (Production)

```bash
# Deploy to development
make k8s-deploy

# Check status
make k8s-status

# View logs
make k8s-logs SERVICE=user-service

# Port forward
make k8s-port-forward SERVICE=user-service PORT=8081
```

## 📊 Monitoring

### Access Points

| Service | URL | Credentials |
|---------|-----|-------------|
| Kong Gateway | http://localhost:8000 | - |
| Kong Admin | http://localhost:8001 | - |
| Grafana | http://localhost:3000 | admin/admin |
| Prometheus | http://localhost:9090 | - |
| API Docs | http://localhost:8080 | - |

## 🔐 Security

- **Authentication**: JWT tokens with refresh mechanism
- **Authorization**: RBAC with 5 role levels (OWNER, ADMIN, EDITOR, VIEWER, GUEST)
- **MFA**: TOTP-based multi-factor authentication
- **Rate Limiting**: Per-service limits via Kong + Redis
- **Encryption**: BCrypt for passwords, TLS for transport

## 📚 API Documentation

```bash
# Serve OpenAPI documentation locally
make api-docs
```

Or visit: https://api.archplatform.com/docs

## 📁 Project Structure

```
backend/
├── services/              # Microservices
│   ├── user-service/     # Java - User management
│   ├── project-service/  # Java - Project management
│   ├── property-service/ # Java - Property engine
│   ├── version-service/  # Java - Version control
│   ├── search-service/   # Java - Search
│   ├── collaboration/    # Go - Real-time collab
│   ├── geometry-service/ # Go - Geometry ops
│   ├── script-service/   # Go - Script execution
│   ├── file-service/     # Go - File management
│   ├── notification/     # Go - Notifications
│   └── analytics/        # Go - Analytics
├── shared/               # Shared components
│   ├── proto/           # Protocol Buffers
│   ├── models/          # Generated models
│   └── utils/           # Common utilities
├── deployments/         # Deployment configs
│   ├── docker/         # Docker Compose
│   ├── k8s/            # Kubernetes manifests
│   └── scripts/        # Deployment scripts
├── docs/               # Documentation
│   └── api/           # OpenAPI specs
├── .github/           # GitHub Actions
└── Makefile           # Build automation
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Spring Boot team for the excellent Java framework
- Go team for the efficient programming language
- Kong for the powerful API gateway
- All open-source contributors
