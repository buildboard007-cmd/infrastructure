#!/bin/bash

# Test script for Project User Management (Assignment) functionality

set -e

API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
PROJECT_ID=29  # Maria Resort 1234 (org_id=10)

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
echo "=== Step 1: Verify Project Exists ==="
PROJECT_RESPONSE=$(curl -s -X GET "$API_BASE/projects/$PROJECT_ID" \
  -H "Authorization: Bearer $TOKEN")

echo "$PROJECT_RESPONSE" | jq .
PROJECT_NAME=$(echo "$PROJECT_RESPONSE" | jq -r '.name // empty')

if [ -z "$PROJECT_NAME" ]; then
    echo "❌ Project $PROJECT_ID not found"
    exit 1
fi
echo "✅ Project found: $PROJECT_NAME"

echo ""
echo "=== Step 2: Assign User to Project (POST /projects/29/users) ==="
ASSIGNMENT=$(curl -s -X POST "$API_BASE/projects/$PROJECT_ID/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 20,
    "role_id": 28,
    "is_primary": true
  }')

echo "$ASSIGNMENT" | jq .
ASSIGNMENT_ID=$(echo "$ASSIGNMENT" | jq -r '.id // empty')

if [ -z "$ASSIGNMENT_ID" ] || [ "$ASSIGNMENT_ID" = "null" ]; then
    echo "❌ Failed to create assignment"
    exit 1
fi
echo "✅ Assignment created with ID: $ASSIGNMENT_ID"

echo ""
echo "=== Step 3: Get Project Users (GET /projects/29/users) ==="
PROJECT_USERS=$(curl -s -X GET "$API_BASE/projects/$PROJECT_ID/users" \
  -H "Authorization: Bearer $TOKEN")

echo "$PROJECT_USERS" | jq .
USER_COUNT=$(echo "$PROJECT_USERS" | jq 'length')
echo "✅ Retrieved $USER_COUNT users assigned to project"

# Verify our assignment is in the list
FOUND_ASSIGNMENT=$(echo "$PROJECT_USERS" | jq --arg id "$ASSIGNMENT_ID" '.[] | select(.id == ($id | tonumber))')
if [ -z "$FOUND_ASSIGNMENT" ]; then
    echo "❌ Could not find our assignment in project users list"
    exit 1
fi
echo "✅ Verified assignment $ASSIGNMENT_ID is in project users list"

echo ""
echo "=== Step 4: Update User Role (PUT /projects/29/users/$ASSIGNMENT_ID) ==="
UPDATED_ASSIGNMENT=$(curl -s -X PUT "$API_BASE/projects/$PROJECT_ID/users/$ASSIGNMENT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": 29,
    "is_primary": false
  }')

echo "$UPDATED_ASSIGNMENT" | jq .
UPDATED_ROLE=$(echo "$UPDATED_ASSIGNMENT" | jq -r '.role_id // empty')

if [ "$UPDATED_ROLE" != "29" ]; then
    echo "❌ Role was not updated correctly. Expected: 29, Got: $UPDATED_ROLE"
    exit 1
fi
echo "✅ Assignment updated successfully (role_id: 28 → 29)"

echo ""
echo "=== Step 5: Remove User from Project (DELETE /projects/29/users/$ASSIGNMENT_ID) ==="
DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE/projects/$PROJECT_ID/users/$ASSIGNMENT_ID" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n 1)
RESPONSE_BODY=$(echo "$DELETE_RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
if [ "$RESPONSE_BODY" != "" ]; then
    echo "Response Body: $RESPONSE_BODY"
fi

if [ "$HTTP_CODE" != "204" ]; then
    echo "❌ Expected HTTP 204, got $HTTP_CODE"
    exit 1
fi
echo "✅ User removed from project successfully"

echo ""
echo "=== Step 6: Verify User is Removed (GET /projects/29/users) ==="
USERS_AFTER_DELETE=$(curl -s -X GET "$API_BASE/projects/$PROJECT_ID/users" \
  -H "Authorization: Bearer $TOKEN")

echo "$USERS_AFTER_DELETE" | jq .

# Verify assignment is gone (soft deleted)
FOUND_AFTER_DELETE=$(echo "$USERS_AFTER_DELETE" | jq --arg id "$ASSIGNMENT_ID" '.[] | select(.id == ($id | tonumber))')
if [ -n "$FOUND_AFTER_DELETE" ]; then
    echo "❌ Assignment $ASSIGNMENT_ID still appears in project users (should be soft-deleted)"
    exit 1
fi
echo "✅ Verified assignment is no longer in project users list (soft-deleted)"

echo ""
echo "=== ✅ All Tests Completed Successfully! ==="
echo ""
echo "Summary:"
echo "  - ✅ POST /projects/{projectId}/users - Assign user to project"
echo "  - ✅ GET /projects/{projectId}/users - Get project team members"
echo "  - ✅ PUT /projects/{projectId}/users/{assignmentId} - Update user role"
echo "  - ✅ DELETE /projects/{projectId}/users/{assignmentId} - Remove user from project"
echo ""
echo "All endpoints working correctly with user_assignments table!"