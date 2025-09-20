// Package data provides data access layer implementations for the IAM system.
// This package contains repository interfaces and their concrete implementations
// for interacting with PostgreSQL database and other data sources.
//
// Key responsibilities:
// 1. Database query execution and result mapping
// 2. Data validation and transformation
// 3. Error handling and logging
// 4. Connection pooling and transaction management
//
// All repositories follow the interface pattern for better testability and
// dependency injection throughout the application.
package data

import (
	"database/sql"
	"fmt"
	"infrastructure/lib/models"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// UserRepository defines the contract for user data operations.
// This interface provides methods for retrieving and managing user profiles
// from the IAM database. It abstracts the data access layer to enable
// different implementations (e.g., PostgreSQL, MongoDB, mock for testing).
//
// All methods use Cognito ID as the primary identifier since it's the
// authoritative source from AWS Cognito authentication system.
type UserRepository interface {
	// GetUserProfile retrieves a complete user profile by Cognito ID.
	// Returns the user profile with all associated organizations, locations,
	// and roles, or an error if the user is not found or inactive.
	//
	// Parameters:
	//   - cognitoID: AWS Cognito user UUID (from 'sub' claim)
	//
	// Returns:
	//   - *models.UserProfile: Complete user profile with nested data
	//   - error: Database errors, user not found, or parsing errors
	GetUserProfile(cognitoID string) (*models.UserProfile, error)
}

// UserDao implements UserRepository interface using PostgreSQL database.
// This is the primary implementation used in production environments.
// It provides optimized queries using database views and proper error handling.
//
// Database Connection:
//   - Uses shared *sql.DB connection pool for concurrent access
//   - Implements connection retry and timeout handling
//   - Logs all operations for debugging
type UserDao struct {
	DB     *sql.DB        // PostgreSQL database connection pool
	Logger *logrus.Logger // Structured logger for debugging
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *sql.DB) UserRepository {
	return &UserDao{
		DB:     db,
		Logger: logrus.New(),
	}
}

// GetUserProfile fetches complete user profile data from PostgreSQL database using Cognito ID.
//
// This method uses the new RBAC system with user_assignments table to determine user access.
// It provides optimized access context aggregation for JWT token generation:
// 1. Single query with RBAC context aggregation
// 2. Direct location access based on user assignments
// 3. Support for organization/location/project level permissions
//
// Query Details:
//   - Uses new iam.user_assignments table for RBAC
//   - Aggregates access contexts into array for JWT
//   - Fetches accessible locations based on user assignments
//   - Supports hierarchical permissions (org -> location -> project)
//
// Error Handling:
//   - sql.ErrNoRows: User not found or inactive
//   - JSON parsing errors: Malformed locations data
//   - Database errors: Connection, timeout, constraint violations
func (dao *UserDao) GetUserProfile(cognitoID string) (*models.UserProfile, error) {
	// Step 1: Get user basic info and access contexts for JWT
	userQuery := `
		SELECT
			u.id, u.cognito_id, u.email, u.first_name, u.last_name,
			u.phone, u.job_title, u.status, u.avatar_url, u.org_id, 
			o.name as org_name, u.last_selected_location_id, u.is_super_admin,
			COALESCE(
				array_agg(DISTINCT
					CASE ua.context_type
						WHEN 'organization' THEN 'ORG:' || ua.context_id
						WHEN 'location' THEN 'LOC:' || ua.context_id
						WHEN 'project' THEN 'PROJ:' || ua.context_id
					END
				) FILTER (WHERE ua.context_id IS NOT NULL),
				ARRAY[]::text[]
			) as access_contexts
		FROM iam.users u
		JOIN iam.organizations o ON u.org_id = o.id
		LEFT JOIN iam.user_assignments ua ON u.id = ua.user_id AND ua.is_deleted = false
		WHERE u.cognito_id = $1 
		  AND u.is_deleted = FALSE
		  AND (
			  u.status = 'active'
			  OR u.status = 'pending'
			  OR (u.status = 'pending_org_setup' AND u.is_super_admin = true)
		  )
		GROUP BY u.id, u.cognito_id, u.email, u.first_name, u.last_name, 
				 u.phone, u.job_title, u.status, u.avatar_url, u.org_id, 
				 o.name, u.last_selected_location_id, u.is_super_admin;
`

	dao.Logger.WithFields(logrus.Fields{
		"cognito_id": cognitoID,
		"operation":  "GetUserProfile",
	}).Debug("Fetching user profile with new RBAC system")

	row := dao.DB.QueryRow(userQuery, cognitoID)

	// Initialize profile struct and variables for RBAC access
	var profile models.UserProfile
	var accessContexts StringSlice

	// Scan database row into struct fields
	// Order must match the SELECT statement exactly
	err := row.Scan(
		&profile.UserID,    // Internal user identifier
		&profile.CognitoID, // AWS Cognito 'sub' UUID
		&profile.Email,     // User's email address
		&profile.FirstName, // Personal information
		&profile.LastName,
		&profile.Phone,                  // sql.NullString for optional field
		&profile.JobTitle,               // sql.NullString for optional field
		&profile.Status,                 // Account status (active/inactive/suspended)
		&profile.AvatarURL,              // sql.NullString for optional field
		&profile.OrgID,                  // Organization identifier
		&profile.OrgName,                // Organization display name
		&profile.LastSelectedLocationID, // sql.NullString for optional last selected location
		&profile.IsSuperAdmin,           // SuperAdmin role flag
		&accessContexts,                 // Access contexts array for RBAC
	)

	// Handle database errors with proper classification
	if err != nil {
		if err == sql.ErrNoRows {
			// User not found or inactive - common case, not an error
			dao.Logger.WithFields(logrus.Fields{
				"cognito_id": cognitoID,
				"operation":  "GetUserProfile",
			}).Warn("User not found in database or inactive")
			return nil, fmt.Errorf("user not found: %s", cognitoID)
		}

		// Database connection, query, or scanning error - serious issue
		dao.Logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"operation":  "GetUserProfile",
			"error":      err.Error(),
		}).Error("Error scanning user profile from database")
		return nil, fmt.Errorf("error fetching user profile: %w", err)
	}

	// Step 2: Fetch accessible locations based on user type
	// For super admins: Load ALL locations in their organization
	// For regular users: Load only locations they have access to via RBAC
	if profile.IsSuperAdmin {
		// Super admin: fetch ALL locations in their organization
		locationQuery := `
			SELECT DISTINCT l.id, l.name, l.location_type
			FROM iam.locations l
			WHERE l.org_id = $1 AND l.is_deleted = false
			ORDER BY l.name;
		`

		dao.Logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"org_id":     profile.OrgID,
			"operation":  "GetUserProfile",
		}).Debug("Fetching all organization locations for super admin")

		rows, err := dao.DB.Query(locationQuery, profile.OrgID)
		if err != nil {
			dao.Logger.WithFields(logrus.Fields{
				"cognito_id": cognitoID,
				"org_id":     profile.OrgID,
				"error":      err.Error(),
			}).Error("Error fetching super admin locations")
			profile.Locations = []models.UserLocation{}
		} else {
			defer rows.Close()

			var locations []models.UserLocation
			for rows.Next() {
				var loc models.UserLocation
				if err := rows.Scan(&loc.ID, &loc.Name, &loc.LocationType); err != nil {
					dao.Logger.WithFields(logrus.Fields{
						"cognito_id": cognitoID,
						"error":      err.Error(),
					}).Warn("Error scanning location row")
					continue
				}
				// No roles in location data - roles will be fetched per-project when needed
				locations = append(locations, loc)
			}
			profile.Locations = locations

			dao.Logger.WithFields(logrus.Fields{
				"cognito_id":     cognitoID,
				"org_id":         profile.OrgID,
				"location_count": len(locations),
				"operation":      "GetUserProfile",
			}).Debug("Loaded all organization locations for super admin")
		}
	} else if len(accessContexts) > 0 {
		// Regular user: fetch ONLY locations they have access to based on user_assignments
		// Parse access contexts to determine accessible locations
		var accessibleLocationIds []int64
		var hasOrgAccess bool

		for _, context := range accessContexts {
			parts := strings.Split(context, ":")
			if len(parts) == 2 {
				contextType := parts[0]
				contextIdStr := parts[1]

				switch contextType {
				case "ORG":
					// User has organization-level access - gets all locations in org
					hasOrgAccess = true
				case "LOC":
					// User has specific location access
					if locationId := parseInt64(contextIdStr); locationId > 0 {
						accessibleLocationIds = append(accessibleLocationIds, locationId)
					}
				case "PROJ":
					// User has project access - need to find the location of this project
					if projectId := parseInt64(contextIdStr); projectId > 0 {
						// Get location ID from project
						var projectLocationId int64
						projLocationQuery := `SELECT location_id FROM project.projects WHERE id = $1 AND is_deleted = false`
						err := dao.DB.QueryRow(projLocationQuery, projectId).Scan(&projectLocationId)
						if err == nil && projectLocationId > 0 {
							accessibleLocationIds = append(accessibleLocationIds, projectLocationId)
						}
					}
				}
			}
		}

		// Build location query based on access level
		var locationQuery string
		var queryArgs []interface{}

		if hasOrgAccess {
			// User has org-level access - get all locations in organization
			locationQuery = `
				SELECT DISTINCT l.id, l.name, l.location_type
				FROM iam.locations l
				WHERE l.org_id = $1 AND l.is_deleted = false
				ORDER BY l.name;
			`
			queryArgs = []interface{}{profile.OrgID}
		} else if len(accessibleLocationIds) > 0 {
			// User has specific location access
			// Remove duplicates
			uniqueLocationIds := make(map[int64]bool)
			var deduplicatedIds []int64
			for _, id := range accessibleLocationIds {
				if !uniqueLocationIds[id] {
					uniqueLocationIds[id] = true
					deduplicatedIds = append(deduplicatedIds, id)
				}
			}

			locationQuery = `
				SELECT DISTINCT l.id, l.name, l.location_type
				FROM iam.locations l
				WHERE l.id = ANY($1::bigint[]) AND l.is_deleted = false
				ORDER BY l.name;
			`
			queryArgs = []interface{}{deduplicatedIds}
		} else {
			// User has no location access
			profile.Locations = []models.UserLocation{}
			dao.Logger.WithFields(logrus.Fields{
				"cognito_id": cognitoID,
				"operation":  "GetUserProfile",
			}).Debug("User has no location access based on assignments")
			return &profile, nil
		}

		dao.Logger.WithFields(logrus.Fields{
			"cognito_id":      cognitoID,
			"access_contexts": accessContexts,
			"has_org_access":  hasOrgAccess,
			"location_ids":    accessibleLocationIds,
			"operation":       "GetUserProfile",
		}).Debug("Fetching accessible locations for regular user")

		// Execute location query
		rows, err := dao.DB.Query(locationQuery, queryArgs...)
		if err != nil {
			dao.Logger.WithFields(logrus.Fields{
				"cognito_id": cognitoID,
				"error":      err.Error(),
			}).Error("Error fetching accessible locations for regular user")
			profile.Locations = []models.UserLocation{}
		} else {
			defer rows.Close()

			var locations []models.UserLocation
			for rows.Next() {
				var loc models.UserLocation
				if err := rows.Scan(&loc.ID, &loc.Name, &loc.LocationType); err != nil {
					dao.Logger.WithFields(logrus.Fields{
						"cognito_id": cognitoID,
						"error":      err.Error(),
					}).Warn("Error scanning location row")
					continue
				}
				// No roles in location data - roles will be fetched per-project when needed
				locations = append(locations, loc)
			}
			profile.Locations = locations

			dao.Logger.WithFields(logrus.Fields{
				"cognito_id":     cognitoID,
				"location_count": len(locations),
				"operation":      "GetUserProfile",
			}).Debug("Loaded accessible locations for regular user")
		}
	} else {
		// No access contexts and not super admin - empty locations
		profile.Locations = []models.UserLocation{}
		dao.Logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"operation":  "GetUserProfile",
		}).Debug("No accessible locations for user")
	}

	// Log successful profile fetch
	dao.Logger.WithFields(logrus.Fields{
		"user_id":         profile.UserID,
		"cognito_id":      profile.CognitoID,
		"email":           profile.Email,
		"org_id":          profile.OrgID,
		"org_name":        profile.OrgName,
		"status":          profile.Status,
		"isSuperAdmin":    profile.IsSuperAdmin,
		"locations_count": len(profile.Locations),
		"operation":       "GetUserProfile",
	}).Debug("Successfully fetched user profile")

	return &profile, nil
}

// parseInt64 safely converts string to int64, returns 0 on error
func parseInt64(s string) int64 {
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val
	}
	return 0
}

// StringSlice is a custom type for handling PostgreSQL array values.
// PostgreSQL stores arrays in a specific text format that needs custom parsing.
//
// This type implements the sql.Scanner interface to automatically convert
// PostgreSQL array columns into Go []string slices during database queries.
//
// PostgreSQL Array Format:
//   - Empty array: "{}"
//   - Single item: "{item}"
//   - Multiple items: "{item1,item2,item3}"
//   - With spaces: "{item 1, item 2, item 3}"
//
// Usage Example:
//
//	var roles StringSlice
//	row.Scan(&roles) // Automatically parses PostgreSQL array
//	fmt.Println([]string(roles)) // Convert back to regular []string
type StringSlice []string

// Scan implements the sql.Scanner interface for StringSlice.
// This method is automatically called by database/sql package when scanning
// PostgreSQL array columns into StringSlice variables.
//
// Supported Input Types:
//   - []byte: PostgreSQL driver typically returns arrays as byte arrays
//   - string: Some drivers may return arrays as strings
//   - nil: NULL values are converted to empty arrays
//
// PostgreSQL Array Parsing:
//   - Handles empty arrays: "{}" → []string{}
//   - Parses multiple items: "{a,b,c}" → []string{"a", "b", "c"}
//   - Trims whitespace: "{a, b, c}" → []string{"a", "b", "c"}
//   - Preserves empty strings within arrays
//
// Error Handling:
//   - Returns error for unsupported value types
//   - Does not validate individual array items
//   - Assumes valid PostgreSQL array format
func (s *StringSlice) Scan(value interface{}) error {
	// Handle NULL values
	if value == nil {
		*s = StringSlice([]string{})
		return nil
	}

	switch v := value.(type) {
	case []byte:
		// PostgreSQL typically returns arrays as byte arrays
		// Handle PostgreSQL array format: {item1,item2,item3}
		str := string(v)
		if str == "{}" {
			*s = StringSlice([]string{})
			return nil
		}

		// Parse array by removing braces and splitting on comma
		if len(str) > 2 {
			str = str[1 : len(str)-1] // Remove opening { and closing }
			items := []string{}
			for _, item := range strings.Split(str, ",") {
				items = append(items, strings.TrimSpace(item))
			}
			*s = StringSlice(items)
		} else {
			*s = StringSlice([]string{})
		}
		return nil

	case string:
		// Some PostgreSQL drivers may return arrays as strings
		if v == "{}" {
			*s = StringSlice([]string{})
			return nil
		}

		// Handle similar to []byte case
		if len(v) > 2 {
			v = v[1 : len(v)-1] // Remove opening { and closing }
			items := []string{}
			for _, item := range strings.Split(v, ",") {
				items = append(items, strings.TrimSpace(item))
			}
			*s = StringSlice(items)
		} else {
			*s = StringSlice([]string{})
		}
		return nil

	default:
		// Unsupported type - return error to help with debugging
		return fmt.Errorf("cannot scan %T into StringSlice", value)
	}
}
