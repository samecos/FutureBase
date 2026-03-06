#!/bin/bash
# Architecture Platform Deployment Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
DOCKER_DIR="$SCRIPT_DIR/../docker"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[DEPLOY]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    command -v docker >/dev/null 2>&1 || error "Docker is required but not installed"
    command -v docker-compose >/dev/null 2>&1 || error "Docker Compose is required but not installed"
    
    # Check Docker is running
    docker info >/dev/null 2>&1 || error "Docker is not running"
    
    log "Prerequisites check passed"
}

# Setup environment
setup_env() {
    log "Setting up environment..."
    
    cd "$DOCKER_DIR"
    
    if [ ! -f .env ]; then
        if [ -f .env.example ]; then
            cp .env.example .env
            warn "Created .env from .env.example. Please review and update values."
        else
            error ".env.example not found"
        fi
    fi
    
    # Generate JWT secret if not set
    if ! grep -q "JWT_SECRET=" .env || grep -q "JWT_SECRET=your-256-bit-secret" .env; then
        JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
        sed -i.bak "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env && rm -f .env.bak
        log "Generated new JWT_SECRET"
    fi
}

# Build services
build_services() {
    log "Building services..."
    cd "$DOCKER_DIR"
    docker-compose -f docker-compose.full.yml build
}

# Start infrastructure
start_infrastructure() {
    log "Starting infrastructure services..."
    cd "$DOCKER_DIR"
    docker-compose -f docker-compose.full.yml up -d postgres redis kafka zookeeper elasticsearch minio temporal
    
    log "Waiting for infrastructure to be ready..."
    sleep 30
    
    # Wait for PostgreSQL
    log "Waiting for PostgreSQL..."
    until docker exec archplatform-postgres pg_isready -U postgres; do
        sleep 2
    done
    
    # Wait for Elasticsearch
    log "Waiting for Elasticsearch..."
    until curl -s http://localhost:9200/_cluster/health | grep -q "status"; do
        sleep 2
    done
}

# Start application services
start_services() {
    log "Starting application services..."
    cd "$DOCKER_DIR"
    docker-compose -f docker-compose.full.yml up -d kong user-service project-service property-service version-service search-service
    docker-compose -f docker-compose.full.yml up -d collaboration-service geometry-service script-service file-service notification-service analytics-service
}

# Start monitoring
start_monitoring() {
    log "Starting monitoring services..."
    cd "$DOCKER_DIR"
    docker-compose -f docker-compose.full.yml up -d prometheus grafana
}

# Health check
health_check() {
    log "Performing health checks..."
    
    services=(
        "http://localhost:8000/api/v1/users/health:User Service"
        "http://localhost:8000/api/v1/projects/health:Project Service"
        "http://localhost:9090/-/healthy:Prometheus"
        "http://localhost:3000/api/health:Grafana"
    )
    
    for check in "${services[@]}"; do
        IFS=':' read -r url name <<< "$check"
        if curl -s "$url" > /dev/null; then
            log "✓ $name is healthy"
        else
            warn "✗ $name is not responding"
        fi
    done
}

# Show status
show_status() {
    log "Deployment Status:"
    cd "$DOCKER_DIR"
    docker-compose -f docker-compose.full.yml ps
    
    echo ""
    log "Access Points:"
    echo "  Kong Gateway:     http://localhost:8000"
    echo "  Kong Admin:       http://localhost:8001"
    echo "  Grafana:          http://localhost:3000 (admin/admin)"
    echo "  Prometheus:       http://localhost:9090"
    echo "  MinIO Console:    http://localhost:9001 (minioadmin/minioadmin)"
    echo "  Temporal UI:      http://localhost:8233"
}

# Main deployment
main() {
    case "${1:-all}" in
        infrastructure)
            check_prerequisites
            setup_env
            start_infrastructure
            ;;
        services)
            check_prerequisites
            setup_env
            start_services
            ;;
        monitoring)
            start_monitoring
            ;;
        build)
            check_prerequisites
            build_services
            ;;
        health)
            health_check
            ;;
        all)
            check_prerequisites
            setup_env
            build_services
            start_infrastructure
            start_services
            start_monitoring
            sleep 10
            health_check
            show_status
            ;;
        *)
            echo "Usage: $0 {all|infrastructure|services|monitoring|build|health}"
            echo ""
            echo "Commands:"
            echo "  all           - Full deployment (default)"
            echo "  infrastructure - Start only infrastructure services"
            echo "  services      - Start only application services"
            echo "  monitoring    - Start only monitoring services"
            echo "  build         - Build all service images"
            echo "  health        - Check service health"
            exit 1
            ;;
    esac
}

main "$@"
