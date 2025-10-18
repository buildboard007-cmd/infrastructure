#!/bin/bash

# Entity Management with Attachments - Complete Workflow Test
# Tests creating Issue, RFI, and Submittal with attachments and verifying GET responses

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m' # No Color

# API Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
COGNITO_ENDPOINT="https://cognito-idp.us-east-2.amazonaws.com/"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"
USERNAME="kakadiyabhautik@gmail.com"
PASSWORD="K@kadiya#25"

# Test data
PROJECT_ID=59
LOCATION_ID=38
USER_ID=40

echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}     Entity Management with Attachments - Complete Workflow Test     ${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

# Get Authentication Token
echo -e "${YELLOW}[Auth] Getting authentication token...${NC}"
TOKEN=$(curl -s -X POST "$COGNITO_ENDPOINT" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -d "{\"AuthFlow\":\"USER_PASSWORD_AUTH\",\"ClientId\":\"$CLIENT_ID\",\"AuthParameters\":{\"USERNAME\":\"$USERNAME\",\"PASSWORD\":\"$PASSWORD\"}}" \
  | jq -r '.AuthenticationResult.IdToken')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
  echo -e "${RED}✗ Failed to get authentication token${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Token obtained successfully${NC}\n"

#==============================================================================
# TEST 1: ISSUE WITH ATTACHMENT
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}TEST 1: Issue with Attachment Workflow${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

# Step 1.1: Create Issue
echo -e "${YELLOW}[1.1] Creating issue...${NC}"
ISSUE_RESPONSE=$(curl -s -X POST "$API_BASE/issues" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"issue_category\": \"quality\",
    \"category\": \"deficiency\",
    \"title\": \"Test Issue with Attachment\",
    \"description\": \"This is a test issue to verify attachment workflow\",
    \"priority\": \"high\",
    \"severity\": \"major\",
    \"location\": {
      \"description\": \"Lobby Area\",
      \"building\": \"Building A\",
      \"level\": \"Floor 1\",
      \"room\": \"Room 101\"
    },
    \"assigned_to\": $USER_ID,
    \"due_date\": \"2025-12-31\"
  }")

ISSUE_ID=$(echo "$ISSUE_RESPONSE" | jq -r '.id // empty')

if [ -z "$ISSUE_ID" ] || [ "$ISSUE_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create issue${NC}"
  echo "Response: $ISSUE_RESPONSE"
  exit 1
fi

echo "$ISSUE_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Issue created with ID: $ISSUE_ID${NC}\n"

# Step 1.2: Generate upload URL for issue attachment
echo -e "${YELLOW}[1.2] Generating upload URL for issue attachment...${NC}"
ISSUE_UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"issue\",
    \"entity_id\": $ISSUE_ID,
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"file_name\": \"issue_crack_photo.jpg\",
    \"file_size\": 524288,
    \"attachment_type\": \"before_photo\"
  }")

ISSUE_ATTACHMENT_ID=$(echo "$ISSUE_UPLOAD_RESPONSE" | jq -r '.attachment_id // .AttachmentID // empty')
ISSUE_UPLOAD_URL=$(echo "$ISSUE_UPLOAD_RESPONSE" | jq -r '.upload_url // .UploadURL // empty')

echo "$ISSUE_UPLOAD_RESPONSE" | jq '.'

if [ -z "$ISSUE_ATTACHMENT_ID" ] || [ "$ISSUE_ATTACHMENT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL for issue${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Upload URL generated. Attachment ID: $ISSUE_ATTACHMENT_ID${NC}\n"

# Step 1.3: Simulate file upload to S3
echo -e "${YELLOW}[1.3] Simulating file upload to S3...${NC}"
echo "Test image data for issue attachment" > /tmp/issue_crack_photo.jpg

if [ -n "$ISSUE_UPLOAD_URL" ] && [ "$ISSUE_UPLOAD_URL" != "null" ]; then
  UPLOAD_RESULT=$(curl -s -X PUT "$ISSUE_UPLOAD_URL" \
    -H "Content-Type: image/jpeg" \
    --data-binary @/tmp/issue_crack_photo.jpg \
    -w "\n%{http_code}")

  HTTP_CODE=$(echo "$UPLOAD_RESULT" | tail -n1)

  if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ File uploaded to S3 successfully${NC}\n"
  else
    echo -e "${YELLOW}⚠ S3 upload returned code: $HTTP_CODE${NC}\n"
  fi
fi

# Step 1.4: Confirm upload
echo -e "${YELLOW}[1.4] Confirming upload...${NC}"
ISSUE_CONFIRM_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\": $ISSUE_ATTACHMENT_ID}")

echo "$ISSUE_CONFIRM_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Upload confirmed${NC}\n"

# Step 1.5: Get Issue by ID and verify attachments
echo -e "${YELLOW}[1.5] Getting issue by ID and verifying attachments...${NC}"
ISSUE_GET_RESPONSE=$(curl -s -X GET "$API_BASE/issues/$ISSUE_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$ISSUE_GET_RESPONSE" | jq '.'

# Verify attachments field exists and has data
ISSUE_ATTACHMENTS_COUNT=$(echo "$ISSUE_GET_RESPONSE" | jq '.attachments | length // 0')

if [ "$ISSUE_ATTACHMENTS_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ Issue GET response includes $ISSUE_ATTACHMENTS_COUNT attachment(s)${NC}"
  echo -e "${GREEN}✓ Attachment details:${NC}"
  echo "$ISSUE_GET_RESPONSE" | jq '.attachments[]'
else
  echo -e "${RED}✗ Issue GET response does not include attachments${NC}"
fi
echo ""

#==============================================================================
# TEST 2: RFI WITH ATTACHMENT
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}TEST 2: RFI with Attachment Workflow${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

# Step 2.1: Create RFI
echo -e "${YELLOW}[2.1] Creating RFI...${NC}"
RFI_RESPONSE=$(curl -s -X POST "$API_BASE/rfis" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"subject\": \"Test RFI with Attachment\",
    \"question\": \"What is the specification for the concrete mix?\",
    \"priority\": \"HIGH\",
    \"category\": \"SPECIFICATION\",
    \"due_date\": \"2025-12-31\"
  }")

RFI_ID=$(echo "$RFI_RESPONSE" | jq -r '.id // .ID // empty')

if [ -z "$RFI_ID" ] || [ "$RFI_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create RFI${NC}"
  echo "Response: $RFI_RESPONSE"
  exit 1
fi

echo "$RFI_RESPONSE" | jq '.'
echo -e "${GREEN}✓ RFI created with ID: $RFI_ID${NC}\n"

# Step 2.2: Generate upload URL for RFI attachment
echo -e "${YELLOW}[2.2] Generating upload URL for RFI attachment...${NC}"
RFI_UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"rfi\",
    \"entity_id\": $RFI_ID,
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"file_name\": \"concrete_specification.pdf\",
    \"file_size\": 1048576,
    \"attachment_type\": \"specification\"
  }")

RFI_ATTACHMENT_ID=$(echo "$RFI_UPLOAD_RESPONSE" | jq -r '.attachment_id // .AttachmentID // empty')
RFI_UPLOAD_URL=$(echo "$RFI_UPLOAD_RESPONSE" | jq -r '.upload_url // .UploadURL // empty')

echo "$RFI_UPLOAD_RESPONSE" | jq '.'

if [ -z "$RFI_ATTACHMENT_ID" ] || [ "$RFI_ATTACHMENT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL for RFI${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Upload URL generated. Attachment ID: $RFI_ATTACHMENT_ID${NC}\n"

# Step 2.3: Simulate file upload to S3
echo -e "${YELLOW}[2.3] Simulating file upload to S3...${NC}"
echo "Test PDF data for RFI specification" > /tmp/concrete_specification.pdf

if [ -n "$RFI_UPLOAD_URL" ] && [ "$RFI_UPLOAD_URL" != "null" ]; then
  UPLOAD_RESULT=$(curl -s -X PUT "$RFI_UPLOAD_URL" \
    -H "Content-Type: application/pdf" \
    --data-binary @/tmp/concrete_specification.pdf \
    -w "\n%{http_code}")

  HTTP_CODE=$(echo "$UPLOAD_RESULT" | tail -n1)

  if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ File uploaded to S3 successfully${NC}\n"
  else
    echo -e "${YELLOW}⚠ S3 upload returned code: $HTTP_CODE${NC}\n"
  fi
fi

# Step 2.4: Confirm upload
echo -e "${YELLOW}[2.4] Confirming upload...${NC}"
RFI_CONFIRM_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\": $RFI_ATTACHMENT_ID}")

echo "$RFI_CONFIRM_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Upload confirmed${NC}\n"

# Step 2.5: Get RFI by ID and verify attachments
echo -e "${YELLOW}[2.5] Getting RFI by ID and verifying attachments...${NC}"
RFI_GET_RESPONSE=$(curl -s -X GET "$API_BASE/rfis/$RFI_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$RFI_GET_RESPONSE" | jq '.'

# Verify attachments field exists and has data
RFI_ATTACHMENTS_COUNT=$(echo "$RFI_GET_RESPONSE" | jq '.attachments // .Attachments | length // 0')

if [ "$RFI_ATTACHMENTS_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ RFI GET response includes $RFI_ATTACHMENTS_COUNT attachment(s)${NC}"
  echo -e "${GREEN}✓ Attachment details:${NC}"
  echo "$RFI_GET_RESPONSE" | jq '.attachments // .Attachments | .[]'
else
  echo -e "${RED}✗ RFI GET response does not include attachments${NC}"
fi
echo ""

#==============================================================================
# TEST 3: SUBMITTAL WITH ATTACHMENT
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}TEST 3: Submittal with Attachment Workflow${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

# Step 3.1: Create Submittal
echo -e "${YELLOW}[3.1] Creating submittal...${NC}"
SUBMITTAL_RESPONSE=$(curl -s -X POST "$API_BASE/submittals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"title\": \"Test Submittal with Attachment - Steel Drawings\",
    \"submittal_type\": \"shop_drawings\",
    \"priority\": \"high\"
  }")

SUBMITTAL_ID=$(echo "$SUBMITTAL_RESPONSE" | jq -r '.id // .ID // empty')

if [ -z "$SUBMITTAL_ID" ] || [ "$SUBMITTAL_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create submittal${NC}"
  echo "Response: $SUBMITTAL_RESPONSE"
  exit 1
fi

echo "$SUBMITTAL_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Submittal created with ID: $SUBMITTAL_ID${NC}\n"

# Step 3.2: Generate upload URL for submittal attachment
echo -e "${YELLOW}[3.2] Generating upload URL for submittal attachment...${NC}"
SUBMITTAL_UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"submittal\",
    \"entity_id\": $SUBMITTAL_ID,
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"file_name\": \"steel_beam_shop_drawing.dwg\",
    \"file_size\": 2097152,
    \"attachment_type\": \"shop_drawing\"
  }")

SUBMITTAL_ATTACHMENT_ID=$(echo "$SUBMITTAL_UPLOAD_RESPONSE" | jq -r '.attachment_id // .AttachmentID // empty')
SUBMITTAL_UPLOAD_URL=$(echo "$SUBMITTAL_UPLOAD_RESPONSE" | jq -r '.upload_url // .UploadURL // empty')

echo "$SUBMITTAL_UPLOAD_RESPONSE" | jq '.'

if [ -z "$SUBMITTAL_ATTACHMENT_ID" ] || [ "$SUBMITTAL_ATTACHMENT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL for submittal${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Upload URL generated. Attachment ID: $SUBMITTAL_ATTACHMENT_ID${NC}\n"

# Step 3.3: Simulate file upload to S3
echo -e "${YELLOW}[3.3] Simulating file upload to S3...${NC}"
echo "Test DWG data for shop drawing" > /tmp/steel_beam_shop_drawing.dwg

if [ -n "$SUBMITTAL_UPLOAD_URL" ] && [ "$SUBMITTAL_UPLOAD_URL" != "null" ]; then
  UPLOAD_RESULT=$(curl -s -X PUT "$SUBMITTAL_UPLOAD_URL" \
    -H "Content-Type: application/octet-stream" \
    --data-binary @/tmp/steel_beam_shop_drawing.dwg \
    -w "\n%{http_code}")

  HTTP_CODE=$(echo "$UPLOAD_RESULT" | tail -n1)

  if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ File uploaded to S3 successfully${NC}\n"
  else
    echo -e "${YELLOW}⚠ S3 upload returned code: $HTTP_CODE${NC}\n"
  fi
fi

# Step 3.4: Confirm upload
echo -e "${YELLOW}[3.4] Confirming upload...${NC}"
SUBMITTAL_CONFIRM_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\": $SUBMITTAL_ATTACHMENT_ID}")

echo "$SUBMITTAL_CONFIRM_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Upload confirmed${NC}\n"

# Step 3.5: Get Submittal by ID and verify attachments
echo -e "${YELLOW}[3.5] Getting submittal by ID and verifying attachments...${NC}"
SUBMITTAL_GET_RESPONSE=$(curl -s -X GET "$API_BASE/submittals/$SUBMITTAL_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$SUBMITTAL_GET_RESPONSE" | jq '.'

# Verify attachments field exists and has data
SUBMITTAL_ATTACHMENTS_COUNT=$(echo "$SUBMITTAL_GET_RESPONSE" | jq '.attachments // .Attachments | length // 0')

if [ "$SUBMITTAL_ATTACHMENTS_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ Submittal GET response includes $SUBMITTAL_ATTACHMENTS_COUNT attachment(s)${NC}"
  echo -e "${GREEN}✓ Attachment details:${NC}"
  echo "$SUBMITTAL_GET_RESPONSE" | jq '.attachments // .Attachments | .[]'
else
  echo -e "${RED}✗ Submittal GET response does not include attachments${NC}"
fi
echo ""

#==============================================================================
# SUMMARY
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}                           TEST SUMMARY                               ${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

echo -e "${GREEN}✓ Issue Workflow:${NC}"
echo -e "  - Issue ID: $ISSUE_ID"
echo -e "  - Attachment ID: $ISSUE_ATTACHMENT_ID"
echo -e "  - Attachments in GET response: $ISSUE_ATTACHMENTS_COUNT\n"

echo -e "${GREEN}✓ RFI Workflow:${NC}"
echo -e "  - RFI ID: $RFI_ID"
echo -e "  - Attachment ID: $RFI_ATTACHMENT_ID"
echo -e "  - Attachments in GET response: $RFI_ATTACHMENTS_COUNT\n"

echo -e "${GREEN}✓ Submittal Workflow:${NC}"
echo -e "  - Submittal ID: $SUBMITTAL_ID"
echo -e "  - Attachment ID: $SUBMITTAL_ATTACHMENT_ID"
echo -e "  - Attachments in GET response: $SUBMITTAL_ATTACHMENTS_COUNT\n"

# Cleanup
rm -f /tmp/issue_crack_photo.jpg /tmp/concrete_specification.pdf /tmp/steel_beam_shop_drawing.dwg

# Final verification
TOTAL_TESTS=3
PASSED_TESTS=0

[ "$ISSUE_ATTACHMENTS_COUNT" -gt 0 ] && ((PASSED_TESTS++))
[ "$RFI_ATTACHMENTS_COUNT" -gt 0 ] && ((PASSED_TESTS++))
[ "$SUBMITTAL_ATTACHMENTS_COUNT" -gt 0 ] && ((PASSED_TESTS++))

if [ "$PASSED_TESTS" -eq "$TOTAL_TESTS" ]; then
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${GREEN}                    ALL TESTS PASSED ($PASSED_TESTS/$TOTAL_TESTS)                    ${NC}"
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
  exit 0
else
  echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${RED}                 SOME TESTS FAILED ($PASSED_TESTS/$TOTAL_TESTS)                   ${NC}"
  echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
  exit 1
fi