# SuperAdmin Signup Workflow - Postman Collection

This Postman collection provides a complete test suite for the SuperAdmin signup workflow, including Cognito authentication, database integration, and API Gateway testing.

## 📋 Collection Contents

### Requests in Order:
1. **SuperAdmin Signup** - Register user with `isSuperAdmin=true`
2. **Confirm Email** - Verify email with confirmation code
3. **SuperAdmin Signin** - Authenticate and get JWT tokens
4. **Validate JWT Token Claims** - Verify custom claims from token customizer
5. **Get Organization Info** - Test API Gateway with Cognito auth
6. **Update Organization Name** - Test PUT endpoint 
7. **Cleanup** - Delete test user

## 🚀 Setup Instructions

### 1. Import the Collection
1. Open Postman
2. Click **Import** 
3. Select `SuperAdmin-Signup-Workflow.postman_collection.json`

### 2. Import the Environment
1. Click **Import**
2. Select `Environment_Template.postman_environment.json`
3. Rename it to your preference (e.g., "SuperAdmin Test Environment")

### 3. Configure Environment Variables

The environment is pre-configured with current values, but verify these settings:

#### Required Variables:
- `aws_region`: `us-east-2`
- `cognito_client_id`: `3f0fb5mpivctnvj85tucusf88e`
- `cognito_user_pool_id`: `us-east-2_8JFjvA7xM`
- `api_gateway_url`: `https://YOUR-API-ID.execute-api.us-east-2.amazonaws.com/v1`
- `test_email`: `buildboard007+mayur@gmail.com`
- `test_password`: `Mayur@1234`

#### Auto-populated Variables:
- `access_token` - Auto-set after signin
- `id_token` - Auto-set after signin  
- `refresh_token` - Auto-set after signin
- `user_sub` - Auto-set after signup
- `confirmation_code` - **You must set this manually**

### 4. Get API Gateway URL
Run this command to get your API Gateway URL:
```bash
aws apigateway get-rest-apis --region us-east-2 --query 'items[?name==`Infrastructure API`].[id,name]' --output table
```

Then update the `api_gateway_url` variable to:
```
https://YOUR-API-ID.execute-api.us-east-2.amazonaws.com/v1
```

## 🧪 Running the Tests

### Method 1: Sequential Testing (Recommended)
Run each request in order, following the manual steps:

1. **Run "1. SuperAdmin Signup"**
   - ✅ Should return `UserSub` and `CodeDeliveryDetails`
   - Check console for success message

2. **Check Your Email**
   - Look for AWS Cognito verification email
   - Copy the 6-digit confirmation code

3. **Set Confirmation Code**
   - In the environment, set `confirmation_code` to the code from email
   - Example: `233005`

4. **Run "2. Confirm Email"**
   - ✅ Should return 200 status
   - Email is now verified

5. **Run "3. SuperAdmin Signin"**
   - ✅ Should return JWT tokens
   - Tokens are automatically stored in environment

6. **Run "4. Validate JWT Token Claims"**
   - ✅ Check console for token claims validation
   - Verify custom claims: `isSuperAdmin`, `user_id`, `org_name`, `status`

7. **Run "5. Get Organization Info"**
   - ✅ Should return organization data
   - Tests API Gateway + Cognito authentication

8. **Run "6. Update Organization Name"**
   - ✅ Should update organization name
   - Tests PUT endpoint functionality

9. **Run "7. Cleanup"** (Optional)
   - Deletes the test user from Cognito

### Method 2: Collection Runner
1. Click **Runner** in Postman
2. Select the "SuperAdmin Signup Workflow" collection
3. **Stop at "2. Confirm Email"**
4. Check email and set `confirmation_code` variable
5. Continue with remaining requests

## ✅ Expected Results

### Successful Workflow:
```
✅ 1. SuperAdmin Signup - UserSub created
✅ 2. Confirm Email - Email verified  
✅ 3. SuperAdmin Signin - JWT tokens received
✅ 4. Validate JWT Claims - Custom claims present:
   - isSuperAdmin: true
   - user_id: 1 (or higher)
   - org_name: "System"
   - status: "pending_org_setup"
✅ 5. Get Organization Info - API responds with org data
✅ 6. Update Organization Name - Name updated successfully
```

### Token Validation Console Output:
```
🔍 ID Token Claims:
  Standard Claims:
    sub: 112bd5e0-70a1-70c0-0ca2-72227853a55d
    email: buildboard007+mayur@gmail.com
    email_verified: true
  Custom Claims (from Token Customizer):
    isSuperAdmin: true
    user_id: 1
    org_name: System
    status: pending_org_setup
```

## 🛠️ Troubleshooting

### Common Issues:

#### "UserExistsException"
- Delete existing user: `aws cognito-idp admin-delete-user --user-pool-id us-east-2_8JFjvA7xM --username buildboard007+mayur@gmail.com --region us-east-2`

#### "Invalid confirmation code"
- Double-check the 6-digit code from email
- Ensure no extra spaces in `confirmation_code` variable

#### "Missing custom claims in JWT"
- Check Lambda function logs for database permission errors
- Verify token customizer Lambda is running successfully

#### API Gateway "403 Forbidden"  
- Verify `api_gateway_url` is correct
- Check that `id_token` is being sent in Authorization header

#### Database Connection Issues
- Check Lambda function logs for database permission errors
- Verify RDS security groups allow Lambda access

## 📊 Testing Different Scenarios

### Test Variations:
1. **New User Each Time**: Change `test_email` (e.g., `buildboard007+test2@gmail.com`)
2. **Different Organization Names**: Modify the PUT request body
3. **Token Refresh**: Use refresh token to get new access tokens

### Environment Variables for Testing:
- Create multiple environments (Dev, Staging, Prod)
- Each with different `cognito_client_id`, `api_gateway_url`, etc.

## 🔧 Advanced Usage

### Custom Scripts:
The collection includes JavaScript test scripts that:
- Validate response structure
- Store tokens automatically  
- Parse and validate JWT claims
- Provide detailed console logging

### Monitoring:
- All requests include comprehensive test assertions
- Console logs provide step-by-step feedback
- Failed tests highlight specific issues

## 🏗️ Architecture Tested

This collection validates:
- ✅ **Cognito User Pools** - Registration, confirmation, authentication
- ✅ **Lambda Post-Confirmation** - `infrastructure-user-signup` creates database records
- ✅ **Lambda Token Customizer** - `infrastructure-token-customizer` enriches JWT tokens
- ✅ **Database Integration** - User/organization data stored in PostgreSQL
- ✅ **API Gateway + Cognito Authorizer** - Protected endpoints with JWT validation
- ✅ **Organization Management API** - GET/PUT operations with SuperAdmin permissions

Happy testing! 🎉