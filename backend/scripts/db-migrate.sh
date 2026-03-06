#!/bin/bash
# Database migration script

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_USER=${DB_USER:-"postgres"}
DB_PASSWORD=${DB_PASSWORD:-"postgres"}
DB_NAME=${DB_NAME:-"archplatform"}

JAVA_SERVICES=("user-service" "project-service" "property-service" "version-service" "search-service")

echo -e "${BLUE}🗄️  Database Migration Tool${NC}"
echo ""

# Function to migrate a service
migrate_service() {
    local service=$1
    local service_name=$(echo $service | sed 's/-service//')
    
    echo -e "${BLUE}Migrating $service...${NC}"
    
    cd "services/$service"
    
    # Run Flyway migration via Maven
    ./mvnw flyway:migrate \
        -Dflyway.url="jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}" \
        -Dflyway.user="${DB_USER}" \
        -Dflyway.password="${DB_PASSWORD}" \
        -Dflyway.schemas="${service_name}_service" \
        -Dflyway.locations="filesystem:src/main/resources/db/migration" \
        -q
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $service migrated successfully${NC}"
    else
        echo -e "${RED}✗ $service migration failed${NC}"
        return 1
    fi
    
    cd ../..
}

# Function to validate migrations
validate_migrations() {
    local service=$1
    local service_name=$(echo $service | sed 's/-service//')
    
    echo -e "${BLUE}Validating $service migrations...${NC}"
    
    cd "services/$service"
    
    ./mvnw flyway:validate \
        -Dflyway.url="jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}" \
        -Dflyway.user="${DB_USER}" \
        -Dflyway.password="${DB_PASSWORD}" \
        -Dflyway.schemas="${service_name}_service" \
        -q
    
    cd ../..
}

# Function to show migration info
migration_info() {
    local service=$1
    local service_name=$(echo $service | sed 's/-service//')
    
    echo -e "${BLUE}Migration info for $service:${NC}"
    
    cd "services/$service"
    
    ./mvnw flyway:info \
        -Dflyway.url="jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}" \
        -Dflyway.user="${DB_USER}" \
        -Dflyway.password="${DB_PASSWORD}" \
        -Dflyway.schemas="${service_name}_service"
    
    cd ../..
}

# Function to repair migrations
repair_migrations() {
    local service=$1
    local service_name=$(echo $service | sed 's/-service//')
    
    echo -e "${YELLOW}Repairing $service migrations...${NC}"
    
    cd "services/$service"
    
    ./mvnw flyway:repair \
        -Dflyway.url="jdbc:postgresql://${DB_HOST}:${DB_PORT}/${DB_NAME}" \
        -Dflyway.user="${DB_USER}" \
        -Dflyway.password="${DB_PASSWORD}" \
        -Dflyway.schemas="${service_name}_service" \
        -q
    
    cd ../..
}

# Main script
case "${1:-migrate}" in
    migrate)
        echo -e "${BLUE}Running migrations...${NC}"
        for service in "${JAVA_SERVICES[@]}"; do
            migrate_service "$service"
        done
        echo ""
        echo -e "${GREEN}✅ All migrations completed!${NC}"
        ;;
    
    validate)
        echo -e "${BLUE}Validating migrations...${NC}"
        for service in "${JAVA_SERVICES[@]}"; do
            validate_migrations "$service"
        done
        echo ""
        echo -e "${GREEN}✅ All migrations validated!${NC}"
        ;;
    
    info)
        for service in "${JAVA_SERVICES[@]}"; do
            migration_info "$service"
            echo ""
        done
        ;;
    
    repair)
        echo -e "${YELLOW}Repairing migrations...${NC}"
        for service in "${JAVA_SERVICES[@]}"; do
            repair_migrations "$service"
        done
        echo ""
        echo -e "${GREEN}✅ Migrations repaired!${NC}"
        ;;
    
    clean)
        echo -e "${RED}⚠️  WARNING: This will drop all schema objects!${NC}"
        read -p "Are you sure? (yes/no): " confirm
        if [ "$confirm" == "yes" ]; then
            for service in "${JAVA_SERVICES[@]}"; do
                service_name=$(echo $service | sed 's/-service//')
                echo -e "${YELLOW}Cleaning schema: ${service_name}_service${NC}"
                PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME \
                    -c "DROP SCHEMA IF EXISTS ${service_name}_service CASCADE;"
            done
            echo -e "${GREEN}✅ All schemas cleaned!${NC}"
        else
            echo "Aborted."
        fi
        ;;
    
    *)
        echo "Usage: $0 [migrate|validate|info|repair|clean]"
        echo ""
        echo "Commands:"
        echo "  migrate   - Run pending migrations (default)"
        echo "  validate  - Validate applied migrations"
        echo "  info      - Show migration information"
        echo "  repair    - Repair migration checksums"
        echo "  clean     - Drop all schemas (DANGER!)"
        exit 1
        ;;
esac
