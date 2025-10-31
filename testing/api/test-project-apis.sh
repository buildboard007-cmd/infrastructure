#!/bin/bash

# Comprehensive Project Management API Testing Script
# Tests all Project endpoints with proper error handling and output formatting

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
LOCATION_ID=6

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
echo -e "${YELLOW}Project Management API Test Suite${NC}"
echo -e "${YELLOW}========================================${NC}\n"

# Test 1: Create Project
echo -e "${BLUE}Test 1: Create Project${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"location_id\": $LOCATION_ID,
    \"basic_info\": {
        \"name\": \"Test Project - Residential Complex\",
        \"description\": \"A modern residential complex with sustainable design features\"
    },
    \"project_details\": {
        \"project_stage\": \"pre-construction\",
        \"work_scope\": \"new\",
        \"project_sector\": \"residential\",
        \"delivery_method\": \"design-build\",
        \"square_footage\": 75000,
        \"language\": \"en\",
        \"status\": \"active\"
    },
    \"location\": {
        \"address\": \"456 Residential Ave, Uptown, NY 10002\",
        \"city\": \"New York\",
        \"state\": \"NY\",
        \"zip_code\": \"10002\",
        \"country\": \"USA\"
    },
    \"timeline\": {
        \"start_date\": \"2025-06-01\",
        \"substantial_completion_date\": \"2026-10-15\",
        \"project_finish_date\": \"2026-12-01\"
    },
    \"financial\": {
        \"budget\": 8500000.00
    }
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    PROJECT_ID=$(echo "$BODY" | jq -r '.data.project_id')
    PROJECT_NUMBER=$(echo "$BODY" | jq -r '.data.project_number')
    echo "  Project ID: $PROJECT_ID"
    echo "  Project Number: $PROJECT_NUMBER"
    echo "  Message: $(echo "$BODY" | jq -r '.message')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
    PROJECT_ID=47  # Fallback ID for subsequent tests
fi

# Test 2: Get Project by ID
echo -e "\n${BLUE}Test 2: Get Project by ID${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/projects/$PROJECT_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Project ID: $(echo "$BODY" | jq -r '.project_id')"
    echo "  Name: $(echo "$BODY" | jq -r '.name')"
    echo "  Project Number: $(echo "$BODY" | jq -r '.project_number')"
    echo "  Status: $(echo "$BODY" | jq -r '.status')"
    echo "  Project Stage: $(echo "$BODY" | jq -r '.project_stage')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 3: Get All Projects
echo -e "\n${BLUE}Test 3: Get All Projects${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    TOTAL=$(echo "$BODY" | jq -r '.total')
    PROJECTS_COUNT=$(echo "$BODY" | jq -r '.projects | length')
    echo "  Total Projects: $TOTAL"
    echo "  Projects in response: $PROJECTS_COUNT"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 4: Get Projects by Location
echo -e "\n${BLUE}Test 4: Get Projects by Location${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/projects?location_id=$LOCATION_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    PROJECTS_COUNT=$(echo "$BODY" | jq -r '.projects | length')
    echo "  Projects at location $LOCATION_ID: $PROJECTS_COUNT"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 5: Update Project
echo -e "\n${BLUE}Test 5: Update Project${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X PUT \
  "$API_BASE/projects/$PROJECT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"location_id\": $LOCATION_ID,
    \"basic_info\": {
        \"name\": \"Test Project - Residential Complex (Updated)\",
        \"description\": \"Updated: A modern residential complex with enhanced sustainable design features\"
    },
    \"project_details\": {
        \"project_stage\": \"course-of-construction\",
        \"work_scope\": \"new\",
        \"project_sector\": \"residential\",
        \"delivery_method\": \"design-build\",
        \"square_footage\": 80000,
        \"language\": \"en\",
        \"status\": \"active\"
    },
    \"location\": {
        \"address\": \"456 Residential Ave, Uptown, NY 10002 - Updated\",
        \"city\": \"New York\",
        \"state\": \"NY\",
        \"zip_code\": \"10002\",
        \"country\": \"USA\"
    },
    \"timeline\": {
        \"start_date\": \"2025-06-01\",
        \"substantial_completion_date\": \"2026-11-01\",
        \"project_finish_date\": \"2026-12-15\"
    },
    \"financial\": {
        \"budget\": 9000000.00
    }
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Project ID: $(echo "$BODY" | jq -r '.project_id')"
    echo "  Name: $(echo "$BODY" | jq -r '.name')"
    echo "  Project Stage: $(echo "$BODY" | jq -r '.project_stage')"
    echo "  Square Footage: $(echo "$BODY" | jq -r '.square_footage')"
    echo "  Budget: $(echo "$BODY" | jq -r '.budget')"
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
