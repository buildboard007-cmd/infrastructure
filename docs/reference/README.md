# Reference Documentation

> Quick lookup guides for API endpoints, database tables, JWT claims, error codes, and environment configuration

This directory contains concise, table-based reference materials for common development tasks. For detailed explanations, see the main documentation in `/docs`.

---

## Available References

### [API Endpoints Reference](./api-endpoints.md)
Complete list of all REST API endpoints across 11 Lambda functions.

**Contents:**
- Organization, Location, Role, Permission Management
- User Management & Assignments
- Project, Issue, RFI, Submittal Management
- Centralized Attachment Management
- HTTP methods, paths, access control
- Request/response patterns
- Common status codes

**Use when:** Making API calls, testing endpoints, understanding available operations

---

### [Database Tables Reference](./database-tables.md)
Quick reference for all database tables in IAM and Project schemas.

**Contents:**
- IAM schema tables (users, organizations, locations, roles, permissions, assignments)
- Project schema tables (projects, issues, RFIs, submittals, attachments)
- Column counts, primary keys, purpose
- Key relationships and indexes
- Common audit fields

**Use when:** Writing SQL queries, understanding data structure, designing new features

---

### [JWT Claims Reference](./jwt-claims.md)
Complete guide to JWT token structure, custom claims, and extraction patterns.

**Contents:**
- Standard Cognito claims (sub, email, iss, aud, etc.)
- Custom claims (user_id, org_id, locations, isSuperAdmin, etc.)
- Locations claim encoding/decoding
- Token lifecycle and customization triggers
- Go and JavaScript extraction examples
- Debugging token issues

**Use when:** Implementing authentication, extracting user context, debugging token issues

---

### [Error Codes Reference](./error-codes.md)
Standard HTTP status codes, error messages, and error handling patterns.

**Contents:**
- Standard HTTP status codes (200, 400, 401, 403, 404, 500)
- Error response format
- Common error messages by category
- Go error handling patterns
- Frontend error handling examples
- Best practices and testing

**Use when:** Handling errors, debugging failed requests, implementing error responses

---

### [Environment Configuration Reference](./environment-config.md)
AWS account IDs, regions, API Gateway URLs, Cognito settings, RDS endpoints, S3 buckets.

**Contents:**
- AWS account IDs and regions (Dev/Prod)
- API Gateway endpoints and stage names
- Cognito User Pool IDs and client IDs
- RDS database connection details
- S3 bucket names and key structures
- SSM parameter paths
- Lambda function configuration
- CDK deployment commands

**Use when:** Configuring environments, connecting to services, deploying infrastructure

---

## Quick Lookup Examples

### "How do I call the project list endpoint?"
→ See [api-endpoints.md](./api-endpoints.md) - Project Management section
```bash
GET /projects
Authorization: Bearer <id_token>
```

### "What columns are in the issues table?"
→ See [database-tables.md](./database-tables.md) - project.issues section

### "How do I extract user_id from JWT in Go?"
→ See [jwt-claims.md](./jwt-claims.md) - Extracting Claims in Lambda Functions

### "What does a 403 error mean?"
→ See [error-codes.md](./error-codes.md) - Authorization Errors section

### "What's the Dev API Gateway URL?"
→ See [environment-config.md](./environment-config.md) - API Gateway section
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
```

---

## Related Documentation

For more detailed information, see:

- **[Architecture Documentation](../architecture/)** - System design and patterns
- **[Entity Documentation](../entities/)** - Detailed entity guides (Assignment, Issue, RFI, etc.)
- **[Guides](../guides/)** - How-to guides and workflows
- **[QUICK-START.md](../QUICK-START.md)** - Get started in 5 minutes
- **[README.md](../README.md)** - Complete documentation index

---

## Maintenance

These reference documents should be updated when:
- New endpoints are added or changed
- Database schema is modified
- JWT claims structure changes
- New error patterns are introduced
- Environment configuration changes

**Last Updated:** 2025-10-27