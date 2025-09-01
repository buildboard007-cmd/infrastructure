// Package main implements the AWS Cognito Post-Confirmation Lambda trigger.
//
// This Lambda function handles two distinct user registration scenarios:
// 1. Admin Signup: First user who signs up becomes admin with "pending_org_setup" status
// 2. Invited Users: Users created by admin through AdminCreateUser, activated on confirmation
//
// Key Responsibilities:
// 1. Detect signup type (self-signup vs invite)
// 2. Create user record in IAM database with appropriate status
// 3. For admin signup: Set status to "pending_org_setup"
// 4. For invited users: Update status from "pending" to "active"
// 5. Handle errors gracefully without blocking Cognito signup flow
//
// Admin Signup Flow:
//   - User signs up via Cognito SignUp API
//   - Email verification required
//   - PostConfirmation trigger creates user with "pending_org_setup" status
//   - Admin must complete org setup wizard before full access
//
// Invited User Flow:
//   - Admin creates user via dashboard (status = "pending")
//   - Admin calls Cognito AdminCreateUser (sends invite email)
//   - User sets password and confirms
//   - PostConfirmation trigger updates status to "active"
//
// Database Integration:
//   - Uses PostgreSQL IAM database with transaction support
//   - Stores minimal data: cognito_id, email, status
//   - Additional user data managed by admin during org setup or invitation
//
// Security:
//   - SSL database connections with SSM credential rotation
//   - VPC isolation for database access
//   - Correlation IDs for tracking user signup journeys
package main

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger        *logrus.Logger     // Structured logger for debugging
	isLocal       bool               // Development/local execution flag
	ssmRepository data.SSMRepository // AWS SSM Parameter Store client interface
	ssmParams     map[string]string  // Cached SSM parameters (database config)
	sqlDB         *sql.DB            // PostgreSQL connection pool (reused across invocations)
)

// SignupRequest represents the extracted data from Cognito Post-Confirmation event
type SignupRequest struct {
	CognitoID     string // AWS Cognito user UUID (from 'sub' attribute)
	Email         string // User's email address
	FirstName     string // User's first name
	LastName      string // User's last name
	Phone         string // User's phone number (optional)
	CorrelationID string // Unique ID for tracking this signup process
	SignupType    string // "superadmin_signup" only - other types not supported
}

// UserStatus constants
const (
	StatusPendingOrgSetup = "pending_org_setup"
	StatusActive          = "active"
)

// Handler processes the Cognito Post-Confirmation trigger event.
//
// This function only handles SuperAdmin signup (isSuperAdmin=true):
//
// SuperAdmin Signup:
//   - Creates new user record with "pending_org_setup" status in "System" organization
//   - SuperAdmin must complete org setup wizard before accessing app
//   - All other signup attempts are rejected
//
// Event Processing Flow:
//  1. Extract user data from Cognito event
//  2. Generate correlation ID for tracking
//  3. Determine signup type (admin vs invited)
//  4. Process accordingly (create vs update user)
//  5. Log success/failure for monitoring
//
// Error Handling:
//   - Never return error that would block Cognito confirmation
//   - Log all errors with correlation ID for debugging
//   - Graceful degradation if database operations fail
//
// The function ensures users can always complete Cognito confirmation flow
// even if IAM database operations fail, maintaining system availability.
func Handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
	// Generate correlation ID for tracking this signup process end-to-end
	correlationID := uuid.New().String()

	// Extract user data from Cognito event
	signupRequest, err := extractSignupData(event, correlationID)
	if err != nil {
		// Log error but don't fail Cognito confirmation
		logger.WithFields(logrus.Fields{
			"correlation_id": correlationID,
			"trigger_source": event.TriggerSource,
			"operation":      "Handler",
			"error":          err.Error(),
		}).Error("Failed to extract signup data from Cognito event")
		return event, nil // Return success to Cognito
	}

	// Log incoming signup request
	logger.WithFields(logrus.Fields{
		"correlation_id": signupRequest.CorrelationID,
		"cognito_id":     signupRequest.CognitoID,
		"email":          signupRequest.Email,
		"first_name":     signupRequest.FirstName,
		"last_name":      signupRequest.LastName,
		"phone":          signupRequest.Phone,
		"signup_type":    signupRequest.SignupType,
		"operation":      "Handler",
	}).Debug("Processing Cognito Post-Confirmation event")

	// Process signup based on type
	err = processSignup(signupRequest)
	if err != nil {
		// Log error but don't fail Cognito confirmation
		logger.WithFields(logrus.Fields{
			"correlation_id": signupRequest.CorrelationID,
			"cognito_id":     signupRequest.CognitoID,
			"signup_type":    signupRequest.SignupType,
			"operation":      "Handler",
			"error":          err.Error(),
		}).Error("Failed to process signup, user can still login but may need admin assistance")
		return event, nil // Return success to Cognito
	}

	// Log successful signup completion
	logger.WithFields(logrus.Fields{
		"correlation_id": signupRequest.CorrelationID,
		"cognito_id":     signupRequest.CognitoID,
		"signup_type":    signupRequest.SignupType,
		"operation":      "Handler",
	}).Debug("Successfully completed signup processing")

	return event, nil // Always return success to Cognito
}

// extractSignupData extracts and validates user data from Cognito Post-Confirmation event
func extractSignupData(event events.CognitoEventUserPoolsPostConfirmation, correlationID string) (*SignupRequest, error) {
	// Extract Cognito user UUID from username
	cognitoID := event.UserName
	if cognitoID == "" {
		return nil, fmt.Errorf("cognito ID (username) is empty")
	}

	// Extract email from user attributes
	email := event.Request.UserAttributes["email"]
	if email == "" {
		return nil, fmt.Errorf("email attribute is missing from Cognito event")
	}

	// Extract additional user data from ClientMetadata (passed during signup)
	// ClientMetadata is used to pass additional data that we don't want to store in Cognito
	firstName := ""
	lastName := ""
	phone := ""
	
	if event.Request.ClientMetadata != nil {
		firstName = event.Request.ClientMetadata["firstName"]
		lastName = event.Request.ClientMetadata["lastName"]
		phone = event.Request.ClientMetadata["phone"]
	}
	
	// If ClientMetadata is not available, these fields will remain empty
	// The database allows NULL for phone, and we'll use empty strings for names
	if firstName == "" {
		firstName = "FirstName" // Default placeholder
	}
	if lastName == "" {
		lastName = "LastName" // Default placeholder
	}

	// Determine signup type based on trigger source and user attributes
	signupType := determineSignupType(event)

	return &SignupRequest{
		CognitoID:     cognitoID,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		Phone:         phone,
		CorrelationID: correlationID,
		SignupType:    signupType,
	}, nil
}

// determineSignupType determines if this is a SuperAdmin signup
func determineSignupType(event events.CognitoEventUserPoolsPostConfirmation) string {
	// Check if this is a SuperAdmin signup
	isSuperAdmin := event.Request.UserAttributes["custom:isSuperAdmin"]
	if isSuperAdmin == "true" {
		return "superadmin_signup"
	}

	// All other signups are not supported in this simplified flow
	return "unsupported"
}

// processSignup handles the signup processing based on type
func processSignup(request *SignupRequest) error {
	// Start database transaction for atomic operations
	tx, err := sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error

	// Process based on signup type
	switch request.SignupType {
	case "superadmin_signup":
		err = processSuperAdminSignup(tx, request)
	case "unsupported":
		return fmt.Errorf("only SuperAdmin signup is supported - users must sign up with isSuperAdmin=true")
	default:
		return fmt.Errorf("unknown signup type: %s", request.SignupType)
	}

	if err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"correlation_id": request.CorrelationID,
		"cognito_id":     request.CognitoID,
		"signup_type":    request.SignupType,
		"operation":      "processSignup",
	}).Debug("Successfully processed signup")

	return nil
}

// processSuperAdminSignup handles the SuperAdmin signup flow
func processSuperAdminSignup(tx *sql.Tx, request *SignupRequest) error {
	// Create a new organization for this SuperAdmin
	// Each SuperAdmin gets their own organization
	var orgID int64
	
	// Create a unique organization for this super admin
	// Initial name will be "New Organization" and they can update it later
	err := tx.QueryRow(`
		INSERT INTO iam.organizations (name, org_type, status, created_by, updated_by)
		VALUES ('New Organization', 'general_contractor', 'pending_setup', 1, 1)
		RETURNING id
	`).Scan(&orgID)

	if err != nil {
		return fmt.Errorf("failed to create organization for super admin: %w", err)
	}

	// Create SuperAdmin user record with pending_org_setup status
	// Handle phone as nullable field
	var phone sql.NullString
	if request.Phone != "" {
		phone = sql.NullString{String: request.Phone, Valid: true}
	}
	
	_, err = tx.Exec(`
		INSERT INTO iam.users (
			cognito_id, 
			org_id,
			email, 
			first_name, 
			last_name,
			phone,
			status, 
			is_super_admin,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, request.CognitoID, orgID, request.Email, request.FirstName, request.LastName, phone, StatusPendingOrgSetup, true, 1, 1)

	if err != nil {
		return fmt.Errorf("failed to create SuperAdmin user record: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"correlation_id": request.CorrelationID,
		"cognito_id":     request.CognitoID,
		"email":          request.Email,
		"first_name":     request.FirstName,
		"last_name":      request.LastName,
		"phone":          request.Phone,
		"status":         StatusPendingOrgSetup,
		"isSuperAdmin":   true,
		"operation":      "processSuperAdminSignup",
	}).Debug("Created SuperAdmin user with pending_org_setup status")

	return nil
}

// setupPostgresSQLClient initializes the PostgreSQL database connection
func setupPostgresSQLClient(ssmParams map[string]string) error {
	var err error

	// Create PostgreSQL client using RDS connection parameters from SSM (following token-customizer pattern)
	sqlDB, err = clients.NewPostgresSQLClient(
		ssmParams[constants.DATABASE_RDS_ENDPOINT], // RDS endpoint URL
		ssmParams[constants.DATABASE_PORT],         // PostgreSQL port
		ssmParams[constants.DATABASE_NAME],         // Database name
		ssmParams[constants.DATABASE_USERNAME],     // Database username
		ssmParams[constants.DATABASE_PASSWORD],     // Database password
		ssmParams[constants.SSL_MODE],              // SSL mode
	)
	if err != nil {
		return fmt.Errorf("error creating PostgreSQL client: %w", err)
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}

// main is the Lambda function entry point
func main() {
	lambda.Start(Handler)
}

// init initializes the Lambda function during cold start
func init() {
	var err error

	// Parse environment variables
	isLocal, _ = strconv.ParseBool(os.Getenv("IS_LOCAL"))

	// Setup logging following alerts-functions pattern
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Set log level to ERROR to reduce CloudWatch costs (following alerts-functions pattern)
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.ErrorLevel) // Only log errors to save costs
	}

	logger.WithField("operation", "init").Error("Initializing Cognito User Signup Lambda")

	// Setup SSM client
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Get SSM parameters
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error while getting SSM params from parameter store")
	}

	logger.WithFields(logrus.Fields{
		"operation":    "init",
		"params_count": len(ssmParams),
	}).Debug("Retrieved SSM parameters")

	// Setup PostgreSQL client
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	logger.WithField("operation", "init").Error("User Signup Lambda initialization completed successfully")
}
