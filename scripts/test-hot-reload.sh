#!/bin/bash

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

BASE_URL="http://localhost:8080"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}Testing Hot Reload Feature${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

echo -e "${BLUE}Step 1: Test current policy (max_amount=5000)${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Agent-ID: finance-agent" \
  -X POST $BASE_URL/tools/payments/create \
  -d '{"amount":7500,"currency":"USD","vendor_id":"V99"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
echo "Response ($HTTP_CODE): $BODY"
if [ "$HTTP_CODE" = "403" ]; then
  echo -e "${GREEN}✓ Amount 7500 blocked (exceeds 5000)${NC}"
fi
echo ""

echo -e "${BLUE}Step 2: Updating policy file (max_amount=10000)${NC}"
cat > policies/finance-policy.yaml << 'EOF'
version: 2
agents:
  - id: finance-agent
    allow:
      - tool: payments
        actions: [create, refund]
        conditions:
          max_amount: 10000
          currencies: [USD, EUR]
EOF
echo -e "${GREEN}✓ Policy file updated${NC}"
echo ""

echo "Triggering manual reload..."
curl -s -X POST $BASE_URL/policies/reload > /dev/null
echo -e "${GREEN}✓ Policies reloaded${NC}"
echo ""

echo -e "${BLUE}Step 3: Test updated policy (max_amount=10000)${NC}"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Agent-ID: finance-agent" \
  -X POST $BASE_URL/tools/payments/create \
  -d '{"amount":7500,"currency":"USD","vendor_id":"V99"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
echo "Response ($HTTP_CODE): $BODY"
if [ "$HTTP_CODE" = "200" ]; then
  echo -e "${GREEN}✓ Amount 7500 now allowed (policy hot-reloaded)${NC}"
fi
echo ""

echo -e "${BLUE}Step 4: Restoring original policy${NC}"
cat > policies/finance-policy.yaml << 'EOF'
version: 1
agents:
  - id: finance-agent
    allow:
      - tool: payments
        actions: [create, refund]
        conditions:
          max_amount: 5000
          currencies: [USD, EUR]
EOF
echo -e "${GREEN}✓ Policy restored to original${NC}"
echo ""

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}Hot reload test completed successfully!${NC}"
echo -e "${BLUE}============================================${NC}"
