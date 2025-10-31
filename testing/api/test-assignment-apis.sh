#!/bin/bash

# Comprehensive Assignment Management API Testing Script
# Tests all Assignment endpoints with proper error handling and output formatting

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
PROJECT_ID=47
USER_ID=20
ROLE_ID=8

# Counter for test results
PASSED=0
FAILED=0

# Get authentication token
echo -e "${BLUE}Getting authentication token...${NC}"
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' \
  | jq -r '.AuthenticationResult.IdToken')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo -e "${RED}❌ Failed to get authentication token${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Token retrieved${NC}\n"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Assignment Management API Test Suite${NC}"
echo -e "${YELLOW}========================================${NC}\n"

# Cleanup: Delete any existing test assignments to avoid duplicate errors
echo -e "${BLUE}Cleanup: Removing existing test assignments...${NC}"
DELETED_COUNT=0
curl -s -X GET "$API_BASE/contexts/project/$PROJECT_ID/assignments" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.assignments[] | select(.user_id == 20 or .user_id == 22) | .id' | while read aid; do
  if [ ! -z "$aid" ]; then
    RESULT=$(curl -s -X DELETE "$API_BASE/assignments/$aid" -H "Authorization: Bearer $TOKEN")
    DELETED_COUNT=$((DELETED_COUNT + 1))
    echo "  Deleted assignment ID: $aid"
  fi
done
sleep 1  # Wait for deletions to propagate
echo -e "${GREEN}✅ Cleanup complete${NC}\n"

# Test 1: Create Assignment
echo -e "${BLUE}Test 1: Create Assignment (Project Manager)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": $USER_ID,
    \"role_id\": $ROLE_ID,
    \"context_type\": \"project\",
    \"context_id\": $PROJECT_ID,
    \"trade_type\": \"electrical\",
    \"is_primary\": true,
    \"start_date\": \"2025-01-01\",
    \"end_date\": \"2025-12-31\"
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    ASSIGNMENT_ID=$(echo "$BODY" | jq -r '.id')
    echo "  Assignment ID: $ASSIGNMENT_ID"
    echo "  User Name: $(echo "$BODY" | jq -r '.user_name')"
    echo "  Role Name: $(echo "$BODY" | jq -r '.role_name')"
    echo "  Context Type: $(echo "$BODY" | jq -r '.context_type')"
    echo "  Context Name: $(echo "$BODY" | jq -r '.context_name')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
    ASSIGNMENT_ID=1  # Fallback ID for subsequent tests
fi

# Test 2: Get Assignment by ID
echo -e "\n${BLUE}Test 2: Get Assignment by ID${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/assignments/$ASSIGNMENT_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Assignment ID: $(echo "$BODY" | jq -r '.id')"
    echo "  User Name: $(echo "$BODY" | jq -r '.user_name')"
    echo "  User Email: $(echo "$BODY" | jq -r '.user_email')"
    echo "  Role Name: $(echo "$BODY" | jq -r '.role_name')"
    echo "  Context: $(echo "$BODY" | jq -r '.context_type') - $(echo "$BODY" | jq -r '.context_name')"
    echo "  Is Primary: $(echo "$BODY" | jq -r '.is_primary')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 3: Update Assignment
echo -e "\n${BLUE}Test 3: Update Assignment (change role and trade)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X PUT \
  "$API_BASE/assignments/$ASSIGNMENT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 9,
    "trade_type": "plumbing",
    "is_primary": false,
    "end_date": "2025-06-30"
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Role Name: $(echo "$BODY" | jq -r '.role_name')"
    echo "  Trade Type: $(echo "$BODY" | jq -r '.trade_type')"
    echo "  Is Primary: $(echo "$BODY" | jq -r '.is_primary')"
    echo "  End Date: $(echo "$BODY" | jq -r '.end_date')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 4: Get Context Assignments (Project)
echo -e "\n${BLUE}Test 4: Get Context Assignments for Project${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/contexts/project/$PROJECT_ID/assignments" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    ASSIGNMENTS_COUNT=$(echo "$BODY" | jq -r '.assignments | length')
    echo "  Context Type: $(echo "$BODY" | jq -r '.context_type')"
    echo "  Context Name: $(echo "$BODY" | jq -r '.context_name')"
    echo "  Total Assignments: $ASSIGNMENTS_COUNT"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 5: Create Another Assignment (Site Supervisor)
echo -e "\n${BLUE}Test 5: Create Another Assignment (Site Supervisor with different user)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 22,
    "role_id": 9,
    "context_type": "project",
    "context_id": 47,
    "trade_type": "hvac",
    "is_primary": false,
    "start_date": "2025-02-01"
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    ASSIGNMENT_ID_2=$(echo "$BODY" | jq -r '.id')
    echo "  Assignment ID: $ASSIGNMENT_ID_2"
    echo "  User Name: $(echo "$BODY" | jq -r '.user_name')"
    echo "  Role Name: $(echo "$BODY" | jq -r '.role_name')"
    echo "  Trade Type: $(echo "$BODY" | jq -r '.trade_type')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 6: Verify Multiple Assignments in Context
echo -e "\n${BLUE}Test 6: Verify Multiple Assignments in Project Context${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/contexts/project/$PROJECT_ID/assignments" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    ASSIGNMENTS_COUNT=$(echo "$BODY" | jq -r '.assignments | length')
    echo "  Total Assignments: $ASSIGNMENTS_COUNT"
    echo "  First Assignment: $(echo "$BODY" | jq -r '.assignments[0].user_name') - $(echo "$BODY" | jq -r '.assignments[0].role_name')"
    if [ "$ASSIGNMENTS_COUNT" -gt 1 ]; then
        echo "  Second Assignment: $(echo "$BODY" | jq -r '.assignments[1].user_name') - $(echo "$BODY" | jq -r '.assignments[1].role_name')"
    fi
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test Summary
echo -e "\n${YELLOW}========================================${NC}"
echo -e "${YELLOW}Test Summary${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "Total: $((PASSED + FAILED))"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some tests failed${NC}"
    exit 1
fi
