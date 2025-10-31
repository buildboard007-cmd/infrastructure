package models

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Attachment represents a file attachment for any entity type
type Attachment struct {
	ID             int64     `json:"id"`
	EntityType     string    `json:"entity_type"`     // "project", "issue", "rfi", "submittal"
	EntityID       int64     `json:"entity_id"`       // ID of the entity (project_id, issue_id, etc.)
	ProjectID      int64     `json:"project_id"`      // Always present for hierarchy
	LocationID     int64     `json:"location_id"`     // Always present for hierarchy
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

// AttachmentUploadRequest represents a request to get an upload URL
type AttachmentUploadRequest struct {
	EntityType     string `json:"entity_type" binding:"required,oneof=project issue rfi submittal issue_comment rfi_comment"`
	EntityID       int64  `json:"entity_id"` // Required for most types, can be 0 for issue_comment/rfi_comment (updated after comment creation)
	ProjectID      int64  `json:"project_id" binding:"required"`
	LocationID     int64  `json:"location_id" binding:"required"`
	OrgID          int64  `json:"org_id,omitempty"` // Set from JWT claims
	FileName       string `json:"file_name" binding:"required,max=255"`
	FileSize       int64  `json:"file_size" binding:"required,max=104857600"` // 100MB max
	AttachmentType string `json:"attachment_type" binding:"required"`
}

// AttachmentUploadResponse represents the response with presigned URL
type AttachmentUploadResponse struct {
	AttachmentID int64  `json:"attachment_id"`
	UploadURL    string `json:"upload_url"`
	S3Key        string `json:"s3_key"`
	ExpiresAt    string `json:"expires_at"`
}

// AttachmentConfirmRequest represents a request to confirm upload completion
type AttachmentConfirmRequest struct {
	AttachmentID int64 `json:"attachment_id" binding:"required"`
}

// AttachmentDownloadResponse represents the response with download URL
type AttachmentDownloadResponse struct {
	DownloadURL string `json:"download_url"`
	FileName    string `json:"file_name"`
	FileSize    *int64 `json:"file_size,omitempty"`
	ExpiresAt   string `json:"expires_at"`
}

// AttachmentListResponse represents a paginated list of attachments
type AttachmentListResponse struct {
	Attachments []Attachment `json:"attachments"`
	TotalCount  int          `json:"total_count"`
	Page        int          `json:"page,omitempty"`
	PageSize    int          `json:"page_size,omitempty"`
	HasNext     bool         `json:"has_next"`
	HasPrev     bool         `json:"has_previous"`
}

// Attachment Type constants
const (
	// Project attachment types
	AttachmentTypeProjectDocument = "project_document"
	AttachmentTypeProjectDrawing  = "project_drawing"
	AttachmentTypeProjectPhoto    = "project_photo"
	AttachmentTypeProjectReport   = "project_report"

	// Issue attachment types
	AttachmentTypeBeforePhoto   = "before_photo"
	AttachmentTypeProgressPhoto = "progress_photo"
	AttachmentTypeAfterPhoto    = "after_photo"
	AttachmentTypeIssueDocument = "issue_document"

	// RFI attachment types
	AttachmentTypeRFIQuestion      = "rfi_question"
	AttachmentTypeRFIResponse      = "rfi_response"
	AttachmentTypeRFISupportingDoc = "rfi_supporting_doc"

	// Submittal attachment types
	AttachmentTypeShopDrawing       = "shop_drawing"
	AttachmentTypeProductData       = "product_data"
	AttachmentTypeSample            = "sample"
	AttachmentTypeCertificate       = "certificate"
	AttachmentTypeSubmittalDocument = "submittal_document"

	// General types
	AttachmentTypeOther = "other"
)

// Upload Status constants
const (
	UploadStatusPending  = "pending"
	UploadStatusUploaded = "uploaded"
	UploadStatusFailed   = "failed"
)

// Entity Type constants
const (
	EntityTypeProject      = "project"
	EntityTypeIssue        = "issue"
	EntityTypeRFI          = "rfi"
	EntityTypeSubmittal    = "submittal"
	EntityTypeIssueComment = "issue_comment"
	EntityTypeRFIComment   = "rfi_comment"
)

// GenerateS3Key creates the S3 key based on the hierarchical path structure
func (req *AttachmentUploadRequest) GenerateS3Key() string {
	timestamp := time.Now().Format("20060102150405")
	cleanFileName := strings.ReplaceAll(req.FileName, " ", "_")

	switch req.EntityType {
	case EntityTypeProject:
		// Project's own attachments go in /attachments/ subfolder
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
		// For issue_comment, entity_id will be 0 initially, will use temp path
		if req.EntityID == 0 {
			return fmt.Sprintf("%d/%d/%d/comments/temp/%s_%s",
				req.OrgID, req.LocationID, req.ProjectID, timestamp, cleanFileName)
		}
		return fmt.Sprintf("%d/%d/%d/comments/%d/%s_%s",
			req.OrgID, req.LocationID, req.ProjectID, req.EntityID, timestamp, cleanFileName)
	case EntityTypeRFIComment:
		// For rfi_comment, entity_id will be 0 initially, will use temp path
		if req.EntityID == 0 {
			return fmt.Sprintf("%d/%d/%d/rfi_comments/temp/%s_%s",
				req.OrgID, req.LocationID, req.ProjectID, timestamp, cleanFileName)
		}
		return fmt.Sprintf("%d/%d/%d/rfi_comments/%d/%s_%s",
			req.OrgID, req.LocationID, req.ProjectID, req.EntityID, timestamp, cleanFileName)
	default:
		return ""
	}
}

// GetTableName returns the appropriate database table name for the entity type
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
	case EntityTypeRFIComment:
		return "project.rfi_comment_attachments"
	default:
		return ""
	}
}

// GetEntityIDColumn returns the appropriate foreign key column name for the entity type
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
	case EntityTypeRFIComment:
		return "comment_id"
	default:
		return ""
	}
}

// ValidateFileType checks if the file type is allowed
func ValidateFileType(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))

	allowedExtensions := map[string]bool{
		// Documents
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".txt": true, ".rtf": true,
		// Images
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".tiff": true, ".webp": true,
		// Drawings
		".dwg": true, ".dxf": true, ".dwf": true, ".rvt": true,
		// Archives
		".zip": true, ".rar": true, ".7z": true,
		// 3D Models
		".ifc": true, ".skp": true, ".3ds": true, ".obj": true,
	}

	return allowedExtensions[ext]
}

// GetMimeType returns the MIME type for a file based on its extension
func GetMimeType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))

	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".txt":  "text/plain",
		".rtf":  "application/rtf",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
		".webp": "image/webp",
		".dwg":  "application/acad",
		".dxf":  "application/dxf",
		".dwf":  "model/vnd.dwf",
		".rvt":  "application/vnd.autodesk.revit",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".ifc":  "application/x-step",
		".skp":  "application/vnd.sketchup.skp",
		".3ds":  "application/x-3ds",
		".obj":  "application/x-tgif",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}