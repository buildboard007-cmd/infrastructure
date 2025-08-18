#!/bin/bash

# Pipeline Validation Script
# This script validates your CDK pipeline setup before deployment

set -e

echo "======================================"
echo "CDK Pipeline Pre-Deployment Validation"
echo "======================================"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Validation results
ERRORS=0
WARNINGS=0

# Function to check status
check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
    else
        echo -e "${RED}✗${NC} $1"
        ERRORS=$((ERRORS + 1))
    fi
}

# Function for warnings
warning() {
    echo -e "${YELLOW}⚠${NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

echo "1. Checking Node.js dependencies..."
npm list aws-cdk-lib @aws-cdk/aws-lambda-go-alpha > /dev/null 2>&1
check_status "Required CDK packages installed"

echo ""
echo "2. Checking TypeScript compilation..."
npx tsc --noEmit > /dev/null 2>&1
check_status "TypeScript compiles without errors"

echo ""
echo "3. Checking Go modules..."
cd src && go mod verify > /dev/null 2>&1
check_status "Go modules verified"
cd ..

echo ""
echo "4. Checking Lambda function compilation..."
cd src/infrastructure-api-gateway-cors && go build . > /dev/null 2>&1
check_status "Lambda functions compile successfully"
cd ../..

echo ""
echo "5. Checking CDK synthesis..."
npx cdk synth --quiet > /dev/null 2>&1
check_status "CDK synthesis successful"

echo ""
echo "6. Checking configuration..."

# Check if accounts are different
TOOLS_ACC=$(grep "toolsAccount" config/config.ts | grep -o '[0-9]\{12\}')
DEV_ACC=$(grep "devAccount" config/config.ts | grep -o '[0-9]\{12\}')
PROD_ACC=$(grep "productionAccount" config/config.ts | grep -o '[0-9]\{12\}')

if [ "$TOOLS_ACC" = "$DEV_ACC" ] || [ "$TOOLS_ACC" = "$PROD_ACC" ] || [ "$DEV_ACC" = "$PROD_ACC" ]; then
    warning "Some AWS accounts are the same - this might not be a cross-account setup"
else
    echo -e "${GREEN}✓${NC} AWS accounts are properly configured for cross-account deployment"
fi

# Check GitHub connection
GITHUB_ARN=$(grep "githubConnectionArn" config/config.ts | grep -o 'arn:aws:codeconnections:[^"]*')
if [[ $GITHUB_ARN == *"401448503050"* ]]; then
    echo -e "${GREEN}✓${NC} GitHub connection ARN matches tools account"
else
    warning "GitHub connection ARN doesn't match tools account - verify it's correct"
fi

# Check bootstrap qualifier
QUALIFIER=$(grep "cdkBootstrapQualifier" config/config.ts | grep -o '"[^"]*"' | tail -1 | tr -d '"')
echo -e "${GREEN}✓${NC} CDK bootstrap qualifier: $QUALIFIER"

echo ""
echo "7. Checking AWS CLI configuration..."
aws sts get-caller-identity --profile tools > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} AWS CLI profile 'tools' is configured"
else
    warning "AWS CLI profile 'tools' not configured - needed for deployment"
fi

echo ""
echo "======================================"
echo "Validation Summary"
echo "======================================"

if [ $ERRORS -eq 0 ]; then
    if [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}All checks passed!${NC}"
        echo ""
        echo "Ready for deployment. Next steps:"
        echo "1. Ensure all AWS accounts are bootstrapped:"
        echo "   npm run bootstrap"
        echo "2. Deploy the pipeline:"
        echo "   npm run deploy"
    else
        echo -e "${YELLOW}Validation completed with $WARNINGS warnings${NC}"
        echo "Review the warnings above before proceeding with deployment."
    fi
else
    echo -e "${RED}Validation failed with $ERRORS errors${NC}"
    echo "Fix the errors above before attempting deployment."
    exit 1
fi