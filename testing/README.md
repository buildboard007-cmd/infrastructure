# Testing Guide

## Overview

This directory contains API test scripts for the BuildBoard Infrastructure project.

## Directory Structure

```
testing/
└── api/           # API endpoint test scripts (shell scripts only)
```

## Important Testing Rules

### ✅ DO:
- **Use shell scripts to test API endpoints** (examples in `api/` folder)
- **Use Postman collections** for API testing (located in `/postman/`)
- **Use MCP natural language queries** for database verification
- **Get tokens from Cognito API** (no hardcoded credentials)

### ❌ DO NOT:
- ~~Use Node.js/bash scripts to query the database directly~~ (Use MCP instead!)
- ~~Hardcode passwords in test scripts~~
- ~~Create test files in project root~~ (Use `testing/api/`)

## API Testing

### Test Scripts

All API test scripts are in the `api/` folder:

```bash
cd testing/api

# Test project user management
./test-project-user-management.sh

# Test access control for GET /projects
./test-get-projects-access-control.sh

# Test issue comments
./test-issue-comments.sh

# Test comment attachments
./test-comment-attachment.sh
```

### Test User Credentials

**Primary Test User:**
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`
- User ID: 19
- Org ID: 10
- Super Admin: Yes

### Getting Authentication Tokens

Use the Cognito API to get an ID token:

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

echo $TOKEN
```

### API Base URL

**Dev Environment:**
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
```

### Example API Test

```bash
#!/bin/bash
API_BASE="https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main"

# Get token
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

# Test GET /projects
curl -X GET "${API_BASE}/projects" \
  -H "Authorization: Bearer ${TOKEN}" \
  | jq '.'
```

## Database Verification

### Use MCP (Claude's Built-in Database Tool)

For database queries, **always use MCP** through natural language:

```
"Show me all users in organization 10"
"List projects at location 6"
"What roles does user 19 have?"
"Show the schema for the user_assignments table"
```

### Why MCP Instead of Scripts?

- ✅ **Secure**: No hardcoded credentials
- ✅ **Simple**: Natural language queries
- ✅ **Correct**: Follows CLAUDE.md project rules
- ✅ **Maintained**: Built-in tool, always up to date

## Postman Collections

Located in `/postman/`:

- `AssignmentManagement.postman_collection.json`
- `AttachmentManagement.postman_collection.json`
- `IssueManagement.postman_collection.json`
- `ProjectManagement.postman_collection.json`
- `RFIManagement.postman_collection.json`
- `RolesManagement.postman_collection.json`
- `SubmittalManagement.postman_collection.json`
- `Infrastructure.postman_collection.json`

### Using Postman

1. Import collection into Postman
2. Set environment variables:
   - `API_BASE`: `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
   - `ID_TOKEN`: Get from Cognito authentication
3. Run requests

## Best Practices

1. **Never hardcode credentials** - Use environment variables or Cognito API
2. **Test against Dev environment first** - Don't test directly on Prod
3. **Clean up test data** - Remove test records after testing
4. **Use descriptive test names** - Make it clear what each script tests
5. **Check response status codes** - Verify 200, 400, 403, 404, 500 as expected
6. **Use jq for JSON parsing** - Makes output readable and testable

## Related Documentation

- [Testing Guide](/docs/guides/testing-guide.md) - Comprehensive testing documentation
- [API Usage Guide](/docs/guides/api-usage-guide.md) - API integration patterns
- [Development Workflow](/docs/guides/development-workflow.md) - Development process

---

**Last Updated:** 2025-10-27