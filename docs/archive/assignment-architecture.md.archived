# Unified Assignment Management Architecture

## Overview

The Unified Assignment Management system provides a centralized approach for managing user assignments across all contexts in the construction management platform (organization, project, location, department, equipment, phases). This system replaces multiple separate assignment tables with a single, flexible `user_assignments` table and provides comprehensive APIs for assignment management.

## Architecture Components

### 1. Database Schema

#### Primary Table: `iam.user_assignments`
```sql
CREATE TABLE iam.user_assignments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES iam.users(id),
    role_id BIGINT NOT NULL REFERENCES iam.roles(id),
    context_type VARCHAR(50) NOT NULL, -- 'organization', 'project', 'location', 'department', 'equipment', 'phase'
    context_id BIGINT NOT NULL,        -- ID of the context (project ID, location ID, etc.)
    trade_type VARCHAR(100),           -- Optional: 'electrical', 'plumbing', 'hvac', etc.
    is_primary BOOLEAN DEFAULT FALSE,  -- Is this the primary assignment for this context
    start_date DATE,                   -- When assignment becomes active
    end_date DATE,                     -- When assignment expires
    created_at TIMESTAMP DEFAULT NOW(),
    created_by BIGINT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by BIGINT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- Indexes for performance
CREATE INDEX idx_user_assignments_user_id ON iam.user_assignments(user_id);
CREATE INDEX idx_user_assignments_context ON iam.user_assignments(context_type, context_id);
CREATE INDEX idx_user_assignments_active ON iam.user_assignments(user_id, context_type)
WHERE is_deleted = FALSE AND (start_date IS NULL OR start_date <= NOW())
AND (end_date IS NULL OR end_date >= NOW());
```

#### Context Types Supported
- `organization` - Company-wide roles
- `project` - Project-specific assignments
- `location` - Location/job site assignments
- `department` - Department-specific roles
- `equipment` - Equipment operator assignments
- `phase` - Project phase assignments

### 2. Service Architecture

#### New Service: `infrastructure-assignment-management`
- **Purpose**: Centralized assignment management across all contexts
- **Location**: `/src/infrastructure-assignment-management/`
- **Responsibilities**:
  - CRUD operations for assignments
  - Bulk assignment operations
  - Assignment transfers
  - Permission checking
  - Context validation

#### Integration with Existing Services
- **Token Customizer**: Uses `user_assignments` to populate JWT with accessible locations
- **Project Management**: Can use assignment APIs instead of separate project user tables
- **User Management**: Integration for user lifecycle events

### 3. API Endpoints

#### Base URL: `/api/assignments`

#### Basic CRUD Operations
```
POST   /assignments                    # Create single assignment
GET    /assignments                    # List assignments with filters
GET    /assignments/{assignmentId}     # Get specific assignment
PUT    /assignments/{assignmentId}     # Update assignment
DELETE /assignments/{assignmentId}     # Delete assignment
```

#### Bulk Operations
```
POST   /assignments/bulk               # Create multiple assignments
POST   /assignments/transfer           # Transfer assignments between users
```

#### User-Specific Endpoints
```
GET    /users/{userId}/assignments              # Get all user assignments
GET    /users/{userId}/assignments/active       # Get active assignments only
GET    /users/{userId}/contexts/{contextType}   # Get accessible context IDs
```

#### Context-Specific Endpoints
```
GET    /contexts/{contextType}/{contextId}/assignments  # Get all assignments for context
```

#### Permission Management
```
POST   /permissions/check              # Check user permissions for context
```

### 4. Data Models

#### Core Models

**UserAssignment**
```go
type UserAssignment struct {
    ID          int64          `json:"id"`
    UserID      int64          `json:"user_id"`
    RoleID      int64          `json:"role_id"`
    ContextType string         `json:"context_type"`
    ContextID   int64          `json:"context_id"`
    TradeType   sql.NullString `json:"trade_type,omitempty"`
    IsPrimary   bool           `json:"is_primary"`
    StartDate   sql.NullTime   `json:"start_date,omitempty"`
    EndDate     sql.NullTime   `json:"end_date,omitempty"`
    CreatedAt   time.Time      `json:"created_at"`
    CreatedBy   int64          `json:"created_by"`
    UpdatedAt   time.Time      `json:"updated_at"`
    UpdatedBy   int64          `json:"updated_by"`
    IsDeleted   bool           `json:"is_deleted"`
}
```

**AssignmentResponse** (Enriched with related data)
```go
type AssignmentResponse struct {
    UserAssignment
    UserName        string `json:"user_name,omitempty"`
    UserEmail       string `json:"user_email,omitempty"`
    RoleName        string `json:"role_name,omitempty"`
    ContextName     string `json:"context_name,omitempty"`
    OrganizationID  int64  `json:"organization_id,omitempty"`
    IsActive        bool   `json:"is_active"`
    DaysRemaining   *int   `json:"days_remaining,omitempty"`
}
```

#### Request Models

**CreateAssignmentRequest**
```go
type CreateAssignmentRequest struct {
    UserID      int64  `json:"user_id" binding:"required"`
    RoleID      int64  `json:"role_id" binding:"required"`
    ContextType string `json:"context_type" binding:"required,oneof=organization project location department equipment phase"`
    ContextID   int64  `json:"context_id" binding:"required"`
    TradeType   string `json:"trade_type,omitempty"`
    IsPrimary   bool   `json:"is_primary,omitempty"`
    StartDate   string `json:"start_date,omitempty"` // YYYY-MM-DD
    EndDate     string `json:"end_date,omitempty"`   // YYYY-MM-DD
}
```

**BulkAssignmentRequest**
```go
type BulkAssignmentRequest struct {
    UserIDs     []int64 `json:"user_ids" binding:"required,min=1"`
    RoleID      int64   `json:"role_id" binding:"required"`
    ContextType string  `json:"context_type" binding:"required"`
    ContextID   int64   `json:"context_id" binding:"required"`
    TradeType   string  `json:"trade_type,omitempty"`
    IsPrimary   bool    `json:"is_primary,omitempty"`
    StartDate   string  `json:"start_date,omitempty"`
    EndDate     string  `json:"end_date,omitempty"`
}
```

### 5. Integration with JWT Token System

#### Token Customizer Integration
The Token Customizer (`infrastructure-token-customizer`) integrates with the assignment system to populate JWT tokens with user's accessible locations:

```go
// In GetUserProfile method
query := `
    SELECT COALESCE(
        array_agg(DISTINCT
            CASE ua.context_type
                WHEN 'organization' THEN 'ORG:' || ua.context_id
                WHEN 'location' THEN 'LOC:' || ua.context_id
                WHEN 'project' THEN 'PROJ:' || ua.context_id
            END
        ) FILTER (WHERE ua.context_id IS NOT NULL),
        ARRAY[]::text[]
    ) as access_contexts
    FROM iam.user_assignments ua
    WHERE ua.user_id = $1 AND ua.is_deleted = FALSE
`
```

#### JWT Token Claims
```json
{
  "user_id": "123",
  "org_id": "456",
  "locations": "base64-encoded-json-of-accessible-locations",
  "access_contexts": ["ORG:456", "LOC:789", "PROJ:101", "PROJ:102"]
}
```

### 6. Permission System

#### Permission Inheritance Hierarchy
1. **Organization Level**: Full access to all locations and projects
2. **Location Level**: Access to all projects at that location
3. **Project Level**: Access only to assigned projects
4. **Department Level**: Access to department-specific resources
5. **Equipment Level**: Access to specific equipment
6. **Phase Level**: Access to specific project phases

#### Permission Check Flow
```go
func CheckPermission(userID int64, contextType string, contextID int64, permission string) bool {
    // 1. Check direct assignment to context
    // 2. Check inherited permissions from parent contexts
    // 3. Check organization-level permissions
    // 4. Return final permission decision
}
```

### 7. Migration Strategy

#### Phase 1: Parallel System
- Deploy assignment management service alongside existing systems
- Keep existing `project_user_roles` table for backward compatibility
- Gradually migrate data and APIs

#### Phase 2: Data Migration
```sql
-- Migrate existing project assignments
INSERT INTO iam.user_assignments (
    user_id, role_id, context_type, context_id, trade_type,
    is_primary, start_date, end_date, created_by, updated_by
)
SELECT
    user_id, role_id, 'project', project_id, trade_type,
    is_primary, start_date, end_date, created_by, updated_by
FROM project.project_user_roles
WHERE is_deleted = FALSE;
```

#### Phase 3: API Migration
- Update frontend to use new assignment APIs
- Deprecate old project assignment endpoints
- Remove redundant tables

### 8. Example API Usage

#### Creating a Project Assignment
```bash
curl -X POST /api/assignments \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123,
    "role_id": 8,
    "context_type": "project",
    "context_id": 456,
    "trade_type": "electrical",
    "is_primary": true,
    "start_date": "2024-01-01",
    "end_date": "2024-12-31"
  }'
```

#### Bulk Assigning Team to Project
```bash
curl -X POST /api/assignments/bulk \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": [123, 124, 125],
    "role_id": 10,
    "context_type": "project",
    "context_id": 456,
    "trade_type": "construction",
    "start_date": "2024-01-01"
  }'
```

#### Getting User's Project Assignments
```bash
curl -X GET "/api/users/123/contexts/project" \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### Checking User Permission
```bash
curl -X POST /api/permissions/check \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123,
    "context_type": "project",
    "context_id": 456,
    "permission": "read"
  }'
```

### 9. Benefits of Unified System

#### For Developers
- **Single Source of Truth**: All assignments in one table
- **Consistent APIs**: Same endpoints for all assignment types
- **Easier Maintenance**: One codebase instead of multiple
- **Better Performance**: Optimized queries and indexes

#### For Frontend/UI
- **Unified Authorization**: Single permission check system
- **Consistent UX**: Same assignment interface for all contexts
- **Real-time Updates**: Centralized assignment events
- **Better Performance**: Fewer API calls needed

#### For Business Logic
- **Flexible Roles**: Support for any context type
- **Temporal Assignments**: Start/end date support
- **Bulk Operations**: Efficient team management
- **Permission Inheritance**: Hierarchical access control

### 10. Security Considerations

#### Organization Isolation
- All queries filter by organization ID from JWT
- Context validation ensures resources belong to user's organization
- Role assignments respect organizational boundaries

#### Audit Trail
- All assignment changes tracked with created_by/updated_by
- Soft deletes preserve history
- Comprehensive logging of assignment operations

#### Permission Validation
- Context existence validated before assignment creation
- Role permissions checked against organization roles
- JWT token integration for real-time authorization

### 11. Performance Optimizations

#### Database Optimizations
- Composite indexes on common query patterns
- Partial indexes for active assignments
- Query optimization for large assignment lists

#### Caching Strategy
- User assignments cached in JWT tokens
- Redis caching for frequently accessed assignments
- Background refresh of assignment caches

#### API Optimizations
- Pagination for large result sets
- Bulk operations for efficiency
- Selective field loading based on use case

## Conclusion

The Unified Assignment Management system provides a robust, scalable foundation for managing user access across all contexts in the construction management platform. By centralizing assignment logic and providing consistent APIs, it simplifies development while enabling powerful access control and permission management features.

This architecture supports the platform's evolution from a simple project management tool to a comprehensive construction management solution similar to Procore or Bluebeam, with the flexibility to handle complex organizational structures and permission hierarchies.