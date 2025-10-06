#!/bin/bash

# Entity Management with Attachments - Complete Workflow Test
# Uses correct API formats from Postman collections

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m'

# API Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
COGNITO_ENDPOINT="https://cognito-idp.us-east-2.amazonaws.com/"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"
USERNAME="buildboard007+555@gmail.com"
PASSWORD="Mayur@1234"

# User Context (from JWT token)
PROJECT_ID=49
LOCATION_ID=24
USER_ID=19
ORG_ID=10

echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}     Entity Management with Attachments - Complete Workflow Test     ${NC}"
echo -e "${BLUE}======================================================================${NC}\n"
echo -e "${YELLOW}User Context:${NC}"
echo -e "  - User ID: $USER_ID"
echo -e "  - Org ID: $ORG_ID"
echo -e "  - Location ID: $LOCATION_ID"
echo -e "  - Project ID: $PROJECT_ID\n"

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

# Step 1.1: Create Issue (using Postman format)
echo -e "${YELLOW}[1.1] Creating issue...${NC}"
ISSUE_RESPONSE=$(curl -s -X POST "$API_BASE/issues" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"issue_category\": \"quality\",
    \"category\": \"defect\",
    \"detail_category\": \"finish_defect\",
    \"title\": \"Test Issue with Attachment - Wall crack\",
    \"description\": \"This is a test issue to verify attachment workflow. Visible crack in wall.\",
    \"priority\": \"medium\",
    \"severity\": \"minor\",
    \"location\": {
      \"description\": \"Test room\",
      \"building\": \"Building A\",
      \"level\": \"Floor 1\",
      \"room\": \"Room 101\"
    },
    \"discipline\": \"drywall\",
    \"trade\": \"finishing\",
    \"assigned_to\": $USER_ID,
    \"due_date\": \"2025-12-31\"
  }")

ISSUE_ID=$(echo "$ISSUE_RESPONSE" | jq -r '.id // empty')

if [ -z "$ISSUE_ID" ] || [ "$ISSUE_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create issue${NC}"
  echo "Response: $ISSUE_RESPONSE"
  exit 1
fi

echo "$ISSUE_RESPONSE" | jq '{id, issue_number, title, status, priority}'
echo -e "${GREEN}✓ Issue created with ID: $ISSUE_ID${NC}\n"

# Step 1.2: Upload attachment
echo -e "${YELLOW}[1.2] Uploading attachment for issue...${NC}"
ISSUE_UPLOAD=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"issue\",\"entity_id\":$ISSUE_ID,\"project_id\":$PROJECT_ID,\"location_id\":$LOCATION_ID,\"file_name\":\"crack_photo.jpg\",\"file_size\":524288,\"attachment_type\":\"before_photo\"}")

ISSUE_ATT_ID=$(echo "$ISSUE_UPLOAD" | jq -r '.attachment_id // .AttachmentID')
ISSUE_UPLOAD_URL=$(echo "$ISSUE_UPLOAD" | jq -r '.upload_url // .UploadURL')

if [ -z "$ISSUE_ATT_ID" ] || [ "$ISSUE_ATT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL${NC}"
  exit 1
fi

echo "Test image" > /tmp/crack_photo.jpg
curl -s -X PUT "$ISSUE_UPLOAD_URL" -H "Content-Type: image/jpeg" --data-binary @/tmp/crack_photo.jpg -o /dev/null -w "%{http_code}\n" | grep -q "200" && echo -e "${GREEN}✓ File uploaded to S3${NC}" || echo -e "${YELLOW}⚠ S3 upload may have failed${NC}"

curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\":$ISSUE_ATT_ID}" | jq -r '.status' | grep -q "confirmed" && echo -e "${GREEN}✓ Upload confirmed (Attachment ID: $ISSUE_ATT_ID)${NC}\n"

# Step 1.3: Get Issue and verify attachments
echo -e "${YELLOW}[1.3] Getting issue by ID and verifying attachments...${NC}"
ISSUE_GET=$(curl -s -X GET "$API_BASE/issues/$ISSUE_ID" -H "Authorization: Bearer $TOKEN")
ISSUE_ATT_COUNT=$(echo "$ISSUE_GET" | jq '.attachments | length // 0')

echo "$ISSUE_GET" | jq '{id, title, status, attachments}'

if [ "$ISSUE_ATT_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ Issue GET includes $ISSUE_ATT_COUNT attachment(s)${NC}\n"
else
  echo -e "${RED}✗ Issue GET does NOT include attachments${NC}\n"
fi

#==============================================================================
# TEST 2: RFI WITH ATTACHMENT
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}TEST 2: RFI with Attachment Workflow${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

# Step 2.1: Create RFI (using Postman format)
echo -e "${YELLOW}[2.1] Creating RFI...${NC}"
RFI_RESPONSE=$(curl -s -X POST "$API_BASE/rfis" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"subject\": \"Test RFI with Attachment - Foundation Detail\",
    \"question\": \"What is the specification for the concrete mix at foundation level?\",
    \"description\": \"Need clarification on concrete specifications.\",
    \"category\": \"SPECIFICATION\",
    \"discipline\": \"structural\",
    \"trade_type\": \"concrete\",
    \"project_phase\": \"construction\",
    \"priority\": \"HIGH\",
    \"due_date\": \"2025-12-31\",
    \"location_description\": \"Foundation Area - Grid A-5\"
  }")

RFI_ID=$(echo "$RFI_RESPONSE" | jq -r '.id // .ID // empty')

if [ -z "$RFI_ID" ] || [ "$RFI_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create RFI${NC}"
  echo "Response: $RFI_RESPONSE"
  exit 1
fi

echo "$RFI_RESPONSE" | jq '{id, rfi_number, subject, status, priority}' 2>/dev/null || echo "$RFI_RESPONSE" | jq '{ID, RFINumber, Subject, Status, Priority}'
echo -e "${GREEN}✓ RFI created with ID: $RFI_ID${NC}\n"

# Step 2.2: Upload attachment
echo -e "${YELLOW}[2.2] Uploading attachment for RFI...${NC}"
RFI_UPLOAD=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"rfi\",\"entity_id\":$RFI_ID,\"project_id\":$PROJECT_ID,\"location_id\":$LOCATION_ID,\"file_name\":\"specification.pdf\",\"file_size\":1048576,\"attachment_type\":\"specification\"}")

RFI_ATT_ID=$(echo "$RFI_UPLOAD" | jq -r '.attachment_id // .AttachmentID')
RFI_UPLOAD_URL=$(echo "$RFI_UPLOAD" | jq -r '.upload_url // .UploadURL')

if [ -z "$RFI_ATT_ID" ] || [ "$RFI_ATT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL${NC}"
  echo "Response: $RFI_UPLOAD"
  exit 1
fi

echo "Test PDF" > /tmp/specification.pdf
curl -s -X PUT "$RFI_UPLOAD_URL" -H "Content-Type: application/pdf" --data-binary @/tmp/specification.pdf -o /dev/null -w "%{http_code}\n" | grep -q "200" && echo -e "${GREEN}✓ File uploaded to S3${NC}" || echo -e "${YELLOW}⚠ S3 upload may have failed${NC}"

curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\":$RFI_ATT_ID}" | jq -r '.status' | grep -q "confirmed" && echo -e "${GREEN}✓ Upload confirmed (Attachment ID: $RFI_ATT_ID)${NC}\n"

# Step 2.3: Get RFI and verify attachments
echo -e "${YELLOW}[2.3] Getting RFI by ID and verifying attachments...${NC}"
RFI_GET=$(curl -s -X GET "$API_BASE/rfis/$RFI_ID" -H "Authorization: Bearer $TOKEN")
RFI_ATT_COUNT=$(echo "$RFI_GET" | jq '.attachments // .Attachments | length // 0')

echo "$RFI_GET" | jq '{id, subject, status, attachments}' 2>/dev/null || echo "$RFI_GET" | jq '{ID, Subject, Status, Attachments}'

if [ "$RFI_ATT_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ RFI GET includes $RFI_ATT_COUNT attachment(s)${NC}\n"
else
  echo -e "${RED}✗ RFI GET does NOT include attachments${NC}\n"
fi

#==============================================================================
# TEST 3: SUBMITTAL WITH ATTACHMENT
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}TEST 3: Submittal with Attachment Workflow${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

# Step 3.1: Create Submittal (using Postman format)
echo -e "${YELLOW}[3.1] Creating submittal...${NC}"
SUBMITTAL_RESPONSE=$(curl -s -X POST "$API_BASE/submittals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"package_name\": \"Test Submittal Package\",
    \"csi_division\": \"05\",
    \"csi_section\": \"05 12 00\",
    \"title\": \"Test Submittal with Attachment - Steel Drawings\",
    \"description\": \"Shop drawings for steel beam connections - test submittal\",
    \"submittal_type\": \"shop_drawings\",
    \"specification_section\": \"051200\",
    \"priority\": \"high\",
    \"current_phase\": \"preparation\",
    \"ball_in_court\": \"contractor\",
    \"workflow_status\": \"pending_submission\",
    \"assigned_to\": $USER_ID,
    \"required_approval_date\": \"2025-12-31\"
  }")

SUBMITTAL_ID=$(echo "$SUBMITTAL_RESPONSE" | jq -r '.id // .ID // empty')

if [ -z "$SUBMITTAL_ID" ] || [ "$SUBMITTAL_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create submittal${NC}"
  echo "Response: $SUBMITTAL_RESPONSE"
  exit 1
fi

echo "$SUBMITTAL_RESPONSE" | jq '{id, submittal_number, title, status, priority}' 2>/dev/null || echo "$SUBMITTAL_RESPONSE" | jq '{ID, SubmittalNumber, Title, Status, Priority}'
echo -e "${GREEN}✓ Submittal created with ID: $SUBMITTAL_ID${NC}\n"

# Step 3.2: Upload attachment
echo -e "${YELLOW}[3.2] Uploading attachment for submittal...${NC}"
SUBMITTAL_UPLOAD=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"submittal\",\"entity_id\":$SUBMITTAL_ID,\"project_id\":$PROJECT_ID,\"location_id\":$LOCATION_ID,\"file_name\":\"shop_drawing.dwg\",\"file_size\":2097152,\"attachment_type\":\"shop_drawing\"}")

SUBMITTAL_ATT_ID=$(echo "$SUBMITTAL_UPLOAD" | jq -r '.attachment_id // .AttachmentID')
SUBMITTAL_UPLOAD_URL=$(echo "$SUBMITTAL_UPLOAD" | jq -r '.upload_url // .UploadURL')

if [ -z "$SUBMITTAL_ATT_ID" ] || [ "$SUBMITTAL_ATT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL${NC}"
  exit 1
fi

echo "Test DWG" > /tmp/shop_drawing.dwg
curl -s -X PUT "$SUBMITTAL_UPLOAD_URL" -H "Content-Type: application/octet-stream" --data-binary @/tmp/shop_drawing.dwg -o /dev/null -w "%{http_code}\n" | grep -q "200" && echo -e "${GREEN}✓ File uploaded to S3${NC}" || echo -e "${YELLOW}⚠ S3 upload may have failed${NC}"

curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\":$SUBMITTAL_ATT_ID}" | jq -r '.status' | grep -q "confirmed" && echo -e "${GREEN}✓ Upload confirmed (Attachment ID: $SUBMITTAL_ATT_ID)${NC}\n"

# Step 3.3: Get Submittal and verify attachments
echo -e "${YELLOW}[3.3] Getting submittal by ID and verifying attachments...${NC}"
SUBMITTAL_GET=$(curl -s -X GET "$API_BASE/submittals/$SUBMITTAL_ID" -H "Authorization: Bearer $TOKEN")
SUBMITTAL_ATT_COUNT=$(echo "$SUBMITTAL_GET" | jq '.attachments // .Attachments | length // 0')

echo "$SUBMITTAL_GET" | jq '{id, title, status, attachments}' 2>/dev/null || echo "$SUBMITTAL_GET" | jq '{ID, Title, Status, Attachments}'

if [ "$SUBMITTAL_ATT_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ Submittal GET includes $SUBMITTAL_ATT_COUNT attachment(s)${NC}\n"
else
  echo -e "${RED}✗ Submittal GET does NOT include attachments${NC}\n"
fi

#==============================================================================
# SUMMARY
#==============================================================================
echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}                           TEST SUMMARY                               ${NC}"
echo -e "${BLUE}======================================================================${NC}\n"

echo -e "${GREEN}Issue:${NC} ID=$ISSUE_ID, Attachment ID=$ISSUE_ATT_ID, Attachments in GET=$ISSUE_ATT_COUNT"
echo -e "${GREEN}RFI:${NC} ID=$RFI_ID, Attachment ID=$RFI_ATT_ID, Attachments in GET=$RFI_ATT_COUNT"
echo -e "${GREEN}Submittal:${NC} ID=$SUBMITTAL_ID, Attachment ID=$SUBMITTAL_ATT_ID, Attachments in GET=$SUBMITTAL_ATT_COUNT\n"

# Cleanup
rm -f /tmp/crack_photo.jpg /tmp/specification.pdf /tmp/shop_drawing.dwg

# Final result
PASSED=0
[ "$ISSUE_ATT_COUNT" -gt 0 ] && ((PASSED++))
[ "$RFI_ATT_COUNT" -gt 0 ] && ((PASSED++))
[ "$SUBMITTAL_ATT_COUNT" -gt 0 ] && ((PASSED++))

if [ "$PASSED" -eq 3 ]; then
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${GREEN}                  ALL TESTS PASSED (3/3) ✓                  ${NC}"
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  exit 0
else
  echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${RED}              SOME TESTS FAILED ($PASSED/3)              ${NC}"
  echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  exit 1
fi