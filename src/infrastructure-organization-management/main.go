package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"infrastructure/lib/util"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
// These are initialized once during Lambda cold start and reused across invocations
var (
	logger        *logrus.Logger     // Structured logger for debugging
	isLocal       bool               // Development/local execution flag
	ssmRepository data.SSMRepository // AWS SSM Parameter Store client interface
	ssmParams     map[string]string  // Cached SSM parameters (database config)
	sqlDB         *sql.DB            // PostgreSQL connection pool (reused across invocations)
	orgRepository data.OrgRepository // Organization repository for data operations
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "Handler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
	}).Info("Organization management request received")

	// Get user ID from the JWT token claims (added by token customizer)
	// The API Gateway with Cognito authorizer adds the decoded JWT claims to the request context
	// Debug: Log the entire authorizer context to understand the structure
	logger.WithField("authorizer", request.RequestContext.Authorizer).Debug("Full authorizer context")
	
	var claims map[string]interface{}
	var ok bool
	
	// Try different possible claim locations in the authorizer context
	if authClaims, exists := request.RequestContext.Authorizer["claims"]; exists {
		claims, ok = authClaims.(map[string]interface{})
	}
	
	// If claims not found, try direct access to authorizer (some API Gateway configurations)
	if !ok {
		claims = request.RequestContext.Authorizer
		ok = (claims != nil)
	}
	
	if !ok || claims == nil {
		logger.WithField("authorizer", request.RequestContext.Authorizer).Error("Failed to get claims from authorizer context")
		return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Missing claims"), nil
	}
	
	logger.WithField("claims", claims).Debug("Extracted claims from authorizer context")
	
	// Get the internal user_id from claims (added by token customizer)
	// user_id could be a string or number, so let's try both
	var userIDStr string
	var userID int64
	var err error
	
	if userIDValue, exists := claims["user_id"]; exists {
		logger.WithField("user_id_raw", userIDValue).Debug("Raw user_id from claims")
		
		// Try as string first
		if userIDStr, ok = userIDValue.(string); ok {
			userID, err = strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				logger.WithError(err).Error("Failed to parse user_id string")
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid user_id format"), nil
			}
		} else if userIDFloat, ok := userIDValue.(float64); ok {
			// Try as float64 (JSON numbers are parsed as float64)
			userID = int64(userIDFloat)
		} else {
			logger.WithField("user_id_type", fmt.Sprintf("%T", userIDValue)).Error("user_id has unexpected type")
			return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Invalid user_id type"), nil
		}
	} else {
		logger.WithField("available_claims", fmt.Sprintf("%+v", claims)).Error("user_id not found in claims")
		return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Missing user_id"), nil
	}

	logger.WithField("user_id", userID).Debug("Successfully extracted user_id")

	// Check if user is super admin from claims
	// isSuperAdmin could be boolean or string, so let's try both
	var isSuperAdmin bool
	if superAdminValue, exists := claims["isSuperAdmin"]; exists {
		logger.WithField("isSuperAdmin_raw", superAdminValue).Debug("Raw isSuperAdmin from claims")
		
		if isSuperAdmin, ok = superAdminValue.(bool); !ok {
			// Try as string "true"/"false"
			if superAdminStr, ok := superAdminValue.(string); ok && superAdminStr == "true" {
				isSuperAdmin = true
			}
		}
	}
	
	logger.WithField("isSuperAdmin", isSuperAdmin).Debug("Extracted isSuperAdmin value")
	
	if !isSuperAdmin {
		logger.WithFields(logrus.Fields{
			"user_id": userID,
			"available_claims": fmt.Sprintf("%+v", claims),
		}).Warn("User is not a super admin")
		return util.CreateErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage organization"), nil
	}

	// Handle PUT request to update organization
	if request.HTTPMethod == http.MethodPut {
		return handleUpdateOrganization(ctx, userID, request.Body), nil
	}
	
	// Handle GET request to retrieve organization
	if request.HTTPMethod == http.MethodGet {
		return handleGetOrganization(ctx, userID), nil
	}

	return util.CreateErrorResponse(http.StatusMethodNotAllowed, "Method not allowed"), nil
}

// handleUpdateOrganization handles the PUT request to update organization info
func handleUpdateOrganization(ctx context.Context, userID int64, body string) events.APIGatewayProxyResponse {
	// Parse request body
	var updateReq models.UpdateOrganizationRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse request body")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Validate request
	if updateReq.OrgName == "" || len(updateReq.OrgName) < 3 || len(updateReq.OrgName) > 150 {
		return util.CreateErrorResponse(http.StatusBadRequest, "Organization name must be between 3 and 150 characters")
	}

	// Update organization
	org := &models.Organization{
		OrgName: updateReq.OrgName,
	}

	updatedOrg, err := orgRepository.UpdateOrganization(ctx, userID, org)
	if err != nil {
		logger.WithError(err).Error("Failed to update organization")
		if err.Error() == "unauthorized: user is not a super admin" {
			return util.CreateErrorResponse(http.StatusForbidden, err.Error())
		}
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to update organization")
	}

	// Return success response
	responseBody, _ := json.Marshal(updatedOrg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleGetOrganization handles the GET request to retrieve organization info
func handleGetOrganization(ctx context.Context, userID int64) events.APIGatewayProxyResponse {
	// Get organization
	org, err := orgRepository.GetOrganizationByUserID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to get organization")
		return util.CreateErrorResponse(http.StatusNotFound, "Organization not found")
	}

	// Return success response
	responseBody, _ := json.Marshal(org)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// main is the Lambda function entry point.
// It simply starts the AWS Lambda runtime with our Handler function.
func main() {
	lambda.Start(Handler)
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

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}
