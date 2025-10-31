#!/bin/bash

# Comprehensive Submittal Management API Testing Script
# Tests all Submittal endpoints with proper error handling and output formatting

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
PROJECT_ID=47
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
echo -e "${YELLOW}Submittal Management API Test Suite${NC}"
echo -e "${YELLOW}========================================${NC}\n"

# Test 1: Create Submittal
echo -e "${BLUE}Test 1: Create Submittal${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$API_BASE/submittals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"package_name\": \"Structural Steel Package - Phase 1\",
    \"csi_division\": \"05\",
    \"csi_section\": \"05 12 00\",
    \"title\": \"Shop Drawings for Steel Beam Connections\",
    \"description\": \"Detailed shop drawings for structural steel beam-to-column connections\",
    \"submittal_type\": \"shop_drawings\",
    \"specification_section\": \"051200\",
    \"priority\": \"high\",
    \"current_phase\": \"preparation\",
    \"ball_in_court\": \"contractor\",
    \"workflow_status\": \"pending_submission\",
    \"assigned_to\": 1,
    \"reviewer\": 2,
    \"approver\": 3,
    \"required_approval_date\": \"2025-03-15\",
    \"fabrication_start_date\": \"2025-04-01\",
    \"installation_date\": \"2025-05-15\",
    \"delivery_tracking\": {
        \"anticipated_delivery_date\": \"2025-05-10\",
        \"order_date\": \"2025-03-20\",
        \"delivery_status\": \"not_ordered\",
        \"vendor_info\": {
            \"name\": \"Steel Fabricators Inc\",
            \"contact_email\": \"contact@steelfab.com\",
            \"contact_phone\": \"+1-555-0123\"
        }
    },
    \"team_assignments\": {
        \"lead_architect\": 2,
        \"lead_engineer\": 4,
        \"project_manager\": 1,
        \"contractor_rep\": 5
    },
    \"linked_drawings\": {
        \"drawing_numbers\": [\"S-101\", \"S-102\"],
        \"drawing_revisions\": [\"R1\", \"R1\"],
        \"detail_references\": [\"DET-1\", \"DET-2\"]
    },
    \"submittal_references\": {
        \"specification_sections\": [\"051200\", \"051300\"],
        \"related_submittals\": [],
        \"related_rfis\": [],
        \"related_issues\": []
    },
    \"tags\": [\"urgent\", \"structural\", \"phase1\"]
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "201" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    SUBMITTAL_ID=$(echo "$BODY" | jq -r '.id')
    SUBMITTAL_NUMBER=$(echo "$BODY" | jq -r '.submittal_number')
    echo "  Submittal ID: $SUBMITTAL_ID"
    echo "  Submittal Number: $SUBMITTAL_NUMBER"
    echo "  Title: $(echo "$BODY" | jq -r '.title')"
    echo "  Workflow Status: $(echo "$BODY" | jq -r '.workflow_status')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
    SUBMITTAL_ID=1  # Fallback ID for subsequent tests
fi

# Test 2: Get Submittal by ID
echo -e "\n${BLUE}Test 2: Get Submittal by ID${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/submittals/$SUBMITTAL_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Submittal Number: $(echo "$BODY" | jq -r '.submittal_number')"
    echo "  Title: $(echo "$BODY" | jq -r '.title')"
    echo "  Workflow Status: $(echo "$BODY" | jq -r '.workflow_status')"
    echo "  Priority: $(echo "$BODY" | jq -r '.priority')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 3: List Submittals for project
echo -e "\n${BLUE}Test 3: List Submittals for project${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/contexts/project/$PROJECT_ID/submittals?page=1&limit=20&sort=created_at&order=desc" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    TOTAL=$(echo "$BODY" | jq -r '.total_count')
    PAGE=$(echo "$BODY" | jq -r '.page')
    PAGE_SIZE=$(echo "$BODY" | jq -r '.page_size')
    SUBMITTALS_COUNT=$(echo "$BODY" | jq -r '.submittals | length')
    echo "  Total Submittals: $TOTAL"
    echo "  Page: $PAGE"
    echo "  Page Size: $PAGE_SIZE"
    echo "  Submittals in response: $SUBMITTALS_COUNT"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 4: List Submittals with filters
echo -e "\n${BLUE}Test 4: List Submittals with filters (status=pending_submission, priority=high)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/contexts/project/$PROJECT_ID/submittals?status=pending_submission&priority=high&page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    FILTERED_COUNT=$(echo "$BODY" | jq -r '.submittals | length')
    echo "  Filtered Submittals: $FILTERED_COUNT"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 5: Update Submittal (Regular Update)
echo -e "\n${BLUE}Test 5: Update Submittal (change priority and description)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X PUT \
  "$API_BASE/submittals/$SUBMITTAL_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": "critical",
    "description": "Updated: Detailed shop drawings with revised connection details per architect comments",
    "delivery_tracking": {
        "anticipated_delivery_date": "2025-05-08",
        "order_date": "2025-03-18",
        "delivery_status": "ordered",
        "tracking_number": "TRK-123456"
    }
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Priority: $(echo "$BODY" | jq -r '.priority')"
    echo "  Workflow Status: $(echo "$BODY" | jq -r '.workflow_status')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 6: Submit for Review (Workflow Action)
echo -e "\n${BLUE}Test 6: Submit for Review (Workflow Action)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST \
  "$API_BASE/submittals/$SUBMITTAL_ID/workflow" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "submit_for_review",
    "comments": "Submittal is ready for architectural review",
    "next_reviewer": 2,
    "ball_in_court_transfer": "architect"
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Workflow Status: $(echo "$BODY" | jq -r '.workflow_status')"
    echo "  Ball in Court: $(echo "$BODY" | jq -r '.ball_in_court')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 7: Approve as Noted (Workflow Action)
echo -e "\n${BLUE}Test 7: Approve as Noted (Workflow Action)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST \
  "$API_BASE/submittals/$SUBMITTAL_ID/workflow" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "approve_as_noted",
    "comments": "Submittal approved with minor revisions noted",
    "conditions": "1. Verify weld symbols. 2. Add fireproofing notes. 3. Confirm bolt grades.",
    "ball_in_court_transfer": "contractor"
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Workflow Status: $(echo "$BODY" | jq -r '.workflow_status')"
    echo "  Ball in Court: $(echo "$BODY" | jq -r '.ball_in_court')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 8: Get Submittal Statistics
echo -e "\n${BLUE}Test 8: Get Submittal Statistics${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X GET \
  "$API_BASE/contexts/project/$PROJECT_ID/submittals/stats" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Total: $(echo "$BODY" | jq -r '.total')"
    echo "  By Status: $(echo "$BODY" | jq -r '.by_status | keys | join(", ")')"
    echo "  Overdue: $(echo "$BODY" | jq -r '.overdue')"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 9: Generate Upload URL for Submittal Attachment
echo -e "\n${BLUE}Test 9: Generate Upload URL for Submittal Attachment${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST \
  "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"submittal\",
    \"entity_id\": $SUBMITTAL_ID,
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"file_name\": \"steel_connection_details_rev_a.pdf\",
    \"file_size\": 2548736,
    \"attachment_type\": \"shop_drawing\"
  }")

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    ATTACHMENT_ID=$(echo "$BODY" | jq -r '.attachment_id')
    echo "  Attachment ID: $ATTACHMENT_ID"
    echo "  Upload URL generated successfully"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "$BODY" | jq '.'
    FAILED=$((FAILED + 1))
fi

# Test 10: Revise and Resubmit (Workflow Action)
echo -e "\n${BLUE}Test 10: Revise and Resubmit (Workflow Action)${NC}"
RESULT=$(curl -s -w "\nHTTP:%{http_code}" -X POST \
  "$API_BASE/submittals/$SUBMITTAL_ID/workflow" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "revise_resubmit",
    "comments": "Several issues need to be addressed before approval",
    "revision_notes": "1. Connection details do not match drawings. 2. Missing seismic bracing. 3. Update weld specs.",
    "ball_in_court_transfer": "contractor"
  }')

HTTP_CODE=$(echo "$RESULT" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESULT" | grep -v "HTTP:")

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ PASSED${NC}"
    echo "  Workflow Status: $(echo "$BODY" | jq -r '.workflow_status')"
    echo "  Ball in Court: $(echo "$BODY" | jq -r '.ball_in_court')"
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
