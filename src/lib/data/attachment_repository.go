package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"time"

	"github.com/sirupsen/logrus"
)

// AttachmentRepository defines the interface for attachment operations
type AttachmentRepository interface {
	CreateAttachment(ctx context.Context, attachment *models.Attachment) (*models.Attachment, error)
	GetAttachment(ctx context.Context, attachmentID int64, entityType string) (*models.Attachment, error)
	GetAttachmentsByEntity(ctx context.Context, entityType string, entityID int64, filters map[string]string) ([]models.Attachment, error)
	UpdateAttachmentStatus(ctx context.Context, attachmentID int64, entityType string, status string) error
	SoftDeleteAttachment(ctx context.Context, attachmentID int64, entityType string, userID int64) error
	VerifyAttachmentAccess(ctx context.Context, attachmentID int64, entityType string, orgID int64) (bool, error)
}

// AttachmentDao implements the AttachmentRepository interface
type AttachmentDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreateAttachment creates a new attachment record in the appropriate table
func (dao *AttachmentDao) CreateAttachment(ctx context.Context, attachment *models.Attachment) (*models.Attachment, error) {
	tableName := models.GetTableName(attachment.EntityType)
	entityIDColumn := models.GetEntityIDColumn(attachment.EntityType)

	if tableName == "" || entityIDColumn == "" {
		return nil, fmt.Errorf("unsupported entity type: %s", attachment.EntityType)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, file_name, file_path, file_size, file_type, attachment_type,
			uploaded_by, created_by, updated_by, created_at, updated_at, is_deleted
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at
	`, tableName, entityIDColumn)

	now := time.Now()
	var id int64
	var createdAt, updatedAt time.Time

	err := dao.DB.QueryRowContext(ctx, query,
		attachment.EntityID,
		attachment.FileName,
		attachment.FilePath,
		attachment.FileSize,
		attachment.FileType,
		attachment.AttachmentType,
		attachment.UploadedBy,
		attachment.CreatedBy,
		attachment.UpdatedBy,
		now,
		now,
		false,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"entity_type": attachment.EntityType,
			"entity_id":   attachment.EntityID,
			"file_name":   attachment.FileName,
		}).Error("Failed to create attachment")
		return nil, err
	}

	attachment.ID = id
	attachment.CreatedAt = createdAt
	attachment.UpdatedAt = updatedAt
	attachment.IsDeleted = false

	dao.Logger.WithFields(logrus.Fields{
		"attachment_id": id,
		"entity_type":   attachment.EntityType,
		"entity_id":     attachment.EntityID,
		"file_name":     attachment.FileName,
	}).Info("Attachment created successfully")

	return attachment, nil
}

// GetAttachment retrieves a specific attachment by ID
func (dao *AttachmentDao) GetAttachment(ctx context.Context, attachmentID int64, entityType string) (*models.Attachment, error) {
	tableName := models.GetTableName(entityType)
	entityIDColumn := models.GetEntityIDColumn(entityType)

	if tableName == "" || entityIDColumn == "" {
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}

	query := fmt.Sprintf(`
		SELECT
			id, %s, file_name, file_path, file_size, file_type, attachment_type,
			uploaded_by, created_at, created_by, updated_at, updated_by, is_deleted
		FROM %s
		WHERE id = $1 AND is_deleted = false
	`, entityIDColumn, tableName)

	var attachment models.Attachment
	attachment.EntityType = entityType

	err := dao.DB.QueryRowContext(ctx, query, attachmentID).Scan(
		&attachment.ID,
		&attachment.EntityID,
		&attachment.FileName,
		&attachment.FilePath,
		&attachment.FileSize,
		&attachment.FileType,
		&attachment.AttachmentType,
		&attachment.UploadedBy,
		&attachment.CreatedAt,
		&attachment.CreatedBy,
		&attachment.UpdatedAt,
		&attachment.UpdatedBy,
		&attachment.IsDeleted,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("attachment not found")
		}
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"attachment_id": attachmentID,
			"entity_type":   entityType,
		}).Error("Failed to get attachment")
		return nil, err
	}

	return &attachment, nil
}

// GetAttachmentsByEntity retrieves all attachments for a specific entity
func (dao *AttachmentDao) GetAttachmentsByEntity(ctx context.Context, entityType string, entityID int64, filters map[string]string) ([]models.Attachment, error) {
	tableName := models.GetTableName(entityType)
	entityIDColumn := models.GetEntityIDColumn(entityType)

	if tableName == "" || entityIDColumn == "" {
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}

	// Build query with optional attachment_type filter
	baseQuery := fmt.Sprintf(`
		SELECT
			id, %s, file_name, file_path, file_size, file_type, attachment_type,
			uploaded_by, created_at, created_by, updated_at, updated_by, is_deleted
		FROM %s
		WHERE %s = $1 AND is_deleted = false
	`, entityIDColumn, tableName, entityIDColumn)

	var query string
	var args []interface{}
	args = append(args, entityID)

	// Add attachment_type filter if provided
	if attachmentType, exists := filters["attachment_type"]; exists && attachmentType != "" {
		query = baseQuery + " AND attachment_type = $2 ORDER BY created_at DESC"
		args = append(args, attachmentType)
	} else {
		query = baseQuery + " ORDER BY created_at DESC"
	}

	rows, err := dao.DB.QueryContext(ctx, query, args...)
	if err != nil {
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"entity_type": entityType,
			"entity_id":   entityID,
		}).Error("Failed to get attachments by entity")
		return nil, err
	}
	defer rows.Close()

	var attachments []models.Attachment

	for rows.Next() {
		var attachment models.Attachment
		attachment.EntityType = entityType

		err := rows.Scan(
			&attachment.ID,
			&attachment.EntityID,
			&attachment.FileName,
			&attachment.FilePath,
			&attachment.FileSize,
			&attachment.FileType,
			&attachment.AttachmentType,
			&attachment.UploadedBy,
			&attachment.CreatedAt,
			&attachment.CreatedBy,
			&attachment.UpdatedAt,
			&attachment.UpdatedBy,
			&attachment.IsDeleted,
		)

		if err != nil {
			dao.Logger.WithError(err).WithFields(logrus.Fields{
				"entity_type": entityType,
				"entity_id":   entityID,
			}).Error("Failed to scan attachment row")
			return nil, err
		}

		attachments = append(attachments, attachment)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"entity_type": entityType,
			"entity_id":   entityID,
		}).Error("Row iteration error")
		return nil, err
	}

	dao.Logger.WithFields(logrus.Fields{
		"entity_type":      entityType,
		"entity_id":        entityID,
		"attachments_count": len(attachments),
	}).Debug("Retrieved attachments for entity")

	return attachments, nil
}

// UpdateAttachmentStatus updates the upload status of an attachment
func (dao *AttachmentDao) UpdateAttachmentStatus(ctx context.Context, attachmentID int64, entityType string, status string) error {
	tableName := models.GetTableName(entityType)

	if tableName == "" {
		return fmt.Errorf("unsupported entity type: %s", entityType)
	}

	// Note: Since upload_status column doesn't exist in current schema,
	// we'll skip this for now and assume uploads are successful when confirmed
	dao.Logger.WithFields(logrus.Fields{
		"attachment_id": attachmentID,
		"entity_type":   entityType,
		"status":        status,
	}).Debug("Attachment status update requested (not implemented in current schema)")

	return nil
}

// SoftDeleteAttachment marks an attachment as deleted
func (dao *AttachmentDao) SoftDeleteAttachment(ctx context.Context, attachmentID int64, entityType string, userID int64) error {
	tableName := models.GetTableName(entityType)

	if tableName == "" {
		return fmt.Errorf("unsupported entity type: %s", entityType)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET is_deleted = true, updated_by = $2, updated_at = $3
		WHERE id = $1 AND is_deleted = false
	`, tableName)

	result, err := dao.DB.ExecContext(ctx, query, attachmentID, userID, time.Now())
	if err != nil {
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"attachment_id": attachmentID,
			"entity_type":   entityType,
			"user_id":       userID,
		}).Error("Failed to soft delete attachment")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("attachment not found or already deleted")
	}

	dao.Logger.WithFields(logrus.Fields{
		"attachment_id": attachmentID,
		"entity_type":   entityType,
		"user_id":       userID,
	}).Info("Attachment soft deleted successfully")

	return nil
}

// VerifyAttachmentAccess verifies that the user's organization has access to the attachment
func (dao *AttachmentDao) VerifyAttachmentAccess(ctx context.Context, attachmentID int64, entityType string, orgID int64) (bool, error) {
	// Get the project_id associated with this attachment through the entity
	var projectID int64
	var query string

	switch entityType {
	case models.EntityTypeProject:
		// For project attachments, check directly in projects table
		query = `
			SELECT p.id
			FROM project.projects p
			JOIN project.project_attachments pa ON p.id = pa.project_id
			WHERE pa.id = $1 AND p.org_id = $2 AND pa.is_deleted = false
		`
	case models.EntityTypeIssue:
		query = `
			SELECT p.id
			FROM project.projects p
			JOIN project.issues i ON p.id = i.project_id
			JOIN project.issue_attachments ia ON i.id = ia.issue_id
			WHERE ia.id = $1 AND p.org_id = $2 AND ia.is_deleted = false
		`
	case models.EntityTypeRFI:
		query = `
			SELECT p.id
			FROM project.projects p
			JOIN project.rfis r ON p.id = r.project_id
			JOIN project.rfi_attachments ra ON r.id = ra.rfi_id
			WHERE ra.id = $1 AND p.org_id = $2 AND ra.is_deleted = false
		`
	case models.EntityTypeSubmittal:
		query = `
			SELECT p.id
			FROM project.projects p
			JOIN project.submittals s ON p.id = s.project_id
			JOIN project.submittal_attachments sa ON s.id = sa.submittal_id
			WHERE sa.id = $1 AND p.org_id = $2 AND sa.is_deleted = false
		`
	default:
		return false, fmt.Errorf("unsupported entity type: %s", entityType)
	}

	err := dao.DB.QueryRowContext(ctx, query, attachmentID, orgID).Scan(&projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No access
		}
		dao.Logger.WithError(err).WithFields(logrus.Fields{
			"attachment_id": attachmentID,
			"entity_type":   entityType,
			"org_id":        orgID,
		}).Error("Failed to verify attachment access")
		return false, err
	}

	return true, nil
}