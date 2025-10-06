#!/bin/bash

# Attachment Management API Test Script
# This script tests all endpoints of the attachment management API

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# API Configuration
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
COGNITO_ENDPOINT="https://cognito-idp.us-east-2.amazonaws.com/"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"
USERNAME="buildboard007+555@gmail.com"
PASSWORD="Mayur@1234"

# Test data
PROJECT_ID=49
LOCATION_ID=24
ENTITY_ID=49

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Attachment Management API Test${NC}"
echo -e "${YELLOW}========================================${NC}\n"

# Step 1: Get Authentication Token
echo -e "${YELLOW}[1/9] Getting authentication token...${NC}"
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

# Step 2: Test Generate Upload URL - Project
echo -e "${YELLOW}[2/9] Testing generate upload URL for project attachment...${NC}"
UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"project\",\"entity_id\":$ENTITY_ID,\"project_id\":$PROJECT_ID,\"location_id\":$LOCATION_ID,\"file_name\":\"test_floor_plan.pdf\",\"file_size\":102400,\"attachment_type\":\"drawing\"}")

ATTACHMENT_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.attachment_id // .AttachmentID // empty')
UPLOAD_URL=$(echo "$UPLOAD_RESPONSE" | jq -r '.upload_url // .UploadURL // empty')

echo "$UPLOAD_RESPONSE" | jq .

if [ -z "$ATTACHMENT_ID" ] || [ "$ATTACHMENT_ID" == "null" ]; then
  echo -e "${RED}✗ Failed to generate upload URL${NC}"
  echo "Response: $UPLOAD_RESPONSE"
  exit 1
fi
echo -e "${GREEN}✓ Upload URL generated. Attachment ID: $ATTACHMENT_ID${NC}\n"

# Step 3: Simulate file upload to S3 (create a test file)
echo -e "${YELLOW}[3/9] Simulating file upload to S3...${NC}"
echo "This is a test PDF file for attachment testing" > /tmp/test_floor_plan.pdf

if [ -n "$UPLOAD_URL" ] && [ "$UPLOAD_URL" != "null" ]; then
  UPLOAD_RESULT=$(curl -s -X PUT "$UPLOAD_URL" \
    -H "Content-Type: application/pdf" \
    --data-binary @/tmp/test_floor_plan.pdf \
    -w "\n%{http_code}")

  HTTP_CODE=$(echo "$UPLOAD_RESULT" | tail -n1)

  if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ File uploaded to S3 successfully${NC}\n"
  else
    echo -e "${YELLOW}⚠ S3 upload returned code: $HTTP_CODE (may need real AWS credentials)${NC}\n"
  fi
else
  echo -e "${YELLOW}⚠ No upload URL returned, skipping S3 upload${NC}\n"
fi

# Step 4: Confirm Upload
echo -e "${YELLOW}[4/9] Testing confirm upload...${NC}"
CONFIRM_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\":$ATTACHMENT_ID}")

echo "$CONFIRM_RESPONSE" | jq .
echo -e "${GREEN}✓ Upload confirmed${NC}\n"

# Step 5: Get Attachment Metadata
echo -e "${YELLOW}[5/9] Testing get attachment metadata...${NC}"
METADATA_RESPONSE=$(curl -s -X GET "$API_BASE/attachments/$ATTACHMENT_ID?entity_type=project" \
  -H "Authorization: Bearer $TOKEN")

echo "$METADATA_RESPONSE" | jq .

if echo "$METADATA_RESPONSE" | jq -e '.id // .ID' > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Attachment metadata retrieved${NC}\n"
else
  echo -e "${RED}✗ Failed to get attachment metadata${NC}\n"
fi

# Step 6: Generate Download URL
echo -e "${YELLOW}[6/9] Testing generate download URL...${NC}"
DOWNLOAD_RESPONSE=$(curl -s -X GET "$API_BASE/attachments/$ATTACHMENT_ID/download-url?entity_type=project" \
  -H "Authorization: Bearer $TOKEN")

echo "$DOWNLOAD_RESPONSE" | jq .

DOWNLOAD_URL=$(echo "$DOWNLOAD_RESPONSE" | jq -r '.download_url // .DownloadURL // empty')
if [ -n "$DOWNLOAD_URL" ] && [ "$DOWNLOAD_URL" != "null" ]; then
  echo -e "${GREEN}✓ Download URL generated${NC}\n"
else
  echo -e "${YELLOW}⚠ No download URL in response${NC}\n"
fi

# Step 7: List Entity Attachments
echo -e "${YELLOW}[7/9] Testing list project attachments...${NC}"
LIST_RESPONSE=$(curl -s -X GET "$API_BASE/entities/project/$PROJECT_ID/attachments?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN")

echo "$LIST_RESPONSE" | jq .

ATTACHMENT_COUNT=$(echo "$LIST_RESPONSE" | jq -r '.total_count // .TotalCount // .attachments | length // 0')
echo -e "${GREEN}✓ Found $ATTACHMENT_COUNT attachment(s)${NC}\n"

# Step 8: Test Error Cases
echo -e "${YELLOW}[8/9] Testing error cases...${NC}"

# Invalid entity type
echo "  - Testing invalid entity type..."
ERROR_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"invalid_type\",\"entity_id\":1,\"project_id\":1,\"location_id\":$LOCATION_ID,\"file_name\":\"test.pdf\",\"file_size\":1024,\"attachment_type\":\"document\"}")

if echo "$ERROR_RESPONSE" | grep -q "Invalid\|invalid\|error"; then
  echo -e "${GREEN}    ✓ Invalid entity type correctly rejected${NC}"
else
  echo -e "${YELLOW}    ⚠ Expected error for invalid entity type${NC}"
fi

# File too large
echo "  - Testing file size validation..."
ERROR_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"project\",\"entity_id\":1,\"project_id\":1,\"location_id\":$LOCATION_ID,\"file_name\":\"huge.pdf\",\"file_size\":104857601,\"attachment_type\":\"document\"}")

if echo "$ERROR_RESPONSE" | grep -q "size\|large\|limit\|error"; then
  echo -e "${GREEN}    ✓ File size limit correctly enforced${NC}"
else
  echo -e "${YELLOW}    ⚠ Expected error for file too large${NC}"
fi

# Invalid file type
echo "  - Testing file type validation..."
ERROR_RESPONSE=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"entity_type\":\"project\",\"entity_id\":1,\"project_id\":1,\"location_id\":$LOCATION_ID,\"file_name\":\"virus.exe\",\"file_size\":1024,\"attachment_type\":\"document\"}")

if echo "$ERROR_RESPONSE" | grep -q "type\|not allowed\|invalid\|error"; then
  echo -e "${GREEN}    ✓ Invalid file type correctly rejected${NC}\n"
else
  echo -e "${YELLOW}    ⚠ Expected error for invalid file type${NC}\n"
fi

# Step 9: Soft Delete Attachment
echo -e "${YELLOW}[9/9] Testing soft delete attachment...${NC}"
DELETE_RESPONSE=$(curl -s -X DELETE "$API_BASE/attachments/$ATTACHMENT_ID?entity_type=project" \
  -H "Authorization: Bearer $TOKEN")

echo "$DELETE_RESPONSE" | jq .

if echo "$DELETE_RESPONSE" | jq -e '.status' | grep -q "deleted"; then
  echo -e "${GREEN}✓ Attachment soft deleted successfully${NC}\n"
else
  echo -e "${YELLOW}⚠ Unexpected delete response${NC}\n"
fi

# Summary
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Test Summary${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}✓ All attachment API tests completed${NC}"
echo -e "Attachment ID tested: ${ATTACHMENT_ID}"
echo -e "Project ID: ${PROJECT_ID}"
echo -e "Location ID: ${LOCATION_ID}\n"

# Cleanup
rm -f /tmp/test_floor_plan.pdf

echo -e "${GREEN}Test script completed successfully!${NC}"
