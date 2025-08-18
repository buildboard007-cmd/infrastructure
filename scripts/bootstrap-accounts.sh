#!/bin/bash

# CDK Bootstrap Script for Multi-Account Setup
# This script bootstraps all three AWS accounts for CDK cross-account deployment

set -e

# Configuration
QUALIFIER="hnb659fds"  # You can change this to a unique value if desired
REGION="us-east-2"
TOOLS_ACCOUNT="401448503050"
DEV_ACCOUNT="521805123898"
PROD_ACCOUNT="186375394147"

echo "========================================="
echo "CDK Bootstrap for Multi-Account Pipeline"
echo "========================================="
echo "Qualifier: $QUALIFIER"
echo "Region: $REGION"
echo "Tools Account: $TOOLS_ACCOUNT"
echo "Dev Account: $DEV_ACCOUNT"
echo "Prod Account: $PROD_ACCOUNT"
echo "========================================="

# Step 1: Bootstrap Tools Account (Pipeline Account)
echo ""
echo "Step 1: Bootstrapping Tools Account ($TOOLS_ACCOUNT)..."
echo "This account will host the CDK Pipeline"
aws sts get-caller-identity --profile tools
npx cdk bootstrap aws://$TOOLS_ACCOUNT/$REGION \
  --profile tools \
  --qualifier $QUALIFIER \
  --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess

# Step 2: Bootstrap Dev Account with trust relationship to Tools account
echo ""
echo "Step 2: Bootstrapping Dev Account ($DEV_ACCOUNT)..."
echo "Setting up trust relationship with Tools account for cross-account deployment"
aws sts get-caller-identity --profile dev
npx cdk bootstrap aws://$DEV_ACCOUNT/$REGION \
  --profile dev \
  --qualifier $QUALIFIER \
  --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess \
  --trust $TOOLS_ACCOUNT

# Step 3: Bootstrap Prod Account with trust relationship to Tools account
echo ""
echo "Step 3: Bootstrapping Prod Account ($PROD_ACCOUNT)..."
echo "Setting up trust relationship with Tools account for cross-account deployment"
aws sts get-caller-identity --profile prod
npx cdk bootstrap aws://$PROD_ACCOUNT/$REGION \
  --profile prod \
  --qualifier $QUALIFIER \
  --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess \
  --trust $TOOLS_ACCOUNT

echo ""
echo "========================================="
echo "Bootstrap Complete!"
echo "========================================="
echo ""
echo "Bootstrap stacks created:"
echo "- Tools: CDKToolkit in account $TOOLS_ACCOUNT"
echo "- Dev: CDKToolkit in account $DEV_ACCOUNT (trusts Tools)"
echo "- Prod: CDKToolkit in account $PROD_ACCOUNT (trusts Tools)"
echo ""
echo "Next steps:"
echo "1. Verify/Create GitHub connection in Tools account"
echo "2. Deploy the pipeline: npm run deploy"