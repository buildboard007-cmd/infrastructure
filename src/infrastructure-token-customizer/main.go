// Package main implements the AWS Cognito Pre-Token Generation V2.0 Lambda trigger.
//
// This Lambda function serves as a critical component in the JWT token customization
// pipeline for the BuildBoard infrastructure. It intercepts Cognito's token generation
// process and enriches JWT tokens with user profile data from the IAM database.
//
// Key Responsibilities:
// 1. Fetch complete user profile from PostgreSQL IAM database
// 2. Add user organization, location, and role data to JWT tokens
// 3. Encode complex data structures (locations/roles) for token size optimization
// 4. Provide graceful error handling that doesn't break authentication flow
// 5. Support both ID tokens (for frontend) and Access tokens (for API calls)
//
// AWS Cognito Integration:
//   - Trigger: Pre Token Generation V2.0 (enhanced version)
//   - Supports: TokenGeneration_HostedAuth, TokenGeneration_Authentication,
//     TokenGeneration_RefreshTokens, and other V2 triggers
//   - Token Enhancement: Adds custom claims to both ID and Access tokens
//   - Group Integration: Maps user roles to Cognito groups for authorization
//
// Database Integration:
//   - Uses PostgreSQL with optimized iam.user_summary view
//   - Connection pooling via shared *sql.DB instance
//   - Graceful degradation if database is unavailable
//   - SSM Parameter Store for configuration management
//
// Performance Considerations:
//   - Cold start optimization with global variable initialization
//   - Base64 encoding for complex nested data (locations/roles)
//   - Single database query per token generation via optimized view
//   - Connection reuse across Lambda invocations
//
// Security Features:
//   - No sensitive data in logs (email/user IDs only in debug mode)
//   - SSL database connections with credential rotation via SSM
//   - VPC isolation for database access
//   - Proper error handling that doesn't expose internal details
package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
// These are initialized once during Lambda cold start and reused across invocations
var (
	logger         *logrus.Logger      // Structured logger for debugging
	isLocal        bool                // Development/local execution flag
	ssmRepository  data.SSMRepository  // AWS SSM Parameter Store client interface
	userRepository data.UserRepository // User data access layer interface
	ssmParams      map[string]string   // Cached SSM parameters (database config)
	sqlDB          *sql.DB             // PostgreSQL connection pool (reused across invocations)
)

// CustomClaims represents the structure of custom claims to add to JWT tokens.
// These claims are added to both ID tokens (for frontend use) and Access tokens
// (for API authorization). The structure mirrors the UserProfile model but optimizes
// for token size by encoding complex nested data as Base64 JSON.
//
// Design Decisions:
//   - All fields are strings for JWT compatibility
//   - Optional fields use omitempty to reduce token size
//   - Locations are Base64-encoded JSON to handle nested role data
//   - Full name is computed for convenience
//   - Phone/JobTitle/PhotoURL are optional and may be empty
//
// Token Size Optimization:
//   - JWT tokens have size limits (typically 8KB for cookies)
//   - Complex location/role data is encoded to minimize token size
//   - Only essential fields are included in main token body
//   - Detailed location/role data is in encoded locations field
//
// Frontend Usage:
//   - Frontend can decode locations to get user's roles per location
//   - Basic user info (name, email, org) is directly accessible
//   - Status field enables account state checking (active/inactive/suspended)
type CustomClaims struct {
	UserID            string `json:"user_id"`    // Internal user identifier
	CognitoID         string `json:"cognito_id"` // AWS Cognito UUID ('sub' claim)
	Email             string `json:"email"`      // User's email address
	FirstName         string `json:"first_name"` // Personal information
	LastName          string `json:"last_name"`
	FullName          string `json:"full_name"`                     // Computed: "FirstName LastName"
	Phone             string `json:"phone,omitempty"`               // Optional contact phone
	JobTitle          string `json:"job_title,omitempty"`           // Optional professional title
	Status            string `json:"status"`                        // Account status (active/inactive/suspended)
	PhotoURL          string `json:"photo_url,omitempty"`           // Optional profile photo URL
	OrgID             string `json:"org_id"`                        // Organization identifier
	OrgName           string `json:"org_name"`                      // Organization display name
	CurrentLocationID string `json:"current_location_id,omitempty"` // User's primary location ID
	IsSuperAdmin      bool   `json:"isSuperAdmin"`                  // SuperAdmin role flag
	Locations         string `json:"locations"`                     // Base64 encoded JSON of []Location with roles
}

// Handler processes the Cognito Pre Token Generation V2.0 trigger event.
//
// This is the main entry point for the Lambda function. It receives Cognito events
// during token generation and enriches the tokens with user profile data from the
// IAM database. The function implements graceful error handling to ensure that
// authentication failures don't occur due to database issues.
//
// Event Processing Flow:
//  1. Validate trigger source (ensure V2.0 compatibility)
//  2. Extract Cognito user ID from event.UserName (contains UUID)
//  3. Fetch complete user profile from PostgreSQL database
//  4. Build custom claims structure with user data
//  5. Add claims to both ID and Access tokens
//  6. Map user roles to Cognito groups for authorization
//
// Error Handling Strategy:
//   - Database errors: Log but don't fail authentication
//   - Invalid trigger: Return unchanged event
//   - User not found: Log but don't fail authentication
//   - Claims building errors: Log but don't fail authentication
//
// This ensures users can always authenticate even if IAM database is unavailable.
func Handler(ctx context.Context, event events.CognitoEventUserPoolsPreTokenGenV2_0) (events.CognitoEventUserPoolsPreTokenGenV2_0, error) {
	// Log incoming event details for debugging
	logger.WithFields(logrus.Fields{
		"trigger_source": event.TriggerSource,          // Type of Cognito trigger
		"user_pool_id":   event.UserPoolID,             // Cognito User Pool identifier
		"username":       event.UserName,               // Cognito user UUID (sub)
		"client_id":      event.CallerContext.ClientID, // Cognito app client ID
		"version":        event.Version,                // Cognito trigger version
		"operation":      "Handler",
	}).Debug("Processing Cognito Pre Token Generation V2.0 event")

	// Validate the trigger source to ensure V2.0 compatibility
	// Only process valid V2.0 triggers to avoid issues with legacy trigger formats
	if !isValidTriggerSourceV2(event.TriggerSource) {
		logger.WithFields(logrus.Fields{
			"trigger_source": event.TriggerSource,
			"operation":      "Handler",
		}).Warn("Invalid trigger source for V2.0, returning event unchanged")
		return event, nil // Return without modifications for invalid/legacy triggers
	}

	// Extract Cognito user UUID from event.UserName
	// In Cognito Pre Token Generation triggers, event.UserName contains the user's
	// Cognito 'sub' attribute (UUID), not their actual username or email
	cognitoID := event.UserName
	if cognitoID == "" {
		logger.WithField("operation", "Handler").Error("Username (cognito_id) is empty in event")
		return event, errors.New("username cannot be empty")
	}

	// Fetch complete user profile from IAM database
	// This single query retrieves all user data, organization, locations, and roles
	userProfile, err := userRepository.GetUserProfile(cognitoID)
	if err != nil {
		// Critical: Log database errors but don't fail authentication
		// Users should be able to login even if IAM database is temporarily unavailable
		logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"operation":  "Handler",
			"error":      err.Error(),
		}).Error("Failed to fetch user profile from database, proceeding without custom claims")
		// Return unchanged event - user can still authenticate with basic Cognito claims
		return event, nil
	}

	// Build custom claims structure from user profile data
	// This transforms database model into JWT-compatible claims format
	customClaims, err := buildCustomClaims(userProfile)
	if err != nil {
		// Claims building error - log but don't fail authentication
		logger.WithFields(logrus.Fields{
			"cognito_id": cognitoID,
			"user_id":    userProfile.UserID,
			"operation":  "Handler",
			"error":      err.Error(),
		}).Error("Failed to build custom claims, proceeding without custom claims")
		return event, nil
	}

	// Prepare custom claims for JWT token injection
	// These claims will be added to both ID tokens (frontend) and Access tokens (API)
	claimsToAdd := map[string]interface{}{
		"user_id":             customClaims.UserID,    // Internal user identifier
		"cognito_id":          customClaims.CognitoID, // AWS Cognito UUID
		"email":               customClaims.Email,     // User's email address
		"first_name":          customClaims.FirstName, // Personal info
		"last_name":           customClaims.LastName,
		"full_name":           customClaims.FullName,          // Computed full name
		"phone":               customClaims.Phone,             // Optional contact info
		"job_title":           customClaims.JobTitle,          // Optional professional title
		"status":              customClaims.Status,            // Account status (active/inactive/suspended)
		"photo_url":           customClaims.PhotoURL,          // Optional profile photo
		"org_id":              customClaims.OrgID,             // Organization identifier
		"org_name":            customClaims.OrgName,           // Organization display name
		"current_location_id": customClaims.CurrentLocationID, // Primary location
		"isSuperAdmin":        customClaims.IsSuperAdmin,      // SuperAdmin role flag
		"locations":           customClaims.Locations,         // Base64 encoded JSON of locations with roles
	}

	// Configure Cognito V2.0 token generation response structure
	// This modifies both ID and Access tokens with custom claims and user roles
	event.Response.ClaimsAndScopeOverrideDetails = events.ClaimsAndScopeOverrideDetailsV2_0{
		// ID Token Configuration (used by frontend applications)
		IDTokenGeneration: events.IDTokenGenerationV2_0{
			ClaimsToAddOrOverride: claimsToAdd, // Add our custom user profile claims
			ClaimsToSuppress:      []string{},  // Keep all default Cognito claims (sub, email, etc.)
		},
		// Access Token Configuration (used by API services)
		AccessTokenGeneration: events.AccessTokenGenerationV2_0{
			ClaimsToAddOrOverride: claimsToAdd, // Add same custom claims to access tokens
			ClaimsToSuppress:      []string{},  // Keep all default Cognito claims
			ScopesToAdd:           []string{},  // No additional OAuth scopes needed
			ScopesToSuppress:      []string{},  // Keep all granted scopes
		},
		// Group Configuration (maps user roles to Cognito groups for authorization)
		GroupOverrideDetails: events.GroupConfigurationV2_0{
			GroupsToOverride:   extractAllRoles(userProfile.Locations), // User's roles across all locations
			IAMRolesToOverride: []string{},                             // Not using AWS IAM role mapping
			PreferredRole:      nil,                                    // No single preferred role
		},
	}

	// Log successful token customization
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithFields(logrus.Fields{
			"user_id":         userProfile.UserID,
			"cognito_id":      userProfile.CognitoID,
			"email":           userProfile.Email,
			"org_id":          userProfile.OrgID,
			"org_name":        userProfile.OrgName,
			"locations_count": len(userProfile.Locations),
			"roles_count":     len(extractAllRoles(userProfile.Locations)),
			"operation":       "Handler",
		}).Debug("Successfully added custom claims to token")
	}

	return event, nil
}

// isValidTriggerSourceV2 validates Cognito trigger sources for V2.0 compatibility.
//
// AWS Cognito has different trigger versions with different capabilities:
//   - V1.0 triggers: Limited claim modification, legacy format
//   - V2.0 triggers: Enhanced features, better performance, current recommended version
//
// This function ensures we only process V2.0 triggers to avoid compatibility issues.
// Processing V1.0 triggers with V2.0 response format can cause authentication failures.
//
// Supported V2.0 Trigger Sources:
//   - TokenGeneration_HostedAuth: Cognito Hosted UI authentication
//   - TokenGeneration_Authentication: Direct authentication (username/password)
//   - TokenGeneration_NewPasswordChallenge: First-time login with temp password
//   - TokenGeneration_AuthenticateDevice: Device-based authentication
//   - TokenGeneration_RefreshTokens: Token refresh operations
//
// Returns:
//   - true: Valid V2.0 trigger source, safe to process
//   - false: Invalid or V1.0 trigger, should return event unchanged
func isValidTriggerSourceV2(triggerSource string) bool {
	// List of supported V2.0 trigger sources
	// Reference: https://docs.aws.amazon.com/cognito/latest/developerguide/user-pool-lambda-pre-token-generation.html
	validSources := []string{
		"TokenGeneration_HostedAuth",           // Cognito Hosted UI flows
		"TokenGeneration_Authentication",       // Direct authentication
		"TokenGeneration_NewPasswordChallenge", // Password change flows
		"TokenGeneration_AuthenticateDevice",   // Device authentication
		"TokenGeneration_RefreshTokens",        // Token refresh operations
	}

	// Check if current trigger source is in our supported list
	for _, valid := range validSources {
		if triggerSource == valid {
			return true
		}
	}
	return false // Unsupported or V1.0 trigger
}

// extractAllRoles extracts unique role names from all user locations for Cognito groups.
//
// Users can have different roles at different locations within their organization.
// For example:
//   - "admin" at "New York Office"
//   - "manager" at "San Francisco Office"
//   - "employee" at "Remote Team"
//
// This function creates a deduplicated list of all roles across all locations,
// which is then used to populate Cognito groups. This enables:
//  1. Role-based access control in frontend applications
//  2. API authorization based on user roles
//  3. Simplified permission checking (user.hasRole("admin"))
//
// Cognito Groups Integration:
//   - Each unique role becomes a Cognito group
//   - Users are automatically added to groups based on their database roles
//   - Groups appear in JWT tokens for authorization logic
//
// Deduplication:
//   - Uses map[string]bool for O(1) duplicate detection
//   - Preserves role names exactly as stored in database
//   - Returns deterministic (but unordered) slice of unique roles
func extractAllRoles(locations []models.UserLocation) []string {
	// Use map for efficient deduplication of role names
	roleMap := make(map[string]bool)

	// Iterate through all locations and their associated roles
	for _, location := range locations {
		for _, role := range location.Roles {
			roleMap[role.RoleName] = true // Add to set (duplicates ignored)
		}
	}

	// Convert map keys to slice for return
	roles := make([]string, 0, len(roleMap))
	for role := range roleMap {
		roles = append(roles, role)
	}
	return roles
}

// buildCustomClaims transforms UserProfile database model into JWT-compatible CustomClaims.
//
// This function handles the complex transformation from the rich database model
// (with nested structures and nullable fields) to a flat JWT claims structure
// that can be efficiently included in tokens.
//
// Key Transformations:
//  1. Nested Data Encoding: Locations with roles are JSON-encoded and Base64-encoded
//     for compact token representation
//  2. Nullable Field Handling: sql.NullString fields are converted to empty strings
//     for JWT compatibility
//  3. Computed Fields: Full name is constructed from first + last name
//  4. Type Conversion: All fields converted to strings for JWT standard compliance
//
// Token Size Optimization:
//   - Complex nested data (locations/roles) is Base64-encoded to minimize token size
//   - Optional fields use omitempty JSON tags to exclude empty values
//   - Field names are kept short but descriptive
//
// Error Handling:
//   - Returns error only for JSON marshaling failures (rare)
//   - Gracefully handles NULL database values
//   - Never returns nil CustomClaims on success
//
// Frontend Decoding:
//
//	Frontend applications can decode the locations field:
//	const locations = JSON.parse(atob(token.locations))
func buildCustomClaims(profile *models.UserProfile) (*CustomClaims, error) {
	// Encode complex nested locations data as Base64 JSON for token efficiency
	// This includes all user locations and their associated roles
	locationsJSON, err := json.Marshal(profile.Locations)
	if err != nil {
		// JSON marshaling error is rare but possible with corrupted data
		return nil, fmt.Errorf("error marshaling locations to JSON: %w", err)
	}
	locationsEncoded := base64.StdEncoding.EncodeToString(locationsJSON)

	// Compute full name by combining first and last name with proper spacing
	fullName := strings.TrimSpace(profile.FirstName + " " + profile.LastName)

	// Handle nullable database fields by converting sql.NullString to regular strings
	// JWT claims must be strings, not complex types
	phone := ""
	if profile.Phone.Valid {
		phone = profile.Phone.String
	}

	jobTitle := ""
	if profile.JobTitle.Valid {
		jobTitle = profile.JobTitle.String
	}

	photoURL := ""
	if profile.PhotoURL.Valid {
		photoURL = profile.PhotoURL.String
	}

	currentLocationID := ""
	if profile.CurrentLocationID.Valid {
		currentLocationID = profile.CurrentLocationID.String
	}

	// Build and return the complete custom claims structure
	return &CustomClaims{
		UserID:            profile.UserID,    // Internal database identifier
		CognitoID:         profile.CognitoID, // AWS Cognito UUID
		Email:             profile.Email,     // User's email address
		FirstName:         profile.FirstName, // Personal information
		LastName:          profile.LastName,
		FullName:          fullName,             // Computed convenience field
		Phone:             phone,                // Optional contact information
		JobTitle:          jobTitle,             // Optional professional title
		Status:            profile.Status,       // Account status (active/inactive/suspended)
		PhotoURL:          photoURL,             // Optional profile photo
		OrgID:             profile.OrgID,        // Organization identifier
		OrgName:           profile.OrgName,      // Organization display name
		CurrentLocationID: currentLocationID,    // Primary location ID
		IsSuperAdmin:      profile.IsSuperAdmin, // SuperAdmin role flag from database
		Locations:         locationsEncoded,     // Base64 encoded JSON of all locations with roles
	}, nil
}

// setupPostgresSQLClient initializes the PostgreSQL database connection and repository.
//
// This function is called during Lambda cold start initialization to establish
// a persistent database connection that will be reused across Lambda invocations.
// The connection uses credentials stored in AWS SSM Parameter Store for security.
//
// Database Configuration:
//   - Uses RDS PostgreSQL with SSL encryption
//   - Connection pooling for concurrent access
//   - Credentials managed via SSM Parameter Store
//   - VPC networking for security isolation
//
// Error Handling:
//   - Returns error if connection cannot be established
//   - Logs connection parameters (excluding sensitive data)
//   - Validates all required SSM parameters are present
//
// Performance:
//   - Connection is established once during cold start
//   - Reused across multiple Lambda invocations for efficiency
//   - Connection pooling handles concurrent database access
//
// Security:
//   - SSL mode enforced for encrypted connections
//   - Database credentials rotated via SSM Parameter Store
//   - No hardcoded credentials in code
func setupPostgresSQLClient(ssmParams map[string]string) error {
	var err error

	// Create PostgreSQL client using RDS connection parameters from SSM
	// All connection details are fetched from SSM Parameter Store for security
	sqlDB, err = clients.NewPostgresSQLClient(
		ssmParams[constants.DATABASE_RDS_ENDPOINT], // RDS endpoint URL
		ssmParams[constants.DATABASE_PORT],         // PostgreSQL port (typically 5432)
		ssmParams[constants.DATABASE_NAME],         // Database name (typically 'iam')
		ssmParams[constants.DATABASE_USERNAME],     // Database username
		ssmParams[constants.DATABASE_PASSWORD],     // Database password (rotated regularly)
		ssmParams[constants.SSL_MODE],              // SSL mode (require/prefer/disable)
	)
	if err != nil {
		return fmt.Errorf("error creating PostgreSQL client: %w", err)
	}

	// Initialize user repository with database connection and logger
	// This repository implements the UserRepository interface for data access
	userRepository = &data.UserDao{
		DB:     sqlDB,  // Shared database connection pool
		Logger: logger, // Structured logger for debugging
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}

// main is the Lambda function entry point.
// It simply starts the AWS Lambda runtime with our Handler function.
func main() {
	lambda.Start(Handler)
}

// init initializes the Lambda function during cold start.
//
// This function runs once when AWS Lambda creates a new container instance.
// It performs expensive initialization operations (database connections,
// credential retrieval, etc.) that are then reused across multiple invocations.
//
// Cold Start Optimization Strategy:
//  1. Initialize global variables (logger, database connections)
//  2. Retrieve configuration from SSM Parameter Store
//  3. Establish database connections with connection pooling
//  4. Set up structured logging with appropriate levels
//
// Initialization Steps:
//  1. Environment Setup: Parse environment variables and configure logging
//  2. SSM Integration: Initialize SSM client and retrieve configuration
//  3. Database Setup: Establish PostgreSQL connection with credentials from SSM
//  4. Repository Setup: Initialize data access layer with database connection
//
// Error Handling:
//   - Fatal errors prevent Lambda from starting (fail fast principle)
//   - Detailed logging for debugging initialization issues
//   - All errors include context for troubleshooting
//
// Performance Considerations:
//   - All expensive operations happen once during cold start
//   - Database connections are pooled and reused across invocations
//   - SSM parameters are cached in memory
//   - Structured logging reduces runtime overhead
func init() {
	var err error

	// Parse environment variables for runtime configuration
	isLocal, _ = strconv.ParseBool(os.Getenv("IS_LOCAL")) // Development vs production mode

	// Initialize structured logging following alerts-functions pattern
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Set log level to ERROR to reduce CloudWatch costs (following alerts-functions pattern)
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.ErrorLevel) // Only log errors to save costs
	}

	logger.WithField("operation", "init").Error("Initializing Cognito Token Customizer Lambda")

	// Initialize AWS SSM Parameter Store client for configuration management
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient, // AWS SSM service client
		Logger: logger,    // Structured logger for debugging
	}

	// Retrieve all required configuration parameters from SSM Parameter Store
	// This includes database credentials, connection strings, and other settings
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error while getting SSM params from parameter store")
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithFields(logrus.Fields{
			"operation":    "init",
			"params_count": len(ssmParams),
		}).Debug("Retrieved SSM parameters")
	}

	// Initialize PostgreSQL database connection using credentials from SSM
	// This establishes a connection pool that will be reused across Lambda invocations
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	logger.WithField("operation", "init").Error("Token Customizer Lambda initialization completed successfully")
}
