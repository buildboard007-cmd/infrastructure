package data

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"strings"

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
	UpdateLocation(ctx context.Context, locationID, orgID int64, updateReq *models.UpdateLocationRequest, userID int64) (*models.Location, error)
	
	// DeleteLocation soft deletes a location (removes user assignments but keeps location record)
	DeleteLocation(ctx context.Context, locationID, orgID int64, userID int64) error
	
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

	// Set defaults
	locationType := location.LocationType
	if locationType == "" {
		locationType = "office"
	}
	
	status := location.Status
	if status == "" {
		status = "active"
	}
	
	country := location.Country
	if country == "" {
		country = "USA"
	}

	// Create the location
	var locationID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO iam.locations (
			org_id, name, location_type, address, city, state, zip_code, country, status,
			created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, orgID, location.Name, locationType, location.Address, location.City, location.State, 
		location.ZipCode, country, status, userID, userID).Scan(
		&locationID, &location.CreatedAt, &location.UpdatedAt)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id": userID,
			"org_id":  orgID,
			"name":    location.Name,
			"error":   err.Error(),
		}).Error("Failed to create location")
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	// Get SuperAdmin role ID for the organization
	var superAdminRoleID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM iam.roles WHERE name = 'SuperAdmin' AND org_id = $1
	`, orgID).Scan(&superAdminRoleID)

	if err != nil {
		dao.Logger.WithError(err).Error("Failed to get SuperAdmin role ID")
		return nil, fmt.Errorf("failed to get SuperAdmin role: %w", err)
	}

	// Assign the location to the creator with SuperAdmin role
	_, err = tx.ExecContext(ctx, `
		INSERT INTO iam.location_user_roles (location_id, user_id, role_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5)
	`, locationID, userID, superAdminRoleID, userID, userID)

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
	location.ID = locationID
	location.OrgID = orgID
	location.CreatedBy = userID
	location.UpdatedBy = userID

	// Check if user should be activated after creating first location
	err = dao.checkAndUpdateUserStatusAfterLocation(ctx, userID, orgID)
	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"org_id":      orgID,
			"location_id": locationID,
			"warning":     err.Error(),
		}).Warn("Failed to check/update user status after location creation")
		// Don't return error as location creation was successful
	}

	dao.Logger.WithFields(logrus.Fields{
		"location_id": locationID,
		"org_id":      orgID,
		"user_id":     userID,
		"name":        location.Name,
	}).Info("Successfully created location and assigned to creator")

	return location, nil
}

// GetLocationsByOrg retrieves all locations for a specific organization
func (dao *LocationDao) GetLocationsByOrg(ctx context.Context, orgID int64) ([]models.Location, error) {
	query := `
		SELECT id, org_id, name, location_type, address, city, state, zip_code, country, 
		       status, created_at, created_by, updated_at, updated_by
		FROM iam.locations
		WHERE org_id = $1 AND is_deleted = FALSE
		ORDER BY name ASC
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
			&location.ID,
			&location.OrgID,
			&location.Name,
			&location.LocationType,
			&location.Address,
			&location.City,
			&location.State,
			&location.ZipCode,
			&location.Country,
			&location.Status,
			&location.CreatedAt,
			&location.CreatedBy,
			&location.UpdatedAt,
			&location.UpdatedBy,
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
		SELECT id, org_id, name, location_type, address, city, state, zip_code, country, 
		       status, created_at, created_by, updated_at, updated_by
		FROM iam.locations
		WHERE id = $1 AND org_id = $2 AND is_deleted = FALSE
	`

	err := dao.DB.QueryRowContext(ctx, query, locationID, orgID).Scan(
		&location.ID,
		&location.OrgID,
		&location.Name,
		&location.LocationType,
		&location.Address,
		&location.City,
		&location.State,
		&location.ZipCode,
		&location.Country,
		&location.Status,
		&location.CreatedAt,
		&location.CreatedBy,
		&location.UpdatedAt,
		&location.UpdatedBy,
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

// UpdateLocation updates an existing location with the provided fields
func (dao *LocationDao) UpdateLocation(ctx context.Context, locationID, orgID int64, updateReq *models.UpdateLocationRequest, userID int64) (*models.Location, error) {
	// Build dynamic update query based on provided fields
	setParts := []string{"updated_by = $1", "updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{userID}
	argIndex := 2
	
	if updateReq.Name != "" {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, updateReq.Name)
		argIndex++
	}
	if updateReq.LocationType != "" {
		setParts = append(setParts, fmt.Sprintf("location_type = $%d", argIndex))
		args = append(args, updateReq.LocationType)
		argIndex++
	}
	if updateReq.Address != "" {
		setParts = append(setParts, fmt.Sprintf("address = $%d", argIndex))
		args = append(args, updateReq.Address)
		argIndex++
	}
	if updateReq.City != "" {
		setParts = append(setParts, fmt.Sprintf("city = $%d", argIndex))
		args = append(args, updateReq.City)
		argIndex++
	}
	if updateReq.State != "" {
		setParts = append(setParts, fmt.Sprintf("state = $%d", argIndex))
		args = append(args, updateReq.State)
		argIndex++
	}
	if updateReq.ZipCode != "" {
		setParts = append(setParts, fmt.Sprintf("zip_code = $%d", argIndex))
		args = append(args, updateReq.ZipCode)
		argIndex++
	}
	if updateReq.Country != "" {
		setParts = append(setParts, fmt.Sprintf("country = $%d", argIndex))
		args = append(args, updateReq.Country)
		argIndex++
	}
	if updateReq.Status != "" {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, updateReq.Status)
		argIndex++
	}
	
	// Add WHERE conditions
	args = append(args, locationID, orgID)
	
	query := fmt.Sprintf(`
		UPDATE iam.locations 
		SET %s
		WHERE id = $%d AND org_id = $%d AND is_deleted = FALSE
		RETURNING id, org_id, name, location_type, address, city, state, zip_code, country, 
		          status, created_at, created_by, updated_at, updated_by
	`, strings.Join(setParts, ", "), argIndex, argIndex+1)

	var updatedLocation models.Location
	err := dao.DB.QueryRowContext(ctx, query, args...).Scan(
		&updatedLocation.ID,
		&updatedLocation.OrgID,
		&updatedLocation.Name,
		&updatedLocation.LocationType,
		&updatedLocation.Address,
		&updatedLocation.City,
		&updatedLocation.State,
		&updatedLocation.ZipCode,
		&updatedLocation.Country,
		&updatedLocation.Status,
		&updatedLocation.CreatedAt,
		&updatedLocation.CreatedBy,
		&updatedLocation.UpdatedAt,
		&updatedLocation.UpdatedBy,
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
		"location_name": updatedLocation.Name,
	}).Info("Successfully updated location")

	return &updatedLocation, nil
}

// DeleteLocation soft deletes a location (sets is_deleted = TRUE)
func (dao *LocationDao) DeleteLocation(ctx context.Context, locationID, orgID int64, userID int64) error {
	// Soft delete the location
	result, err := dao.DB.ExecContext(ctx, `
		UPDATE iam.locations 
		SET is_deleted = TRUE, updated_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND org_id = $3 AND is_deleted = FALSE
	`, userID, locationID, orgID)

	if err != nil {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
			"error":       err.Error(),
		}).Error("Failed to delete location")
		return fmt.Errorf("failed to delete location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		dao.Logger.WithFields(logrus.Fields{
			"location_id": locationID,
			"org_id":      orgID,
		}).Warn("Location not found for deletion")
		return fmt.Errorf("location not found")
	}

	dao.Logger.WithFields(logrus.Fields{
		"location_id": locationID,
		"org_id":      orgID,
		"user_id":     userID,
	}).Info("Successfully soft deleted location")


	return nil
}

// VerifyLocationAccess checks if a user has access to a specific location
func (dao *LocationDao) VerifyLocationAccess(ctx context.Context, userID, locationID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM iam.location_user_roles 
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

// checkAndUpdateUserStatusAfterLocation checks if user should be activated after creating a location
// User becomes active when they have updated their org AND created at least one location
func (dao *LocationDao) checkAndUpdateUserStatusAfterLocation(ctx context.Context, userID, orgID int64) error {
	// Check if user is still in pending_org_setup status and org has been updated from default
	var userStatus string
	var orgName string
	
	query := `
		SELECT u.status, o.name
		FROM iam.users u
		JOIN iam.organizations o ON u.org_id = o.id
		WHERE u.id = $1 AND u.org_id = $2
	`
	
	err := dao.DB.QueryRowContext(ctx, query, userID, orgID).Scan(&userStatus, &orgName)
	if err != nil {
		return fmt.Errorf("failed to check user status and org name: %w", err)
	}
	
	// If user is pending_org_setup and org has been updated from default name, activate them
	if userStatus == "pending_org_setup" && orgName != "New Organization" {
		_, err = dao.DB.ExecContext(ctx, `
			UPDATE iam.users
			SET status = 'active', updated_by = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2 AND status = 'pending_org_setup'
		`, userID, userID)
		
		if err != nil {
			return fmt.Errorf("failed to update user status to active: %w", err)
		}
		
		dao.Logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"org_id":   orgID,
			"org_name": orgName,
		}).Info("User status updated to active after organization update and location creation")
	}
	
	return nil
}