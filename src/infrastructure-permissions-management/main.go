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
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

// Global variables for Lambda cold start optimization
var (
	logger                *logrus.Logger
	isLocal               bool
	ssmRepository         data.SSMRepository
	ssmParams             map[string]string
	sqlDB                 *sql.DB
	permissionRepository  data.PermissionRepository
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "Handler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
	}).Info("Permissions management request received")

	// Get user ID and org ID from the JWT token claims
	var claims map[string]interface{}
	var ok bool

	// Try different possible claim locations in the authorizer context
	if authClaims, exists := request.RequestContext.Authorizer["claims"]; exists {
		claims, ok = authClaims.(map[string]interface{})
	}

	// If claims not found, try direct access to authorizer
	if !ok {
		claims = request.RequestContext.Authorizer
		ok = (claims != nil)
	}

	if !ok || claims == nil {
		logger.Error("Failed to get claims from authorizer context")
		return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Missing claims"), nil
	}

	// Get the internal user_id from claims
	var userID int64
	if userIDValue, exists := claims["user_id"]; exists {
		if userIDStr, ok := userIDValue.(string); ok {
			var err error
			userID, err = strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				logger.WithError(err).Error("Failed to parse user_id string")
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid user_id format"), nil
			}
		} else if userIDFloat, ok := userIDValue.(float64); ok {
			userID = int64(userIDFloat)
		} else {
			logger.Error("user_id has unexpected type")
			return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Invalid user_id type"), nil
		}
	} else {
		logger.Error("user_id not found in claims")
		return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Missing user_id"), nil
	}

	// Get the org_id from claims
	var orgID int64
	if orgIDValue, exists := claims["org_id"]; exists {
		if orgIDStr, ok := orgIDValue.(string); ok {
			var err error
			orgID, err = strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				logger.WithError(err).Error("Failed to parse org_id string")
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid org_id format"), nil
			}
		} else if orgIDFloat, ok := orgIDValue.(float64); ok {
			orgID = int64(orgIDFloat)
		} else {
			logger.Error("org_id has unexpected type")
			return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Invalid org_id type"), nil
		}
	} else {
		logger.Error("org_id not found in claims")
		return util.CreateErrorResponse(http.StatusUnauthorized, "Unauthorized: Missing org_id"), nil
	}

	// Check if user is super admin
	var isSuperAdmin bool
	if superAdminValue, exists := claims["isSuperAdmin"]; exists {
		if isSuperAdmin, ok = superAdminValue.(bool); !ok {
			if superAdminStr, ok := superAdminValue.(string); ok && superAdminStr == "true" {
				isSuperAdmin = true
			}
		}
	}

	if !isSuperAdmin {
		logger.WithField("user_id", userID).Warn("User is not a super admin")
		return util.CreateErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage permissions"), nil
	}

	// Route based on HTTP method and path
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		// POST /permissions - Create new permission
		return handleCreatePermission(ctx, userID, orgID, request.Body), nil
		
	case http.MethodGet:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// GET /permissions/{id} - Get specific permission
			permissionID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid permission ID"), nil
			}
			return handleGetPermission(ctx, permissionID, orgID), nil
		} else {
			// GET /permissions - Get all permissions for org
			return handleGetPermissions(ctx, orgID), nil
		}
		
	case http.MethodPut:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// PUT /permissions/{id} - Update permission
			permissionID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid permission ID"), nil
			}
			return handleUpdatePermission(ctx, permissionID, orgID, request.Body), nil
		} else {
			return util.CreateErrorResponse(http.StatusBadRequest, "Permission ID required for update"), nil
		}
		
	case http.MethodDelete:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// DELETE /permissions/{id} - Delete permission
			permissionID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid permission ID"), nil
			}
			return handleDeletePermission(ctx, permissionID, orgID), nil
		} else {
			return util.CreateErrorResponse(http.StatusBadRequest, "Permission ID required for deletion"), nil
		}
		
	default:
		return util.CreateErrorResponse(http.StatusMethodNotAllowed, "Method not allowed"), nil
	}
}

// handleCreatePermission handles POST /permissions
func handleCreatePermission(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreatePermissionRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create permission request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if createReq.PermissionName == "" || len(createReq.PermissionName) < 2 || len(createReq.PermissionName) > 100 {
		return util.CreateErrorResponse(http.StatusBadRequest, "Permission name must be between 2 and 100 characters")
	}

	// Create permission object
	permission := &models.Permission{
		PermissionName: createReq.PermissionName,
		Description:    createReq.Description,
	}

	// Create permission
	createdPermission, err := permissionRepository.CreatePermission(ctx, orgID, permission)
	if err != nil {
		logger.WithError(err).Error("Failed to create permission")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to create permission")
	}

	// Return success response
	responseBody, _ := json.Marshal(createdPermission)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleGetPermissions handles GET /permissions
func handleGetPermissions(ctx context.Context, orgID int64) events.APIGatewayProxyResponse {
	permissions, err := permissionRepository.GetPermissionsByOrg(ctx, orgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get permissions")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to get permissions")
	}

	response := models.PermissionListResponse{
		Permissions: permissions,
		Total:       len(permissions),
	}

	responseBody, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleGetPermission handles GET /permissions/{id}
func handleGetPermission(ctx context.Context, permissionID, orgID int64) events.APIGatewayProxyResponse {
	permission, err := permissionRepository.GetPermissionByID(ctx, permissionID, orgID)
	if err != nil {
		if err.Error() == "permission not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Permission not found")
		}
		logger.WithError(err).Error("Failed to get permission")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to get permission")
	}

	responseBody, _ := json.Marshal(permission)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleUpdatePermission handles PUT /permissions/{id}
func handleUpdatePermission(ctx context.Context, permissionID, orgID int64, body string) events.APIGatewayProxyResponse {
	var updateReq models.UpdatePermissionRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update permission request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Create permission object with updates
	permission := &models.Permission{
		PermissionName: updateReq.PermissionName,
		Description:    updateReq.Description,
	}

	updatedPermission, err := permissionRepository.UpdatePermission(ctx, permissionID, orgID, permission)
	if err != nil {
		if err.Error() == "permission not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Permission not found")
		}
		logger.WithError(err).Error("Failed to update permission")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to update permission")
	}

	responseBody, _ := json.Marshal(updatedPermission)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleDeletePermission handles DELETE /permissions/{id}
func handleDeletePermission(ctx context.Context, permissionID, orgID int64) events.APIGatewayProxyResponse {
	err := permissionRepository.DeletePermission(ctx, permissionID, orgID)
	if err != nil {
		if err.Error() == "permission not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Permission not found")
		}
		logger.WithError(err).Error("Failed to delete permission")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to delete permission")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNoContent,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// main is the Lambda function entry point
func main() {
	lambda.Start(Handler)
}

func init() {
	var err error

	isLocal = parseIsLocal()

	// Logger Setup
	logger = setupLogger(isLocal)

	// Initialize AWS SSM Parameter Store client
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Retrieve all required configuration parameters from SSM
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

	// Initialize PostgreSQL database connection
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	logger.WithField("operation", "init").Info("Permissions Management Lambda initialization completed successfully")
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
	sqlDB, err = clients.NewPostgresSQLClient(
		ssmParams[constants.DATABASE_RDS_ENDPOINT],
		ssmParams[constants.DATABASE_PORT],
		ssmParams[constants.DATABASE_NAME],
		ssmParams[constants.DATABASE_USERNAME],
		ssmParams[constants.DATABASE_PASSWORD],
		ssmParams[constants.SSL_MODE],
	)
	if err != nil {
		return fmt.Errorf("error creating PostgreSQL client: %w", err)
	}

	// Initialize repository
	permissionRepository = &data.PermissionDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}