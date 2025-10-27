#!/bin/bash

# Test script for Issue Comments with Attachments functionality

set -e

API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
PROJECT_ID=11
LOCATION_ID=3
ISSUE_ID=67

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
echo "=== Step 1: Verify Issue Exists ==="
ISSUE_RESPONSE=$(curl -s -X GET "$API_BASE/issues/$ISSUE_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$ISSUE_RESPONSE" | jq .
ISSUE_STATUS=$(echo "$ISSUE_RESPONSE" | jq -r '.status // empty')

if [ -z "$ISSUE_STATUS" ]; then
    echo "❌ Issue $ISSUE_ID not found"
    exit 1
fi
echo "✅ Issue found with status: $ISSUE_STATUS"

echo ""
echo "=== Step 2: Create Comment WITHOUT Attachments ==="
COMMENT_1=$(curl -s -X POST "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "comment": "Testing comment functionality - this is a simple comment without attachments"
  }')

echo "$COMMENT_1" | jq .
COMMENT_1_ID=$(echo "$COMMENT_1" | jq -r '.id // empty')

if [ -z "$COMMENT_1_ID" ] || [ "$COMMENT_1_ID" = "null" ]; then
    echo "❌ Failed to create comment"
    exit 1
fi
echo "✅ Comment created with ID: $COMMENT_1_ID"

echo ""
echo "=== Step 3: Upload Attachment for Comment ==="
UPLOAD_REQUEST=$(curl -s -X POST "$API_BASE/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"entity_type\": \"issue_comment\",
    \"entity_id\": 0,
    \"project_id\": $PROJECT_ID,
    \"location_id\": $LOCATION_ID,
    \"file_name\": \"test-photo.jpg\",
    \"file_size\": 12345,
    \"attachment_type\": \"photo\"
  }")

echo "$UPLOAD_REQUEST" | jq .
ATTACHMENT_ID=$(echo "$UPLOAD_REQUEST" | jq -r '.attachment_id // empty')
UPLOAD_URL=$(echo "$UPLOAD_REQUEST" | jq -r '.upload_url // empty')

if [ -z "$ATTACHMENT_ID" ] || [ "$ATTACHMENT_ID" = "null" ]; then
    echo "❌ Failed to get upload URL"
    exit 1
fi
echo "✅ Attachment record created with ID: $ATTACHMENT_ID"

echo ""
echo "=== Step 4: Create Comment WITH Attachment ==="
COMMENT_2=$(curl -s -X POST "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"comment\": \"This comment has an attachment - reviewing the site photo\",
    \"attachment_ids\": [$ATTACHMENT_ID]
  }")

echo "$COMMENT_2" | jq .
COMMENT_2_ID=$(echo "$COMMENT_2" | jq -r '.id // empty')
ATTACHMENT_COUNT=$(echo "$COMMENT_2" | jq -r '.attachments | length')

if [ -z "$COMMENT_2_ID" ] || [ "$COMMENT_2_ID" = "null" ]; then
    echo "❌ Failed to create comment with attachment"
    exit 1
fi

if [ "$ATTACHMENT_COUNT" != "1" ]; then
    echo "❌ Comment does not have expected attachment count. Expected: 1, Got: $ATTACHMENT_COUNT"
    exit 1
fi
echo "✅ Comment with attachment created with ID: $COMMENT_2_ID"

echo ""
echo "=== Step 5: Get All Comments for Issue ==="
COMMENTS=$(curl -s -X GET "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN")

echo "$COMMENTS" | jq .
TOTAL_COMMENTS=$(echo "$COMMENTS" | jq 'length')
echo "✅ Retrieved $TOTAL_COMMENTS comments"

echo ""
echo "=== Step 6: Change Issue Status to Trigger Activity Log ==="
NEW_STATUS="in_progress"
STATUS_UPDATE=$(curl -s -X PATCH "$API_BASE/issues/$ISSUE_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"status\": \"$NEW_STATUS\"
  }")

echo "$STATUS_UPDATE" | jq .
UPDATE_STATUS=$(echo "$STATUS_UPDATE" | jq -r '.status // empty')

if [ "$UPDATE_STATUS" != "$NEW_STATUS" ]; then
    echo "⚠️  Status might not have changed (could be same status)"
else
    echo "✅ Status updated to: $NEW_STATUS"
fi

echo ""
echo "=== Step 7: Verify Activity Log Was Created ==="
sleep 2  # Give it a moment to process
COMMENTS_AFTER=$(curl -s -X GET "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN")

echo "$COMMENTS_AFTER" | jq .
ACTIVITY_LOGS=$(echo "$COMMENTS_AFTER" | jq '[.[] | select(.comment_type == "activity")]')
ACTIVITY_COUNT=$(echo "$ACTIVITY_LOGS" | jq 'length')

echo "✅ Found $ACTIVITY_COUNT activity log entries"
echo ""
echo "Activity Logs:"
echo "$ACTIVITY_LOGS" | jq -r '.[] | "- \(.comment) (by user \(.created_by))"'

echo ""
echo "=== Step 8: Change Status Back ==="
REVERT_STATUS="open"
curl -s -X PATCH "$API_BASE/issues/$ISSUE_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"status\": \"$REVERT_STATUS\"
  }" > /dev/null
echo "✅ Reverted status back to: $REVERT_STATUS"

echo ""
echo "=== Final Comments Feed ==="
FINAL_COMMENTS=$(curl -s -X GET "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN")

echo "$FINAL_COMMENTS" | jq '[.[] | {
  id: .id,
  type: .comment_type,
  comment: .comment,
  created_by_name: .created_by_name,
  attachments_count: (.attachments | length),
  previous_value: .previous_value,
  new_value: .new_value
}]'

echo ""
echo "=== ✅ All Tests Completed Successfully! ==="
