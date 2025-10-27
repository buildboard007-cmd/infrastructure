#!/bin/bash
set -e

API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"

echo "=== Getting Authentication Token ==="
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' | jq -r '.AuthenticationResult.IdToken')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "❌ Failed to get authentication token"
    exit 1
fi
echo "✅ Token obtained"

echo ""
echo "=== Testing Issue 90 - Recent Comments ==="
curl -s -X GET "$API_BASE/issues/90" \
  -H "Authorization: Bearer $TOKEN" | jq '.comments[0:3] | .[] | {id, comment, attachments}'

echo ""
echo "=== Testing Issue 67 - Comment with Attachment ==="
curl -s -X GET "$API_BASE/issues/67" \
  -H "Authorization: Bearer $TOKEN" | jq '.comments[0:2] | .[] | {id, comment, attachments}'