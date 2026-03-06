#!/bin/bash
# Seed data script for development and testing

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
API_URL=${API_URL:-"http://localhost:8000"}
ADMIN_EMAIL=${ADMIN_EMAIL:-"admin@archplatform.local"}
ADMIN_PASSWORD=${ADMIN_PASSWORD:-"Admin123!"}

echo -e "${BLUE}🌱 ArchPlatform Seed Data Script${NC}"
echo "API URL: $API_URL"
echo ""

# Helper function for API calls
call_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    local token=$4
    
    local auth_header=""
    if [ -n "$token" ]; then
        auth_header="-H Authorization: Bearer $token"
    fi
    
    if [ -n "$data" ]; then
        curl -s -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            $auth_header \
            -d "$data"
    else
        curl -s -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            $auth_header
    fi
}

# Step 1: Create admin user
echo -e "${BLUE}Step 1: Creating admin user...${NC}"
ADMIN_RESPONSE=$(call_api "POST" "/api/v1/auth/register" "{
    \"username\": \"admin\",
    \"email\": \"$ADMIN_EMAIL\",
    \"password\": \"$ADMIN_PASSWORD\",
    \"firstName\": \"System\",
    \"lastName\": \"Administrator\"
}")

echo "$ADMIN_RESPONSE" | jq . 2>/dev/null || echo "$ADMIN_RESPONSE"

# Step 2: Login as admin
echo -e "${BLUE}Step 2: Logging in as admin...${NC}"
LOGIN_RESPONSE=$(call_api "POST" "/api/v1/auth/login" "{
    \"username\": \"admin\",
    \"password\": \"$ADMIN_PASSWORD\"
}")

ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.accessToken' 2>/dev/null)

if [ "$ADMIN_TOKEN" == "null" ] || [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${RED}❌ Failed to login as admin${NC}"
    echo "$LOGIN_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Logged in successfully${NC}"

# Step 3: Create sample users
echo -e "${BLUE}Step 3: Creating sample users...${NC}"

USERS=(
    "john.doe:john@example.com:John:Doe"
    "jane.smith:jane@example.com:Jane:Smith"
    "bob.wilson:bob@example.com:Bob:Wilson"
    "alice.jones:alice@example.com:Alice:Jones"
    "charlie.brown:charlie@example.com:Charlie:Brown"
)

for user in "${USERS[@]}"; do
    IFS=':' read -r username email firstName lastName <<< "$user"
    
    RESPONSE=$(call_api "POST" "/api/v1/auth/register" "{
        \"username\": \"$username\",
        \"email\": \"$email\",
        \"password\": \"Password123\",
        \"firstName\": \"$firstName\",
        \"lastName\": \"$lastName\"
    }")
    
    if echo "$RESPONSE" | grep -q "id"; then
        echo -e "${GREEN}✓ Created user: $username${NC}"
    else
        echo -e "${YELLOW}⚠ User $username may already exist${NC}"
    fi
done

# Step 4: Create sample projects
echo -e "${BLUE}Step 4: Creating sample projects...${NC}"

PROJECTS=(
    "Downtown Office Building:15-story commercial office building:New York:commercial"
    "Riverside Apartments:Luxury residential complex:Chicago:residential"
    "Tech Hub Campus:Modern tech company headquarters:San Francisco:commercial"
    "Community Hospital:Regional healthcare facility:Boston:healthcare"
    "Shopping Mall Renovation:Major retail space upgrade:Miami:retail"
)

for project in "${PROJECTS[@]}"; do
    IFS=':' read -r name description location type <<< "$project"
    
    RESPONSE=$(call_api "POST" "/api/v1/projects" "{
        \"name\": \"$name\",
        \"description\": \"$description\",
        \"location\": \"$location\",
        \"tags\": [\"$type\", \"active\"]
    }" "$ADMIN_TOKEN")
    
    PROJECT_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null)
    
    if [ "$PROJECT_ID" != "null" ] && [ -n "$PROJECT_ID" ]; then
        echo -e "${GREEN}✓ Created project: $name${NC}"
        
        # Add random members to project
        MEMBER_COUNT=$((RANDOM % 3 + 1))
        for ((i=1; i<=MEMBER_COUNT; i++)); do
            USER_INDEX=$((RANDOM % ${#USERS[@]}))
            IFS=':' read -r member_username member_email _ _ <<< "${USERS[$USER_INDEX]}"
            
            # Login as member to get ID
            MEMBER_LOGIN=$(call_api "POST" "/api/v1/auth/login" "{
                \"username\": \"$member_username\",
                \"password\": \"Password123\"
            }")
            
            MEMBER_ID=$(echo "$MEMBER_LOGIN" | jq -r '.user.id' 2>/dev/null)
            
            if [ "$MEMBER_ID" != "null" ] && [ -n "$MEMBER_ID" ]; then
                ROLES=("EDITOR" "VIEWER" "ADMIN")
                ROLE=${ROLES[$RANDOM % ${#ROLES[@]}]}
                
                call_api "POST" "/api/v1/projects/$PROJECT_ID/members" "{
                    \"userId\": \"$MEMBER_ID\",
                    \"role\": \"$ROLE\"
                }" "$ADMIN_TOKEN" > /dev/null
            fi
        done
    else
        echo -e "${RED}❌ Failed to create project: $name${NC}"
        echo "$RESPONSE"
    fi
done

# Step 5: Create property templates (if property service is available)
echo -e "${BLUE}Step 5: Creating property templates...${NC}"

# This would require property service to be running
# For now, just print a message
echo -e "${YELLOW}⚠ Property templates require property service to be running${NC}"

# Summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ Seed data creation completed!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Created:${NC}"
echo "  • 1 Admin user"
echo "  • 5 Sample users"
echo "  • 5 Sample projects with members"
echo ""
echo -e "${BLUE}Login credentials:${NC}"
echo "  Admin: $ADMIN_EMAIL / $ADMIN_PASSWORD"
echo "  Users: <username>@example.com / Password123"
echo ""
echo -e "${BLUE}You can now login and start exploring the API.${NC}"
