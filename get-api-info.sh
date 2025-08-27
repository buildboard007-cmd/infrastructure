#!/bin/bash

echo "🌐 Getting API Gateway Information"
echo "=================================="

REGION="us-east-2"

# Get API ID and Name
API_INFO=$(aws apigateway get-rest-apis --region $REGION --query 'items[?name==`Infrastructure API`].[id,name]' --output text)
API_ID=$(echo $API_INFO | awk '{print $1}')
API_NAME=$(echo $API_INFO | awk '{print $2}')

if [ -z "$API_ID" ]; then
    echo "❌ Infrastructure API not found"
    exit 1
fi

echo "✅ API Name: $API_NAME"
echo "✅ API ID: $API_ID"

# Get Stage
STAGE=$(aws apigateway get-stages --rest-api-id $API_ID --region $REGION --query 'item[0].stageName' --output text)
echo "✅ Stage: $STAGE"

# Get Resources
echo ""
echo "📋 API Resources:"
aws apigateway get-resources --rest-api-id $API_ID --region $REGION --query 'items[*].[pathPart,resourceMethods]' --output table

# Construct URLs
BASE_URL="https://$API_ID.execute-api.$REGION.amazonaws.com/$STAGE"

echo ""
echo "🔗 API Gateway URLs:"
echo "Base URL: $BASE_URL"
echo "Organization GET: $BASE_URL/org"
echo "Organization PUT: $BASE_URL/org"
echo ""

echo "📋 For Postman Environment:"
echo "api_gateway_url = $BASE_URL"
echo ""

echo "🧪 Test with cURL (replace YOUR_JWT_TOKEN):"
echo "curl -X GET \"$BASE_URL/org\" -H \"Authorization: YOUR_JWT_TOKEN\" -H \"Content-Type: application/json\""