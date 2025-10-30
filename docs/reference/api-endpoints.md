# API Endpoints Reference

> Complete list of all REST API endpoints across 11 Lambda functions

**Base URL (Dev):** `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`

**Authentication:** All endpoints require Cognito JWT ID Token via `Authorization: Bearer <token>` header

---

## Organization Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| GET | `/org` | Get organization details | Organization members |
| PUT | `/org` | Update organization | Organization admins |

---

## Location Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| GET | `/locations` | List all locations in organization | Organization members |
| POST | `/locations` | Create new location | Organization admins |
| GET | `/locations/{id}` | Get location details | Organization members |
| PUT | `/locations/{id}` | Update location | Organization admins |

---

## Role Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| GET | `/roles` | List organization roles | Organization members |
| POST | `/roles` | Create custom role | Organization admins |
| GET | `/roles/{id}` | Get role details | Organization members |
| PUT | `/roles/{id}` | Update role | Organization admins |
| POST | `/roles/{id}/permissions` | Assign permission to role | Organization admins |

---

## Permission Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| GET | `/permissions` | List all permissions | Organization admins |
| POST | `/permissions` | Create custom permission | Super Admin only |
| GET | `/permissions/{id}` | Get permission details | Organization admins |
| PUT | `/permissions/{id}` | Update permission | Super Admin only |

---

## User Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| GET | `/users` | List organization users | Organization members |
| POST | `/users` | Create new user | Organization admins |
| GET | `/users/{userId}` | Get user details | Organization members |
| PUT | `/users/{userId}` | Update user | Organization admins |
| PATCH | `/users/{userId}/reset-password` | Reset user password | Organization admins |
| PATCH | `/users/{userId}/location` | Update user location | User self or admin |
| PUT | `/users/{userId}/selected-location/{locationId}` | Set user's selected location | User self |

---

## Assignment Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| POST | `/assignments` | Create assignment (assign user to org/location/project) | Context admins |
| GET | `/assignments/{assignmentId}` | Get assignment details | Context members |
| PUT | `/assignments/{assignmentId}` | Update assignment | Context admins |
| DELETE | `/assignments/{assignmentId}` | Remove assignment | Context admins |
| GET | `/contexts/{contextType}/{contextId}/assignments` | Get all assignments for context (organization/location/project) | Context members |

**Context Types:** `organization`, `location`, `project`

---

## Project Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| GET | `/projects` | List projects (filtered by user access) | Organization members |
| POST | `/projects` | Create new project | Location/Org admins |
| GET | `/projects/{projectId}` | Get project details | Project team members |
| PUT | `/projects/{projectId}` | Update project | Project managers |
| GET | `/projects/{projectId}/issues` | Get project issues | Project team members |
| POST | `/projects/{projectId}/issues` | Create issue in project | Project team members |
| GET | `/projects/{projectId}/users` | Get project team | Project team members |
| POST | `/projects/{projectId}/users` | Assign user to project | Project managers |
| PUT | `/projects/{projectId}/users/{assignmentId}` | Update project user role | Project managers |

---

## Issue Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| POST | `/issues` | Create issue | Project team members |
| GET | `/issues/{issueId}` | Get issue details | Project team members |
| PUT | `/issues/{issueId}` | Update issue | Project team members |
| PATCH | `/issues/{issueId}/status` | Update issue status only | Project team members |
| POST | `/issues/{issueId}/comments` | Add comment to issue | Project team members |
| GET | `/issues/{issueId}/comments` | Get issue comments and activity | Project team members |

**Issue Statuses:** `open`, `in_progress`, `ready_for_review`, `closed`, `rejected`, `on_hold`

**Issue Priorities:** `critical`, `high`, `medium`, `low`, `planned`

---

## RFI Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| POST | `/rfis` | Create RFI | Project team members |
| GET | `/rfis/{rfiId}` | Get RFI details | Project team members |
| PUT | `/rfis/{rfiId}` | Update RFI | RFI submitter/assignee |
| POST | `/rfis/{rfiId}/comments` | Add comment to RFI | Project team members |
| GET | `/contexts/{contextType}/{contextId}/rfis` | Get RFIs for project/location/org | Context members |

**RFI Statuses:** `draft`, `open`, `in_review`, `answered`, `closed`

**Context Types:** `project`, `location`, `organization`

---

## Submittal Management

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| POST | `/submittals` | Create submittal | Project team members |
| GET | `/submittals/{submittalId}` | Get submittal details | Project team members |
| PUT | `/submittals/{submittalId}` | Update submittal | Submittal creator |
| POST | `/submittals/{submittalId}/workflow` | Execute workflow action (submit/review/approve/reject) | Workflow assignees |
| GET | `/contexts/{contextType}/{contextId}/submittals` | Get submittals for project/location/org | Context members |
| GET | `/contexts/{contextType}/{contextId}/submittals/stats` | Get submittal statistics | Context members |
| GET | `/contexts/{contextType}/{contextId}/submittals/export` | Export submittals (CSV/Excel) | Context members |

**Submittal Statuses:** `draft`, `submitted`, `under_review`, `approved`, `approved_as_noted`, `rejected`, `revise_and_resubmit`

**Workflow Actions:** `submit_for_review`, `approve`, `approve_as_noted`, `reject`, `revise_and_resubmit`

---

## Attachment Management (Centralized)

| Method | Path | Description | Access Control |
|--------|------|-------------|----------------|
| POST | `/attachments/upload-url` | Generate pre-signed S3 upload URL | Authenticated users |
| POST | `/attachments/confirm` | Confirm attachment upload and save metadata | Authenticated users |
| GET | `/attachments/{id}` | Get attachment metadata | Entity access |
| DELETE | `/attachments/{id}` | Delete attachment | Attachment uploader or admin |
| GET | `/attachments/{id}/download-url` | Generate pre-signed download URL | Entity access |
| GET | `/entities/{type}/{id}/attachments` | Get all attachments for entity | Entity access |

**Entity Types:** `issue`, `issue_comment`, `rfi`, `rfi_comment`, `submittal`, `project`

**Attachment Types:**
- Issues: `before_photo`, `progress_photo`, `after_photo`, `issue_document`
- Comments: `photo`, `document`
- RFIs: `document`, `drawing`, `specification`
- Submittals: `technical_data`, `product_data`, `sample`, `shop_drawing`, `other`

---

## Request/Response Patterns

### Standard Success Responses

```json
// Single resource
{
  "message": "Success message",
  "data": { /* resource object */ }
}

// List of resources
{
  "message": "Success message",
  "data": [ /* array of resources */ ],
  "count": 42
}
```

### Pagination (where applicable)

Query parameters: `?limit=50&offset=0`

### Filtering (where applicable)

- **Projects:** `?location_id=6`
- **Issues:** `?status=open&priority=high&assigned_to=19`
- **RFIs:** `?status=open`
- **Submittals:** `?status=submitted`

### Sorting (where applicable)

Query parameter: `?sort_by=created_at&sort_order=desc`

---

## Common HTTP Status Codes

| Code | Meaning | Usage |
|------|---------|-------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST |
| 400 | Bad Request | Invalid request body or parameters |
| 403 | Forbidden | User lacks permission for resource |
| 404 | Not Found | Resource doesn't exist |
| 500 | Internal Server Error | Server-side error |

---

## Notes

1. **Access Control:** All endpoints enforce hierarchical access control via `iam.user_assignments` table
2. **Soft Deletes:** DELETE endpoints set `is_deleted=true` rather than hard deleting
3. **Audit Fields:** All mutations include `created_by`, `updated_by`, `created_at`, `updated_at`
4. **Organization Isolation:** All data is scoped to user's organization (multi-tenant)
5. **Super Admin:** Super Admin users can access all organizations
6. **Context Types:** `organization`, `location`, `project` define access scope levels

---

**Last Updated:** 2025-10-27