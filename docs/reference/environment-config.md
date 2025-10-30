# Environment Configuration Reference

> AWS account IDs, regions, API Gateway URLs, Cognito pool IDs, RDS endpoints, S3 buckets, and SSM parameters

---

## AWS Accounts

### Development Account
- **Account ID:** `521805123898`
- **Region:** `us-east-2` (Ohio)
- **Profile Name:** `dev`
- **Stage Name:** `Dev`

### Production Account
- **Account ID:** `186375394147`
- **Region:** `us-east-2` (Ohio)
- **Profile Name:** `prod`
- **Stage Name:** `Prod`

---

## API Gateway

### Development
- **API Gateway ID:** `74zc1md7sc`
- **Base URL:** `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
- **Stage Name:** `main`
- **REST API Name:** `Infrastructure-API-Dev`
- **Deployment Method:** `npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev`

### Production
- **Base URL:** TBD (not yet deployed)
- **Stage Name:** `main`
- **REST API Name:** `Infrastructure-API-Prod`

### API Endpoints

See [api-endpoints.md](./api-endpoints.md) for complete endpoint listing.

**Example Requests:**

```bash
# Get projects
curl -X GET \
  https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects \
  -H "Authorization: Bearer $ID_TOKEN"

# Create issue
curl -X POST \
  https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/issues \
  -H "Authorization: Bearer $ID_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"project_id": 1, "title": "Test Issue", ...}'
```

---

## AWS Cognito

### Development User Pool
- **User Pool ID:** `us-east-2_VkTLMp9RZ`
- **User Pool ARN:** `arn:aws:cognito-idp:us-east-2:521805123898:userpool/us-east-2_VkTLMp9RZ`
- **App Client ID:** `3f0fb5mpivctnvj85tucusf88e`
- **Region:** `us-east-2`

### Cognito Domain
- **Domain Prefix:** TBD
- **Hosted UI URL:** TBD

### Token Endpoints

```bash
# Authentication endpoint
POST https://cognito-idp.us-east-2.amazonaws.com/

# Token refresh endpoint
POST https://cognito-idp.us-east-2.amazonaws.com/
```

### Pre-Token Generation Lambda
- **Trigger Type:** Pre Token Generation V2.0
- **Lambda Function:** `infrastructure-token-customizer`
- **Purpose:** Inject custom claims (user_id, org_id, locations, etc.) into JWT tokens

### Post Confirmation Lambda
- **Trigger Type:** Post Confirmation
- **Lambda Function:** `infrastructure-user-signup`
- **Purpose:** Create user record in `iam.users` table after Cognito signup

---

## RDS PostgreSQL

### Development Database
- **Endpoint:** `appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com`
- **Port:** `5432`
- **Database Name:** `appdb`
- **Admin User:** `appdb_admin`
- **SSL Mode:** `require`
- **Schemas:** `iam`, `project`

### Connection String Format

```bash
# Via psql
psql "host=appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com \
      port=5432 \
      dbname=appdb \
      user=appdb_admin \
      sslmode=require"

# Connection URL
postgresql://appdb_admin:PASSWORD@appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com:5432/appdb?sslmode=require
```

### Go Connection (via Lambda)

```go
import "infrastructure/lib/clients"

sqlDB, err := clients.NewPostgresSQLClient(
    ssmParams[constants.DATABASE_RDS_ENDPOINT],
    ssmParams[constants.DATABASE_PORT],
    ssmParams[constants.DATABASE_NAME],
    ssmParams[constants.DATABASE_USERNAME],
    ssmParams[constants.DATABASE_PASSWORD],
    ssmParams[constants.SSL_MODE],
)
```

---

## S3 Buckets

### Attachments Bucket (Development)
- **Bucket Name:** `buildboard-attachments-dev`
- **Region:** `us-east-2`
- **Purpose:** Store all file attachments (issues, RFIs, submittals, projects)
- **Lifecycle Policy:** TBD

### S3 Key Structure

```
# Issue attachments
issues/{issue_id}/{timestamp}_{filename}

# Issue comment attachments
issue_comments/{comment_id}/{timestamp}_{filename}

# RFI attachments
rfis/{rfi_id}/{timestamp}_{filename}

# Submittal attachments
submittals/{submittal_id}/{timestamp}_{filename}

# Project attachments
projects/{project_id}/{timestamp}_{filename}
```

### Pre-signed URL Generation

```go
import "infrastructure/lib/clients"

s3Client := clients.NewS3Client(isLocal)

// Upload URL (PUT)
uploadURL, err := s3Client.GeneratePresignedPutURL(bucketName, s3Key, 15*time.Minute)

// Download URL (GET)
downloadURL, err := s3Client.GeneratePresignedGetURL(bucketName, s3Key, 15*time.Minute)
```

---

## SSM Parameter Store

All sensitive configuration is stored in AWS Systems Manager Parameter Store.

### Parameter Paths (Development)

| Parameter Name | Path | Description | Type |
|----------------|------|-------------|------|
| `DATABASE_RDS_ENDPOINT` | `/infrastructure/dev/database/rds-endpoint` | RDS endpoint URL | String |
| `DATABASE_PORT` | `/infrastructure/dev/database/port` | Database port (5432) | String |
| `DATABASE_NAME` | `/infrastructure/dev/database/name` | Database name (appdb) | String |
| `DATABASE_USERNAME` | `/infrastructure/dev/database/username` | Database admin username | String |
| `DATABASE_PASSWORD` | `/infrastructure/dev/database/password` | Database password | SecureString |
| `SSL_MODE` | `/infrastructure/dev/database/ssl-mode` | SSL mode (require) | String |
| `COGNITO_USER_POOL_ID` | `/infrastructure/dev/cognito/user-pool-id` | Cognito User Pool ID | String |
| `COGNITO_CLIENT_ID` | `/infrastructure/dev/cognito/client-id` | Cognito App Client ID | String |
| `S3_BUCKET_ATTACHMENTS` | `/infrastructure/dev/s3/attachments-bucket` | S3 attachments bucket name | String |

### Accessing SSM Parameters in Lambda

```go
package data

import (
    "context"
    "infrastructure/lib/clients"
    "infrastructure/lib/constants"
)

// Get all SSM parameters
ssmClient := clients.NewSSMClient(isLocal)
ssmRepository := &SSMDao{
    SSM:    ssmClient,
    Logger: logger,
}

params, err := ssmRepository.GetParameters()
if err != nil {
    logger.Fatal("Failed to get SSM parameters")
}

// Access specific parameters
rdsEndpoint := params[constants.DATABASE_RDS_ENDPOINT]
dbPort := params[constants.DATABASE_PORT]
cognitoPoolID := params[constants.COGNITO_USER_POOL_ID]
```

### Adding New Parameters

```bash
# Via AWS CLI
aws ssm put-parameter \
  --name "/infrastructure/dev/new-param" \
  --value "param-value" \
  --type "String" \
  --profile dev

# For secure strings
aws ssm put-parameter \
  --name "/infrastructure/dev/secure-param" \
  --value "secret-value" \
  --type "SecureString" \
  --profile dev
```

---

## Lambda Functions

### Development Deployment

All Lambda functions are deployed as part of the CDK stack.

| Function Name | Handler | Runtime | Timeout | Memory | Purpose |
|---------------|---------|---------|---------|--------|---------|
| `infrastructure-organization-management` | Bootstrap | Go 1.x (provided.al2) | 30s | 512 MB | Organization CRUD |
| `infrastructure-location-management` | Bootstrap | Go 1.x | 30s | 512 MB | Location CRUD |
| `infrastructure-role-management` | Bootstrap | Go 1.x | 30s | 512 MB | Role CRUD |
| `infrastructure-permission-management` | Bootstrap | Go 1.x | 30s | 512 MB | Permission CRUD |
| `infrastructure-user-management` | Bootstrap | Go 1.x | 30s | 512 MB | User CRUD + Cognito |
| `infrastructure-assignment-management` | Bootstrap | Go 1.x | 30s | 512 MB | Assignment CRUD |
| `infrastructure-project-management` | Bootstrap | Go 1.x | 30s | 512 MB | Project CRUD |
| `infrastructure-issue-management` | Bootstrap | Go 1.x | 30s | 512 MB | Issue CRUD + comments |
| `infrastructure-rfi-management` | Bootstrap | Go 1.x | 30s | 512 MB | RFI CRUD + workflow |
| `infrastructure-submittal-management` | Bootstrap | Go 1.x | 30s | 512 MB | Submittal CRUD + workflow |
| `infrastructure-attachment-management` | Bootstrap | Go 1.x | 30s | 512 MB | Centralized file handling |
| `infrastructure-token-customizer` | Bootstrap | Go 1.x | 10s | 256 MB | Cognito token customization |
| `infrastructure-user-signup` | Bootstrap | Go 1.x | 10s | 256 MB | Post-signup user creation |

### Lambda Environment Variables

Each Lambda function has the following environment variables:

```bash
IS_LOCAL=false
LOG_LEVEL=ERROR  # or DEBUG for verbose logging
STAGE=Dev
```

### IAM Roles

Lambda functions have IAM roles with permissions for:
- CloudWatch Logs (write logs)
- SSM Parameter Store (read parameters)
- RDS Data API (database access)
- S3 (generate pre-signed URLs)
- Cognito (user management)
- VPC (database connectivity)

---

## VPC Configuration

### Development VPC
- **VPC ID:** TBD
- **Subnets:** Private subnets for Lambda and RDS
- **Security Groups:**
  - Lambda SG: Outbound to RDS
  - RDS SG: Inbound from Lambda SG on port 5432

---

## Test Users

### Primary Test User (Super Admin)
- **Email:** `buildboard007+555@gmail.com`
- **Password:** `Mayur@1234`
- **User ID:** `19`
- **Organization ID:** `10`
- **Is Super Admin:** `true`
- **Status:** `active`

### Additional Test Users
Create via `/users` endpoint or Cognito console.

---

## CDK Deployment

### Build and Deploy Commands

```bash
# Navigate to infrastructure directory
cd /Users/mayur/git_personal/infrastructure

# Install dependencies
npm install

# Build TypeScript CDK code
npm run build

# Deploy to Dev
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Deploy specific stack
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage/InfrastructureStack" --profile dev

# List all stacks
npx cdk list --profile dev

# Diff changes before deploy
npx cdk diff --profile dev

# Destroy stack (careful!)
npx cdk destroy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

### CDK Context Values

Located in `cdk.json`:

```json
{
  "apiName": "Infrastructure-API",
  "apiStageName": "main",
  "stages": [
    {
      "name": "Dev",
      "account": "521805123898",
      "region": "us-east-2"
    },
    {
      "name": "Prod",
      "account": "186375394147",
      "region": "us-east-2"
    }
  ]
}
```

---

## Local Development

### Environment Setup

```bash
# Set AWS profile
export AWS_PROFILE=dev

# Set local flag for Lambda functions
export IS_LOCAL=true

# Set debug logging
export LOG_LEVEL=DEBUG
```

### Running Lambda Locally

```bash
# Build Go binary
cd src/infrastructure-project-management
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go

# Test with SAM CLI (if configured)
sam local invoke -e test-event.json
```

### Testing API Locally

Use Postman collections in `/postman` directory.

---

## Monitoring & Logging

### CloudWatch Log Groups

Each Lambda function has its own log group:

```
/aws/lambda/infrastructure-organization-management
/aws/lambda/infrastructure-location-management
/aws/lambda/infrastructure-role-management
/aws/lambda/infrastructure-permission-management
/aws/lambda/infrastructure-user-management
/aws/lambda/infrastructure-assignment-management
/aws/lambda/infrastructure-project-management
/aws/lambda/infrastructure-issue-management
/aws/lambda/infrastructure-rfi-management
/aws/lambda/infrastructure-submittal-management
/aws/lambda/infrastructure-attachment-management
/aws/lambda/infrastructure-token-customizer
/aws/lambda/infrastructure-user-signup
```

### Viewing Logs

```bash
# Via AWS CLI
aws logs tail /aws/lambda/infrastructure-project-management \
  --follow \
  --profile dev

# Filter for errors
aws logs filter-events \
  --log-group-name /aws/lambda/infrastructure-project-management \
  --filter-pattern "ERROR" \
  --profile dev
```

### CloudWatch Metrics

- Lambda invocations
- Lambda errors
- Lambda duration
- API Gateway 4xx/5xx errors
- API Gateway latency

---

## Security

### IAM Policies
- Least privilege principle
- Lambda execution roles scoped per function
- RDS access limited to Lambda security group
- S3 bucket policies restrict access to Lambda roles only

### Encryption
- **RDS:** Encrypted at rest with AWS-managed keys
- **S3:** Server-side encryption (SSE-S3)
- **SSM:** SecureString parameters use KMS encryption
- **Cognito:** Tokens are JWT signed with RS256

### Network Security
- Lambda functions in private VPC subnets
- RDS in private subnets, no public access
- API Gateway with Cognito authorizer (no API keys)

---

## Frontend Configuration

### React Environment Variables

```bash
# .env.development
REACT_APP_API_BASE_URL=https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
REACT_APP_COGNITO_USER_POOL_ID=us-east-2_VkTLMp9RZ
REACT_APP_COGNITO_CLIENT_ID=3f0fb5mpivctnvj85tucusf88e
REACT_APP_COGNITO_REGION=us-east-2
REACT_APP_STAGE=dev
```

### Frontend Location

```bash
# Frontend repository
/Users/mayur/git_personal/ui/frontend
```

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| API returns 502 Bad Gateway | Lambda timeout or crash | Check CloudWatch logs for errors |
| Database connection timeout | VPC/security group misconfiguration | Verify Lambda and RDS in same VPC |
| SSM parameter not found | Parameter doesn't exist | Check parameter name and path |
| S3 access denied | IAM role missing permissions | Update Lambda execution role |
| Cognito token expired | Token older than 1 hour | Refresh token |

### Debug Checklist

1. ✅ Check CloudWatch logs for Lambda errors
2. ✅ Verify SSM parameters are set correctly
3. ✅ Confirm database is accessible from Lambda
4. ✅ Check IAM role has necessary permissions
5. ✅ Validate request body matches expected format
6. ✅ Ensure JWT token is valid and not expired
7. ✅ Verify user has correct assignments in database

---

**Last Updated:** 2025-10-27