#!/bin/bash

# Pre-push validation script
# This catches most errors before deploying to AWS

set -e

echo "========================================"
echo "Pre-Push Validation for AWS CodePipeline"
echo "========================================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ERRORS=0

check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
    else
        echo -e "${RED}✗${NC} $1"
        ERRORS=$((ERRORS + 1))
    fi
}

echo "1. TypeScript compilation..."
npx tsc --noEmit > /dev/null 2>&1
check_status "TypeScript compiles without errors"

echo ""
echo "2. CDK Synthesis (Main Pipeline)..."
npx cdk synth --app "npx ts-node bin/app.ts" --quiet > /dev/null 2>&1
check_status "CDK pipeline synthesizes successfully"

echo ""
echo "3. Go modules verification..."
cd src && go mod verify > /dev/null 2>&1
check_status "Go modules are valid"
cd ..

echo ""
echo "4. Go compilation..."
cd src/infrastructure-api-gateway-cors && go build . > /dev/null 2>&1
check_status "Lambda functions compile"
cd ../..

echo ""
echo "5. Go tests..."
cd src && go test ./... > /dev/null 2>&1
check_status "All Go tests pass"
cd ..

echo ""
echo "6. CDK Context validation..."
npx cdk context --clear > /dev/null 2>&1 && npx cdk synth --app "npx ts-node bin/app.ts" --quiet > /dev/null 2>&1
check_status "CDK synthesis with fresh context"

echo ""
echo "========================================"
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}All validations passed!${NC}"
    echo ""
    echo "✅ Safe to push to AWS CodePipeline"
    echo ""
    echo "Next steps:"
    echo "1. git add ."
    echo "2. git commit -m 'Your message'"
    echo "3. git push origin main"
else
    echo -e "${RED}$ERRORS validation(s) failed!${NC}"
    echo ""
    echo "❌ Fix errors before pushing to AWS"
    exit 1
fi