#!/bin/bash
# =============================================================================
# Kubric Validation Script - Test All Fixes
# =============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== Kubric Validation Suite ===${NC}"
echo ""

# Test 1: Validate docker-compose.prod.yml syntax
echo -e "${YELLOW}[1/4] Validating docker-compose.prod.yml...${NC}"
docker compose -f docker-compose.prod.yml config > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ docker-compose.prod.yml syntax valid${NC}"
else
    echo -e "${RED}✗ docker-compose.prod.yml syntax invalid${NC}"
    exit 1
fi

# Test 2: Build frontend image
echo -e "${YELLOW}[2/4] Building frontend image...${NC}"
cd frontend
docker build -t kubric-web:test -f ../Dockerfile.web . > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Frontend image built successfully${NC}"
else
    echo -e "${RED}✗ Frontend build failed${NC}"
    cd ..
    exit 1
fi
cd ..

# Test 3: Check KAI Python modules exist
echo -e "${YELLOW}[3/4] Checking KAI Python modules...${NC}"
if [ -f "kai/deploy/docker_manager.py" ] && [ -f "kai/house/monitor.py" ]; then
    echo -e "${GREEN}✓ KAI autonomy modules present${NC}"
else
    echo -e "${RED}✗ KAI autonomy modules missing${NC}"
    exit 1
fi

# Test 4: Verify service names in compose file
echo -e "${YELLOW}[4/4] Verifying service names...${NC}"
if grep -q "kai-python:" docker-compose.prod.yml; then
    echo -e "${GREEN}✓ Service name 'kai-python' found in docker-compose.prod.yml${NC}"
else
    echo -e "${RED}✗ Service name 'kai-python' not found${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}=== All Validations Passed ===${NC}"
echo ""
echo "Next steps:"
echo "  1. docker compose -f docker-compose.prod.yml up -d"
echo "  2. docker compose -f docker-compose.prod.yml ps"
echo "  3. docker exec -it kubric-uidr-kai-python-1 python --version"
echo ""
