# Testing Guide

## Table of Contents

- [Overview](#overview)
- [Test Environment Setup](#test-environment-setup)
- [Getting Authentication Tokens](#getting-authentication-tokens)
- [Using Postman Collections](#using-postman-collections)
- [Test Script Locations and Usage](#test-script-locations-and-usage)
- [Testing with MCP Database Queries](#testing-with-mcp-database-queries)
- [Test User Credentials](#test-user-credentials)
- [Testing Access Control](#testing-access-control)
- [Testing Attachments](#testing-attachments)
- [Creating Test Data](#creating-test-data)
- [Debugging API Errors](#debugging-api-errors)
- [Common Testing Scenarios](#common-testing-scenarios)
- [Testing Checklist](#testing-checklist)

---

## Overview

This guide covers all testing approaches for the BuildBoard infrastructure system, including:

- API endpoint testing with curl and Postman
- Access control verification
- File attachment upload/download testing
- Database query validation with MCP
- Creating and managing test data
- Debugging common API errors

### Testing Philosophy

1. **Test in Dev first** - Always test in development environment before production
2. **Use real data patterns** - Test with realistic data structures
3. **Cover access levels** - Test as super admin, org admin, location user, project user
4. **Verify error cases** - Test validation, authorization, and error handling
5. **Clean up test data** - Remove test data after testing (soft delete)

---

## Test Environment Setup

### API Base URLs

**Development:**
```bash
export API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
```

**Production:**
```bash
export API_BASE="https://api.buildboard.com"
```

### Required Tools

```bash
# curl - API testing
curl --version

# jq - JSON parsing
jq --version

# AWS CLI - CloudWatch logs
aws --version

# Postman - GUI testing (optional)
```

### Environment Variables

Set up testing environment:

```bash
# In ~/.bashrc or ~/.zshrc
export API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
export TEST_EMAIL="buildboard007+555@gmail.com"
export TEST_PASSWORD="Mayur@1234"
export COGNITO_CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"

# Function to get token quickly
get_token() {
    curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
      -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
      -H "Content-Type: application/x-amz-json-1.1" \
      -d "{
        \"AuthFlow\":\"USER_PASSWORD_AUTH\",
        \"ClientId\":\"$COGNITO_CLIENT_ID\",
        \"AuthParameters\":{
          \"USERNAME\":\"$TEST_EMAIL\",
          \"PASSWORD\":\"$TEST_PASSWORD\"
        }
      }" | jq -r '.AuthenticationResult.IdToken'
}

# Usage
export TOKEN=$(get_token)
```

---

## Getting Authentication Tokens

### Method 1: Using curl (Recommended)

**Get ID Token:**

```bash
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "AuthFlow":"USER_PASSWORD_AUTH",
    "ClientId":"3f0fb5mpivctnvj85tucusf88e",
    "AuthParameters":{
      "USERNAME":"buildboard007+555@gmail.com",
      "PASSWORD":"Mayur@1234"
    }
  }' | jq -r '.AuthenticationResult.IdToken')

# Verify token was retrieved
echo $TOKEN

# Use in API calls
curl -X GET "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

**Get Access Token (if needed):**

```bash
ACCESS_TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "AuthFlow":"USER_PASSWORD_AUTH",
    "ClientId":"3f0fb5mpivctnvj85tucusf88e",
    "AuthParameters":{
      "USERNAME":"buildboard007+555@gmail.com",
      "PASSWORD":"Mayur@1234"
    }
  }' | jq -r '.AuthenticationResult.AccessToken')
```

**Important:** API Gateway requires **ID tokens**, not access tokens.

### Method 2: Using test script

```bash
# Create token helper script
cat > /Users/mayur/git_personal/infrastructure/testing/utilities/get-token.sh << 'EOF'
#!/bin/bash

EMAIL="${1:-buildboard007+555@gmail.com}"
PASSWORD="${2:-Mayur@1234}"
CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"

TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d "{
    \"AuthFlow\":\"USER_PASSWORD_AUTH\",
    \"ClientId\":\"$CLIENT_ID\",
    \"AuthParameters\":{
      \"USERNAME\":\"$EMAIL\",
      \"PASSWORD\":\"$PASSWORD\"
    }
  }" | jq -r '.AuthenticationResult.IdToken')

echo "$TOKEN"
EOF

chmod +x /Users/mayur/git_personal/infrastructure/testing/utilities/get-token.sh

# Usage
TOKEN=$(./testing/utilities/get-token.sh)
```

### Method 3: Decode and inspect token

**Decode JWT token:**

```bash
# Decode token to see claims
echo $TOKEN | cut -d. -f2 | base64 -d | jq .

# You should see:
# {
#   "sub": "cognito-user-id",
#   "email": "buildboard007+555@gmail.com",
#   "user_id": 19,
#   "org_id": 10,
#   "isSuperAdmin": true,
#   "locations": [...],
#   ...
# }
```

---

## Using Postman Collections

### Setting Up Postman

**1. Import Collections:**

Import all collections from `/Users/mayur/git_personal/infrastructure/postman/`:

- `Infrastructure.postman_collection.json` - General endpoints
- `ProjectManagement.postman_collection.json` - Project APIs
- `IssueManagement.postman_collection.json` - Issue APIs
- `RFIManagement.postman_collection.json` - RFI APIs
- `SubmittalManagement.postman_collection.json` - Submittal APIs
- `AttachmentManagement.postman_collection.json` - Attachment APIs
- `AssignmentManagement.postman_collection.json` - Assignment APIs
- `RolesManagement.postman_collection.json` - Role APIs

**2. Create Environment:**

Create a new environment with these variables:

```json
{
  "name": "BuildBoard Dev",
  "values": [
    {
      "key": "base_url",
      "value": "https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main",
      "enabled": true
    },
    {
      "key": "access_token",
      "value": "",
      "enabled": true
    },
    {
      "key": "user_email",
      "value": "buildboard007+555@gmail.com",
      "enabled": true
    },
    {
      "key": "user_id",
      "value": "19",
      "enabled": true
    },
    {
      "key": "org_id",
      "value": "10",
      "enabled": true
    },
    {
      "key": "location_id",
      "value": "6",
      "enabled": true
    },
    {
      "key": "project_id",
      "value": "29",
      "enabled": true
    }
  ]
}
```

**3. Get Token in Postman:**

Use the Infrastructure collection's "Get ID Token" request, or:

1. Run curl command to get token
2. Copy token value
3. Paste into `access_token` environment variable

**4. Test Request:**

Select any request in a collection, ensure:
- Environment is selected (BuildBoard Dev)
- Authorization header uses `{{access_token}}`
- URL uses `{{base_url}}`

Click "Send" to test.

### Postman Tips

**Auto-update token:**

Add to collection Pre-request Script:

```javascript
// Check if token expired (optional)
const token = pm.environment.get("access_token");
if (!token || isTokenExpired(token)) {
    // Get new token
    pm.sendRequest({
        url: 'https://cognito-idp.us-east-2.amazonaws.com/',
        method: 'POST',
        header: {
            'X-Amz-Target': 'AWSCognitoIdentityProviderService.InitiateAuth',
            'Content-Type': 'application/x-amz-json-1.1'
        },
        body: {
            mode: 'raw',
            raw: JSON.stringify({
                AuthFlow: 'USER_PASSWORD_AUTH',
                ClientId: '3f0fb5mpivctnvj85tucusf88e',
                AuthParameters: {
                    USERNAME: 'buildboard007+555@gmail.com',
                    PASSWORD: 'Mayur@1234'
                }
            })
        }
    }, (err, res) => {
        if (!err) {
            pm.environment.set("access_token", res.json().AuthenticationResult.IdToken);
        }
    });
}
```

**Save response values:**

In Tests tab:

```javascript
// Save project ID from response
if (pm.response.code === 200) {
    const response = pm.response.json();
    if (response.project && response.project.id) {
        pm.environment.set("project_id", response.project.id);
    }
}
```

---

## Test Script Locations and Usage

### Test Script Directory Structure

```
/Users/mayur/git_personal/infrastructure/testing/
├── api/                          # API endpoint tests
│   ├── test-project-user-management.sh
│   ├── test-get-projects-access-control.sh
│   ├── test-issue-comments.sh
│   ├── test-comment-attachment.sh
│   ├── test-attachment-api.sh
│   ├── test-entity-with-attachments.sh
│   └── ...
├── auth/                         # Authentication tests
├── database/                     # Database validation
└── utilities/                    # Helper scripts
    └── get-token.sh
```

### Running Test Scripts

**1. Project Management Tests:**

```bash
cd /Users/mayur/git_personal/infrastructure/testing/api

# Test project user assignments
./test-project-user-management.sh

# Test project access control
./test-get-projects-access-control.sh
```

**2. Issue Management Tests:**

```bash
# Test issue creation and comments
./test-issue-comments.sh

# Test issue with attachments
./test-entity-with-attachments.sh
```

**3. Attachment Tests:**

```bash
# Test attachment upload/download
./test-attachment-api.sh

# Test comment attachments
./test-comment-attachment.sh

# Test attachment type filtering
./test-attachment-type-filter.sh
```

### Creating New Test Scripts

**Template for API test script:**

```bash
#!/bin/bash

# Set up environment
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"
COGNITO_CLIENT_ID="3f0fb5mpivctnvj85tucusf88e"

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Testing Your Feature ===${NC}\n"

# Get authentication token
echo -e "${BLUE}Getting authentication token...${NC}"
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "AuthFlow":"USER_PASSWORD_AUTH",
    "ClientId":"'$COGNITO_CLIENT_ID'",
    "AuthParameters":{
      "USERNAME":"buildboard007+555@gmail.com",
      "PASSWORD":"Mayur@1234"
    }
  }' | jq -r '.AuthenticationResult.IdToken')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo -e "${RED}Failed to get authentication token${NC}"
    exit 1
fi
echo -e "${GREEN}Token obtained successfully${NC}\n"

# Test 1: Your first test
echo -e "${BLUE}Test 1: Testing GET endpoint${NC}"
RESPONSE=$(curl -s -X GET "$API_BASE/your-endpoint" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "Response: $RESPONSE" | jq .

# Check response
if echo "$RESPONSE" | jq -e '.error' > /dev/null; then
    echo -e "${RED}FAILED: Error in response${NC}"
else
    echo -e "${GREEN}PASSED: GET endpoint successful${NC}"
fi
echo ""

# Test 2: Your second test
echo -e "${BLUE}Test 2: Testing POST endpoint${NC}"
RESPONSE=$(curl -s -X POST "$API_BASE/your-endpoint" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "field1": "value1",
    "field2": "value2"
  }')

echo "Response: $RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.id' > /dev/null; then
    RESOURCE_ID=$(echo "$RESPONSE" | jq -r '.id')
    echo -e "${GREEN}PASSED: POST endpoint successful, created ID: $RESOURCE_ID${NC}"
else
    echo -e "${RED}FAILED: Could not create resource${NC}"
fi
echo ""

echo -e "${BLUE}=== Test Complete ===${NC}"
```

**Save and run:**

```bash
chmod +x testing/api/test-your-feature.sh
./testing/api/test-your-feature.sh
```

---

## Testing with MCP Database Queries

### Using MCP for Verification

MCP (Model Context Protocol) allows natural language database queries. Use it to verify test results.

**Examples:**

```
"Show me all projects at location 6"
"What are the active user assignments for user 19?"
"List all issues for project 29"
"Show me the schema of the user_assignments table"
"How many RFIs are in status 'open'?"
"Get the attachments for issue 90"
```

### Verifying Test Data

**After creating a project:**

```bash
# Create project via API
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"location_id": 6, "basic_info": {"name": "Test Project"}}' \
  | jq .

# Verify in database via MCP
# Ask: "Show me the project named 'Test Project' in location 6"
```

**After assigning user to project:**

```bash
# Assign user
curl -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 20, "role_id": 2, "context_type": "project", "context_id": 29}' \
  | jq .

# Verify in database
# Ask: "Show me all project-level assignments for user 20"
```

### Common MCP Queries for Testing

**User verification:**
```
"Show user 19's profile including organization and super admin status"
"List all users in organization 10"
"What assignments does user 19 have?"
```

**Project verification:**
```
"Show all projects at location 6"
"Get project 29 details including location and organization"
"How many projects exist in organization 10?"
```

**Assignment verification:**
```
"Show all user_assignments for context_type 'project' and context_id 29"
"List users assigned to project 29"
"What locations can user 19 access?"
```

**Issue verification:**
```
"Show all issues for project 29"
"Get issue 90 with its comments"
"List open issues assigned to user 19"
```

**Attachment verification:**
```
"Show attachments for entity_type 'issue' and entity_id 90"
"List all attachments uploaded by user 19"
"How many attachments are in the system?"
```

---

## Test User Credentials

### Primary Test User (Super Admin)

**Credentials:**
- **Email:** `buildboard007+555@gmail.com`
- **Password:** `Mayur@1234`
- **User ID:** 19
- **Organization ID:** 10
- **Is Super Admin:** Yes
- **Access Level:** Full access to all resources in organization 10

**Use for:**
- Testing super admin features
- Creating test data
- Testing all endpoints without access restrictions
- Initial setup and configuration

### Creating Additional Test Users

**For testing different access levels:**

```sql
-- Via MCP, request:
"Create a new user with email test_org_admin@example.com in organization 10 with org-level access"

-- Or manually:
INSERT INTO iam.users (cognito_id, org_id, email, first_name, last_name, is_super_admin, created_by, updated_by)
VALUES ('test-cognito-id', 10, 'test_org_admin@example.com', 'Test', 'OrgAdmin', false, 19, 19)
RETURNING id;

-- Assign org-level access
INSERT INTO iam.user_assignments (user_id, role_id, context_type, context_id, org_id, created_by, updated_by)
VALUES (NEW_USER_ID, 1, 'organization', 10, 10, 19, 19);
```

**Test user scenarios:**

1. **Organization-level user:**
   - Context: `organization`, `context_id = org_id`
   - Should see: All locations and projects in org

2. **Location-level user:**
   - Context: `location`, `context_id = location_id`
   - Should see: Only projects at assigned location(s)

3. **Project-level user:**
   - Context: `project`, `context_id = project_id`
   - Should see: Only assigned project(s)

---

## Testing Access Control

### Access Control Test Scenarios

#### Scenario 1: Super Admin Access

**Test:**

```bash
# Super admin should see all projects
TOKEN=$(./testing/utilities/get-token.sh buildboard007+555@gmail.com Mayur@1234)

# Without location_id - should return all projects
curl -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.projects | length'

# With location_id - should return projects at that location
curl -X GET "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.projects | length'
```

**Expected:** All projects in organization (or filtered by location if provided)

#### Scenario 2: Organization-Level Access

**Setup:**

```bash
# Create org-level user assignment
curl -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 20,
    "role_id": 1,
    "context_type": "organization",
    "context_id": 10
  }' | jq .
```

**Test:**

```bash
# Get token for org-level user (if Cognito account exists)
TOKEN_ORG=$(./testing/utilities/get-token.sh org_user@example.com password)

# Should see all projects
curl -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN_ORG" \
  | jq '.projects'
```

**Expected:** All projects in organization

#### Scenario 3: Location-Level Access

**Setup:**

```bash
# Create location-level user assignment
curl -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 21,
    "role_id": 2,
    "context_type": "location",
    "context_id": 6
  }' | jq .
```

**Test:**

```bash
TOKEN_LOC=$(./testing/utilities/get-token.sh location_user@example.com password)

# Without location_id - should require location
curl -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN_LOC" \
  | jq .

# With correct location_id - should succeed
curl -X GET "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN_LOC" \
  | jq '.projects'

# With different location_id - should fail
curl -X GET "$API_BASE/projects?location_id=7" \
  -H "Authorization: Bearer $TOKEN_LOC" \
  | jq .
```

**Expected:**
- No location_id: Error or empty list
- Correct location_id: Projects at location 6
- Different location_id: 403 Forbidden

#### Scenario 4: Project-Level Access

**Setup:**

```bash
# Create project-level user assignment
curl -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 22,
    "role_id": 3,
    "context_type": "project",
    "context_id": 29
  }' | jq .
```

**Test:**

```bash
TOKEN_PROJ=$(./testing/utilities/get-token.sh project_user@example.com password)

# Should see only assigned project
curl -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq '.projects'

# Filter by location - should still only see assigned project
curl -X GET "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq '.projects'

# Try to access unassigned project - should fail
curl -X GET "$API_BASE/projects/30" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq .
```

**Expected:**
- List projects: Only project 29
- Get specific project: 200 for project 29, 403 for others

### Testing Authorization Failures

**Test cases:**

```bash
# 1. No token - should return 401
curl -X GET "$API_BASE/projects"

# 2. Invalid token - should return 401
curl -X GET "$API_BASE/projects" \
  -H "Authorization: Bearer invalid_token"

# 3. Wrong organization - should return 403
curl -X GET "$API_BASE/projects/999" \
  -H "Authorization: Bearer $TOKEN"

# 4. Deleted resource - should return 404
curl -X GET "$API_BASE/projects/999999" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Testing Attachments

### Upload Attachment Test

**Create test file:**

```bash
# Create test image
echo "Test image content" > /tmp/test-image.png

# Or use actual file
cp ~/Pictures/test.jpg /tmp/test-image.jpg
```

**Upload to issue:**

```bash
# Get base64 encoded file
FILE_CONTENT=$(base64 -i /tmp/test-image.png)

# Upload attachment
curl -X POST "$API_BASE/attachments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "issue",
    "entity_id": 90,
    "file_name": "test-image.png",
    "file_content": "'$FILE_CONTENT'",
    "file_type": "image/png",
    "attachment_type": "issue_photo"
  }' | jq .
```

**Expected response:**

```json
{
  "id": 123,
  "entity_type": "issue",
  "entity_id": 90,
  "file_name": "test-image.png",
  "file_path": "s3://bucket/path/to/file",
  "file_size": 1024,
  "attachment_type": "issue_photo",
  "uploaded_by": 19,
  "created_at": "2025-10-27T12:00:00Z"
}
```

### Download Attachment Test

```bash
# Get attachment details
ATTACHMENT=$(curl -s -X GET "$API_BASE/attachments?entity_type=issue&entity_id=90" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.attachments[0]')

ATTACHMENT_ID=$(echo $ATTACHMENT | jq -r '.id')

# Get download URL
DOWNLOAD_URL=$(curl -s -X GET "$API_BASE/attachments/$ATTACHMENT_ID/download" \
  -H "Authorization: Bearer $TOKEN" \
  | jq -r '.download_url')

# Download file
curl -o /tmp/downloaded-file.png "$DOWNLOAD_URL"

# Verify file
file /tmp/downloaded-file.png
```

### List Attachments Test

```bash
# List all attachments for an issue
curl -X GET "$API_BASE/attachments?entity_type=issue&entity_id=90" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# Filter by attachment type
curl -X GET "$API_BASE/attachments?entity_type=issue&entity_id=90&attachment_type=issue_photo" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# List comment attachments
curl -X GET "$API_BASE/attachments?entity_type=issue_comment&entity_id=123" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .
```

### Delete Attachment Test

```bash
# Soft delete attachment
curl -X DELETE "$API_BASE/attachments/$ATTACHMENT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# Verify deletion
curl -X GET "$API_BASE/attachments/$ATTACHMENT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# Should return 404 or is_deleted: true
```

### Complete Attachment Test Script

See `/Users/mayur/git_personal/infrastructure/testing/api/test-attachment-api.sh` for a comprehensive attachment test.

---

## Creating Test Data

### Creating Test Projects

```bash
# Create project
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "location_id": 6,
    "basic_info": {
      "name": "Test Project - Auto",
      "description": "Created for testing"
    },
    "project_details": {
      "project_stage": "planning",
      "work_scope": "new_construction"
    },
    "timeline": {
      "start_date": "2025-01-01",
      "planned_end_date": "2025-12-31"
    }
  }' | jq .
```

### Creating Test Issues

```bash
PROJECT_ID=29

# Create issue
curl -X POST "$API_BASE/projects/$PROJECT_ID/issues" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Issue",
    "description": "Created for testing",
    "issue_type": "quality_issue",
    "priority": "medium",
    "status": "open"
  }' | jq .
```

### Creating Test RFIs

```bash
# Create RFI
curl -X POST "$API_BASE/projects/$PROJECT_ID/rfis" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "Test RFI",
    "question": "What is the answer to this test question?",
    "priority": "normal",
    "status": "open",
    "due_date": "2025-11-01"
  }' | jq .
```

### Creating Test Users

```bash
# Create user (super admin only)
curl -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testuser@example.com",
    "first_name": "Test",
    "last_name": "User",
    "cognito_id": "temp-cognito-id"
  }' | jq .
```

### Batch Creating Test Data

**Script to create multiple test entities:**

```bash
#!/bin/bash

TOKEN=$(./testing/utilities/get-token.sh)
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"

echo "Creating 5 test projects..."

for i in {1..5}; do
    RESPONSE=$(curl -s -X POST "$API_BASE/projects" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{
        \"location_id\": 6,
        \"basic_info\": {
          \"name\": \"Test Project $i\",
          \"description\": \"Auto-generated test project\"
        }
      }")

    PROJECT_ID=$(echo $RESPONSE | jq -r '.project.id')
    echo "Created project $i with ID: $PROJECT_ID"
done

echo "Test data creation complete"
```

---

## Debugging API Errors

### Common Error Codes

| Code | Meaning | Common Causes |
|------|---------|---------------|
| 400 | Bad Request | Invalid input, missing required fields |
| 401 | Unauthorized | Missing or invalid token |
| 403 | Forbidden | No permission to access resource |
| 404 | Not Found | Resource doesn't exist or is deleted |
| 500 | Internal Server Error | Backend error, database issue |

### Debugging 400 Bad Request

**Check request body:**

```bash
# Validate JSON
echo '{"field": "value"}' | jq .

# Check required fields
curl -v -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "location_id": 6
  }' | jq .

# Error message should indicate missing fields
```

**Common causes:**
- Missing required fields
- Invalid field values (wrong type, out of range)
- Malformed JSON
- Invalid date format

### Debugging 401 Unauthorized

**Check token:**

```bash
# Decode token
echo $TOKEN | cut -d. -f2 | base64 -d | jq .

# Check expiration
# Tokens expire after 1 hour

# Get fresh token
TOKEN=$(./testing/utilities/get-token.sh)
```

**Common causes:**
- Token expired (get new token)
- Wrong token type (use ID token, not access token)
- Token not sent in Authorization header
- Cognito configuration issue

### Debugging 403 Forbidden

**Check access:**

```bash
# Verify user has access to resource
# Via MCP: "Show user 19's assignments"

# Check if resource belongs to same org
# Via MCP: "Get project 29's organization"

# Verify user is authenticated
echo $TOKEN | cut -d. -f2 | base64 -d | jq '.user_id, .org_id, .isSuperAdmin'
```

**Common causes:**
- User not assigned to project/location
- Resource in different organization
- Super admin check failing
- Access control logic error

### Debugging 404 Not Found

**Check resource exists:**

```bash
# Via MCP
# "Does project 29 exist and is it deleted?"
# "Show me issue 90 including deleted status"

# Check ID in request
curl -v -X GET "$API_BASE/projects/INVALID_ID" \
  -H "Authorization: Bearer $TOKEN"
```

**Common causes:**
- Resource ID doesn't exist
- Resource is soft-deleted
- Wrong endpoint path
- Typo in resource ID

### Debugging 500 Internal Server Error

**Check CloudWatch logs:**

```bash
# Tail Lambda logs
aws logs tail /aws/lambda/infrastructure-project-management \
  --since 5m \
  --follow \
  --profile dev \
  --region us-east-2

# Look for error messages
aws logs tail /aws/lambda/infrastructure-project-management \
  --since 30m \
  --profile dev \
  --region us-east-2 \
  | grep -i error
```

**Check database:**

```bash
# Via MCP
# "Is the database connection working?"
# "Show me recent errors in database logs"
```

**Common causes:**
- Database connection timeout
- SQL query error
- Null pointer dereference
- Missing environment variable
- AWS service permission denied

### Using Verbose Mode

**Get detailed request/response:**

```bash
# Use -v flag for verbose output
curl -v -X GET "$API_BASE/projects?location_id=6" \
  -H "Authorization: Bearer $TOKEN" \
  2>&1 | tee debug.log

# Shows:
# - Request headers
# - Response headers
# - Status code
# - Response body
```

### Testing Error Handling

**Intentionally trigger errors:**

```bash
# 1. Missing required field
curl -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"basic_info": {}}' \
  | jq .

# 2. Invalid ID format
curl -X GET "$API_BASE/projects/abc" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# 3. Deleted resource
curl -X GET "$API_BASE/projects/999999" \
  -H "Authorization: Bearer $TOKEN" \
  | jq .

# 4. Forbidden resource
# (as project-level user, try to access different project)
curl -X GET "$API_BASE/projects/30" \
  -H "Authorization: Bearer $TOKEN_PROJ" \
  | jq .
```

---

## Common Testing Scenarios

### Scenario 1: Complete Project Workflow

```bash
# 1. Create project
PROJECT=$(curl -s -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "location_id": 6,
    "basic_info": {"name": "Test Project", "description": "Testing"}
  }')
PROJECT_ID=$(echo $PROJECT | jq -r '.project.id')
echo "Created project: $PROJECT_ID"

# 2. Assign user to project
curl -s -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": 20,
    \"role_id\": 2,
    \"context_type\": \"project\",
    \"context_id\": $PROJECT_ID
  }" | jq .

# 3. Create issue
ISSUE=$(curl -s -X POST "$API_BASE/projects/$PROJECT_ID/issues" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Issue",
    "description": "First issue",
    "issue_type": "quality_issue",
    "priority": "high",
    "status": "open"
  }')
ISSUE_ID=$(echo $ISSUE | jq -r '.issue.id')
echo "Created issue: $ISSUE_ID"

# 4. Add comment to issue
curl -s -X POST "$API_BASE/issues/$ISSUE_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "comment_text": "This is a test comment"
  }' | jq .

# 5. Verify via MCP
# "Show me all issues for project $PROJECT_ID"
# "List users assigned to project $PROJECT_ID"
```

### Scenario 2: Access Control Verification

```bash
# As super admin - create project
PROJECT_ID=$(curl -s -X POST "$API_BASE/projects" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"location_id": 6, "basic_info": {"name": "Access Test Project"}}' \
  | jq -r '.project.id')

# Assign project-level user
curl -s -X POST "$API_BASE/assignments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\": 22, \"role_id\": 3, \"context_type\": \"project\", \"context_id\": $PROJECT_ID}" \
  | jq .

# Test as project-level user
TOKEN_USER=$(./testing/utilities/get-token.sh project_user@example.com password)

# Should succeed - assigned project
curl -s -X GET "$API_BASE/projects/$PROJECT_ID" \
  -H "Authorization: Bearer $TOKEN_USER" \
  | jq .

# Should fail - different project
curl -s -X GET "$API_BASE/projects/999" \
  -H "Authorization: Bearer $TOKEN_USER" \
  | jq .
```

### Scenario 3: Attachment Upload/Download

See `/Users/mayur/git_personal/infrastructure/testing/api/test-attachment-api.sh`

### Scenario 4: Pagination Testing

```bash
# Create multiple projects
for i in {1..25}; do
    curl -s -X POST "$API_BASE/projects" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"location_id\": 6, \"basic_info\": {\"name\": \"Project $i\"}}" \
      > /dev/null
done

# Test pagination
curl -s -X GET "$API_BASE/projects?location_id=6&page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.pagination'

curl -s -X GET "$API_BASE/projects?location_id=6&page=2&limit=10" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.pagination'
```

---

## Testing Checklist

### Before Deployment

- [ ] Get authentication token successfully
- [ ] All GET endpoints return 200 for valid requests
- [ ] All POST endpoints create resources correctly
- [ ] All PUT endpoints update resources correctly
- [ ] All DELETE endpoints soft-delete correctly
- [ ] Authorization checks work (403 for unauthorized)
- [ ] Validation works (400 for invalid input)
- [ ] Error messages are clear and helpful
- [ ] CloudWatch logs show no errors
- [ ] Database queries via MCP confirm data state

### Access Control Tests

- [ ] Super admin can access all resources
- [ ] Org-level user sees all org resources
- [ ] Location-level user sees only assigned location resources
- [ ] Project-level user sees only assigned project resources
- [ ] Users cannot access resources in different organizations
- [ ] Proper 403 errors for unauthorized access

### Data Integrity Tests

- [ ] Required fields are enforced
- [ ] Default values are applied
- [ ] Foreign key relationships maintained
- [ ] Soft deletes work (is_deleted flag)
- [ ] Timestamps (created_at, updated_at) populated
- [ ] Auto-generated numbers are unique

### Edge Cases

- [ ] Empty request body handled
- [ ] Invalid JSON handled
- [ ] Missing required fields handled
- [ ] Invalid IDs handled (non-existent, wrong format)
- [ ] Deleted resources return 404
- [ ] Expired tokens return 401
- [ ] Very long field values handled
- [ ] Special characters in fields handled

---

**Last Updated:** 2025-10-27