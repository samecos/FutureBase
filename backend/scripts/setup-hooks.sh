#!/bin/bash
# Setup Git Hooks and Development Tools

set -e

echo "🚀 Setting up development environment..."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if we're in the right directory
if [ ! -f "Makefile" ]; then
    echo -e "${RED}Error: Please run this script from the backend directory${NC}"
    exit 1
fi

cd "$(dirname "$0")/.."

echo -e "${BLUE}📦 Installing pre-commit hooks...${NC}"

# Check if pip is available
if ! command -v pip &> /dev/null; then
    echo -e "${YELLOW}⚠️ pip not found. Please install Python and pip first.${NC}"
    echo "   Windows: https://www.python.org/downloads/"
    echo "   Linux: sudo apt-get install python3-pip"
    exit 1
fi

# Install pre-commit
pip install pre-commit --quiet

# Install hooks
pre-commit install

echo -e "${GREEN}✅ Pre-commit hooks installed!${NC}"

echo -e "${BLUE}🔧 Installing Go tools...${NC}"

# Install golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
else
    echo -e "${GREEN}✓ golangci-lint already installed${NC}"
fi

# Install goimports
if ! command -v goimports &> /dev/null; then
    echo "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
else
    echo -e "${GREEN}✓ goimports already installed${NC}"
fi

# Install buf (protobuf linter)
if ! command -v buf &> /dev/null; then
    echo "Installing buf..."
    go install github.com/bufbuild/buf/cmd/buf@latest
else
    echo -e "${GREEN}✓ buf already installed${NC}"
fi

echo -e "${GREEN}✅ Go tools installed!${NC}"

echo -e "${BLUE}☕ Setting up Maven wrapper...${NC}"

# Check for Maven
if ! command -v mvn &> /dev/null; then
    echo -e "${YELLOW}⚠️ Maven not found. Please install Maven:${NC}"
    echo "   Windows: choco install maven"
    echo "   Linux: sudo apt-get install maven"
    echo "   Mac: brew install maven"
else
    echo -e "${GREEN}✓ Maven found: $(mvn -v | head -1)${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Setup complete!${NC}"
echo ""
echo -e "${BLUE}📚 Quick Start:${NC}"
echo ""
echo "  Format Java code:    mvn spotless:apply -f services/user-service/pom.xml"
echo "  Check Java style:    mvn checkstyle:check -f services/user-service/pom.xml"
echo "  Format Go code:      go fmt ./... (in service directory)"
echo "  Lint Go code:        golangci-lint run"
echo "  Run pre-commit:      pre-commit run --all-files"
echo ""
echo -e "${YELLOW}⚠️ Note: Pre-commit hooks will run automatically on git commit.${NC}"
echo "   If you need to skip hooks temporarily: git commit --no-verify"
