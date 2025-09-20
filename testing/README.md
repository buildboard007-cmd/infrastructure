# Testing Guidelines & Organization

## Directory Structure

```
testing/
├── api/           # API endpoint testing
├── auth/          # Authentication & JWT testing
├── database/      # Database queries & checks
├── utilities/     # Helper scripts & explanations
└── README.md      # This file
```

## Testing Process

### IMPORTANT RULES
1. **Never create test files in project root** - always use organized directories
2. **Never scatter files in MCP or other directories** - everything goes in `/testing/`
3. **Always clean up after testing** - remove temporary test files
4. **Use MCP server for database operations** - no direct DB connection scripts
5. **Follow naming conventions** - descriptive names with purpose

### File Categories

#### API Testing (`/testing/api/`)
- API endpoint functionality
- Route validation
- Request/response testing
- API Gateway integration tests

**Examples:** `test-user-endpoints.js`, `verify-api-routes.js`

#### Authentication Testing (`/testing/auth/`)
- JWT token validation
- Cognito integration
- Authorization checks
- User signup/signin flows

**Examples:** `test-jwt-tokens.js`, `verify-cognito-auth.js`

#### Database Testing (`/testing/database/`)
- Data validation
- Query testing
- Schema verification
- User/org/project checks

**Examples:** `check-user-data.js`, `verify-org-structure.js`

#### Utilities (`/testing/utilities/`)
- Helper scripts
- Data explanation tools
- MCP server tests
- General debugging tools

**Examples:** `explain-data-structure.js`, `test-mcp-connection.js`

## Creating New Test Files

### DO:
```bash
# Create in appropriate directory
touch testing/api/test-new-endpoint.js

# Use descriptive names
touch testing/auth/verify-super-admin-flow.js

# Follow existing patterns
```

### DON'T:
```bash
# Never create in root
touch test-something.js

# Avoid generic names
touch debug.js

# Don't scatter files randomly
touch random-location/test.js
```

## Common Testing Patterns

### API Testing Template
```javascript
// testing/api/test-endpoint-name.js
const https = require('https');

async function testEndpoint() {
    const options = {
        hostname: '74zc1md7sc.execute-api.us-east-2.amazonaws.com',
        path: '/main/endpoint',
        method: 'GET',
        headers: {
            'Authorization': 'Bearer ' + process.env.ID_TOKEN
        }
    };
    // Test implementation
}

testEndpoint().catch(console.error);
```

### Database Query Template
```javascript
// testing/database/check-data-name.js
// NOTE: Use MCP server instead of direct DB connection
// Ask Claude to run database queries using MCP
console.log('Use: "How many users are active in the database?"');
```

### Authentication Template
```javascript
// testing/auth/test-auth-name.js
// JWT and Cognito testing patterns
```

## Cleanup Process

After testing is complete:
1. Remove temporary files: `rm testing/*/temp-*.js`
2. Keep only reusable test scripts
3. Update documentation if new patterns are established

## Best Practices

- **Use environment variables** for sensitive data
- **Document test purpose** in file headers
- **Keep tests simple** and focused
- **Use existing infrastructure** (MCP, CDK patterns)
- **Clean up after yourself**

## Integration with Project

- Tests complement the main Go backend in `/src`
- Use CDK infrastructure patterns from `/lib`
- Leverage MCP server for database operations
- Follow project's authentication patterns