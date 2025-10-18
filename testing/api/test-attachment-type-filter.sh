#!/bin/bash

# Test Attachment Type Filter
# Tests the attachment_type query parameter for filtering attachments

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}==================================================================${NC}"
echo -e "${BLUE}Testing Attachment Type Filter${NC}"
echo -e "${BLUE}==================================================================${NC}"

# Configuration
API_BASE_URL="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
USER_POOL_ID="us-east-2_8JFjvA7xM"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"
USERNAME="buildboard007+555@gmail.com"
PASSWORD="Mayur@1234"

# Test data
PROJECT_ID=49
LOCATION_ID=24

# Get JWT token
echo -e "\n${YELLOW}[1] Getting authentication token...${NC}"
AUTH_RESPONSE=$(curl -s -X POST \
  "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d "{
    \"AuthFlow\": \"USER_PASSWORD_AUTH\",
    \"ClientId\": \"${CLIENT_ID}\",
    \"AuthParameters\": {
      \"USERNAME\": \"${USERNAME}\",
      \"PASSWORD\": \"${PASSWORD}\"
    }
  }")

TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.AuthenticationResult.IdToken')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
  echo -e "${RED}✗ Failed to get token${NC}"
  echo "$AUTH_RESPONSE" | jq .
  exit 1
fi
echo -e "${GREEN}✓ Token obtained${NC}"

# Step 1: Upload a DOCUMENT attachment
echo -e "\n${YELLOW}[2] Creating DOCUMENT attachment...${NC}"
DOCUMENT_UPLOAD_RESPONSE=$(curl -s -X POST \
  "${API_BASE_URL}/attachments/upload-url" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"project\",
    \"entity_id\": ${PROJECT_ID},
    \"project_id\": ${PROJECT_ID},
    \"location_id\": ${LOCATION_ID},
    \"file_name\": \"project_specification.pdf\",
    \"file_size\": 1048576,
    \"attachment_type\": \"document\"
  }")

echo "$DOCUMENT_UPLOAD_RESPONSE" | jq .

DOCUMENT_ATTACHMENT_ID=$(echo "$DOCUMENT_UPLOAD_RESPONSE" | jq -r '.attachment_id')
DOCUMENT_UPLOAD_URL=$(echo "$DOCUMENT_UPLOAD_RESPONSE" | jq -r '.upload_url')

if [ -z "$DOCUMENT_ATTACHMENT_ID" ] || [ "$DOCUMENT_ATTACHMENT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create document attachment${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Document attachment created: ID ${DOCUMENT_ATTACHMENT_ID}${NC}"

# Upload file to S3
echo "Test PDF content" > /tmp/project_specification.pdf
curl -s -X PUT "$DOCUMENT_UPLOAD_URL" \
  -H "Content-Type: application/pdf" \
  --data-binary @/tmp/project_specification.pdf > /dev/null

# Confirm upload
curl -s -X POST "${API_BASE_URL}/attachments/confirm" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\": ${DOCUMENT_ATTACHMENT_ID}}" > /dev/null

echo -e "${GREEN}✓ Document uploaded and confirmed${NC}"

# Step 2: Upload a DRAWING attachment
echo -e "\n${YELLOW}[3] Creating DRAWING attachment...${NC}"
DRAWING_UPLOAD_RESPONSE=$(curl -s -X POST \
  "${API_BASE_URL}/attachments/upload-url" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"project\",
    \"entity_id\": ${PROJECT_ID},
    \"project_id\": ${PROJECT_ID},
    \"location_id\": ${LOCATION_ID},
    \"file_name\": \"architectural_plan.dwg\",
    \"file_size\": 2097152,
    \"attachment_type\": \"drawing\"
  }")

echo "$DRAWING_UPLOAD_RESPONSE" | jq .

DRAWING_ATTACHMENT_ID=$(echo "$DRAWING_UPLOAD_RESPONSE" | jq -r '.attachment_id')
DRAWING_UPLOAD_URL=$(echo "$DRAWING_UPLOAD_RESPONSE" | jq -r '.upload_url')

if [ -z "$DRAWING_ATTACHMENT_ID" ] || [ "$DRAWING_ATTACHMENT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to create drawing attachment${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Drawing attachment created: ID ${DRAWING_ATTACHMENT_ID}${NC}"

# Upload file to S3
echo "Test DWG content" > /tmp/architectural_plan.dwg
curl -s -X PUT "$DRAWING_UPLOAD_URL" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @/tmp/architectural_plan.dwg > /dev/null

# Confirm upload
curl -s -X POST "${API_BASE_URL}/attachments/confirm" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\": ${DRAWING_ATTACHMENT_ID}}" > /dev/null

echo -e "${GREEN}✓ Drawing uploaded and confirmed${NC}"

# Step 3: Test filters
echo -e "\n${BLUE}==================================================================${NC}"
echo -e "${BLUE}Testing Filters${NC}"
echo -e "${BLUE}==================================================================${NC}"

# Test 1: Get ALL attachments
echo -e "\n${YELLOW}[4] Get ALL project attachments (no filter)...${NC}"
ALL_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/entities/project/${PROJECT_ID}/attachments" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$ALL_RESPONSE" | jq .

ALL_COUNT=$(echo "$ALL_RESPONSE" | jq '.attachments | length')
echo -e "${GREEN}✓ Total attachments: ${ALL_COUNT}${NC}"

# Test 2: Get only DOCUMENTS
echo -e "\n${YELLOW}[5] Get only DOCUMENT attachments...${NC}"
DOCUMENT_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/entities/project/${PROJECT_ID}/attachments?attachment_type=document" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$DOCUMENT_RESPONSE" | jq .

DOCUMENT_COUNT=$(echo "$DOCUMENT_RESPONSE" | jq '.attachments | length')
echo -e "${GREEN}✓ Document attachments: ${DOCUMENT_COUNT}${NC}"

# Verify only documents are returned
ONLY_DOCUMENTS=$(echo "$DOCUMENT_RESPONSE" | jq '[.attachments[] | select(.attachment_type != "document")] | length')
if [ "$ONLY_DOCUMENTS" -eq 0 ]; then
  echo -e "${GREEN}✓ Filter working correctly - only documents returned${NC}"
else
  echo -e "${RED}✗ Filter not working - non-documents found${NC}"
fi

# Test 3: Get only DRAWINGS
echo -e "\n${YELLOW}[6] Get only DRAWING attachments...${NC}"
DRAWING_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/entities/project/${PROJECT_ID}/attachments?attachment_type=drawing" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$DRAWING_RESPONSE" | jq .

DRAWING_COUNT=$(echo "$DRAWING_RESPONSE" | jq '.attachments | length')
echo -e "${GREEN}✓ Drawing attachments: ${DRAWING_COUNT}${NC}"

# Verify only drawings are returned
ONLY_DRAWINGS=$(echo "$DRAWING_RESPONSE" | jq '[.attachments[] | select(.attachment_type != "drawing")] | length')
if [ "$ONLY_DRAWINGS" -eq 0 ]; then
  echo -e "${GREEN}✓ Filter working correctly - only drawings returned${NC}"
else
  echo -e "${RED}✗ Filter not working - non-drawings found${NC}"
fi

# Summary
echo -e "\n${BLUE}==================================================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}==================================================================${NC}"
echo -e "Total attachments: ${ALL_COUNT}"
echo -e "Document attachments: ${DOCUMENT_COUNT}"
echo -e "Drawing attachments: ${DRAWING_COUNT}"

# Cleanup
rm -f /tmp/project_specification.pdf /tmp/architectural_plan.dwg

echo -e "\n${GREEN}==================================================================${NC}"
echo -e "${GREEN}Attachment Type Filter Test Completed!${NC}"
echo -e "${GREEN}==================================================================${NC}"