#!/bin/bash

# Comprehensive RFI API Testing Script
# Tests all RFI endpoints with proper error handling and output formatting

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
PROJECT_ID=47
LOCATION_ID=22

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
echo -e "${YELLOW}RFI API Comprehensive Test Suite${NC}"
echo -e "${YELLOW}========================================${NC}\n"

# Test 1: Create RFI with DRAFT status
echo -e "${BLUE}Test 1: Create RFI (DRAFT status)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$API_BASE/rfis" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"subject\": \"Test DRAFT RFI - Structural Beam Question\",
    \"description\": \"Need clarification on beam specifications before finalizing design\",
    \"category\": \"DESIGN\",
    \"priority\": \"MEDIUM\",
    \"status\": \"DRAFT\"
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    DRAFT_RFI_ID=$(echo "$BODY" | jq -r '.id')
    RFI_NUMBER=$(echo "$BODY" | jq -r '.rfi_number')
    echo "  RFI ID: $DRAFT_RFI_ID"
    echo "  RFI Number: $RFI_NUMBER (should be null)"
    echo "  Status: $(echo "$BODY" | jq -r '.status')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 2: Create RFI with OPEN status
echo -e "\n${BLUE}Test 2: Create RFI (OPEN status)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$API_BASE/rfis" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"subject\": \"Test OPEN RFI - HVAC System Location\",
    \"description\": \"Requires immediate clarification on HVAC placement\",
    \"category\": \"DESIGN\",
    \"priority\": \"HIGH\",
    \"status\": \"OPEN\"
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    OPEN_RFI_ID=$(echo "$BODY" | jq -r '.id')
    OPEN_RFI_NUMBER=$(echo "$BODY" | jq -r '.rfi_number')
    echo "  RFI ID: $OPEN_RFI_ID"
    echo "  RFI Number: $OPEN_RFI_NUMBER"
    echo "  Status: $(echo "$BODY" | jq -r '.status')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 3: List RFIs for project
echo -e "\n${BLUE}Test 3: List RFIs for project${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/projects/$PROJECT_ID/rfis" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    COUNT=$(echo "$BODY" | jq '. | length')
    echo "  Total RFIs: $COUNT"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 4: Get RFI by ID
echo -e "\n${BLUE}Test 4: Get RFI by ID${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/rfis/$OPEN_RFI_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Subject: $(echo "$BODY" | jq -r '.subject')"
    echo "  RFI Number: $(echo "$BODY" | jq -r '.rfi_number')"
    echo "  Status: $(echo "$BODY" | jq -r '.status')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 5: Update RFI priority
echo -e "\n${BLUE}Test 5: Update RFI (change priority)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X PUT \
  "$API_BASE/rfis/$OPEN_RFI_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": "CRITICAL"
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Priority: $(echo "$BODY" | jq -r '.priority')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 6: Update DRAFT to OPEN (should generate RFI number)
echo -e "\n${BLUE}Test 6: Update DRAFT to OPEN (generate RFI number)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X PUT \
  "$API_BASE/rfis/$DRAFT_RFI_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "OPEN"}')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    NEW_RFI_NUMBER=$(echo "$BODY" | jq -r '.rfi_number')
    echo "  Generated RFI Number: $NEW_RFI_NUMBER"
    echo "  Status: $(echo "$BODY" | jq -r '.status')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 7: Add Comment to RFI
echo -e "\n${BLUE}Test 7: Add Comment to RFI${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST \
  "$API_BASE/rfis/$OPEN_RFI_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "comment": "Reviewed with architect and structural engineer. Awaiting final approval."
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    COMMENT_ID=$(echo "$BODY" | jq -r '.id')
    echo "  Comment ID: $COMMENT_ID"
    echo "  Created By: $(echo "$BODY" | jq -r '.created_by_name')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 8: Verify RFI includes comments
echo -e "\n${BLUE}Test 8: Verify RFI includes comments${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/rfis/$OPEN_RFI_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    HAS_COMMENTS=$(echo "$BODY" | jq 'has("comments")')
    if [ "$HAS_COMMENTS" == "true" ]; then
        echo -e "${GREEN}✅ PASSED${NC}"
        COMMENT_COUNT=$(echo "$BODY" | jq '.comments | length')
        echo "  Comments field present with $COMMENT_COUNT comment(s)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAILED - Missing comments field${NC}"
        FAILED=$((FAILED + 1))
    fi
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 9: List RFIs with filters
echo -e "\n${BLUE}Test 9: List RFIs with filters (status=OPEN)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/projects/$PROJECT_ID/rfis?status=OPEN" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    COUNT=$(echo "$BODY" | jq '. | length')
    echo "  Filtered RFIs (OPEN): $COUNT"
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
