#!/bin/bash

TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' | jq -r '.AuthenticationResult.IdToken')

echo "=== Checking Submittal 3 for attachments field ==="
RESPONSE=$(curl -s -X GET "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/submittals/3" -H "Authorization: Bearer $TOKEN")

echo "Has 'attachments' field:"
echo "$RESPONSE" | jq 'has("attachments")'

echo ""
echo "Has 'attachment_count' field:"
echo "$RESPONSE" | jq 'has("attachment_count")'

echo ""
echo "Attachment count value:"
echo "$RESPONSE" | jq '.attachment_count // "field not present"'