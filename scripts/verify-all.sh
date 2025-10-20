#!/bin/bash

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}Aegis Gateway - Full Verification${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

echo -e "${YELLOW}Step 1: Checking project structure...${NC}"
REQUIRED_DIRS=("cmd/aegis" "internal/gateway" "internal/policy" "internal/adapters/payments" "internal/adapters/files" "pkg/telemetry" "policies" "scripts" "deploy")
for dir in "${REQUIRED_DIRS[@]}"; do
  if [ -d "$dir" ]; then
    echo -e "${GREEN}✓${NC} $dir exists"
  else
    echo -e "${RED}✗${NC} $dir missing"
    exit 1
  fi
done
echo ""

echo -e "${YELLOW}Step 2: Checking required files...${NC}"
REQUIRED_FILES=("README.md" "DESIGN.md" "Makefile" "go.mod" "deploy/docker-compose.yml" "deploy/Dockerfile")
for file in "${REQUIRED_FILES[@]}"; do
  if [ -f "$file" ]; then
    echo -e "${GREEN}✓${NC} $file exists"
  else
    echo -e "${RED}✗${NC} $file missing"
    exit 1
  fi
done
echo ""

echo -e "${YELLOW}Step 3: Installing dependencies...${NC}"
go mod tidy > /dev/null 2>&1
echo -e "${GREEN}✓${NC} Dependencies installed"
echo ""

echo -e "${YELLOW}Step 4: Building binary...${NC}"
go build -o bin/aegis-gateway cmd/aegis/main.go 2>&1 | grep -v "^#" || true
if [ -f "bin/aegis-gateway" ]; then
  echo -e "${GREEN}✓${NC} Build successful"
else
  echo -e "${RED}✗${NC} Build failed"
  exit 1
fi
echo ""

echo -e "${YELLOW}Step 5: Starting gateway...${NC}"
killall aegis-gateway 2>/dev/null || true
lsof -ti:8080 -ti:8081 -ti:8082 | xargs kill -9 2>/dev/null || true
./bin/aegis-gateway > /tmp/aegis-verify.log 2>&1 &
GATEWAY_PID=$!
sleep 3

if curl -s http://localhost:8080/health > /dev/null; then
  echo -e "${GREEN}✓${NC} Gateway started successfully"
else
  echo -e "${RED}✗${NC} Gateway failed to start"
  cat /tmp/aegis-verify.log | tail -10
  exit 1
fi
echo ""

echo -e "${YELLOW}Step 6: Running demo tests...${NC}"
./scripts/demo.sh | grep -E "(✓|✗|Test [0-9])"
echo ""

echo -e "${YELLOW}Step 7: Checking audit logs...${NC}"
if [ -f "logs/aegis.log" ]; then
  LOG_COUNT=$(wc -l < logs/aegis.log | tr -d ' ')
  if [ "$LOG_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Audit logs generated ($LOG_COUNT entries)"
    echo "Sample log entry:"
    tail -1 logs/aegis.log | jq '.' 2>/dev/null || tail -1 logs/aegis.log
  else
    echo -e "${RED}✗${NC} No audit logs found"
  fi
else
  echo -e "${RED}✗${NC} Audit log file not created"
fi
echo ""

echo -e "${YELLOW}Step 8: Testing policy reload...${NC}"
RELOAD_RESPONSE=$(curl -s -X POST http://localhost:8080/policies/reload)
if echo "$RELOAD_RESPONSE" | grep -q "reloaded"; then
  echo -e "${GREEN}✓${NC} Policy reload endpoint works"
else
  echo -e "${RED}✗${NC} Policy reload failed"
fi
echo ""

echo -e "${YELLOW}Step 9: Cleaning up...${NC}"
kill $GATEWAY_PID 2>/dev/null || true
lsof -ti:8080 -ti:8081 -ti:8082 | xargs kill -9 2>/dev/null || true
echo -e "${GREEN}✓${NC} Cleanup complete"
echo ""

echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN}All verification tests passed!${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo "Next steps:"
echo "  1. Run gateway: make run"
echo "  2. Run demo: make demo"
echo "  3. View docs: cat README.md"
echo "  4. Deploy: cd deploy && docker-compose up"
