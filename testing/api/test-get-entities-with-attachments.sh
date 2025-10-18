#!/bin/bash

# Test if GET by ID returns attachments for Issue, RFI, and Submittal

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}==================================================================${NC}"
echo -e "${BLUE}Testing GET by ID - Attachments in Response${NC}"
echo -e "${BLUE}==================================================================${NC}"

# Configuration
API_BASE_URL="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"
USERNAME="buildboard007+555@gmail.com"
PASSWORD="Mayur@1234"

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
  exit 1
fi
echo -e "${GREEN}✓ Token obtained${NC}"

# Test Issue with attachment (we created issue 75 earlier with attachment)
echo -e "\n${BLUE}==================================================================${NC}"
echo -e "${BLUE}Testing Issue GET by ID${NC}"
echo -e "${BLUE}==================================================================${NC}"

ISSUE_ID=75
echo -e "\n${YELLOW}[2] Getting Issue ${ISSUE_ID}...${NC}"
ISSUE_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/issues/${ISSUE_ID}" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$ISSUE_RESPONSE" | jq .

HAS_ATTACHMENTS=$(echo "$ISSUE_RESPONSE" | jq 'has("attachments")')
ATTACHMENT_COUNT=$(echo "$ISSUE_RESPONSE" | jq '.attachments | length // 0')

if [ "$HAS_ATTACHMENTS" == "true" ] && [ "$ATTACHMENT_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ Issue returns attachments (count: ${ATTACHMENT_COUNT})${NC}"
else
  echo -e "${RED}✗ Issue does NOT return attachments${NC}"
fi

# Test RFI with attachment (we created RFI 84 earlier with attachment)
echo -e "\n${BLUE}==================================================================${NC}"
echo -e "${BLUE}Testing RFI GET by ID${NC}"
echo -e "${BLUE}==================================================================${NC}"

RFI_ID=84
echo -e "\n${YELLOW}[3] Getting RFI ${RFI_ID}...${NC}"
RFI_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/rfis/${RFI_ID}" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$RFI_RESPONSE" | jq .

HAS_ATTACHMENTS=$(echo "$RFI_RESPONSE" | jq 'has("attachments")')
ATTACHMENT_COUNT=$(echo "$RFI_RESPONSE" | jq '.attachments | length // 0')

if [ "$HAS_ATTACHMENTS" == "true" ] && [ "$ATTACHMENT_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ RFI returns attachments (count: ${ATTACHMENT_COUNT})${NC}"
else
  echo -e "${RED}✗ RFI does NOT return attachments${NC}"
fi

# Test Submittal - need to find one with attachment
echo -e "\n${BLUE}==================================================================${NC}"
echo -e "${BLUE}Testing Submittal GET by ID${NC}"
echo -e "${BLUE}==================================================================${NC}"

# First, let's query to find a submittal with attachments
echo -e "\n${YELLOW}[4] Finding submittal with attachments...${NC}"
SUBMITTAL_WITH_ATTACHMENT=$(curl -s "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "query": "SELECT s.id FROM project.submittals s JOIN project.submittal_attachments sa ON s.id = sa.submittal_id WHERE s.is_deleted = false AND sa.is_deleted = false LIMIT 1"
  }' 2>/dev/null || echo "{}")

# For now, let's test with submittal ID 1 (if it exists)
SUBMITTAL_ID=1
echo -e "\n${YELLOW}[5] Getting Submittal ${SUBMITTAL_ID}...${NC}"
SUBMITTAL_RESPONSE=$(curl -s -X GET \
  "${API_BASE_URL}/submittals/${SUBMITTAL_ID}" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$SUBMITTAL_RESPONSE" | jq .

HAS_ATTACHMENTS=$(echo "$SUBMITTAL_RESPONSE" | jq 'has("attachments")')
ATTACHMENT_COUNT=$(echo "$SUBMITTAL_RESPONSE" | jq '.attachments | length // 0')

if [ "$HAS_ATTACHMENTS" == "true" ]; then
  if [ "$ATTACHMENT_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Submittal returns attachments (count: ${ATTACHMENT_COUNT})${NC}"
  else
    echo -e "${YELLOW}⚠ Submittal has 'attachments' field but it's empty${NC}"
  fi
else
  echo -e "${RED}✗ Submittal does NOT return attachments field${NC}"
fi

echo -e "\n${BLUE}==================================================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}==================================================================${NC}"

echo -e "\nNote: If attachments are missing, you need to rebuild and deploy:"
echo -e "${YELLOW}npm run build${NC}"
echo -e "${YELLOW}npx cdk deploy \"Infrastructure/Dev/Infrastructure-AppStage\" --profile dev${NC}"