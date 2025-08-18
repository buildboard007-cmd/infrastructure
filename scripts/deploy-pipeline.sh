#!/bin/bash

# Deploy CDK Pipeline to Tools Account

set -e

echo "========================================="
echo "Deploying CDK Pipeline to Tools Account"
echo "========================================="

# Verify we're using the tools profile
echo "Checking AWS credentials..."
aws sts get-caller-identity --profile tools

# Build the project
echo "Building the project..."
npm run build

# Synthesize the CDK app
echo "Synthesizing CDK app..."
npx cdk synth --profile tools

# Deploy the pipeline stack to tools account
echo "Deploying pipeline to Tools account..."
npx cdk deploy --profile tools --require-approval never

echo ""
echo "========================================="
echo "Pipeline Deployment Complete!"
echo "========================================="
echo ""
echo "Pipeline deployed to Tools account (401448503050)"
echo "The pipeline will automatically deploy to:"
echo "- Dev account (521805123898)"
echo "- Prod account (186375394147) - with manual approval"
echo ""
echo "Check the pipeline status in AWS CodePipeline console"