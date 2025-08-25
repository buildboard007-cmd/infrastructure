#!/bin/bash

echo "🔍 Getting Cognito Configuration from AWS Dev Account"
echo "================================================="

# Set region
REGION="us-east-2"

echo "📍 Region: $REGION"
echo ""

# Get User Pool ID
echo "1️⃣ Getting User Pool ID..."
USER_POOL_ID=$(aws cognito-idp list-user-pools --max-results 10 --region $REGION --query 'UserPools[?Name==`Users`].Id' --output text)

if [ -z "$USER_POOL_ID" ]; then
    echo "❌ User Pool not found. Looking for all pools..."
    aws cognito-idp list-user-pools --max-results 10 --region $REGION --query 'UserPools[].[Name,Id]' --output table
    exit 1
else
    echo "✅ User Pool ID: $USER_POOL_ID"
fi

echo ""

# Get Client ID
echo "2️⃣ Getting Client ID..."
CLIENT_ID=$(aws cognito-idp list-user-pool-clients --user-pool-id $USER_POOL_ID --region $REGION --query 'UserPoolClients[0].ClientId' --output text)

if [ -z "$CLIENT_ID" ]; then
    echo "❌ Client not found"
    exit 1
else
    echo "✅ Client ID: $CLIENT_ID"
fi

echo ""

# Get Hosted UI Domain (if exists)
echo "3️⃣ Getting Hosted UI Domain..."
DOMAIN_PREFIX=$(aws cognito-idp describe-user-pool --user-pool-id $USER_POOL_ID --region $REGION --query 'UserPool.Domain' --output text 2>/dev/null)

if [ "$DOMAIN_PREFIX" != "None" ] && [ ! -z "$DOMAIN_PREFIX" ]; then
    echo "✅ Domain Prefix: $DOMAIN_PREFIX"
    echo "✅ Hosted UI URL: https://$DOMAIN_PREFIX.auth.$REGION.amazoncognito.com"
else
    echo "ℹ️  No custom domain configured"
fi

echo ""

# Show User Pool Configuration
echo "4️⃣ User Pool Configuration:"
aws cognito-idp describe-user-pool --user-pool-id $USER_POOL_ID --region $REGION --query '{
    UserPoolName: UserPool.Name,
    SelfSignUpEnabled: UserPool.Policies.PasswordPolicy,
    AutoVerifiedAttributes: UserPool.AutoVerifiedAttributes,
    CustomAttributes: UserPool.Schema[?AttributeDataType==`String` || AttributeDataType==`Boolean`].{Name: Name, DataType: AttributeDataType}
}' --output table

echo ""

# Show configuration for test script
echo "🧪 CONFIGURATION FOR TEST SCRIPT"
echo "================================="
echo "Copy these values into test-superadmin-signup.js:"
echo ""
echo "const CONFIG = {"
echo "    region: '$REGION',"
echo "    userPoolId: '$USER_POOL_ID',"
echo "    clientId: '$CLIENT_ID',"
echo "    testEmail: 'bildboard007+mayur@gmail.com',"
echo "    testPassword: 'Mayur@1234'"
echo "};"
echo ""

# Test AWS credentials
echo "5️⃣ Testing AWS Credentials..."
ACCOUNT_ID=$(aws sts get-caller-identity --query 'Account' --output text 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "✅ AWS Credentials working"
    echo "✅ Account ID: $ACCOUNT_ID"
else
    echo "❌ AWS Credentials not configured properly"
    echo "Run: aws configure"
fi

echo ""
echo "🚀 Ready to test! Run: node test-superadmin-signup.js"