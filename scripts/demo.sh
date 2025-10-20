#!/bin/bash

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

BASE_URL="http://localhost:8080"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}Aegis Gateway Demo - Policy Enforcement${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

sleep 2

echo -e "${BLUE}Test 1: Blocked high-value payment (exceeds max_amount=5000)${NC}"
echo "Request: amount=50000, currency=USD"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Agent-ID: finance-agent" \
  -X POST $BASE_URL/tools/payments/create \
  -d '{"amount":50000,"currency":"USD","vendor_id":"V99"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
echo "Response ($HTTP_CODE): $BODY"
if [ "$HTTP_CODE" = "403" ]; then
  echo -e "${GREEN}✓ Test 1 PASSED: Request blocked as expected${NC}"
else
  echo -e "${RED}✗ Test 1 FAILED: Expected 403, got $HTTP_CODE${NC}"
fi
echo ""

sleep 1

echo -e "${BLUE}Test 2: Allowed payment within limits (amount < 5000)${NC}"
echo "Request: amount=1000, currency=USD"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Agent-ID: finance-agent" \
  -X POST $BASE_URL/tools/payments/create \
  -d '{"amount":1000,"currency":"USD","vendor_id":"V42","memo":"Office supplies"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
echo "Response ($HTTP_CODE): $BODY"
if [ "$HTTP_CODE" = "200" ]; then
  echo -e "${GREEN}✓ Test 2 PASSED: Payment created successfully${NC}"
else
  echo -e "${RED}✗ Test 2 FAILED: Expected 200, got $HTTP_CODE${NC}"
fi
echo ""

sleep 1

echo -e "${BLUE}Test 3: Allowed HR file read inside /hr-docs/${NC}"
echo "Request: path=/hr-docs/employee-handbook.pdf"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Agent-ID: hr-agent" \
  -X POST $BASE_URL/tools/files/read \
  -d '{"path":"/hr-docs/employee-handbook.pdf"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
echo "Response ($HTTP_CODE): $BODY"
if [ "$HTTP_CODE" = "200" ]; then
  echo -e "${GREEN}✓ Test 3 PASSED: File read successfully${NC}"
else
  echo -e "${RED}✗ Test 3 FAILED: Expected 200, got $HTTP_CODE${NC}"
fi
echo ""

sleep 1

echo -e "${BLUE}Test 4: Blocked HR file read outside /hr-docs/${NC}"
echo "Request: path=/legal/contract.docx"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Agent-ID: hr-agent" \
  -X POST $BASE_URL/tools/files/read \
  -d '{"path":"/legal/contract.docx"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
echo "Response ($HTTP_CODE): $BODY"
if [ "$HTTP_CODE" = "403" ]; then
  echo -e "${GREEN}✓ Test 4 PASSED: Request blocked as expected${NC}"
else
  echo -e "${RED}✗ Test 4 FAILED: Expected 403, got $HTTP_CODE${NC}"
fi
echo ""

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}Demo completed. Check logs/aegis.log for audit trail${NC}"
echo -e "${BLUE}============================================${NC}"
