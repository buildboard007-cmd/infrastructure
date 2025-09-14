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

	// Step 2: Fetch accessible locations based on RBAC contexts
	if len(accessContexts) > 0 {
		// Build location query based on access contexts
		locationQuery := `
			SELECT DISTINCT 
				l.id, l.name, l.location_type,
				COUNT(DISTINCT p.id) as project_count
			FROM iam.locations l
			LEFT JOIN project.projects p ON l.id = p.location_id AND p.is_deleted = false
			WHERE (
		`
		
		var queryConditions []string
		var queryArgs []interface{}
		argIndex := 1
		
		// Parse access contexts and build query conditions
		var projectIds []int64
		var locationIds []int64
		
		for _, context := range accessContexts {
			parts := strings.Split(context, ":")
			if len(parts) == 2 {
				contextType := parts[0]
				contextId := parts[1]
				
				switch contextType {
				case "ORG":
					queryConditions = append(queryConditions, fmt.Sprintf("l.org_id = $%d", argIndex))
					queryArgs = append(queryArgs, contextId)
					argIndex++
				case "LOC":
					locationIds = append(locationIds, parseInt64(contextId))
				case "PROJ":
					projectIds = append(projectIds, parseInt64(contextId))
				}
			}
		}
		
		// Add location IDs condition
		if len(locationIds) > 0 {
			queryConditions = append(queryConditions, fmt.Sprintf("l.id = ANY($%d::bigint[])", argIndex))
			queryArgs = append(queryArgs, locationIds)
			argIndex++
		}
		
		// Add project-based location access
		if len(projectIds) > 0 {
			queryConditions = append(queryConditions, fmt.Sprintf("p.id = ANY($%d::bigint[])", argIndex))
			queryArgs = append(queryArgs, projectIds)
			argIndex++
		}
		
		if len(queryConditions) > 0 {
			locationQuery += strings.Join(queryConditions, " OR ")
		} else {
			locationQuery += "1=0" // No access
		}
		
		locationQuery += `
			) AND l.is_deleted = false
			GROUP BY l.id, l.name, l.location_type
			ORDER BY l.name;
		`
		
		// Execute location query
		rows, err := dao.DB.Query(locationQuery, queryArgs...)
		if err != nil {
			dao.Logger.WithFields(logrus.Fields{
				"cognito_id": cognitoID,
				"error":      err.Error(),
			}).Error("Error fetching user locations")
			profile.Locations = []models.UserLocation{}
		} else {
			defer rows.Close()
			
			var locations []models.UserLocation
			for rows.Next() {
				var loc models.UserLocation
				var projectCount int
				if err := rows.Scan(&loc.ID, &loc.Name, &loc.LocationType, &projectCount); err != nil {
					dao.Logger.WithFields(logrus.Fields{
						"cognito_id": cognitoID,
						"error":      err.Error(),
					}).Warn("Error scanning location row")
					continue
				}
				locations = append(locations, loc)
			}
			profile.Locations = locations
		}
	} else {
		// No access contexts - empty locations
		profile.Locations = []models.UserLocation{}
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
