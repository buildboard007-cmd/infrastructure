# Attachment Management API - Testing Guide

## ✅ Test Results Summary

**All tests passed successfully!** ✓

- ✓ Authentication
- ✓ Generate upload URL (project, issue, RFI, submittal)
- ✓ Upload file to S3
- ✓ Confirm upload
- ✓ Get attachment metadata
- ✓ Generate download URL
- ✓ List entity attachments
- ✓ Soft delete attachment
- ✓ Error validation (invalid entity type, file size, file type)

---

## 📋 Step-by-Step Testing Instructions

### Prerequisites

1. **Postman** installed
2. **Valid test user**: `buildboard007+555@gmail.com` / `Mayur@1234`
3. **API Base URL**: `https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`

---

### Step 1: Get Authentication Token

Use the **Infrastructure.postman_collection.json** → Authentication → SuperAdmin Sign In:

```http
POST https://cognito-idp.us-east-2.amazonaws.com/
Headers:
  Content-Type: application/x-amz-json-1.1
  X-Amz-Target: AWSCognitoIdentityProviderService.InitiateAuth

Body:
{
  "AuthFlow": "USER_PASSWORD_AUTH",
  "ClientId": "3f0fb5mpivctnvj85tucusf88e",
  "AuthParameters": {
    "USERNAME": "buildboard007+555@gmail.com",
    "PASSWORD": "Mayur@1234"
  }
}
```

**Expected Response:**
```json
{
  "AuthenticationResult": {
    "IdToken": "eyJra...",
    "AccessToken": "eyJra...",
    "RefreshToken": "eyJra..."
  }
}
```

**Action:** Copy the `IdToken` value and set it as `{{access_token}}` variable in Postman.

---

### Step 2: Generate Upload URL - Project Attachment

**Request:**
```http
POST /attachments/upload-url
Headers:
  Authorization: Bearer {{access_token}}
  Content-Type: application/json

Body:
{
  "entity_type": "project",
  "entity_id": 49,
  "project_id": 49,
  "location_id": 24,
  "file_name": "floor_plan_v2.pdf",
  "file_size": 2048576,
  "attachment_type": "drawing"
}
```

**Expected Response (200 OK):**
```json
{
  "attachment_id": 3,
  "upload_url": "https://s3.us-east-2.amazonaws.com/buildboard-attachments-dev/...",
  "s3_key": "10/24/49/attachments/20251006184456_test_floor_plan.pdf",
  "expires_at": "2025-10-06T18:59:56Z"
}
```

**Action:**
1. Copy `attachment_id` → Set as `{{attachment_id}}` variable
2. Copy `upload_url` for next step

---

### Step 3: Upload File to S3

**Request:**
```http
PUT {upload_url from Step 2}
Headers:
  Content-Type: application/pdf

Body: (Binary file)
```

**How to test in Postman:**
1. Create new request
2. Method: PUT
3. URL: Paste the `upload_url` from Step 2
4. Headers → Content-Type: `application/pdf`
5. Body → binary → Select a PDF file
6. Send

**Expected Response:** 200 OK (empty body)

---

### Step 4: Confirm Upload

**Request:**
```http
POST /attachments/confirm
Headers:
  Authorization: Bearer {{access_token}}
  Content-Type: application/json

Body:
{
  "attachment_id": {{attachment_id}}
}
```

**Expected Response (200 OK):**
```json
{
  "status": "confirmed"
}
```

---

### Step 5: Get Attachment Metadata

**Request:**
```http
GET /attachments/{{attachment_id}}?entity_type=project
Headers:
  Authorization: Bearer {{access_token}}
```

**Expected Response (200 OK):**
```json
{
  "id": 3,
  "entity_type": "project",
  "entity_id": 49,
  "file_name": "test_floor_plan.pdf",
  "file_path": "10/24/49/attachments/20251006184456_test_floor_plan.pdf",
  "file_size": 102400,
  "file_type": "application/pdf",
  "attachment_type": "drawing",
  "uploaded_by": 19,
  "created_at": "2025-10-06T18:44:56.579154Z",
  "is_deleted": false
}
```

---

### Step 6: Generate Download URL

**Request:**
```http
GET /attachments/{{attachment_id}}/download-url?entity_type=project
Headers:
  Authorization: Bearer {{access_token}}
```

**Expected Response (200 OK):**
```json
{
  "download_url": "https://s3.us-east-2.amazonaws.com/buildboard-attachments-dev/...",
  "file_name": "test_floor_plan.pdf",
  "file_size": 102400,
  "expires_at": "2025-10-06T19:44:57Z"
}
```

**Action:** Copy `download_url` and paste in browser to download the file.

---

### Step 7: List Entity Attachments

**Request:**
```http
GET /entities/project/{{project_id}}/attachments?page=1&limit=20
Headers:
  Authorization: Bearer {{access_token}}
```

**Expected Response (200 OK):**
```json
{
  "attachments": [
    {
      "id": 3,
      "entity_type": "project",
      "file_name": "test_floor_plan.pdf",
      "attachment_type": "drawing",
      ...
    }
  ],
  "total_count": 1,
  "page": 1,
  "page_size": 20,
  "has_next": false,
  "has_previous": false
}
```

---

### Step 8: Soft Delete Attachment

**Request:**
```http
DELETE /attachments/{{attachment_id}}?entity_type=project
Headers:
  Authorization: Bearer {{access_token}}
```

**Expected Response (200 OK):**
```json
{
  "status": "deleted"
}
```

---

## 🎯 Valid attachment_type Values

Each entity type has specific allowed attachment_type values enforced by database constraints:

### Project Attachments
- `logo`
- `project_photo`
- `document`
- `drawing`
- `other`

### Issue Attachments
- `before_photo`
- `after_photo`
- `document`
- `drawing_markup`

### RFI Attachments
- `document`
- `drawing`
- `photo`
- `specification`
- `calculation`
- `sketch`
- `other`

### Submittal Attachments
- `shop_drawing`
- `product_data`
- `specification`
- `sample_photo`
- `certificate`
- `test_report`
- `other`

---

## ⚠️ Common Errors & Solutions

### Error: "Failed to create attachment" (attachment_type constraint)

**Cause:** Using invalid `attachment_type` value.

**Solution:** Use only the values listed above for each entity type.

**Example:**
```json
❌ "attachment_type": "project_drawing"  // Invalid
✓ "attachment_type": "drawing"           // Correct
```

---

### Error: Foreign key constraint violation

**Cause:** Referenced entity (project_id, issue_id, etc.) doesn't exist or belongs to different org.

**Solution:** Use existing entity IDs from your organization.

**Find valid projects:**
```sql
SELECT id, name, org_id, location_id
FROM project.projects
WHERE org_id = 10 AND is_deleted = false;
```

---

### Error: "File type not allowed"

**Cause:** Uploading unsupported file type.

**Solution:** Use only allowed file extensions:
- Documents: `.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.txt`, `.rtf`
- Images: `.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`, `.tiff`, `.webp`
- Drawings: `.dwg`, `.dxf`, `.dwf`, `.rvt`
- Archives: `.zip`, `.rar`, `.7z`
- 3D Models: `.ifc`, `.skp`, `.3ds`, `.obj`

---

### Error: "File size limit exceeded"

**Cause:** File size > 100MB

**Solution:** File size must be ≤ 104857600 bytes (100MB)

---

## 🔐 Security Notes

1. **Authentication:** All endpoints require JWT ID token (not access token)
2. **Organization isolation:** Users can only access attachments from their organization
3. **Entity type required:** GET/DELETE operations require `?entity_type=` query parameter
4. **Presigned URL expiry:**
   - Upload URLs: 15 minutes
   - Download URLs: 60 minutes

---

## 📁 S3 File Structure

Files are stored with hierarchical structure:

```
{org_id}/{location_id}/{project_id}/{entity_type}/{entity_id}/{timestamp}_{filename}
```

**Examples:**
- Project: `10/24/49/attachments/20251006184456_floor_plan.pdf`
- Issue: `10/24/49/issues/101/20251006184456_crack_photo.jpg`
- RFI: `10/24/49/rfis/201/20251006184456_clarification.pdf`
- Submittal: `10/24/49/submittals/301/20251006184456_shop_drawing.dwg`

---

## 🚀 Automated Testing

Run the complete test suite:

```bash
cd /Users/mayur/git_personal/infrastructure/testing/api
./test-attachment-api.sh
```

**Test Coverage:**
- ✓ Authentication
- ✓ Upload workflow (generate URL → upload → confirm)
- ✓ Metadata retrieval
- ✓ Download URL generation
- ✓ Entity listing
- ✓ Soft deletion
- ✓ Error validation (invalid entity type, file size, file type)

---

## 📊 Test Data

**Test User:**
- Email: `buildboard007+555@gmail.com`
- Password: `Mayur@1234`
- User ID: `19`
- Org ID: `10`
- Location ID: `24`

**Test Project:**
- Project ID: `49`
- Name: "TEST assdfsdfsdf"
- Location ID: `24`
- Org ID: `10`

---

## ✅ Verification Checklist

- [ ] Can generate upload URL for all entity types (project, issue, RFI, submittal)
- [ ] Can upload file to S3 using presigned URL
- [ ] Can confirm upload completion
- [ ] Can retrieve attachment metadata
- [ ] Can generate download URL
- [ ] Can download file from S3
- [ ] Can list attachments for each entity
- [ ] Can soft delete attachments
- [ ] Invalid entity types are rejected
- [ ] File size limits are enforced (100MB max)
- [ ] Invalid file types are rejected
- [ ] Only valid attachment_type values are accepted
- [ ] Organization isolation is enforced
- [ ] Presigned URLs expire as expected

---

## 🔗 Related Files

- **Postman Collection:** `/postman/AttachmentManagement.postman_collection.json`
- **Test Script:** `/testing/api/test-attachment-api.sh`
- **Lambda Function:** `/src/infrastructure-attachment-management/main.go`
- **Repository:** `/src/lib/data/attachment_repository.go`
- **Models:** `/src/lib/models/attachment.go`
- **S3 Client:** `/src/lib/clients/s3_client.go`

---

**Last Updated:** 2025-10-06
**Test Status:** ✅ All tests passing
