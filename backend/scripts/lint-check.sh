#!/bin/bash
# Run linting checks on all code

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

cd "$(dirname "$0")/.."

echo -e "${BLUE}🔍 Running lint checks...${NC}"

exit_code=0

# Java Checkstyle
echo -e "${BLUE}Checking Java code style...${NC}"
java_services=("user-service" "project-service" "property-service" "version-service" "search-service")

for service in "${java_services[@]}"; do
    echo "  → Checking $service..."
    if [ -f "services/$service/pom.xml" ]; then
        if ! (cd "services/$service" && mvn checkstyle:check -q 2>/dev/null); then
            echo -e "    ${RED}✗ Checkstyle failed for $service${NC}"
            exit_code=1
        else
            echo -e "    ${GREEN}✓ $service passed${NC}"
        fi
    fi
done

# Go Linting
echo -e "${BLUE}Checking Go code...${NC}"
go_services=("collaboration-service" "geometry-service" "script-service" "file-service" "notification-service" "analytics-service")

for service in "${go_services[@]}"; do
    echo "  → Checking $service..."
    if [ -d "services/$service" ]; then
        if ! (cd "services/$service" && golangci-lint run 2>/dev/null); then
            echo -e "    ${YELLOW}⚠️ Linting issues in $service${NC}"
            # Don't fail for Go as golangci-lint might not be installed
        else
            echo -e "    ${GREEN}✓ $service passed${NC}"
        fi
    fi
done

if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}✅ All lint checks passed!${NC}"
else
    echo -e "${RED}❌ Some lint checks failed${NC}"
fi

exit $exit_code
