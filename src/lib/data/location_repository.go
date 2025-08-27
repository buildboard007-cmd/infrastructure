package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"

	"github.com/sirupsen/logrus"
)

// LocationRepository defines the interface for location data operations
type LocationRepository interface {
	// CreateLocation creates a new location in the organization and assigns it to the creator with SuperAdmin role
	CreateLocation(ctx context.Context, userID, orgID int64, location *models.Location) (*models.Location, error)
	
	// GetLocationsByOrg retrieves all locations for a specific organization
	GetLocationsByOrg(ctx context.Context, orgID int64) ([]models.Location, error)
	
	// GetLocationByID retrieves a specific location by ID (with org validation)
	GetLocationByID(ctx context.Context, locationID, orgID int64) (*models.Location, error)
	
	// UpdateLocation updates an existing location
	UpdateLocation(ctx context.Context, locationID, orgID int64, location *models.Location) (*models.Location, error)
	
	// DeleteLocation soft deletes a location (removes user assignments but keeps location record)
	DeleteLocation(ctx context.Context, locationID, orgID int64) error
	
	// VerifyLocationAccess verifies if a user has access to a specific location
	VerifyLocationAccess(ctx context.Context, userID, locationID int64) (bool, error)
}

// LocationDao implements LocationRepository interface using PostgreSQL
type LocationDao struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// CreateLocation creates a new location and automatically assigns it to the creator with SuperAdmin role
func (dao *LocationDao) CreateLocation(ctx context.Context, userID, orgID int64, location *models.Location) (*models.Location, error) {
	// Start transaction for atomic operation
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for location creation")
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the location
	var locationID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO iam.location (org_id, location_name, address, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING location_id, created_at, updated_at
	`, orgID, location.LocationName, location.Address, userID).Scan(
		&locationID, &location.CreatedAt, &location.UpdatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"org_id":        orgID,
			"location_name": location.LocationName,
			"error":         err.Error(),
		}).Error("Failed to create location")
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	// Get SuperAdmin role ID
	var superAdminRoleID int64
	err = tx.QueryRowContext(ctx, `
		SELECT role_id FROM iam.role WHERE role_name = 'SuperAdmin'
	`).Scan(&superAdminRoleID)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get SuperAdmin role ID")
		return nil, fmt.Errorf("failed to get SuperAdmin role: %w", err)
	}

	// Assign the location to the creator with SuperAdmin role
	_, err = tx.ExecContext(ctx, `
		INSERT INTO iam.user_location (user_id, location_id, role_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userID, locationID, superAdminRoleID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"location_id": locationID,
			"role_id":     superAdminRoleID,
			"error":       err.Error(),
		}).Error("Failed to assign location to user")
		return nil, fmt.Errorf("failed to assign location to user: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit location creation transaction")
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Populate the response
	location.LocationID = locationID
	location.OrgID = orgID
	location.CreatedBy = userID

	dao.Logger.WithFields(logrus.Fields{
		"location_id":   locationID,
		"org_id":        orgID,
		"user_id":       userID,
		"location_name": location.LocationName,
	}).Info("Successfully created location and assigned to creator")

	return location, nil
}

// GetLocationsByOrg retrieves all locations for a specific organization
func (dao *LocationDao) GetLocationsByOrg(ctx context.Context, orgID int64) ([]models.Location, error) {
	query := `
		SELECT location_id, org_id, location_name, address, created_by, created_at, updated_at
		FROM iam.location
		WHERE org_id = $1
		ORDER BY location_name ASC
	`

	rows, err := dao.DB.QueryContext(ctx, query, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"org_id": orgID,
			"error":  err.Error(),
		}).Error("Failed to query locations")
		return nil, fmt.Errorf("failed to query locations: %w", err)
	}
	defer rows.Close()

	var locations []models.Location
	for rows.Next() {
		var location models.Location
		err := rows.Scan(
			&location.LocationID,
			&location.OrgID,
			&location.LocationName,
			&location.Address,
			&location.CreatedBy,
			&location.CreatedAt,
			&location.UpdatedAt,
		)
		if err != nil {
			dao.Logger.WithError(err).Error("Failed to scan location row")
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		locations = append(locations, location)
	}

	if err = rows.Err(); err != nil {
		dao.Logger.WithError(err).Error("Error iterating location rows")
		return nil, fmt.Errorf("error iterating locations: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"org_id": orgID,
		"count":  len(locations),
	}).Debug("Successfully retrieved locations for organization")

	return locations, nil
}

// GetLocationByID retrieves a specific location by ID with organization validation
func (dao *LocationDao) GetLocationByID(ctx context.Context, locationID, orgID int64) (*models.Location, error) {
	var location models.Location
	query := `
		SELECT location_id, org_id, location_name, address, created_by, created_at, updated_at
		FROM iam.location
		WHERE location_id = $1 AND org_id = $2
	`

	err := dao.DB.QueryRowContext(ctx, query, locationID, orgID).Scan(
		&location.LocationID,
		&location.OrgID,
		&location.LocationName,
		&location.Address,
		&location.CreatedBy,
		&location.CreatedAt,
		&location.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
		}).Warn("Location not found")
		return nil, fmt.Errorf("location not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
			"error":       err.Error(),
		}).Error("Failed to get location")
		return nil, fmt.Errorf("failed to get location: %w", err)
	}

	return &location, nil
}

// UpdateLocation updates an existing location
func (dao *LocationDao) UpdateLocation(ctx context.Context, locationID, orgID int64, location *models.Location) (*models.Location, error) {
	query := `
		UPDATE iam.location 
		SET location_name = $1, address = $2, updated_at = NOW()
		WHERE location_id = $3 AND org_id = $4
		RETURNING location_id, org_id, location_name, address, created_by, created_at, updated_at
	`

	var updatedLocation models.Location
	err := dao.DB.QueryRowContext(ctx, query,
		location.LocationName,
		location.Address,
		locationID,
		orgID,
	).Scan(
		&updatedLocation.LocationID,
		&updatedLocation.OrgID,
		&updatedLocation.LocationName,
		&updatedLocation.Address,
		&updatedLocation.CreatedBy,
		&updatedLocation.CreatedAt,
		&updatedLocation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
		}).Warn("Location not found for update")
		return nil, fmt.Errorf("location not found")
	}

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
			"error":       err.Error(),
		}).Error("Failed to update location")
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"location_id":   locationID,
		"org_id":        orgID,
		"location_name": updatedLocation.LocationName,
	}).Info("Successfully updated location")

	return &updatedLocation, nil
}

// DeleteLocation removes a location and all its user assignments
func (dao *LocationDao) DeleteLocation(ctx context.Context, locationID, orgID int64) error {
	// Start transaction
	tx, err := dao.DB.BeginTx(ctx, nil)
	if err != nil {
		dao.Logger.WithError(err).Error("Failed to start transaction for location deletion")
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, remove all user-location assignments
	_, err = tx.ExecContext(ctx, `
		DELETE FROM iam.user_location WHERE location_id = $1
	`, locationID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"error":       err.Error(),
		}).Error("Failed to remove user-location assignments")
		return fmt.Errorf("failed to remove user assignments: %w", err)
	}

	// Then delete the location (with org validation)
	result, err := tx.ExecContext(ctx, `
		DELETE FROM iam.location WHERE location_id = $1 AND org_id = $2
	`, locationID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
			"error":       err.Error(),
		}).Error("Failed to delete location")
		return fmt.Errorf("failed to delete location: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
		}).Warn("Location not found for deletion")
		return fmt.Errorf("location not found")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		dao.Logger.WithError(err).Error("Failed to commit location deletion transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dao.Logger.WithFields(logrus.Fields{
		"location_id": locationID,
		"org_id":      orgID,
	}).Info("Successfully deleted location and all assignments")

	return nil
}

// VerifyLocationAccess checks if a user has access to a specific location
func (dao *LocationDao) VerifyLocationAccess(ctx context.Context, userID, locationID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM iam.user_location 
		WHERE user_id = $1 AND location_id = $2
	`

	err := dao.DB.QueryRowContext(ctx, query, userID, locationID).Scan(&count)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"location_id": locationID,
			"error":       err.Error(),
		}).Error("Failed to verify location access")
		return false, fmt.Errorf("failed to verify location access: %w", err)
	}

	return count > 0, nil
}