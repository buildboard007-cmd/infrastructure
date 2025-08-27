# Organization Management API - Postman Collection

This Postman collection provides comprehensive testing for the Organization Management API endpoints with Cognito authentication.

## Files

- `Organization_Management_API.postman_collection.json` - Main collection with all API tests
- `Environment_Template.postman_environment.json` - Environment template with required variables
- `README.md` - This instruction file

## Setup Instructions

### 1. Import Collection and Environment

1. Open Postman
2. Import the collection: `Organization_Management_API.postman_collection.json`
3. Import the environment: `Environment_Template.postman_environment.json`
4. Select the environment in the top-right dropdown

### 2. Configure Environment Variables

Update these variables in your Postman environment:

#### Required Variables (Get from CDK deployment outputs):
- `api_gateway_url` - Your API Gateway URL
  - Example: `https://abc123xyz.execute-api.us-east-2.amazonaws.com/v1`
  - Find in: AWS Console → API Gateway → Your API → Stages → v1
  
- `cognito_hosted_ui_url` - Cognito Hosted UI domain
  - Example: `https://infrastructure-users-dev.auth.us-east-2.amazoncognito.com`
  - Find in: CDK outputs or AWS Console → Cognito → User Pools → Domain
  
- `cognito_client_id` - Cognito User Pool Client ID
  - Example: `1a2b3c4d5e6f7g8h9i0j`
  - Find in: CDK outputs or AWS Console → Cognito → User Pools → App clients

#### Optional Variables:
- `redirect_uri` - OAuth redirect URI (default: `https://localhost:3000/callback`)
  - Must match what's configured in your Cognito User Pool Client

### 3. Authentication Flow

#### Step 1: Get Authorization URL
1. Run `Authentication → Get Cognito Auth URL`
2. Check the Console tab for the authentication URL
3. Copy the URL and paste it in your browser

#### Step 2: Login and Get Code
1. Login using your credentials in the browser
2. After successful login, you'll be redirected to your redirect URI
3. Copy the `code` parameter from the redirect URL
4. Set this code in the `authorization_code` environment variable

#### Step 3: Exchange Code for Tokens
1. Run `Authentication → Exchange Code for Tokens`
2. This will automatically store your access tokens in the environment
3. Tokens are valid for 24 hours

#### Step 4: Use the API
Now you can run the organization management endpoints!

## Available Endpoints

### Organization Management
- **GET /org** - Retrieve organization information
- **PUT /org** - Update organization name
- **OPTIONS /org** - CORS preflight request

### Test Scenarios
- Valid organization updates
- Invalid data validation
- Unauthorized requests
- Unsupported HTTP methods

## Usage Tips

### Token Management
- Tokens are automatically stored after authentication
- Use `Authentication → Refresh Access Token` if tokens expire
- The collection checks token expiry before requests

### Testing Flow
1. **First Time Setup**: Run authentication flow to get tokens
2. **Get Current Org**: Run `GET /org` to see current organization (likely "system")
3. **Update Organization**: Run `PUT /org` to change from "system" to your company name
4. **Verify Update**: Run `GET /org` again to confirm the change

### Expected Behavior

#### New User (pending_org_setup):
```json
{
  "org_id": 1,
  "org_name": "system",
  "created_at": "2023-12-01T10:00:00Z",
  "updated_at": "2023-12-01T10:00:00Z"
}
```

#### After Organization Setup:
```json
{
  "org_id": 1,
  "org_name": "My Company Inc",
  "created_at": "2023-12-01T10:00:00Z",
  "updated_at": "2023-12-01T10:30:00Z"
}
```

### Common Issues

#### 401 Unauthorized
- Check if your access token is valid
- Run the refresh token request
- Re-authenticate if refresh fails

#### 403 Forbidden
- Ensure your user has `isSuperAdmin = true`
- Only super admins can manage organizations

#### 400 Bad Request
- Check request body format
- Organization name must be 3-150 characters

#### 500 Internal Server Error
- Check Lambda logs in CloudWatch
- Verify database connectivity
- Check if organization management Lambda is deployed

## Testing Checklist

- [ ] Authentication flow works
- [ ] GET /org returns organization data
- [ ] PUT /org updates organization name
- [ ] PUT /org validates input (min 3 characters)
- [ ] PUT /org changes user status to 'active'
- [ ] CORS requests work properly
- [ ] Unauthorized requests are rejected
- [ ] Invalid HTTP methods are rejected

## Troubleshooting

### CDK Deployment Outputs
After deploying your CDK stack, get the required values:

```bash
# Get CDK outputs
cdk deploy --outputs-file outputs.json

# Look for:
# - UserPoolId
# - UserPoolClientId
# - HostedUIDomain
# - API Gateway URL
```

### CloudWatch Logs
Check Lambda function logs for debugging:
- `infrastructure-organization-management`
- `infrastructure-token-customizer` 
- `infrastructure-user-signup`
- `infrastructure-api-gateway-cors`

### Database Verification
Verify the organization was updated in your database:

```sql
SELECT * FROM iam.organization;
SELECT * FROM iam.users WHERE isSuperAdmin = true;
```

## Security Notes

- Access tokens are stored in Postman environment (keep secure)
- Only super admin users can access these endpoints
- All requests require valid Cognito JWT tokens
- CORS is configured for specific origins only

## Support

For issues:
1. Check CloudWatch logs for Lambda errors
2. Verify Cognito configuration
3. Confirm database connectivity
4. Test individual components in isolation