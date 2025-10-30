# Infrastructure Deployment Guide

## Overview

This document provides step-by-step instructions for deploying the BuildBoard infrastructure to AWS environments. Follow these exact commands to ensure consistent deployments across different environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Deployment Commands](#deployment-commands)
- [Verification Steps](#verification-steps)
- [Troubleshooting](#troubleshooting)
- [Rollback Procedures](#rollback-procedures)

## Prerequisites

### Required Tools
- **Node.js** (v18 or later)
- **Go** (v1.21 or later) 
- **AWS CDK CLI** (v2.x)
- **AWS CLI** configured with profiles

### AWS Profiles Required
- `dev` - Development environment (Account: 521805123898)
- `prod` - Production environment (Account: 186375394147)

### Verify Prerequisites
```bash
# Check Node.js version
node --version

# Check Go version  
go version

# Check CDK version
npx cdk --version

# Verify AWS profiles
aws configure list-profiles

# Test AWS access
aws sts get-caller-identity --profile dev
aws sts get-caller-identity --profile prod
```

## Environment Setup

### Repository Structure
```
infrastructure/
├── bin/           # CDK app entry point
├── lib/           # CDK construct definitions
├── src/           # Lambda function source code
├── docs/          # Documentation
├── cdk.json       # CDK configuration
└── package.json   # Dependencies
```

### Working Directory
Always run deployment commands from the infrastructure root directory:
```bash
cd /Users/mayur/git_personal/infrastructure
```

## Deployment Commands

### 1. Build and Validate

#### Clean Build
```bash
# Install/update dependencies
npm ci

# Build TypeScript and Go code
npm run build

# Validate CDK synthesis
npx cdk synth
```

#### Quick Validation (Skip full build)
```bash
# Just validate CDK configuration
npx cdk list
```

### 2. Deploy to Development

#### Standard Dev Deployment
```bash
# Deploy to dev environment (Account: 521805123898)
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

#### Dev Deployment with Confirmation Skip
```bash
# Skip approval prompts (use with caution)
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev --require-approval never
```

### 3. Deploy to Production

#### Production Deployment (Manual Approval Required)
```bash
# Deploy to prod environment (Account: 186375394147) 
cd .. && npx cdk deploy "Infrastructure/Prod/Infrastructure-AppStage" --profile prod
```

#### Deploy All Environments
```bash
# Deploy to both dev and prod (will prompt for each)
cd .. && npx cdk deploy --all
```

## Stack Names and Identifiers

### CDK Stack Paths
- **Development:** `Infrastructure/Dev/Infrastructure-AppStage`
- **Production:** `Infrastructure/Prod/Infrastructure-AppStage`

### CloudFormation Stack Names
- **Development:** `Dev-Infrastructure-AppStage`
- **Production:** `Prod-Infrastructure-AppStage`

### AWS Account Mapping
- **Dev Environment:** 521805123898 (us-east-2)
- **Prod Environment:** 186375394147 (us-east-2)

## Verification Steps

### Post-Deployment Verification

#### 1. Check Stack Status
```bash
# Verify dev deployment
aws cloudformation describe-stacks --stack-name "Dev-Infrastructure-AppStage" --profile dev --region us-east-2

# Verify prod deployment  
aws cloudformation describe-stacks --stack-name "Prod-Infrastructure-AppStage" --profile prod --region us-east-2
```

#### 2. Test Lambda Functions
```bash
# List deployed functions (dev)
aws lambda list-functions --profile dev --region us-east-2 | grep "infrastructure"

# List deployed functions (prod)
aws lambda list-functions --profile prod --region us-east-2 | grep "infrastructure"
```

#### 3. Verify API Gateway
```bash
# Get API Gateway info (dev)
aws apigateway get-rest-apis --profile dev --region us-east-2

# Get API Gateway info (prod)
aws apigateway get-rest-apis --profile prod --region us-east-2
```

### Expected Lambda Functions
After successful deployment, these Lambda functions should exist:
- `infrastructure-api-gateway-cors`
- `infrastructure-token-customizer`
- `infrastructure-user-signup`
- `infrastructure-organization-management`
- `infrastructure-location-management`
- `infrastructure-roles-management`
- `infrastructure-permissions-management`
- `infrastructure-project-management`
- `infrastructure-user-management`
- `infrastructure-issue-management`
- `infrastructure-rfi-management`

## Troubleshooting

### Common Issues

#### 1. Build Errors
```bash
# Go compilation errors
Error: bash exited with status 1
```

**Solution:**
```bash
# Check Go syntax in Lambda functions
cd src/infrastructure-*
go build .

# Fix compilation errors before deploying
```

#### 2. AWS Profile Issues
```bash
# Profile not found or access denied
Error: The provided profile was not found or is not configured
```

**Solution:**
```bash
# Configure AWS profile
aws configure --profile dev
aws configure --profile prod

# Verify credentials
aws sts get-caller-identity --profile dev
```

#### 3. Permission Errors
```bash
# CDK bootstrap required
Error: Need to perform AWS CDK bootstrap
```

**Solution:**
```bash
# Bootstrap CDK in target accounts
npx cdk bootstrap --profile dev
npx cdk bootstrap --profile prod
```

#### 4. Stack Drift or Lock Issues
```bash
# Stack is in UPDATE_IN_PROGRESS state
Error: Stack is currently being updated
```

**Solution:**
```bash
# Wait for current operation to complete, or cancel if stuck
aws cloudformation cancel-update-stack --stack-name "Dev-Infrastructure-AppStage" --profile dev --region us-east-2
```

### Debugging Commands

#### View Stack Events
```bash
# Monitor deployment progress (dev)
aws cloudformation describe-stack-events --stack-name "Dev-Infrastructure-AppStage" --profile dev --region us-east-2 --max-items 20

# Monitor deployment progress (prod)
aws cloudformation describe-stack-events --stack-name "Prod-Infrastructure-AppStage" --profile prod --region us-east-2 --max-items 20
```

#### View Lambda Logs
```bash
# View recent Lambda logs
aws logs tail /aws/lambda/infrastructure-user-management --since 1h --profile dev --region us-east-2
```

## Rollback Procedures

### Emergency Rollback

#### 1. Rollback to Previous Version
```bash
# Checkout previous Git commit
git checkout HEAD~1

# Deploy previous version
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

#### 2. CloudFormation Rollback
```bash
# Cancel current update and rollback (if deployment is stuck)
aws cloudformation cancel-update-stack --stack-name "Dev-Infrastructure-AppStage" --profile dev --region us-east-2

# If stack is in failed state, continue rollback
aws cloudformation continue-update-rollback --stack-name "Dev-Infrastructure-AppStage" --profile dev --region us-east-2
```

### Safe Rollback Process
1. **Stop Traffic:** Update Route 53 to point away from affected environment
2. **Rollback Code:** Deploy previous working version
3. **Verify Function:** Test critical paths
4. **Restore Traffic:** Update Route 53 back to working environment
5. **Post-Mortem:** Document what went wrong and how to prevent it

## Best Practices

### Pre-Deployment Checklist
- [ ] Code changes committed and pushed to Git
- [ ] All tests passing locally
- [ ] Go code compiles without errors (`npm run build`)
- [ ] CDK synthesis successful (`npx cdk synth`)
- [ ] AWS credentials valid for target environment
- [ ] No ongoing deployments in target environment

### Deployment Workflow
1. **Always deploy to Dev first**
2. **Verify functionality in Dev** 
3. **Only then deploy to Prod**
4. **Monitor logs after each deployment**
5. **Keep deployment windows short**

### Security Considerations
- Use least-privilege IAM policies
- Rotate AWS access keys regularly  
- Never commit AWS credentials to Git
- Use AWS Secrets Manager for sensitive configuration
- Enable CloudTrail for audit logging

## Quick Reference

### Essential Commands
```bash
# Quick deployment to dev
cd /Users/mayur/git_personal/infrastructure
npm run build
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Quick deployment to prod
cd /Users/mayur/git_personal/infrastructure  
npm run build
cd .. && npx cdk deploy "Infrastructure/Prod/Infrastructure-AppStage" --profile prod

# Emergency rollback
git checkout HEAD~1
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

### Environment URLs
- **Dev API Base:** `https://api-dev.buildboard.com`
- **Prod API Base:** `https://api.buildboard.com`
- **Frontend Dev:** `http://localhost:3001` 
- **Frontend Prod:** `https://app.buildboard.com`

---

## Support

For deployment issues or questions:
1. Check this documentation first
2. Review CloudFormation stack events in AWS Console  
3. Check Lambda function logs in CloudWatch
4. Verify AWS permissions and profile configuration

**Note:** This deployment guide is specific to the BuildBoard infrastructure setup and should be updated whenever deployment procedures change.