# Attachment Management System

## Overview

The Attachment Management system provides centralized file handling for all entities in the construction management platform. It uses a **unified API** with **entity-specific database tables**, ensuring data isolation while maintaining a consistent upload/download workflow.

**Key Features:**
- Centralized S3 storage with hierarchical folder structure
- Presigned URLs for secure direct upload/download
- Entity-specific attachment tables for data integrity
- Support for multiple entity types (issues, RFIs, submittals, comments, projects)
- Soft-delete functionality
- Access control based on organization membership
- File type validation and MIME type detection

**Architecture Pattern:**
- One API service handles all attachment operations
- Dynamic table routing based on `entity_type`
- Entity-specific foreign key constraints
- Consistent S3 path structure: `{org_id}/{location_id}/{project_id}/{entity_type}/{entity_id}/{timestamp}_{filename}`

---

## Supported Entities

The system supports attachments for the following entities:

| Entity Type | Database Table | Foreign Key Column | Example Use Case |
|------------|----------------|-------------------|------------------|
| `issue` | `project.issue_attachments` | `issue_id` | Deficiency photos, repair documentation |
| `rfi` | `project.rfi_attachments` | `rfi_id` | Drawings, specifications, clarification docs |
| `submittal` | `project.submittal_attachments` | `submittal_id` | Shop drawings, product data, samples |
| `issue_comment` | `project.issue_comment_attachments` | `comment_id` | Photos added to issue comments |
| `project` | `project.project_attachments` | `project_id` | General project documents |

---

## Database Schema

### Entity-Specific Attachment Tables

#### 1. Issue Attachments (`project.issue_attachments`)

```sql
CREATE TABLE project.issue_attachments (
    id                BIGSERIAL PRIMARY KEY,
    issue_id          BIGINT NOT NULL REFERENCES project.issues(id),
    file_name         VARCHAR(255) NOT NULL,
    file_path         VARCHAR(500) NOT NULL,
    file_size         BIGINT,
    file_type         VARCHAR(50),
    attachment_type   VARCHAR(50) NOT NULL DEFAULT 'before_photo',
    uploaded_by       BIGINT NOT NULL REFERENCES iam.users(id),
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by        BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted        BOOLEAN NOT NULL DEFAULT FALSE
);

-- Valid attachment_type values for issues:
-- 'before_photo', 'progress_photo', 'after_photo', 'issue_document'
```

#### 2. RFI Attachments (`project.rfi_attachments`)

```sql
CREATE TABLE project.rfi_attachments (
    id                BIGSERIAL PRIMARY KEY,
    rfi_id            BIGINT NOT NULL REFERENCES project.rfis(id),
    file_name         VARCHAR(255) NOT NULL,
    file_path         VARCHAR(500),
    file_size         BIGINT,
    file_type         VARCHAR(50),
    attachment_type   VARCHAR(50) DEFAULT 'document',
    uploaded_by       BIGINT NOT NULL REFERENCES iam.users(id),
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by        BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted        BOOLEAN NOT NULL DEFAULT FALSE,

    -- Legacy fields (maintained for compatibility)
    description       TEXT,
    s3_bucket         VARCHAR(255),
    s3_key            VARCHAR(500),
    s3_url            TEXT,
    upload_date       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Valid attachment_type values for RFIs:
-- 'rfi_question', 'rfi_response', 'rfi_supporting_doc'
```

#### 3. Submittal Attachments (`project.submittal_attachments`)

```sql
CREATE TABLE project.submittal_attachments (
    id                BIGSERIAL PRIMARY KEY,
    submittal_id      BIGINT NOT NULL REFERENCES project.submittals(id),
    file_name         VARCHAR(255) NOT NULL,
    file_path         VARCHAR(500) NOT NULL,
    file_size         BIGINT,
    file_type         VARCHAR(50),
    attachment_type   VARCHAR(50) NOT NULL DEFAULT 'other',
    uploaded_by       BIGINT NOT NULL REFERENCES iam.users(id),
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by        BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted        BOOLEAN NOT NULL DEFAULT FALSE
);

-- Valid attachment_type values for submittals:
-- 'shop_drawing', 'product_data', 'sample', 'certificate', 'submittal_document'
```

#### 4. Issue Comment Attachments (`project.issue_comment_attachments`)

```sql
CREATE TABLE project.issue_comment_attachments (
    id                BIGSERIAL PRIMARY KEY,
    comment_id        BIGINT NULL REFERENCES project.issue_comments(id),
    file_name         VARCHAR(255) NOT NULL,
    file_path         VARCHAR(500) NOT NULL,
    file_size         BIGINT,
    file_type         VARCHAR(50),
    attachment_type   VARCHAR(50) NOT NULL DEFAULT 'photo',
    uploaded_by       BIGINT NOT NULL REFERENCES iam.users(id),
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by        BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted        BOOLEAN NOT NULL DEFAULT FALSE
);

-- Note: comment_id can be NULL initially for pre-upload scenario
-- Gets updated when comment is created with attachment_ids
```

#### 5. Project Attachments (`project.project_attachments`)

```sql
CREATE TABLE project.project_attachments (
    id                BIGSERIAL PRIMARY KEY,
    project_id        BIGINT NOT NULL REFERENCES project.projects(id),
    file_name         VARCHAR(255) NOT NULL,
    file_path         VARCHAR(500) NOT NULL,
    file_size         BIGINT,
    file_type         VARCHAR(50),
    attachment_type   VARCHAR(50) NOT NULL,
    uploaded_by       BIGINT NOT NULL REFERENCES iam.users(id),
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        BIGINT NOT NULL REFERENCES iam.users(id),
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by        BIGINT NOT NULL REFERENCES iam.users(id),
    is_deleted        BOOLEAN NOT NULL DEFAULT FALSE
);

-- Valid attachment_type values for projects:
-- 'project_document', 'project_drawing', 'project_photo', 'project_report'
```

---

## Data Models

### Core Attachment Model

```go
// Attachment represents a file attachment for any entity type
type Attachment struct {
    ID             int64     `json:"id"`
    EntityType     string    `json:"entity_type"`     // "project", "issue", "rfi", "submittal", "issue_comment"
    EntityID       int64     `json:"entity_id"`       // ID of the entity
    ProjectID      int64     `json:"project_id"`      // For hierarchy
    LocationID     int64     `json:"location_id"`     // For hierarchy
    OrgID          int64     `json:"org_id"`          // Organization ID
    FileName       string    `json:"file_name"`       // Original filename
    FilePath       string    `json:"file_path"`       // S3 key
    FileSize       *int64    `json:"file_size,omitempty"`
    FileType       *string   `json:"file_type,omitempty"`
    MimeType       *string   `json:"mime_type,omitempty"`
    AttachmentType string    `json:"attachment_type"` // Category of attachment
    UploadedBy     int64     `json:"uploaded_by"`
    UploadStatus   string    `json:"upload_status"`   // "pending", "uploaded", "failed"
    CreatedAt      time.Time `json:"created_at"`
    CreatedBy      int64     `json:"created_by"`
    UpdatedAt      time.Time `json:"updated_at"`
    UpdatedBy      int64     `json:"updated_by"`
    IsDeleted      bool      `json:"is_deleted"`
}
```

### Upload Request Model

```go
type AttachmentUploadRequest struct {
    EntityType     string `json:"entity_type" binding:"required,oneof=project issue rfi submittal issue_comment"`
    EntityID       int64  `json:"entity_id"` // Can be 0 for issue_comment (temp upload)
    ProjectID      int64  `json:"project_id" binding:"required"`
    LocationID     int64  `json:"location_id" binding:"required"`
    OrgID          int64  `json:"org_id,omitempty"` // Set from JWT claims
    FileName       string `json:"file_name" binding:"required,max=255"`
    FileSize       int64  `json:"file_size" binding:"required,max=104857600"` // 100MB max
    AttachmentType string `json:"attachment_type" binding:"required"`
}
```

### Upload Response Model

```go
type AttachmentUploadResponse struct {
    AttachmentID int64  `json:"attachment_id"`
    UploadURL    string `json:"upload_url"`    // Presigned S3 URL (15 min expiry)
    S3Key        string `json:"s3_key"`        // S3 object key
    ExpiresAt    string `json:"expires_at"`    // ISO 8601 timestamp
}
```

### Download Response Model

```go
type AttachmentDownloadResponse struct {
    DownloadURL string `json:"download_url"`   // Presigned S3 URL (60 min expiry)
    FileName    string `json:"file_name"`
    FileSize    *int64 `json:"file_size,omitempty"`
    ExpiresAt   string `json:"expires_at"`     // ISO 8601 timestamp
}
```

---

## API Endpoints

**Base URL:** `https://{api-gateway-url}/prod` or `https://{api-gateway-url}/main`

**Authentication:** All endpoints require JWT ID token in `Authorization: Bearer {token}` header

### Core Operations

#### 1. Generate Upload URL

```http
POST /attachments/upload-url
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "entity_type": "issue",
  "entity_id": 72,
  "project_id": 49,
  "location_id": 24,
  "file_name": "wall_crack_photo.jpg",
  "file_size": 524288,
  "attachment_type": "before_photo"
}

Response (200 OK):
{
  "attachment_id": 6,
  "upload_url": "https://s3.amazonaws.com/bucket/...",
  "s3_key": "10/24/49/issues/72/20251006201530_wall_crack_photo.jpg",
  "expires_at": "2025-10-06T20:30:30Z"
}
```

**Special Case: Issue Comment Attachments**

For comments, you can pre-upload attachments with `entity_id: 0`:

```json
{
  "entity_type": "issue_comment",
  "entity_id": 0,
  "project_id": 49,
  "location_id": 24,
  "file_name": "comment_photo.jpg",
  "file_size": 123456,
  "attachment_type": "photo"
}
```

The attachment record is created with `comment_id = NULL` and later linked when the comment is created.

#### 2. Confirm Upload

```http
POST /attachments/confirm
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "attachment_id": 6
}

Response (200 OK):
{
  "status": "confirmed"
}
```

#### 3. Get Attachment Metadata

```http
GET /attachments/{id}?entity_type={entity_type}
Authorization: Bearer {jwt_token}

Response (200 OK):
{
  "id": 6,
  "entity_type": "issue",
  "entity_id": 72,
  "file_name": "wall_crack_photo.jpg",
  "file_path": "10/24/49/issues/72/20251006201530_wall_crack_photo.jpg",
  "file_size": 524288,
  "file_type": "image/jpeg",
  "attachment_type": "before_photo",
  "uploaded_by": 19,
  "created_at": "2025-10-06T20:15:30Z",
  "is_deleted": false
}
```

**Note:** `entity_type` query parameter is required for dynamic table routing.

#### 4. Generate Download URL

```http
GET /attachments/{id}/download-url?entity_type={entity_type}
Authorization: Bearer {jwt_token}

Response (200 OK):
{
  "download_url": "https://s3.amazonaws.com/bucket/...",
  "file_name": "wall_crack_photo.jpg",
  "file_size": 524288,
  "expires_at": "2025-10-06T21:15:30Z"
}
```

**URL Expiry:** Download URLs are valid for 60 minutes.

#### 5. Delete Attachment (Soft Delete)

```http
DELETE /attachments/{id}?entity_type={entity_type}
Authorization: Bearer {jwt_token}

Response (200 OK):
{
  "status": "deleted"
}
```

**Note:** This is a soft delete. The file remains in S3 but `is_deleted` is set to `TRUE`.

### Entity-Based Queries

#### 6. List Attachments for Entity

```http
GET /entities/{type}/{id}/attachments?attachment_type={optional_filter}
Authorization: Bearer {jwt_token}

Example: GET /entities/issue/72/attachments

Response (200 OK):
{
  "attachments": [
    {
      "id": 6,
      "file_name": "wall_crack_photo.jpg",
      "file_path": "10/24/49/issues/72/20251006201530_wall_crack_photo.jpg",
      "file_size": 524288,
      "file_type": "image/jpeg",
      "attachment_type": "before_photo",
      "created_at": "2025-10-06T20:15:30Z"
    }
  ],
  "total_count": 1,
  "page": 1,
  "page_size": 20,
  "has_next": false,
  "has_previous": false
}
```

**Query Parameters:**
- `attachment_type`: Filter by attachment type
- `page`: Page number (default: 1)
- `limit`: Results per page (default: 20, max: 100)

---

## S3 Integration

### S3 Path Structure

The system uses a hierarchical folder structure for organized storage:

```
{org_id}/{location_id}/{project_id}/{entity_type}/{entity_id}/{timestamp}_{filename}

Examples:
10/24/49/issues/72/20251006201530_wall_crack_photo.jpg
10/24/49/rfis/81/20251006202000_foundation_plan.pdf
10/24/49/submittals/9/20251006202500_steel_beam_details.dwg
10/24/49/comments/45/20251006203000_comment_photo.jpg
10/24/49/attachments/20251006203500_project_manual.pdf  (project-level)
```

**Path Generation Logic:**

```go
func (req *AttachmentUploadRequest) GenerateS3Key() string {
    timestamp := time.Now().Format("20060102150405")
    cleanFileName := strings.ReplaceAll(req.FileName, " ", "_")

    switch req.EntityType {
    case EntityTypeProject:
        return fmt.Sprintf("%d/%d/%d/attachments/%s_%s",
            req.OrgID, req.LocationID, req.ProjectID, timestamp, cleanFileName)
    case EntityTypeIssue:
        return fmt.Sprintf("%d/%d/%d/issues/%d/%s_%s",
            req.OrgID, req.LocationID, req.ProjectID, req.EntityID, timestamp, cleanFileName)
    case EntityTypeRFI:
        return fmt.Sprintf("%d/%d/%d/rfis/%d/%s_%s",
            req.OrgID, req.LocationID, req.ProjectID, req.EntityID, timestamp, cleanFileName)
    case EntityTypeSubmittal:
        return fmt.Sprintf("%d/%d/%d/submittals/%d/%s_%s",
            req.OrgID, req.LocationID, req.ProjectID, req.EntityID, timestamp, cleanFileName)
    case EntityTypeIssueComment:
        if req.EntityID == 0 {
            return fmt.Sprintf("%d/%d/%d/comments/temp/%s_%s",
                req.OrgID, req.LocationID, req.ProjectID, timestamp, cleanFileName)
        }
        return fmt.Sprintf("%d/%d/%d/comments/%d/%s_%s",
            req.OrgID, req.LocationID, req.ProjectID, req.EntityID, timestamp, cleanFileName)
    }
}
```

### Presigned URLs

**Upload URL (15-minute expiry):**
```go
uploadURL, err := s3Client.GenerateUploadURL(s3Key, 15*time.Minute)
```

**Download URL (60-minute expiry):**
```go
downloadURL, err := s3Client.GenerateDownloadURL(s3Key, 60*time.Minute)
```

### Upload Flow

```
1. Client: POST /attachments/upload-url
   ↓
2. API: Create DB record, generate S3 key, return presigned URL
   ↓
3. Client: PUT to presigned S3 URL (direct upload, bypasses API Gateway)
   ↓
4. Client: POST /attachments/confirm (optional verification)
   ↓
5. Client: GET entity endpoint (attachments included in response)
```

---

## Repository Methods

**File:** `/Users/mayur/git_personal/infrastructure/src/lib/data/attachment_repository.go`

### Interface Definition

```go
type AttachmentRepository interface {
    CreateAttachment(ctx context.Context, attachment *models.Attachment) (*models.Attachment, error)
    GetAttachment(ctx context.Context, attachmentID int64, entityType string) (*models.Attachment, error)
    GetAttachmentsByEntity(ctx context.Context, entityType string, entityID int64, filters map[string]string) ([]models.Attachment, error)
    UpdateAttachmentStatus(ctx context.Context, attachmentID int64, entityType string, status string) error
    SoftDeleteAttachment(ctx context.Context, attachmentID int64, entityType string, userID int64) error
    VerifyAttachmentAccess(ctx context.Context, attachmentID int64, entityType string, orgID int64) (bool, error)
}
```

### Key Methods

#### CreateAttachment

Creates attachment record in the appropriate entity-specific table:

```go
func (dao *AttachmentDao) CreateAttachment(ctx context.Context, attachment *models.Attachment) (*models.Attachment, error) {
    tableName := models.GetTableName(attachment.EntityType)
    entityIDColumn := models.GetEntityIDColumn(attachment.EntityType)

    // For issue_comment with entity_id = 0, use NULL
    var entityIDValue interface{}
    if attachment.EntityType == models.EntityTypeIssueComment && attachment.EntityID == 0 {
        entityIDValue = nil
    } else {
        entityIDValue = attachment.EntityID
    }

    query := fmt.Sprintf(`
        INSERT INTO %s (
            %s, file_name, file_path, file_size, file_type, attachment_type,
            uploaded_by, created_by, updated_by, created_at, updated_at, is_deleted
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING id, created_at, updated_at
    `, tableName, entityIDColumn)

    // Execute query...
}
```

#### VerifyAttachmentAccess

Validates organization access through entity relationships:

```go
func (dao *AttachmentDao) VerifyAttachmentAccess(ctx context.Context, attachmentID int64, entityType string, orgID int64) (bool, error) {
    // Step 1: Check if attachment exists
    existsQuery := `SELECT EXISTS(SELECT 1 FROM {table} WHERE id = $1 AND is_deleted = false)`

    // Step 2: Verify org access through entity → project → org relationship
    accessQuery := `
        SELECT p.id
        FROM project.projects p
        JOIN project.{entity_table} e ON p.id = e.project_id
        JOIN {attachment_table} a ON e.id = a.{entity_id_column}
        WHERE a.id = $1 AND p.org_id = $2 AND a.is_deleted = false
    `

    // Returns: (hasAccess bool, error)
    // - Attachment not found: (false, "attachment not found")
    // - Access denied: (false, "access denied")
    // - Has access: (true, nil)
}
```

---

## Entity Type Pattern

### Dynamic Table Routing

The system uses helper functions to map entity types to database tables:

```go
// GetTableName returns the database table for the entity type
func GetTableName(entityType string) string {
    switch entityType {
    case EntityTypeProject:
        return "project.project_attachments"
    case EntityTypeIssue:
        return "project.issue_attachments"
    case EntityTypeRFI:
        return "project.rfi_attachments"
    case EntityTypeSubmittal:
        return "project.submittal_attachments"
    case EntityTypeIssueComment:
        return "project.issue_comment_attachments"
    default:
        return ""
    }
}

// GetEntityIDColumn returns the foreign key column name
func GetEntityIDColumn(entityType string) string {
    switch entityType {
    case EntityTypeProject:
        return "project_id"
    case EntityTypeIssue:
        return "issue_id"
    case EntityTypeRFI:
        return "rfi_id"
    case EntityTypeSubmittal:
        return "submittal_id"
    case EntityTypeIssueComment:
        return "comment_id"
    default:
        return ""
    }
}
```

### Supported Entity Types (Constants)

```go
const (
    EntityTypeProject      = "project"
    EntityTypeIssue        = "issue"
    EntityTypeRFI          = "rfi"
    EntityTypeSubmittal    = "submittal"
    EntityTypeIssueComment = "issue_comment"
)
```

---

## File Validation

### Allowed File Types

```go
func ValidateFileType(fileName string) bool {
    ext := strings.ToLower(filepath.Ext(fileName))

    allowedExtensions := map[string]bool{
        // Documents
        ".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
        ".txt": true, ".rtf": true,

        // Images
        ".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
        ".tiff": true, ".webp": true,

        // Drawings
        ".dwg": true, ".dxf": true, ".dwf": true, ".rvt": true,

        // Archives
        ".zip": true, ".rar": true, ".7z": true,

        // 3D Models
        ".ifc": true, ".skp": true, ".3ds": true, ".obj": true,
    }

    return allowedExtensions[ext]
}
```

### MIME Type Detection

```go
func GetMimeType(fileName string) string {
    ext := strings.ToLower(filepath.Ext(fileName))

    mimeTypes := map[string]string{
        ".pdf":  "application/pdf",
        ".jpg":  "image/jpeg",
        ".jpeg": "image/jpeg",
        ".png":  "image/png",
        ".dwg":  "application/acad",
        ".dxf":  "application/dxf",
        ".rvt":  "application/vnd.autodesk.revit",
        ".zip":  "application/zip",
        // ... more mappings
    }

    if mimeType, exists := mimeTypes[ext]; exists {
        return mimeType
    }
    return "application/octet-stream"
}
```

### File Size Limits

- **Maximum file size:** 100MB (104,857,600 bytes)
- **Validation:** Enforced in request model binding
- **Recommended:** Client-side validation before upload

---

## Access Control

### Organization-Based Access

All attachment operations validate organization membership:

1. **Upload:** Project must belong to user's organization
2. **Download:** Attachment's entity must belong to user's organization
3. **Delete:** Attachment's entity must belong to user's organization

### Access Validation Flow

```
User Request → JWT Claims (org_id, user_id)
    ↓
Attachment ID → Attachment Table (entity_id)
    ↓
Entity Table (project_id) → Project Table (org_id)
    ↓
Compare: JWT org_id == Project org_id
    ↓
Grant/Deny Access
```

### Error Responses

```json
// Attachment not found
{
  "error": "Attachment not found",
  "status": 404
}

// Access denied
{
  "error": "Access denied to this attachment",
  "status": 403
}

// Project validation failure
{
  "error": "Project does not belong to your organization",
  "status": 403
}
```

---

## Testing

### Postman Collection

**File:** `/Users/mayur/git_personal/infrastructure/postman/AttachmentManagement.postman_collection.json`

**Collection includes:**
- Generate upload URL for each entity type
- Upload file to S3
- Confirm upload
- Get attachment metadata
- Generate download URL
- List entity attachments
- Delete attachment
- Comment attachment workflow

### Test Scripts

#### test-comment-attachment.sh

**File:** `/Users/mayur/git_personal/infrastructure/testing/api/test-comment-attachment.sh`

Basic test for comment attachment upload:

```bash
#!/bin/bash
TOKEN=$(curl -s -X POST "https://cognito-idp.us-east-2.amazonaws.com/" \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"AuthFlow":"USER_PASSWORD_AUTH","ClientId":"...","AuthParameters":{"USERNAME":"...","PASSWORD":"..."}}' \
  | jq -r '.AuthenticationResult.IdToken')

echo "Testing comment attachment upload..."
curl -s -X POST "https://api-url/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entity_type":"issue_comment","entity_id":0,"project_id":11,"location_id":3,"file_name":"test-photo.jpg","file_size":12345,"attachment_type":"photo"}' \
  | jq .
```

### Manual Testing Steps

**Complete Upload/Download Workflow:**

```bash
# 1. Get authentication token
TOKEN="your_jwt_token"

# 2. Generate upload URL
UPLOAD_RESPONSE=$(curl -X POST "https://api-url/attachments/upload-url" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "issue",
    "entity_id": 72,
    "project_id": 49,
    "location_id": 24,
    "file_name": "test.jpg",
    "file_size": 12345,
    "attachment_type": "before_photo"
  }')

UPLOAD_URL=$(echo $UPLOAD_RESPONSE | jq -r '.upload_url')
ATTACHMENT_ID=$(echo $UPLOAD_RESPONSE | jq -r '.attachment_id')

# 3. Upload file to S3
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: image/jpeg" \
  --data-binary @test.jpg

# 4. Confirm upload
curl -X POST "https://api-url/attachments/confirm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"attachment_id\": $ATTACHMENT_ID}"

# 5. Get download URL
curl -X GET "https://api-url/attachments/$ATTACHMENT_ID/download-url?entity_type=issue" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Recent Improvements

### Issue Comment Support (2025-10-27)

**What Changed:**
- Added `issue_comment` entity type support
- Implemented pre-upload pattern (entity_id = 0)
- `comment_id` can be NULL initially
- Updated validation to allow entity_id = 0 for comments

**Why:** Comments need attachments before the comment itself is created. Users upload files first, then submit the comment with `attachment_ids[]`.

### Improved Error Handling

**Enhanced VerifyAttachmentAccess:**
- Separate checks for "not found" vs "access denied"
- Detailed error messages for database failures
- Graceful handling of unsupported entity types

**Example:**
```go
if strings.Contains(err.Error(), "attachment not found") {
    return api.ErrorResponse(http.StatusNotFound, "Attachment not found", logger), nil
}
if strings.Contains(err.Error(), "access denied") {
    return api.ErrorResponse(http.StatusForbidden, "Access denied to this attachment", logger), nil
}
```

---

## Best Practices

### 1. Upload Workflow

```
✅ DO: Create entity → Upload attachments → Retrieve entity
✅ DO: Pre-upload for comments (entity_id = 0)
✅ DO: Use presigned URLs for direct S3 upload
✅ DO: Validate file types client-side before upload

❌ DON'T: Upload before entity exists (except comments)
❌ DON'T: Upload files > 100MB
❌ DON'T: Modify S3 keys manually
```

### 2. File Management

```
✅ DO: Soft delete (set is_deleted = true)
✅ DO: Compress images before upload when appropriate
✅ DO: Show upload progress for large files

❌ DON'T: Hard delete attachments
❌ DON'T: Upload duplicate files
```

### 3. Error Recovery

```
✅ DO: Retry failed uploads
✅ DO: Handle expired presigned URLs
✅ DO: Validate entity exists before upload

❌ DON'T: Recreate entity on attachment failure
```

### 4. Security

```
✅ DO: Validate organization access
✅ DO: Use presigned URLs (time-limited)
✅ DO: Validate file types and sizes

❌ DON'T: Expose S3 bucket publicly
❌ DON'T: Share presigned URLs across organizations
```

---

## Code References

**Models:** `/Users/mayur/git_personal/infrastructure/src/lib/models/attachment.go`
**API Handler:** `/Users/mayur/git_personal/infrastructure/src/infrastructure-attachment-management/main.go`
**Repository:** `/Users/mayur/git_personal/infrastructure/src/lib/data/attachment_repository.go`
**Postman:** `/Users/mayur/git_personal/infrastructure/postman/AttachmentManagement.postman_collection.json`

---

## Summary

The Attachment Management system provides a **centralized, secure, and scalable** approach to file handling across all construction management entities. Key design decisions:

1. **One API, Multiple Tables:** Unified API with entity-specific storage
2. **Presigned URLs:** Direct S3 upload/download bypassing API Gateway
3. **Hierarchical S3 Structure:** Organized by org → location → project → entity
4. **Access Control:** Organization-based validation through entity relationships
5. **Soft Deletes:** Data retention with logical deletion
6. **Pre-upload Support:** Flexible workflow for comment attachments

This architecture balances **consistency** (unified API), **data integrity** (entity-specific tables), and **performance** (direct S3 access).