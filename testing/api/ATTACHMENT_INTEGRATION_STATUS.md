# Attachment Integration Status Report

## Executive Summary

Checked all management APIs (Project, Issue, RFI, Submittal) to verify attachment integration in GET endpoints.

**Status:** ⚠️ **Partial Integration** - Only RFI and Submittal are fetching attachments

---

## Current Status by Entity

### ✅ RFI Management - **COMPLETE**
**File:** `/src/infrastructure-rfi-management/main.go`

**Status:** Attachments ARE being fetched in GET /rfis/{id}

**Implementation:**
```go
// Line 138-143
// Enrich with comments and attachments
comments, _ := rfiRepository.GetRFIComments(ctx, rfiID)
attachments, _ := rfiRepository.GetRFIAttachments(ctx, rfiID)

rfi.Comments = comments
rfi.Attachments = attachments
```

**Response includes:**
```json
{
  "id": 1,
  "title": "Clarification Request",
  "attachments": [
    {
      "id": 1,
      "filename": "clarification.pdf",
      "file_size": 512000,
      "s3_url": "...",
      ...
    }
  ],
  "comments": [...],
  ...
}
```

✅ **No action needed**

---

### ✅ Submittal Management - **COMPLETE**
**File:** `/src/infrastructure-submittal-management/main.go`

**Status:** Attachments ARE being fetched in repository layer

**Implementation:**
```go
// File: /src/lib/data/submittal_repository.go
// Line 248-250
attachments, _ := dao.GetSubmittalAttachments(ctx, submittalID)
submittal.Attachments = attachments
submittal.AttachmentCount = len(attachments)
```

**Response Model:**
```go
type SubmittalResponse struct {
    ...
    Attachments     []SubmittalAttachment `json:"attachments,omitempty"`
    AttachmentCount int                   `json:"attachment_count,omitempty"`
    ...
}
```

✅ **No action needed**

---

### ❌ Issue Management - **MISSING**
**File:** `/src/infrastructure-issue-management/main.go`

**Status:** Attachments are NOT being fetched in GET /issues/{id}

**Current Implementation:**
```go
// Line 168-191 (handleGetIssue)
func handleGetIssue(ctx context.Context, issueID, orgID int64) events.APIGatewayProxyResponse {
    issue, err := issueRepository.GetIssueByID(ctx, issueID)
    if err != nil {
        // error handling...
    }

    // Validate issue belongs to org
    // ... validation code ...

    return api.SuccessResponse(http.StatusOK, issue, logger)
    // ❌ Missing: attachment fetching
}
```

**Issue Model has attachment field:**
```go
// /src/lib/models/issue.go
type Issue struct {
    ...
    Attachments []string `json:"attachments,omitempty"`
    ...
}
```

⚠️ **ACTION REQUIRED:** Need to add attachment fetching

**Recommended Fix:**
```go
func handleGetIssue(ctx context.Context, issueID, orgID int64) events.APIGatewayProxyResponse {
    issue, err := issueRepository.GetIssueByID(ctx, issueID)
    if err != nil {
        if err.Error() == "issue not found" {
            return api.ErrorResponse(http.StatusNotFound, "Issue not found", logger)
        }
        logger.WithError(err).Error("Failed to get issue")
        return api.ErrorResponse(http.StatusInternalServerError, "Failed to get issue", logger)
    }

    // Validate issue belongs to org
    var projectOrgID int64
    err = sqlDB.QueryRowContext(ctx, `
        SELECT org_id FROM project.projects
        WHERE id = $1 AND is_deleted = FALSE
    `, issue.ProjectID).Scan(&projectOrgID)

    if err != nil || projectOrgID != orgID {
        return api.ErrorResponse(http.StatusForbidden, "Issue does not belong to your organization", logger)
    }

    // ✅ ADD THIS: Fetch attachments using centralized attachment repository
    attachments, _ := attachmentRepository.GetAttachmentsByEntity(ctx, "issue", issueID, nil)
    issue.Attachments = convertAttachmentsToStrings(attachments) // Or use full attachment objects

    return api.SuccessResponse(http.StatusOK, issue, logger)
}
```

---

### ❌ Project Management - **ARCHITECTURE DECISION**
**File:** `/src/infrastructure-project-management/main.go`

**Status:** Project attachments explicitly removed, using centralized service

**Comments in code:**
```go
// Line 71
// Project attachment endpoints removed - now handled by centralized attachment management service

// Line 224
// Project attachment handlers removed - now handled by centralized attachment management service
```

**Question:** Should GET /projects/{id} include attachments in response?

**Options:**

**Option A: Keep separate** (Current approach)
- Pros: Clean separation, clients call `/entities/project/{id}/attachments` explicitly
- Cons: Extra API call needed to get complete project view

**Option B: Include in GET response**
- Pros: Single API call for complete project data
- Cons: May slow down project list queries if not careful

**Recommendation:**
- **For GET /projects/{id}** (single project): ✅ Include attachments
- **For GET /projects** (list): ❌ Don't include (use separate call)

**Implementation would be similar to RFI/Submittal:**
```go
func handleGetProject(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
    // ... existing project fetch code ...

    // Add attachments if fetching single project
    if isSingleProjectFetch {
        attachments, _ := attachmentRepository.GetAttachmentsByEntity(ctx, "project", projectID, nil)
        project.Attachments = attachments
    }

    return api.SuccessResponse(http.StatusOK, project, logger), nil
}
```

---

## Summary Table

| Entity    | GET Endpoint | Attachments Fetched? | Action Needed |
|-----------|--------------|----------------------|---------------|
| RFI       | GET /rfis/{id} | ✅ Yes | None |
| Submittal | GET /submittals/{id} | ✅ Yes | None |
| Issue     | GET /issues/{id} | ❌ No | **Add attachment fetching** |
| Project   | GET /projects/{id} | ❌ No | **Design decision needed** |

---

## Recommendations

### Priority 1: Fix Issue Management
**Immediate action required** - Issues should return attachments just like RFIs and Submittals do.

1. Update `handleGetIssue` in `/src/infrastructure-issue-management/main.go`
2. Initialize `attachmentRepository` in the init() function
3. Fetch attachments using `GetAttachmentsByEntity(ctx, "issue", issueID, nil)`
4. Add to response

### Priority 2: Decide on Project Attachments
**Design decision needed** - Should projects include attachments in GET response?

Suggested approach:
- GET `/projects/{id}` → Include attachments (single project detail view)
- GET `/projects` → Don't include (performance for list views)
- Keep `/entities/project/{id}/attachments` for dedicated attachment management

### Priority 3: Update Postman Collections
Once Issue and Project are updated, update Postman collections to show attachment responses in examples.

---

## Testing Verification

After implementing fixes, verify with these tests:

### Test Issue Attachments
```bash
# 1. Create issue attachment
POST /attachments/upload-url
{
  "entity_type": "issue",
  "entity_id": 101,
  "project_id": 49,
  "location_id": 24,
  "file_name": "crack_photo.jpg",
  "file_size": 1048576,
  "attachment_type": "before_photo"
}

# 2. Get issue and verify attachments field is populated
GET /issues/101

# Expected response should include:
{
  "id": 101,
  "title": "Wall crack in lobby",
  "attachments": [
    {
      "id": 1,
      "file_name": "crack_photo.jpg",
      ...
    }
  ],
  ...
}
```

### Test Project Attachments
```bash
# 1. Create project attachment
POST /attachments/upload-url
{
  "entity_type": "project",
  "entity_id": 49,
  "project_id": 49,
  "location_id": 24,
  "file_name": "floor_plan.pdf",
  "file_size": 2048576,
  "attachment_type": "drawing"
}

# 2. Get project and verify attachments
GET /projects/49

# Expected response should include attachments array
```

---

## Database Schema Notes

All attachment tables exist and are ready:
- ✅ `project.project_attachments`
- ✅ `project.issue_attachments`
- ✅ `project.rfi_attachments`
- ✅ `project.submittal_attachments`

Centralized attachment repository (`/src/lib/data/attachment_repository.go`) supports all entity types through `GetAttachmentsByEntity()` method.

---

## Next Steps

1. **Implement Issue attachment fetching** (30 min)
2. **Decide on Project attachment strategy** (discussion)
3. **Implement Project attachment fetching if approved** (30 min)
4. **Test all endpoints** (15 min)
5. **Update Postman collections with examples** (15 min)
6. **Update API documentation** (15 min)

**Total estimated time:** 2 hours

---

**Report Date:** 2025-10-06
**Reviewed APIs:** Project, Issue, RFI, Submittal Management
**Status:** 50% Complete (2/4 entities fetching attachments)