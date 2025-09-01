package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"infrastructure/lib/api"
	"infrastructure/lib/auth"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"infrastructure/lib/util"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/sirupsen/logrus"
)

// Handler struct contains all dependencies for the Lambda function
type Handler struct {
	DB            *sql.DB
	Logger        *logrus.Logger
	CognitoClient *cognitoidentityprovider.Client
	UserPoolID    string
}

// Global variables for Lambda cold start optimization
// These are initialized once during Lambda cold start and reused across invocations
var (
	logger        *logrus.Logger     // Structured logger for debugging
	isLocal       bool               // Development/local execution flag
	ssmRepository data.SSMRepository // AWS SSM Parameter Store client interface
	ssmParams     map[string]string  // Cached SSM parameters (database config)
	sqlDB         *sql.DB            // PostgreSQL connection pool (reused across invocations)
	orgRepository data.OrgRepository // Organization repository for data operations
	handler       *Handler           // Main handler instance
)

func LambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "LambdaHandler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
		"resource":  request.Resource,
	}).Info("Infrastructure management request received")

	// Route based on path prefix
	if strings.HasPrefix(request.Path, "/users") || strings.HasPrefix(request.Resource, "/users") {
		return handler.handleUserRoutes(ctx, request)
	}

	// Organization management routes
	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	if !claims.IsSuperAdmin {
		logger.WithField("user_id", claims.UserID).Warn("User is not a super admin")
		return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage organization", logger), nil
	}

	// Handle PUT request to update organization
	if request.HTTPMethod == http.MethodPut {
		return handleUpdateOrganization(ctx, claims.UserID, request.Body), nil
	}
	
	// Handle GET request to retrieve organization
	if request.HTTPMethod == http.MethodGet {
		return handleGetOrganization(ctx, claims.UserID), nil
	}

	return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
}

// handleUpdateOrganization handles the PUT request to update organization info
func handleUpdateOrganization(ctx context.Context, userID int64, body string) events.APIGatewayProxyResponse {
	// Parse request body
	var updateReq models.UpdateOrganizationRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse request body")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate request
	if updateReq.OrgName == "" || len(updateReq.OrgName) < 3 || len(updateReq.OrgName) > 150 {
		return api.ErrorResponse(http.StatusBadRequest, "Organization name must be between 3 and 150 characters", logger)
	}

	// Update organization
	org := &models.Organization{
		OrgName: updateReq.OrgName,
	}

	updatedOrg, err := orgRepository.UpdateOrganization(ctx, userID, org)
	if err != nil {
		logger.WithError(err).Error("Failed to update organization")
		if err.Error() == "unauthorized: user is not a super admin" {
			return api.ErrorResponse(http.StatusForbidden, err.Error(), logger)
		}
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update organization", logger)
	}

	return api.SuccessResponse(http.StatusOK, updatedOrg, logger)
}

// handleGetOrganization handles the GET request to retrieve organization info
func handleGetOrganization(ctx context.Context, userID int64) events.APIGatewayProxyResponse {
	// Get organization
	org, err := orgRepository.GetOrganizationByUserID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to get organization")
		return api.ErrorResponse(http.StatusNotFound, "Organization not found", logger)
	}

	return api.SuccessResponse(http.StatusOK, org, logger)
}

// main is the Lambda function entry point.
// It simply starts the AWS Lambda runtime with our Handler function.
func main() {
	lambda.Start(LambdaHandler)
}

func init() {
	var err error

	isLocal = parseIsLocal()

	// --- Logger Setup ---
	logger = setupLogger(isLocal)

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

	logger.WithFields(logrus.Fields{
		"operation":    "init",
		"params_count": len(ssmParams),
	}).Debug("Retrieved SSM parameters")

	// Initialize PostgreSQL database connection using credentials from SSM
	// This establishes a connection pool that will be reused across Lambda invocations
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	logger.WithField("operation", "init").Error("Organization Management Lambda initialization completed successfully")
}

func parseIsLocal() bool {
	isLocal, _ := strconv.ParseBool(os.Getenv("IS_LOCAL"))
	return isLocal
}

func setupLogger(isLocal bool) *logrus.Logger {
	logger := logrus.New()
	util.SetLogLevel(logger, os.Getenv("LOG_LEVEL"))
	logger.SetFormatter(&logrus.JSONFormatter{PrettyPrint: isLocal})
	return logger
}

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

	// Initialize org repository with database connection and logger
	// This repository implements the OrgRepository interface for data access
	orgRepository = &data.OrgDao{
		DB:     sqlDB,  // Shared database connection pool
		Logger: logger, // Structured logger for debugging
	}

	// Initialize Cognito client
	cognitoClient := clients.NewCognitoIdentityProviderClient(isLocal)
	
	// Get User Pool ID from SSM parameters
	userPoolID := ssmParams[constants.COGNITO_USER_POOL_ID]
	if userPoolID == "" {
		logger.Fatal("COGNITO_USER_POOL_ID not found in SSM parameters")
	}

	// Initialize handler with all dependencies
	handler = &Handler{
		DB:            sqlDB,
		Logger:        logger,
		CognitoClient: cognitoClient,
		UserPoolID:    userPoolID,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}
