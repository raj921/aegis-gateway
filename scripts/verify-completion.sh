#!/bin/bash

# Aegis Gateway - Completion Verification Script
# This script verifies that all requirements are met

echo "============================================"
echo "Aegis Gateway - Completion Verification"
echo "============================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
PASSED=0
FAILED=0

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED++))
}

echo "1. Checking Project Structure..."
if [ -d "cmd/aegis" ] && [ -d "internal/gateway" ] && [ -d "internal/policy" ] && [ -d "internal/adapters" ]; then
    check_pass "Project structure is correct"
else
    check_fail "Project structure is missing directories"
fi

echo ""
echo "2. Checking Policy Files..."
POLICY_COUNT=$(find policies -name "*.yaml" 2>/dev/null | wc -l)
if [ $POLICY_COUNT -ge 1 ]; then
    check_pass "Policy files exist ($POLICY_COUNT files)"
else
    check_fail "Policy files are missing"
fi

echo ""
echo "3. Checking Docker Setup..."
if [ -f "deploy/docker-compose.yml" ] && [ -f "deploy/Dockerfile" ] && [ -f "deploy/otel-collector-config.yaml" ]; then
    check_pass "Docker configuration complete"
else
    check_fail "Docker configuration incomplete"
fi

echo ""
echo "4. Checking Documentation..."
DOCS=("README.md" "DESIGN_DOCUMENT.md" "QUICK_START.md" "COMPLETION_SUMMARY.md")
DOC_COUNT=0
for doc in "${DOCS[@]}"; do
    if [ -f "$doc" ]; then
        ((DOC_COUNT++))
    fi
done
if [ $DOC_COUNT -eq ${#DOCS[@]} ]; then
    check_pass "All documentation files present ($DOC_COUNT/${#DOCS[@]})"
else
    check_fail "Missing documentation files ($DOC_COUNT/${#DOCS[@]})"
fi

echo ""
echo "5. Running Unit Tests..."
if go test ./... > /dev/null 2>&1; then
    check_pass "All unit tests passing"
else
    check_fail "Some unit tests failing"
fi

echo ""
echo "6. Checking Test Files..."
TEST_FILES=$(find . -name "*_test.go" | wc -l)
if [ $TEST_FILES -ge 4 ]; then
    check_pass "Test files present ($TEST_FILES files)"
else
    check_fail "Insufficient test coverage ($TEST_FILES files)"
fi

echo ""
echo "7. Checking Go Module..."
if [ -f "go.mod" ] && [ -f "go.sum" ]; then
    check_pass "Go module configuration present"
else
    check_fail "Go module configuration missing"
fi

echo ""
echo "8. Checking Scripts..."
if [ -f "scripts/demo.sh" ] && [ -x "scripts/demo.sh" ]; then
    check_pass "Demo script present and executable"
else
    check_fail "Demo script missing or not executable"
fi

echo ""
echo "9. Checking Makefile..."
if [ -f "Makefile" ]; then
    MAKE_TARGETS=$(grep -c "^[a-z-]*:" Makefile)
    if [ $MAKE_TARGETS -ge 8 ]; then
        check_pass "Makefile with $MAKE_TARGETS targets"
    else
        check_fail "Makefile has insufficient targets"
    fi
else
    check_fail "Makefile missing"
fi

echo ""
echo "10. Checking Code Quality..."
# Check for common issues
if grep -r "TODO\|FIXME\|XXX" internal/ cmd/ pkg/ 2>/dev/null | grep -v Binary > /dev/null; then
    check_fail "Found TODO/FIXME comments in code"
else
    check_pass "No TODO/FIXME comments found"
fi

echo ""
echo "============================================"
echo "Verification Summary"
echo "============================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
else
    echo -e "${GREEN}Failed: $FAILED${NC}"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed! Project is complete.${NC}"
    exit 0
else
    echo -e "${RED}✗ Some checks failed. Please review.${NC}"
    exit 1
fi
