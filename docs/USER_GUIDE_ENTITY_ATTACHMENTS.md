# User Guide: Creating Entities with Attachments

## Overview

This guide explains how to create Issues, RFIs, and Submittals with attachments in the Construction Management System. The process follows industry-standard workflows used by Procore, Bluebeam, and other construction management platforms.

---

## 🔑 Prerequisites

### 1. Authentication
You need a valid JWT ID token from Cognito:

```bash
# Get authentication token
aws cognito-idp admin-initiate-auth \
  --user-pool-id <USER_POOL_ID> \
  --client-id <CLIENT_ID> \
  --auth-flow ADMIN_NO_SRP_AUTH \
  --auth-parameters USERNAME=<email>,PASSWORD=<password> \
  --profile dev \
  --region us-east-2 \
  --query 'AuthenticationResult.IdToken' \
  --output text
```

### 2. Required Context
- **User ID**: From JWT token claims
- **Organization ID**: From JWT token claims
- **Project ID**: Valid project the user has access to
- **Location ID**: Valid location within the project

---

## 📋 Workflow Overview

Creating an entity with attachments is a **2-step process**:

```
Step 1: Create Entity (Issue/RFI/Submittal)
   ↓
Step 2: Upload Attachment(s) to the Entity
   ↓
Step 3: Retrieve Entity (with attachments included)
```

This approach allows:
- ✅ Creating drafts and adding attachments later
- ✅ Handling large construction files efficiently
- ✅ Adding multiple attachments over time
- ✅ Recovering from failed uploads without losing entity data

---

## 🔧 1. ISSUE with Attachments

### Step 1: Create Issue

**Endpoint:** `POST /projects/{projectId}/issues`

**Request:**
```json
{
  "title": "Wall crack in conference room",
  "description": "Large crack observed in west wall",
  "issue_type": "deficiency",
  "priority": "high",
  "due_date": "2025-10-15",
  "assigned_to": "25",
  "location_id": 24
}
```

**Response:**
```json
{
  "id": 72,
  "issue_number": "PROJ-2025-0005-DE-0001",
  "title": "Wall crack in conference room",
  "status": "open",
  "priority": "high",
  "project_id": 49,
  "created_at": "2025-10-06T20:15:30Z"
}
```

**Save the `id` field (72) - you'll need it for attachments.**

---

### Step 2: Upload Attachment(s)

For each file you want to attach:

#### Step 2a: Generate Upload URL

**Endpoint:** `POST /attachments/upload-url`

**Request:**
```json
{
  "entity_type": "issue",
  "entity_id": 72,
  "project_id": 49,
  "location_id": 24,
  "file_name": "wall_crack_photo.jpg",
  "file_size": 524288,
  "file_type": "image/jpeg",
  "attachment_type": "before_photo",
  "description": "Photo of crack before repair"
}
```

**Valid `attachment_type` values for Issues:**
- `before_photo` - Photo taken before work
- `after_photo` - Photo taken after work
- `document` - General document
- `drawing` - Technical drawing
- `other` - Other file types

**Response:**
```json
{
  "attachment_id": 6,
  "upload_url": "https://s3.amazonaws.com/...",
  "file_path": "10/24/49/issues/72/20251006201530_wall_crack_photo.jpg",
  "expires_in": 300
}
```

#### Step 2b: Upload File to S3

```bash
curl -X PUT "<upload_url>" \
  -H "Content-Type: image/jpeg" \
  --data-binary @wall_crack_photo.jpg
```

#### Step 2c: Confirm Upload

**Endpoint:** `POST /attachments/{attachmentId}/confirm`

**Request:**
```json
{
  "entity_type": "issue"
}
```

**Response:**
```json
{
  "message": "Upload confirmed successfully",
  "attachment_id": 6,
  "status": "active"
}
```

---

### Step 3: Retrieve Issue with Attachments

**Endpoint:** `GET /projects/{projectId}/issues/{issueId}`

**Response:**
```json
{
  "id": 72,
  "issue_number": "PROJ-2025-0005-DE-0001",
  "title": "Wall crack in conference room",
  "status": "open",
  "priority": "high",
  "attachments": [
    {
      "id": 6,
      "issue_id": 72,
      "file_name": "wall_crack_photo.jpg",
      "file_path": "10/24/49/issues/72/20251006201530_wall_crack_photo.jpg",
      "file_size": 524288,
      "file_type": "image/jpeg",
      "attachment_type": "before_photo",
      "description": "Photo of crack before repair",
      "uploaded_by": 19,
      "created_at": "2025-10-06T20:15:30Z"
    }
  ]
}
```

---

## 📝 2. RFI with Attachments

### Step 1: Create RFI

**Endpoint:** `POST /projects/{projectId}/rfis`

**Request:**
```json
{
  "subject": "Foundation rebar spacing clarification",
  "question": "What is the required rebar spacing for the foundation at Grid A?",
  "priority": "HIGH",
  "category": "DESIGN",
  "due_date": "2025-10-20",
  "assigned_to": "30",
  "location_id": 24,
  "distribution_list": ["engineer@example.com", "pm@example.com"]
}
```

**Valid `priority` values:**
- `LOW`, `MEDIUM`, `HIGH`, `URGENT`

**Valid `category` values:**
- `DESIGN`, `SPECIFICATION`, `SCHEDULE`, `COORDINATION`, `GENERAL`, `SUBMITTAL`, `CHANGE_EVENT`

**Response:**
```json
{
  "id": 81,
  "rfi_number": "RFI-2025-0023",
  "subject": "Foundation rebar spacing clarification",
  "status": "DRAFT",
  "priority": "HIGH",
  "project_id": 49,
  "created_at": "2025-10-06T20:20:00Z"
}
```

**Save the `id` field (81) - you'll need it for attachments.**

---

### Step 2: Upload Attachment(s)

#### Step 2a: Generate Upload URL

**Endpoint:** `POST /attachments/upload-url`

**Request:**
```json
{
  "entity_type": "rfi",
  "entity_id": 81,
  "project_id": 49,
  "location_id": 24,
  "file_name": "foundation_plan.pdf",
  "file_size": 1048576,
  "file_type": "application/pdf",
  "attachment_type": "drawing",
  "description": "Foundation plan sheet A-101"
}
```

**Valid `attachment_type` values for RFIs:**
- `question` - Supporting documentation for the question
- `specification` - Specification document
- `drawing` - Technical drawing
- `document` - General document
- `photo` - Photograph
- `other` - Other file types

**Response:**
```json
{
  "attachment_id": 1,
  "upload_url": "https://s3.amazonaws.com/...",
  "file_path": "10/24/49/rfis/81/20251006202000_foundation_plan.pdf",
  "expires_in": 300
}
```

#### Step 2b: Upload File to S3

```bash
curl -X PUT "<upload_url>" \
  -H "Content-Type: application/pdf" \
  --data-binary @foundation_plan.pdf
```

#### Step 2c: Confirm Upload

**Endpoint:** `POST /attachments/{attachmentId}/confirm`

**Request:**
```json
{
  "entity_type": "rfi"
}
```

---

### Step 3: Retrieve RFI with Attachments

**Endpoint:** `GET /projects/{projectId}/rfis/{rfiId}`

**Response:**
```json
{
  "id": 81,
  "rfi_number": "RFI-2025-0023",
  "subject": "Foundation rebar spacing clarification",
  "status": "DRAFT",
  "priority": "HIGH",
  "attachments": [
    {
      "id": 1,
      "rfi_id": 81,
      "file_name": "foundation_plan.pdf",
      "file_path": "10/24/49/rfis/81/20251006202000_foundation_plan.pdf",
      "file_size": 1048576,
      "file_type": "application/pdf",
      "attachment_type": "drawing",
      "description": "Foundation plan sheet A-101",
      "uploaded_by": 19,
      "created_at": "2025-10-06T20:20:00Z"
    }
  ]
}
```

---

## 📤 3. SUBMITTAL with Attachments

### Step 1: Create Submittal

**Endpoint:** `POST /projects/{projectId}/submittals`

**Request:**
```json
{
  "title": "Structural Steel Shop Drawings",
  "description": "Shop drawings for structural steel beams and columns",
  "spec_section": "05 12 00",
  "submittal_type": "shop_drawing",
  "priority": "high",
  "required_by_date": "2025-10-25",
  "submitted_by": "contractor@example.com",
  "location_id": 24,
  "lead_time_days": 14
}
```

**Valid `submittal_type` values:**
- `shop_drawing`, `product_data`, `sample`, `mix_design`, `test_report`, `certificate`, `other`

**Valid `priority` values:**
- `low`, `medium`, `high`, `urgent`

**Response:**
```json
{
  "id": 9,
  "submittal_number": "SUB-2025-004",
  "title": "Structural Steel Shop Drawings",
  "status": "draft",
  "priority": "high",
  "project_id": 49,
  "created_at": "2025-10-06T20:25:00Z"
}
```

**Save the `id` field (9) - you'll need it for attachments.**

---

### Step 2: Upload Attachment(s)

#### Step 2a: Generate Upload URL

**Endpoint:** `POST /attachments/upload-url`

**Request:**
```json
{
  "entity_type": "submittal",
  "entity_id": 9,
  "project_id": 49,
  "location_id": 24,
  "file_name": "steel_beam_details.dwg",
  "file_size": 2097152,
  "file_type": "application/acad",
  "attachment_type": "shop_drawing",
  "description": "W-beam connection details"
}
```

**Valid `attachment_type` values for Submittals:**
- `shop_drawing` - Shop drawing file
- `product_data` - Product data sheet
- `sample_photo` - Photo of sample
- `calculation` - Engineering calculation
- `certificate` - Certificate document
- `test_report` - Test report document
- `document` - General document
- `other` - Other file types

**Response:**
```json
{
  "attachment_id": 1,
  "upload_url": "https://s3.amazonaws.com/...",
  "file_path": "10/24/49/submittals/9/20251006202500_steel_beam_details.dwg",
  "expires_in": 300
}
```

#### Step 2b: Upload File to S3

```bash
curl -X PUT "<upload_url>" \
  -H "Content-Type: application/acad" \
  --data-binary @steel_beam_details.dwg
```

#### Step 2c: Confirm Upload

**Endpoint:** `POST /attachments/{attachmentId}/confirm`

**Request:**
```json
{
  "entity_type": "submittal"
}
```

---

### Step 3: Retrieve Submittal with Attachments

**Endpoint:** `GET /projects/{projectId}/submittals/{submittalId}`

**Response:**
```json
{
  "id": 9,
  "submittal_number": "SUB-2025-004",
  "title": "Structural Steel Shop Drawings",
  "status": "draft",
  "priority": "high",
  "attachments": [
    {
      "id": 1,
      "submittal_id": 9,
      "file_name": "steel_beam_details.dwg",
      "file_path": "10/24/49/submittals/9/20251006202500_steel_beam_details.dwg",
      "file_size": 2097152,
      "file_type": "application/acad",
      "attachment_type": "shop_drawing",
      "description": "W-beam connection details",
      "uploaded_by": 19,
      "created_at": "2025-10-06T20:25:00Z"
    }
  ]
}
```

---

## 📥 4. Downloading Attachments

To download an attachment, generate a download URL:

**Endpoint:** `POST /attachments/{attachmentId}/download-url`

**Request:**
```json
{
  "entity_type": "issue"
}
```

**Response:**
```json
{
  "download_url": "https://s3.amazonaws.com/...",
  "file_name": "wall_crack_photo.jpg",
  "file_size": 524288,
  "expires_in": 300
}
```

**Download the file:**
```bash
curl "<download_url>" -o wall_crack_photo.jpg
```

---

## 🗑️ 5. Removing Attachments

To soft-delete an attachment:

**Endpoint:** `DELETE /attachments/{attachmentId}`

**Query Parameters:**
- `entity_type=issue` (or `rfi`, `submittal`)

**Response:**
```json
{
  "message": "Attachment soft deleted successfully",
  "attachment_id": 6
}
```

**Note:** This is a soft delete - the file remains in S3 but is marked as deleted in the database.

---

## 🔍 6. Listing All Attachments for an Entity

**For Issues:**
```
GET /attachments?entity_type=issue&entity_id=72
```

**For RFIs:**
```
GET /attachments?entity_type=rfi&entity_id=81
```

**For Submittals:**
```
GET /attachments?entity_type=submittal&entity_id=9
```

**Response:**
```json
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
  "total_count": 1
}
```

---

## 📱 Best Practices

### 1. **Draft → Attach → Submit Workflow**
```
1. Create entity in DRAFT status
2. Upload all required attachments
3. Submit/Update entity to SUBMITTED status
```

### 2. **Multiple Attachments**
Upload multiple files by repeating Step 2 for each file:
```
- Create entity once
- Upload file 1 (generate URL → upload → confirm)
- Upload file 2 (generate URL → upload → confirm)
- Upload file 3 (generate URL → upload → confirm)
- Retrieve entity (all 3 attachments included)
```

### 3. **Large Files**
For files > 100MB:
- Split upload into chunks if needed
- Show progress bar to users
- Allow resume on network failure

### 4. **Mobile Optimization**
- Compress images before upload (if appropriate)
- Use presigned URLs for direct S3 upload (faster than API Gateway)
- Cache entity data locally, sync attachments in background

### 5. **Error Handling**
```javascript
// If attachment upload fails
try {
  // Step 1: Entity created ✓
  const entity = await createIssue(data);

  // Step 2a: Generate URL ✓
  const uploadUrl = await generateUploadUrl(entity.id, file);

  // Step 2b: Upload to S3 (FAILS ✗)
  await uploadToS3(uploadUrl, file);
} catch (error) {
  // Entity still exists - user can retry upload
  // No need to recreate the entire entity
  console.log("Upload failed, but issue was created. Retry upload.");
}
```

### 6. **File Type Validation**
Validate file types before upload:

**Images:** `.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`
**Documents:** `.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`
**Drawings:** `.dwg`, `.dxf`, `.rvt`, `.pdf`
**Other:** `.zip`, `.txt`, `.csv`

### 7. **File Size Limits**
- **Maximum file size:** 100MB per file
- **Maximum attachments per entity:** No hard limit (recommended: 20-30)
- **Total storage per project:** Check with administrator

---

## 🚨 Common Issues & Solutions

### Issue: "Entity not found"
**Solution:** Make sure you're using the correct `entity_id` from Step 1 response.

### Issue: "Upload URL expired"
**Solution:** Upload URLs expire in 5 minutes. Generate a new URL if expired.

### Issue: "Attachment type not allowed"
**Solution:** Check the valid `attachment_type` values for your entity type.

### Issue: "File not uploaded to S3"
**Solution:** Ensure you complete Step 2b (S3 upload) before Step 2c (confirm).

### Issue: "Attachments not showing in GET response"
**Solution:** Make sure you called the `/confirm` endpoint after uploading to S3.

### Issue: "Invalid file_path"
**Solution:** Don't modify the `file_path` - it's auto-generated by the system.

---

## 📊 Complete Example: JavaScript/React

```javascript
// Complete workflow example
async function createIssueWithAttachments(issueData, files) {
  try {
    // Step 1: Create Issue
    const issue = await fetch(`/projects/${projectId}/issues`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(issueData)
    }).then(r => r.json());

    console.log('Issue created:', issue.id);

    // Step 2: Upload each attachment
    for (const file of files) {
      // Step 2a: Generate upload URL
      const uploadData = await fetch('/attachments/upload-url', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          entity_type: 'issue',
          entity_id: issue.id,
          project_id: projectId,
          location_id: locationId,
          file_name: file.name,
          file_size: file.size,
          file_type: file.type,
          attachment_type: 'before_photo'
        })
      }).then(r => r.json());

      // Step 2b: Upload to S3
      await fetch(uploadData.upload_url, {
        method: 'PUT',
        headers: { 'Content-Type': file.type },
        body: file
      });

      // Step 2c: Confirm upload
      await fetch(`/attachments/${uploadData.attachment_id}/confirm`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ entity_type: 'issue' })
      });

      console.log('Attachment uploaded:', file.name);
    }

    // Step 3: Retrieve issue with attachments
    const issueWithAttachments = await fetch(
      `/projects/${projectId}/issues/${issue.id}`,
      {
        headers: { 'Authorization': `Bearer ${token}` }
      }
    ).then(r => r.json());

    console.log('Issue with attachments:', issueWithAttachments);
    return issueWithAttachments;

  } catch (error) {
    console.error('Error:', error);
    throw error;
  }
}

// Usage
const issueData = {
  title: "Wall crack in conference room",
  description: "Large crack observed",
  issue_type: "deficiency",
  priority: "high"
};

const files = [
  document.getElementById('file1').files[0],
  document.getElementById('file2').files[0]
];

createIssueWithAttachments(issueData, files);
```

---

## 📚 Additional Resources

- **API Base URL:** `https://z1pbmjzrb6.execute-api.us-east-2.amazonaws.com/prod`
- **Postman Collection:** `/postman/AttachmentManagement.postman_collection.json`
- **Test Scripts:** `/testing/api/`

---

## 🎯 Summary

**The 2-step process is intentional and follows industry standards:**

1. ✅ **Create entity first** - Get an ID, save as draft
2. ✅ **Upload attachments** - Add files one by one
3. ✅ **Retrieve entity** - GET endpoint includes all attachment metadata

This approach provides flexibility, error recovery, and efficient handling of large construction files.