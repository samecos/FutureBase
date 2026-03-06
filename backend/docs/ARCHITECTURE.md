# Architecture Platform - System Architecture

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Service Communication](#service-communication)
4. [Data Flow](#data-flow)
5. [Security Architecture](#security-architecture)
6. [Deployment Architecture](#deployment-architecture)

---

## Overview

Architecture Platform is a cloud-native microservices application designed for architectural design teams. It provides real-time collaboration, version control, and advanced geometry processing capabilities.

### Key Characteristics

- **11 Microservices**: 5 Java (Spring Boot) + 6 Go services
- **Polyglot Persistence**: PostgreSQL + PostGIS, Redis, Elasticsearch, MinIO, ClickHouse
- **Event-Driven**: Kafka for cross-service communication
- **Real-time**: WebSocket for collaboration
- **Cloud-Native**: Kubernetes-ready with auto-scaling

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client Applications                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Web App    │  │  Desktop App │  │   Mobile App │  │   CAD Plugin │    │
│  │  (React/Vue) │  │   (Electron) │  │(iOS/Android) │  │(AutoCAD/Revit│    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
└─────────┼─────────────────┼─────────────────┼─────────────────┼────────────┘
          │                 │                 │                 │
          └─────────────────┴────────┬────────┴─────────────────┘
                                     │
                              ┌──────▼──────┐
                              │   CDN/WAF   │
                              │  (CloudFlare│
                              └──────┬──────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────────┐
│                        Kubernetes Cluster                                    │
│                                                                              │
│  ┌────────────────────────────────┼────────────────────────────────┐        │
│  │                    Kong API Gateway                             │        │
│  │  ┌─────────────────┬───────────┴───────────┬─────────────────┐ │        │
│  │  │   Rate Limiting │    JWT Validation     │   SSL/TLS       │ │        │
│  │  └─────────────────┴───────────────────────┴─────────────────┘ │        │
│  └────────────────────────────────┼────────────────────────────────┘        │
│                                   │                                          │
│  ┌────────────────────────────────┼────────────────────────────────┐        │
│  │                         Service Mesh                             │        │
│  └────────────────────────────────┼────────────────────────────────┘        │
│                                   │                                          │
│  ┌────────────────────────────────┼────────────────────────────────┐        │
│  │                      Microservices Layer                         │        │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │        │
│  │  │   Java      │  │     Go      │  │     Go      │              │        │
│  │  │  Services   │  │  Services   │  │  Services   │              │        │
│  │  │             │  │             │  │             │              │        │
│  │  │ • User      │  │ • Collab    │  │ • Script    │              │        │
│  │  │ • Project   │  │ • Geometry  │  │ • File      │              │        │
│  │  │ • Property  │  │ • Notif     │  │ • Analytics │              │        │
│  │  │ • Version   │  │             │  │             │              │        │
│  │  │ • Search    │  │             │  │             │              │        │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │        │
│  └──────────────────────────────────────────────────────────────────┘        │
│                                   │                                          │
│  ┌────────────────────────────────┼────────────────────────────────┐        │
│  │                      Data Layer                                  │        │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │        │
│  │  │  PostgreSQL │  │    Redis    │  │    Kafka    │              │        │
│  │  │  + PostGIS  │  │   (Cache)   │  │  (Events)   │              │        │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │        │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │        │
│  │  │Elasticsearch│  │    MinIO    │  │   Temporal  │              │        │
│  │  │   (Search)  │  │  (Objects)  │  │ (Workflows) │              │        │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │        │
│  └──────────────────────────────────────────────────────────────────┘        │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────┐        │
│  │                     Observability Stack                          │        │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │        │
│  │  │ Prometheus  │  │   Grafana   │  │   Jaeger    │              │        │
│  │  │  (Metrics)  │  │(Dashboards) │  │   (Traces)  │              │        │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │        │
│  └──────────────────────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Service Communication

### Synchronous Communication (HTTP/gRPC)

```
┌──────────────┐     HTTP      ┌──────────────┐
│   Client     │──────────────▶│  Kong Gateway │
└──────────────┘               └──────┬───────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
             ┌──────────┐      ┌──────────┐      ┌──────────┐
             │  Java    │      │   Go     │      │   Go     │
             │ Service  │      │ Service  │      │ Service  │
             └──────────┘      └──────────┘      └──────────┘
```

### Asynchronous Communication (Kafka)

```
┌──────────┐   ┌──────────┐   ┌──────────┐
│ Producer │   │  Kafka   │   │ Consumer │
│  Service │──▶│  Topic   │──▶│  Service │
└──────────┘   └──────────┘   └──────────┘
     │                             │
     │  Project Created Event      │
     │  ───────────────────────▶   │
     │                             │
     │                             │ Notify Users
     │                             │ Update Analytics
     │                             │ Index for Search
```

### Real-time Communication (WebSocket)

```
┌──────────┐         ┌──────────────────┐         ┌──────────┐
│  User A  │◀───────▶│  Collaboration   │◀───────▶│  User B  │
│(Browser) │   Yjs   │   Service        │   Yjs   │(Browser) │
└──────────┘  CRDT   │  (WebSocket)     │  CRDT   └──────────┘
                     └──────────────────┘
```

---

## Data Flow

### User Registration Flow

```
┌────────┐     ┌─────────────┐     ┌─────────────┐     ┌────────┐
│ Client │────▶│ Kong Gateway│────▶│User Service │────▶│PostgreSQL
└────────┘     └─────────────┘     └─────────────┘     └────────┘
     │                                  │
     │                                  │
     │  JWT Token                       │ Hash Password (BCrypt)
     │◀─────────────────────────────────│
     │
```

### Design File Upload Flow

```
┌────────┐     ┌─────────────┐     ┌─────────────┐     ┌────────┐
│ Client │────▶│ Kong Gateway│────▶│File Service │────▶│ MinIO  │
└────────┘     └─────────────┘     └─────────────┘     └────────┘
                                              │
                                              │ Event
                                              ▼
                                       ┌─────────────┐
                                       │    Kafka    │
                                       └──────┬──────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
             ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
             │   Search    │          │  Analytics  │          │ Notification│
             │   Service   │          │   Service   │          │   Service   │
             └─────────────┘          └─────────────┘          └─────────────┘
```

### Real-time Collaboration Flow

```
┌─────────┐                    ┌────────────────────┐                    ┌─────────┐
│ User A  │                    │  Collaboration Svc │                    │ User B  │
│(Chrome) │                    │   (Yjs + WebSocket)│                    │(Safari) │
└────┬────┘                    └──────────┬─────────┘                    └────┬────┘
     │                                     │                                   │
     │  Operation: Add Wall                │                                   │
     │────────────────────────────────────▶│                                   │
     │                                     │                                   │
     │                                     │  Apply CRDT Operation             │
     │                                     │  Broadcast to Room                │
     │                                     │──────────────────────────────────▶│
     │                                     │                                   │
     │  Ack: Success                       │                                   │
     │◀────────────────────────────────────│                                   │
     │                                     │                                   │
     │                                     │                                   │ Wall Added
     │                                     │                                   │◀───────────
```

---

## Security Architecture

### Authentication Flow

```
┌─────────┐                            ┌─────────────┐                            ┌─────────┐
│  Client │                            │User Service │                            │  Redis  │
└────┬────┘                            └──────┬──────┘                            └────┬────┘
     │                                         │                                      │
     │  1. POST /auth/login                    │                                      │
     │    {username, password}                 │                                      │
     │────────────────────────────────────────▶│                                      │
     │                                         │                                      │
     │                                         │  2. Validate Credentials            │
     │                                         │     (BCrypt compare)                 │
     │                                         │                                      │
     │                                         │  3. Generate JWT + Refresh          │
     │                                         │     Store in Redis                   │
     │                                         │─────────────────────────────────────▶│
     │                                         │                                      │
     │  4. Return Tokens                       │                                      │
     │    {access_token, refresh_token}        │                                      │
     │◀────────────────────────────────────────│                                      │
     │                                         │                                      │
     │  5. Request with Bearer Token           │                                      │
     │────────────────────────────────────────▶│                                      │
     │                                         │  6. Validate JWT                    │
     │                                         │     Check Redis blacklist            │
     │                                         │◀─────────────────────────────────────│
     │                                         │                                      │
     │  7. Response                            │                                      │
     │◀────────────────────────────────────────│                                      │
```

### Authorization (RBAC)

```
┌─────────────────────────────────────────────────────────────────┐
│                        Permission Hierarchy                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  OWNER                                                          │
│    ├── Full project control                                     │
│    ├── Can delete project                                       │
│    ├── Can manage billing                                       │
│    └── Can transfer ownership                                   │
│                                                                 │
│  ADMIN                                                          │
│    ├── Can manage members (except owner)                        │
│    ├── Can modify all design files                              │
│    └── Can create versions and branches                         │
│                                                                 │
│  EDITOR                                                         │
│    ├── Can modify design files                                  │
│    ├── Can create versions                                      │
│    └── Cannot manage members                                    │
│                                                                 │
│  VIEWER                                                         │
│    ├── Can view design files                                    │
│    ├── Can add comments                                         │
│    └── Cannot modify files                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Deployment Architecture

### Kubernetes Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Production Cluster                                │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                        Ingress Controller                            │    │
│  │                    (NGINX / Traefik / ALB)                           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│  ┌─────────────────────────────────▼───────────────────────────────────┐    │
│  │                         Kong Gateway                                 │    │
│  │                     (3 replicas, HPA enabled)                        │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│  ┌─────────────────────────────────┼───────────────────────────────────┐    │
│  │                                 │                                   │    │
│  ▼                                 ▼                                   ▼    │
│  ┌─────────────┐           ┌─────────────┐           ┌─────────────┐        │
│  │ User Svc    │           │ Project Svc │           │ Collab Svc  │        │
│  │ 3 replicas  │           │ 3 replicas  │           │ 5-20 (HPA)  │        │
│  │ PDB: 1      │           │ PDB: 1      │           │ PDB: 2      │        │
│  └─────────────┘           └─────────────┘           └─────────────┘        │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Stateful Services                               │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │    │
│  │  │PostgreSQL│  │  Redis   │  │  Kafka   │  │  MinIO   │            │    │
│  │  │(StatefulSet)│(Sentinel)│  │(Strimzi) │  │(StatefulSet)         │    │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Monitoring Stack                                │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │    │
│  │  │Prometheus│  │  Grafana │  │  Jaeger  │  │   Loki   │            │    │
│  │  │(VMAgent) │  │(Grafana) │  │(Operator)│  │(Promtail)│            │    │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Frontend** | React/Vue | Web application |
| **API Gateway** | Kong | Routing, auth, rate limiting |
| **Services** | Spring Boot 3.2 | Java microservices |
| **Services** | Go 1.21 | High-performance services |
| **Database** | PostgreSQL 15 + PostGIS | Primary data store |
| **Cache** | Redis 7 | Session, cache, pub/sub |
| **Search** | Elasticsearch 8 | Full-text search |
| **Storage** | MinIO | Object storage |
| **Messaging** | Kafka | Event streaming |
| **Workflows** | Temporal | Long-running workflows |
| **Monitoring** | Prometheus + Grafana | Metrics and dashboards |
| **Tracing** | Jaeger | Distributed tracing |
| **Deployment** | Kubernetes | Container orchestration |
| **CI/CD** | GitHub Actions | Build and deploy |

---

## Performance Characteristics

| Metric | Target | Current |
|--------|--------|---------|
| API Response Time (p95) | < 200ms | ~ 150ms |
| WebSocket Latency | < 50ms | ~ 30ms |
| File Upload Speed | 100MB/s | ~ 80MB/s |
| Concurrent Users | 10,000 | Tested 5,000 |
| Availability | 99.9% | 99.95% |

---

## Scalability Strategy

### Horizontal Scaling

- **Stateless Services**: Scale based on CPU/Memory
- **Collaboration Service**: Scale based on WebSocket connections
- **File Service**: Scale based on upload/download rates

### Vertical Scaling

- **Database**: Read replicas for query optimization
- **Elasticsearch**: Shard allocation based on data volume
- **Redis**: Cluster mode for large datasets

---

## Disaster Recovery

### Backup Strategy

| Component | Frequency | Retention | Method |
|-----------|-----------|-----------|--------|
| PostgreSQL | Hourly | 30 days | WAL archiving |
| MinIO | Daily | 90 days | Cross-region replication |
| Redis | Real-time | 7 days | RDB + AOF |

### Recovery Time Objectives (RTO)

- **Service Recovery**: < 5 minutes
- **Database Recovery**: < 15 minutes
- **Full System Recovery**: < 1 hour
