#!/bin/bash

TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"3f0fb5mpivctnvj85tucusf88e","AuthParameters":{"USERNAME":"buildboard007+555@gmail.com","PASSWORD":"Mayur@1234"}}' | jq -r '.AuthenticationResult.IdToken')

echo "Testing comment attachment upload..."
curl -s -X POST "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entity_type":"issue_comment","entity_id":0,"project_id":11,"location_id":3,"file_name":"test-photo.jpg","file_size":12345,"attachment_type":"photo"}' | jq .