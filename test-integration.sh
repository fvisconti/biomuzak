#!/usr/bin/env bash
# Integration test script for biomuzak
# Tests the full stack: frontend, backend, database, and audio processor

set -e

echo "🧪 biomuzak Integration Test Suite"
echo "=================================="
echo ""

# Configuration
BACKEND_URL=${BACKEND_URL:-http://localhost:8080}
AUDIO_PROCESSOR_URL=${AUDIO_PROCESSOR_URL:-http://localhost:8000}
FRONTEND_URL=${FRONTEND_URL:-http://localhost:3000}

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Test 1: Backend health check
echo "Test 1: Backend Health Check"
if curl -s -f "${BACKEND_URL}/api/health" > /dev/null; then
    RESPONSE=$(curl -s "${BACKEND_URL}/api/health")
    if echo "$RESPONSE" | grep -q '"status":"ok"'; then
        pass "Backend is healthy"
    else
        fail "Backend health check returned unexpected response"
    fi
else
    fail "Backend is not responding"
fi
echo ""

# Test 2: Audio processor health check
echo "Test 2: Audio Processor Health Check"
if curl -s -f "${AUDIO_PROCESSOR_URL}/" > /dev/null; then
    RESPONSE=$(curl -s "${AUDIO_PROCESSOR_URL}/")
    if echo "$RESPONSE" | grep -q "Audio processing service is running"; then
        pass "Audio processor is healthy"
    else
        fail "Audio processor returned unexpected response"
    fi
else
    fail "Audio processor is not responding"
fi
echo ""

# Test 3: Frontend accessibility
echo "Test 3: Frontend Accessibility"
if curl -s -f "${FRONTEND_URL}/" > /dev/null; then
    pass "Frontend is accessible"
else
    fail "Frontend is not accessible"
fi
echo ""

# Test 4: User registration
echo "Test 4: User Registration"
USERNAME="testuser_$(date +%s)_$$"
PASSWORD="testpass123"
EMAIL="${USERNAME}@example.com"

REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BACKEND_URL}/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${USERNAME}\",\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")

HTTP_CODE=$(echo "$REGISTER_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "User registration successful"
else
    fail "User registration failed (HTTP $HTTP_CODE)"
fi
echo ""

# Test 5: User login
echo "Test 5: User Login"
LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BACKEND_URL}/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}")

HTTP_CODE=$(echo "$LOGIN_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$LOGIN_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    # Try to extract token with jq if available, fallback to grep
    if command -v jq &> /dev/null; then
        TOKEN=$(echo "$RESPONSE_BODY" | jq -r '.token // empty')
    else
        TOKEN=$(echo "$RESPONSE_BODY" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -n "$TOKEN" ]; then
        pass "User login successful (token received)"
    else
        fail "User login succeeded but no token received"
    fi
else
    fail "User login failed (HTTP $HTTP_CODE)"
fi
echo ""

# Test 6: Protected endpoint access
echo "Test 6: Protected Endpoint Access"
if [ -n "$TOKEN" ]; then
    LIBRARY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BACKEND_URL}/api/library" \
        -H "Authorization: Bearer ${TOKEN}")
    
    HTTP_CODE=$(echo "$LIBRARY_RESPONSE" | tail -n1)
    if [ "$HTTP_CODE" = "200" ]; then
        pass "Protected endpoint accessible with valid token"
    else
        fail "Protected endpoint access failed (HTTP $HTTP_CODE)"
    fi
else
    fail "Cannot test protected endpoint (no token)"
fi
echo ""

# Test 7: Subsonic API - Ping
echo "Test 7: Subsonic API - Ping"
SUBSONIC_RESPONSE=$(curl -s "${BACKEND_URL}/rest/ping.view?u=${USERNAME}&p=${PASSWORD}&v=1.16.1&c=integration-test")
if echo "$SUBSONIC_RESPONSE" | grep -q 'status="ok"'; then
    pass "Subsonic API ping successful"
else
    fail "Subsonic API ping failed"
fi
echo ""

# Test 8: Subsonic API - Get Music Folders
echo "Test 8: Subsonic API - Get Music Folders"
FOLDERS_RESPONSE=$(curl -s "${BACKEND_URL}/rest/getMusicFolders.view?u=${USERNAME}&p=${PASSWORD}&v=1.16.1&c=integration-test")
if echo "$FOLDERS_RESPONSE" | grep -q 'musicFolders'; then
    pass "Subsonic API getMusicFolders successful"
else
    fail "Subsonic API getMusicFolders failed"
fi
echo ""

# Summary
echo "=================================="
echo "Test Results:"
echo "  Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo "  Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC} 🎉"
    exit 0
else
    echo -e "${RED}Some tests failed${NC} ❌"
    exit 1
fi
