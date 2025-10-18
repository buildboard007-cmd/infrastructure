#!/bin/bash

# Test Download Attachment Workflow
# This script tests the complete download flow

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}==================================================================${NC}"
echo -e "${YELLOW}Testing Attachment Download Workflow${NC}"
echo -e "${YELLOW}==================================================================${NC}"

# Configuration
API_BASE_URL="https://z1pbmjzrb6.execute-api.us-east-2.amazonaws.com/prod"
USER_POOL_ID="us-east-2_8JFjvA7xM"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"
USERNAME="buildboard007+555@gmail.com"
PASSWORD="Mayur@1234"

# Get JWT token
echo -e "\n${YELLOW}[1] Getting authentication token...${NC}"
TOKEN=$(aws cognito-idp admin-initiate-auth \
  --user-pool-id ${USER_POOL_ID} \
  --client-id ${CLIENT_ID} \
  --auth-flow ADMIN_NO_SRP_AUTH \
  --auth-parameters USERNAME=${USERNAME},PASSWORD=${PASSWORD} \
  --profile dev \
  --region us-east-2 \
  --query 'AuthenticationResult.IdToken' \
  --output text)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}✗ Failed to get token${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Token obtained${NC}"

# Test with a known attachment ID (replace with actual ID from your system)
ATTACHMENT_ID=6
ENTITY_TYPE="project"

echo -e "\n${YELLOW}[2] Getting download URL for attachment ${ATTACHMENT_ID}...${NC}"
DOWNLOAD_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/attachments/${ATTACHMENT_ID}/download-url?entity_type=${ENTITY_TYPE}" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json")

echo "Response:"
echo "$DOWNLOAD_RESPONSE" | jq .

# Extract download URL
DOWNLOAD_URL=$(echo "$DOWNLOAD_RESPONSE" | jq -r '.download_url // empty')
FILE_NAME=$(echo "$DOWNLOAD_RESPONSE" | jq -r '.file_name // empty')
FILE_SIZE=$(echo "$DOWNLOAD_RESPONSE" | jq -r '.file_size // empty')

if [ -z "$DOWNLOAD_URL" ] || [ "$DOWNLOAD_URL" == "null" ]; then
  echo -e "${RED}✗ Failed to get download URL${NC}"
  echo "Full response: $DOWNLOAD_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ Download URL generated${NC}"
echo "File: $FILE_NAME ($FILE_SIZE bytes)"

# Download the file
echo -e "\n${YELLOW}[3] Downloading file from S3...${NC}"
OUTPUT_FILE="/tmp/${FILE_NAME}"

# Use wget or curl to download
HTTP_CODE=$(curl -s -w "%{http_code}" -o "$OUTPUT_FILE" "$DOWNLOAD_URL")

if [ "$HTTP_CODE" == "200" ]; then
  ACTUAL_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null)
  echo -e "${GREEN}✓ File downloaded successfully${NC}"
  echo "Saved to: $OUTPUT_FILE"
  echo "Size: $ACTUAL_SIZE bytes"

  # Verify file size matches
  if [ "$ACTUAL_SIZE" == "$FILE_SIZE" ]; then
    echo -e "${GREEN}✓ File size verified${NC}"
  else
    echo -e "${YELLOW}⚠ File size mismatch (expected: $FILE_SIZE, got: $ACTUAL_SIZE)${NC}"
  fi

  # Show file type
  FILE_TYPE=$(file -b "$OUTPUT_FILE")
  echo "File type: $FILE_TYPE"

else
  echo -e "${RED}✗ Download failed with HTTP code: $HTTP_CODE${NC}"
  echo "Response:"
  cat "$OUTPUT_FILE"
  exit 1
fi

echo -e "\n${GREEN}==================================================================${NC}"
echo -e "${GREEN}Download Test Completed Successfully!${NC}"
echo -e "${GREEN}==================================================================${NC}"
