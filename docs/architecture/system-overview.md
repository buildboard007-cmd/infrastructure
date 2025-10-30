# System Overview - BuildBoard Infrastructure

**Last Updated:** 2025-10-27
**Purpose:** High-level architectural overview of the BuildBoard construction management system

---

## Table of Contents
1. [Architecture Philosophy](#architecture-philosophy)
2. [Technology Stack](#technology-stack)
3. [AWS Infrastructure](#aws-infrastructure)
4. [System Architecture](#system-architecture)
5. [Multi-Tenant Design](#multi-tenant-design)
6. [Deployment Topology](#deployment-topology)
7. [System Boundaries](#system-boundaries)
8. [Key Architectural Decisions](#key-architectural-decisions)

---

## Architecture Philosophy

BuildBoard is built on three core architectural principles:

### 1. Serverless-First
- **Event-driven Lambda functions** for compute
- **Pay-per-use pricing model** - no idle server costs
- **Automatic scaling** - handles traffic spikes without manual intervention
- **Reduced operational overhead** - no server maintenance or patching

### 2. Microservices Pattern
- **Domain-driven design** - each Lambda function represents a bounded context
- **Independent deployment** - services can be updated without affecting others
- **Loose coupling** - services communicate via API Gateway and shared database
- **Single responsibility** - each service handles one business domain

### 3. Infrastructure as Code
- **AWS CDK with TypeScript** - type-safe infrastructure definitions
- **Repeatable deployments** - same infrastructure across Dev/Prod
- **Version controlled** - infrastructure changes tracked in Git
- **Automated CI/CD** - deployments through AWS CodePipeline

---

## Technology Stack

### Backend
**Language:** Go (Golang) 1.21+

**Why Go?**
- **Performance** - compiled language with minimal overhead, ideal for Lambda cold starts
- **Concurrency** - built-in goroutines for efficient concurrent operations
- **Type safety** - strong typing prevents runtime errors
- **Small binary size** - fast Lambda deployments and reduced package size
- **Standard library** - excellent built-in HTTP, JSON, and database support
- **Cross-compilation** - easy to build for AWS Lambda's Linux environment

**Architecture Patterns:**
- Repository pattern for data access abstraction
- Dependency injection for testability
- Interface-based design for loose coupling

### Infrastructure
**AWS CDK (Cloud Development Kit) with TypeScript**

**Why CDK over CloudFormation/Terraform?**
- **Type safety** - IDE autocomplete and compile-time checks
- **Programming constructs** - loops, conditionals, functions for DRY code
- **AWS best practices** - built-in security and architecture patterns
- **L2/L3 constructs** - higher-level abstractions reduce boilerplate
- **Direct AWS integration** - official AWS framework, always up-to-date
- **Synthesizes to CloudFormation** - benefits of CloudFormation state management

### Database
**PostgreSQL (AWS RDS)**

**Why PostgreSQL?**
- **ACID compliance** - critical for construction data integrity
- **Rich data types** - JSONB for flexible metadata, arrays, enums
- **Advanced indexing** - GiST, GIN indexes for complex queries
- **Schema support** - logical separation of IAM and project data
- **Mature ecosystem** - proven reliability in production systems
- **Open source** - no vendor lock-in

### Authentication
**AWS Cognito**

**Why Cognito?**
- **Managed service** - no authentication infrastructure to maintain
- **OAuth 2.0 / OIDC** - industry-standard protocols
- **JWT tokens** - stateless authentication for serverless
- **Token customization** - Lambda triggers for custom claims
- **MFA support** - built-in multi-factor authentication
- **Integration with API Gateway** - seamless authorization

### API Layer
**AWS API Gateway**

- RESTful API design
- Cognito authorizer for JWT validation
- Request/response transformation
- CORS handling
- Rate limiting and throttling

### Storage
**Amazon S3**

- Document storage (attachments, photos)
- Static asset hosting
- Versioning for audit trail
- Lifecycle policies for cost optimization

### Frontend
**React (Separate Repository)**

- Located at `/Users/mayur/git_personal/ui/frontend`
- Single Page Application (SPA)
- Consumes REST APIs
- Hosted separately from backend

---

## AWS Infrastructure

### Account Structure

**Development Account**
- Account ID: `521805123898`
- Purpose: Development and testing
- Access: Development team
- Cost optimization: Auto-shutdown of non-production resources

**Production Account**
- Account ID: `186375394147`
- Purpose: Production workloads
- Access: Restricted, audit logged
- High availability configuration

### Regional Deployment
- **Primary Region:** `us-east-2` (Ohio)
- Proximity to target users
- Cost-effective data transfer
- Full service availability

### Infrastructure Components

#### Compute
**AWS Lambda Functions (14 services)**
- `infrastructure-api-gateway-cors` - CORS preflight handler
- `infrastructure-token-customizer` - JWT enrichment with user data
- `infrastructure-user-signup` - User registration handler
- `infrastructure-user-management` - User CRUD operations
- `infrastructure-organization-management` - Organization management
- `infrastructure-location-management` - Location/site management
- `infrastructure-roles-management` - Role definitions
- `infrastructure-permissions-management` - Permission management
- `infrastructure-assignment-management` - User role assignments
- `infrastructure-project-management` - Project lifecycle
- `infrastructure-issue-management` - Issue tracking
- `infrastructure-rfi-management` - RFI (Request for Information) management
- `infrastructure-submittal-management` - Submittal approval workflow
- `infrastructure-attachment-management` - Centralized file management

**Lambda Configuration:**
- Runtime: Go 1.x (custom runtime)
- Memory: 256MB - 512MB per function
- Timeout: 30 seconds (adjustable per function)
- Environment variables loaded from SSM Parameter Store
- VPC configuration for database access

#### API Gateway
- **Base URL (Dev):** `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
- REST API type
- Cognito authorizer for all protected endpoints
- Request validation
- CloudWatch logging enabled

#### Database
- **Host:** `appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com`
- **Engine:** PostgreSQL 14+
- **Instance Class:** db.t3.micro (Dev), db.r5.large (Prod)
- **Storage:** 100GB GP3, auto-scaling enabled
- **Backup:** Automated daily backups, 7-day retention
- **Multi-AZ:** Enabled in Production for high availability
- **Encryption:** At-rest and in-transit encryption enabled

#### Authentication
- **Cognito User Pool:** Primary authentication
- **Client ID:** `3f0fb5mpivctnvj85tucusf88e`
- **Identity Pool:** For temporary AWS credentials
- **Token expiration:**
  - ID Token: 1 hour
  - Refresh Token: 30 days

#### Storage
- **S3 Bucket:** Per-environment buckets
  - `buildboard-dev-attachments`
  - `buildboard-prod-attachments`
- **Versioning:** Enabled
- **Lifecycle:** Archive to Glacier after 90 days
- **Encryption:** SSE-S3 default encryption

#### Secrets Management
- **AWS Systems Manager Parameter Store**
- Database credentials
- API keys
- Configuration parameters
- Automatic rotation support

---

## System Architecture

### Serverless Multi-Service Architecture

```
                                    ┌─────────────────┐
                                    │                 │
                                    │  React Frontend │
                                    │    (SPA)        │
                                    │                 │
                                    └────────┬────────┘
                                             │
                                             │ HTTPS
                                             ▼
                                    ┌─────────────────┐
                                    │                 │
                                    │  API Gateway    │
                                    │  (REST API)     │
                                    │                 │
                                    └────────┬────────┘
                                             │
                          ┌──────────────────┼──────────────────┐
                          │                  │                  │
                          ▼                  ▼                  ▼
                    ┌──────────┐      ┌──────────┐      ┌──────────┐
                    │ Cognito  │      │  Lambda  │      │  Lambda  │
                    │Authorizer│      │ Function │      │ Function │
                    └──────────┘      └─────┬────┘      └─────┬────┘
                                            │                  │
                          ┌─────────────────┴──────────────────┘
                          │
                          ▼
                    ┌──────────────┐
                    │              │
                    │  PostgreSQL  │
                    │     RDS      │
                    │              │
                    └──────────────┘
                          │
                          ▼
                    ┌──────────────┐
                    │      S3      │
                    │  Attachments │
                    └──────────────┘
```

### Request Flow

1. **User Authentication**
   - User logs in via React frontend
   - Cognito validates credentials
   - Returns ID Token and Refresh Token
   - Frontend stores tokens securely

2. **API Request**
   - Frontend sends request with ID Token in Authorization header
   - API Gateway validates token with Cognito Authorizer
   - Token Customizer Lambda enriches token with user metadata
   - Request routed to appropriate Lambda function

3. **Business Logic Execution**
   - Lambda function extracts JWT claims (user_id, org_id, roles)
   - Validates user permissions and access level
   - Executes business logic via repository pattern
   - Database queries with row-level security

4. **Response**
   - Lambda returns JSON response
   - API Gateway applies response transformations
   - Frontend receives and renders data

### Lambda Function Architecture

Each Lambda function follows this structure:

```go
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
)

// Shared libraries
var (
    dbClient    *sql.DB
    repository  Repository
    logger      *Logger
)

// Initialize connections (reused across invocations)
func init() {
    dbClient = initDatabaseConnection()
    repository = NewRepository(dbClient)
    logger = NewLogger()
}

// Handler function
func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Extract JWT claims
    claims := auth.ExtractClaimsFromRequest(request)

    // Route to appropriate function
    switch request.HTTPMethod {
        case "GET": return handleGet(ctx, request, claims)
        case "POST": return handlePost(ctx, request, claims)
        case "PUT": return handlePut(ctx, request, claims)
        case "DELETE": return handleDelete(ctx, request, claims)
    }
}

func main() {
    lambda.Start(handler)
}
```

### Repository Pattern

All data access goes through repository interfaces:

```go
type ProjectRepository interface {
    CreateProject(ctx, orgID, request, userID) (*Project, error)
    GetProjectByID(ctx, projectID, orgID) (*Project, error)
    GetProjectsByLocationID(ctx, locationID, orgID) ([]Project, error)
    UpdateProject(ctx, projectID, request, userID) (*Project, error)
    DeleteProject(ctx, projectID, userID) error
}

// Implementation uses database/sql with prepared statements
// Handles connection pooling, error handling, logging
```

**Benefits:**
- Testability (mock repositories for unit tests)
- Consistent error handling
- Transaction management
- SQL injection prevention
- Performance optimization (prepared statements)

---

## Multi-Tenant Design

### Tenant Isolation

**Organization-Based Multi-Tenancy:**
- Each customer is an `organization` (org_id)
- All data tagged with org_id
- Row-level security enforced in queries
- No shared data between organizations

### Data Isolation Strategy

1. **Database Level**
   - All queries filtered by org_id
   - Foreign keys enforce referential integrity within org
   - Indexes on org_id for query performance

2. **Application Level**
   - JWT claims include org_id
   - All API requests validate org_id match
   - Repository methods always include org_id parameter

3. **Access Control**
   ```go
   // Example: User can only access data in their org
   if claims.OrgID != project.OrgID {
       return ErrorResponse(403, "Access denied")
   }
   ```

### Hierarchical Data Model

```
Organization (org_id: 1)
    ├── Users (org_id: 1)
    ├── Roles (org_id: 1)
    ├── Locations (org_id: 1)
    │   ├── Location A
    │   └── Location B
    │       ├── Projects (org_id: 1, location_id: B)
    │       │   ├── Project 1
    │       │   │   ├── Issues
    │       │   │   ├── RFIs
    │       │   │   ├── Submittals
    │       │   │   └── Attachments
    │       │   └── Project 2
    │       └── User Assignments
```

### Tenant-Aware Queries

All database queries include org_id:

```sql
-- Good: Tenant-isolated query
SELECT * FROM project.projects
WHERE org_id = $1 AND location_id = $2 AND is_deleted = FALSE;

-- Bad: Missing org_id (would expose data across tenants)
SELECT * FROM project.projects
WHERE location_id = $1 AND is_deleted = FALSE;
```

---

## Deployment Topology

### Environment Segregation

**Development Environment**
- AWS Account: 521805123898
- Branch: `develop`
- Deployment: Manual via CDK
- Database: Shared dev database
- Purpose: Active development and testing

**Production Environment**
- AWS Account: 186375394147
- Branch: `main`
- Deployment: Automated via CodePipeline
- Database: Isolated production database
- Purpose: Customer-facing workloads

### CI/CD Pipeline

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│              │      │              │      │              │
│  Git Push    │─────▶│  CodeBuild   │─────▶│  CodeDeploy  │
│  to main     │      │  (Build)     │      │  (Deploy)    │
│              │      │              │      │              │
└──────────────┘      └──────────────┘      └──────────────┘
                             │
                             ├─ npm run build (CDK synth)
                             ├─ go build (compile Lambda functions)
                             ├─ Run tests
                             └─ Create CloudFormation templates
```

**Pipeline Stages:**
1. **Source** - GitHub repository trigger
2. **Build** - Compile Go functions, synthesize CDK
3. **Test** - Run unit and integration tests
4. **Deploy to Staging** - Deploy to staging environment
5. **Manual Approval** - Human approval gate
6. **Deploy to Production** - Deploy to production

### Deployment Commands

**Manual Deployment (Dev):**
```bash
# Build TypeScript infrastructure code
npm run build

# Deploy to development
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev

# Deploy specific stack
npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage/ProjectManagement" --profile dev
```

**Rollback:**
```bash
# CloudFormation automatic rollback on failure
# Manual rollback via AWS Console or CLI
aws cloudformation rollback-stack --stack-name <stack-name>
```

### Infrastructure Updates

All infrastructure changes via CDK:
1. Modify code in `/lib/` directory
2. Run `npm run build` to compile TypeScript
3. Run `npx cdk diff` to preview changes
4. Deploy with `npx cdk deploy`
5. CloudFormation manages state and updates

**CDK Stack Structure:**
```typescript
// Main stack
InfrastructureStack
  ├── NetworkStack (VPC, subnets, security groups)
  ├── DatabaseStack (RDS instance)
  ├── CognitoStack (User pool, identity pool)
  ├── ApiGatewayStack (API Gateway, authorizers)
  └── LambdaStack (14 Lambda functions)
```

---

## System Boundaries

### Frontend/Backend Separation

**Frontend (React SPA)**
- **Responsibilities:**
  - User interface rendering
  - Client-side routing
  - Form validation
  - State management
  - Token storage and refresh
- **Does NOT:**
  - Access database directly
  - Contain business logic
  - Store sensitive data

**Backend (Lambda + API Gateway)**
- **Responsibilities:**
  - Business logic enforcement
  - Data validation
  - Authorization checks
  - Database transactions
  - File storage management
- **Does NOT:**
  - Render UI
  - Store session state (stateless)

### Service Boundaries

Each Lambda function owns its domain:

| Lambda Function | Domain | Data Ownership |
|----------------|--------|----------------|
| user-management | User CRUD | iam.users |
| organization-management | Org CRUD | iam.organizations |
| location-management | Location CRUD | iam.locations |
| roles-management | Role definitions | iam.roles |
| permissions-management | Permission system | iam.permissions |
| assignment-management | User assignments | iam.user_assignments |
| project-management | Project lifecycle | project.projects |
| issue-management | Issue tracking | project.issues |
| rfi-management | RFI workflow | project.rfis |
| submittal-management | Submittal approval | project.submittals |
| attachment-management | File storage | project.attachments (cross-domain) |

**Cross-Cutting Concerns:**
- Authentication: Cognito + token-customizer
- Authorization: JWT claims in each Lambda
- Logging: CloudWatch Logs
- Monitoring: CloudWatch Metrics
- Error handling: Standardized error responses

### External Integrations

**Current:**
- AWS Cognito (authentication)
- AWS S3 (file storage)
- AWS SES (email notifications - planned)

**Future Planned:**
- Procore API integration
- Autodesk Construction Cloud
- Document generation services
- Mobile push notifications

---

## Key Architectural Decisions

### 1. Serverless vs. Container-Based

**Decision:** Serverless (Lambda)

**Rationale:**
- Construction management has variable traffic patterns
- No need for persistent connections (WebSockets)
- Cost efficiency: pay only for actual usage
- Zero server maintenance overhead
- Automatic scaling for concurrent requests
- Fast iteration and deployment cycles

**Trade-offs:**
- Cold start latency (mitigated with provisioned concurrency for critical functions)
- 15-minute execution limit (not an issue for API operations)
- Stateless design required (beneficial for scalability)

### 2. Monolithic Database vs. Database-per-Service

**Decision:** Monolithic PostgreSQL with schema separation

**Rationale:**
- ACID transactions across related entities (project + issues + RFIs)
- Easier data consistency and referential integrity
- Simpler deployment and maintenance
- Cross-domain queries when needed (reporting)
- Logical separation via schemas (iam vs project)

**Trade-offs:**
- Shared database is a coupling point
- Requires coordination for schema changes
- Single point of failure (mitigated with RDS Multi-AZ)

### 3. REST vs. GraphQL

**Decision:** REST API

**Rationale:**
- Simpler to implement and maintain
- Better caching support (HTTP caching)
- API Gateway native support
- Easier for third-party integrations
- Construction industry familiarity with REST

**Trade-offs:**
- Multiple requests for related data (N+1 queries)
- Over-fetching or under-fetching data
- No built-in schema introspection

### 4. Cognito vs. Custom Authentication

**Decision:** AWS Cognito

**Rationale:**
- Managed service: no security updates to maintain
- Compliance certifications (SOC, ISO)
- Built-in features: MFA, password policies, account recovery
- Native API Gateway integration
- OAuth/OIDC standards compliance
- Cost-effective (pay per active user)

**Trade-offs:**
- Less customization flexibility
- AWS vendor lock-in
- Learning curve for token customization

### 5. CDK vs. CloudFormation vs. Terraform

**Decision:** AWS CDK with TypeScript

**Rationale:**
- Type safety prevents configuration errors
- Reusable constructs (DRY principle)
- Programming language features (loops, conditions)
- Built-in best practices and security defaults
- Official AWS support and updates
- Generates CloudFormation (benefits of both)

**Trade-offs:**
- AWS-specific (not multi-cloud)
- Additional build step (TypeScript compilation)
- Steeper learning curve than YAML

### 6. Go vs. Node.js vs. Python for Lambda

**Decision:** Go (Golang)

**Rationale:**
- **Performance:** Compiled language, fast cold starts
- **Type safety:** Catch errors at compile time
- **Concurrency:** Built-in goroutines for parallel operations
- **Small package size:** Faster deployments
- **Standard library:** Excellent HTTP, JSON, database support
- **Team expertise:** Backend team proficiency

**Trade-offs:**
- More verbose than Python
- Smaller ecosystem than Node.js
- Longer development time for complex features

### 7. Soft Deletes vs. Hard Deletes

**Decision:** Soft deletes (is_deleted flag)

**Rationale:**
- Audit trail requirements
- Data recovery capability
- Referential integrity preservation
- Compliance with data retention policies
- Undo operations support

**Trade-offs:**
- Queries must always filter is_deleted = FALSE
- Increased storage requirements
- More complex unique constraints

### 8. Location-First UI Pattern

**Decision:** Users must select location before viewing projects

**Rationale:**
- Matches real-world workflow (users work at one site at a time)
- Simplifies access control logic
- Reduces query complexity
- Improves UI performance (smaller data sets)
- Clear user context for operations

**Trade-offs:**
- Extra click for org-level users
- Cannot view projects across locations simultaneously
- Dashboard aggregation more complex

---

## Performance Considerations

### Lambda Optimization
- Connection pooling in init() function
- Reuse database connections across invocations
- Prepared statements for repeated queries
- Minimal dependencies to reduce cold start time

### Database Optimization
- Comprehensive indexing strategy (see database-schema.md)
- Read replicas for reporting queries (production)
- Connection pooling via RDS Proxy
- Query timeout limits to prevent runaway queries

### API Gateway
- Response caching for GET requests
- Request throttling to prevent abuse
- Payload compression enabled

### S3 Storage
- CloudFront CDN for attachment delivery (production)
- Presigned URLs for direct upload/download
- Lifecycle policies for cost optimization

---

## Security Architecture

### Defense in Depth

1. **Network Layer**
   - VPC isolation for database
   - Security groups restrict access
   - Private subnets for RDS

2. **Authentication Layer**
   - Cognito JWT validation
   - Token expiration (1 hour)
   - Refresh token rotation

3. **Authorization Layer**
   - JWT claims validation
   - Role-based access control
   - Org-level data isolation

4. **Data Layer**
   - Encryption at rest (RDS, S3)
   - Encryption in transit (TLS)
   - Database connection SSL required

5. **Application Layer**
   - Input validation
   - SQL injection prevention (prepared statements)
   - XSS prevention (output encoding)

### Secrets Management
- No credentials in code
- SSM Parameter Store for secrets
- IAM roles for Lambda execution
- Least privilege principle

---

## Monitoring and Observability

### Logging
- CloudWatch Logs per Lambda function
- Structured logging (JSON format)
- Log retention: 30 days (dev), 90 days (prod)

### Metrics
- Lambda execution duration
- Error rates and retries
- Database connection pool utilization
- API Gateway request counts

### Alerting
- CloudWatch Alarms for error thresholds
- SNS notifications to on-call team
- Database CPU/memory alerts

### Tracing
- AWS X-Ray integration (planned)
- Request ID propagation
- End-to-end request tracking

---

## Disaster Recovery

### Backup Strategy
- **Database:** Automated daily backups, 7-day retention
- **Point-in-time recovery:** 5-minute RPO
- **S3 Attachments:** Versioning enabled, cross-region replication (prod)

### Recovery Objectives
- **RTO (Recovery Time Objective):** 4 hours
- **RPO (Recovery Point Objective):** 5 minutes

### High Availability
- Multi-AZ RDS deployment (production)
- S3 cross-region replication
- Lambda automatic retries
- API Gateway multi-AZ by default

---

## Cost Optimization

### Serverless Advantages
- No idle server costs
- Auto-scaling prevents over-provisioning
- Pay per request pricing

### Optimization Strategies
- Right-sized Lambda memory allocation
- RDS instance sizing based on metrics
- S3 lifecycle policies (archive to Glacier)
- CloudWatch log retention policies
- Reserved capacity for predictable workloads (future)

---

## Future Architecture Evolution

### Planned Enhancements
1. **Real-time Updates:** WebSocket API for live notifications
2. **Async Processing:** SQS + Lambda for heavy operations (PDF generation, bulk imports)
3. **Caching Layer:** ElastiCache Redis for frequently accessed data
4. **Search:** Amazon OpenSearch for full-text search
5. **Analytics:** Redshift or Athena for business intelligence
6. **Mobile:** React Native mobile apps
7. **Offline Support:** Progressive Web App (PWA) capabilities

### Scalability Roadmap
- Current: Supports 100+ concurrent users per organization
- 6 months: 500+ concurrent users
- 12 months: 1000+ concurrent users
- Strategy: Horizontal scaling via Lambda concurrency, database read replicas

---

## Related Documentation

- [Database Schema Documentation](./database-schema.md) - Complete database design
- [APPLICATION-ARCHITECTURE.md](../APPLICATION-ARCHITECTURE.md) - Detailed implementation guide
- [CLAUDE.md](../../CLAUDE.md) - Development guidelines for AI assistants
- [Testing README](../../testing/README.md) - Testing procedures and templates

---

**Document Maintainers:** Development Team
**Review Cycle:** Quarterly or after major architectural changes
**Questions:** Contact technical lead