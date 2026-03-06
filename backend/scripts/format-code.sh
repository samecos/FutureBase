#!/bin/bash
# Format all code in the project

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

cd "$(dirname "$0")/.."

echo -e "${BLUE}🔧 Formatting all code...${NC}"

# Java Services
echo -e "${BLUE}Formatting Java services...${NC}"
java_services=("user-service" "project-service" "property-service" "version-service" "search-service")

for service in "${java_services[@]}"; do
    echo "  → Formatting $service..."
    if [ -f "services/$service/pom.xml" ]; then
        (cd "services/$service" && mvn spotless:apply -q 2>/dev/null) || echo "    ⚠️ Skipped $service (Maven not available)"
    fi
done

# Go Services
echo -e "${BLUE}Formatting Go services...${NC}"
go_services=("collaboration-service" "geometry-service" "script-service" "file-service" "notification-service" "analytics-service")

for service in "${go_services[@]}"; do
    echo "  → Formatting $service..."
    if [ -d "services/$service" ]; then
        (cd "services/$service" && go fmt ./... 2>/dev/null) || echo "    ⚠️ Skipped $service"
    fi
done

echo -e "${GREEN}✅ Code formatting complete!${NC}"
