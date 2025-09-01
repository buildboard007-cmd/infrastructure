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

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	if !claims.IsSuperAdmin {
		logger.WithField("user_id", claims.UserID).Warn("User is not a super admin")
		return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage permissions", logger), nil
	}

	// Route based on HTTP method and path
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		// POST /permissions - Create new permission
		return handleCreatePermission(ctx, claims.UserID, claims.OrgID, request.Body), nil
		
	case http.MethodGet:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// GET /permissions/{id} - Get specific permission
			permissionID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid permission ID", logger), nil
			}
			return handleGetPermission(ctx, permissionID, claims.OrgID), nil
		} else {
			// GET /permissions - Get all permissions for org
			return handleGetPermissions(ctx, claims.OrgID), nil
		}
		
	case http.MethodPut:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// PUT /permissions/{id} - Update permission
			permissionID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid permission ID", logger), nil
			}
			return handleUpdatePermission(ctx, permissionID, claims.OrgID, request.Body), nil
		} else {
			return api.ErrorResponse(http.StatusBadRequest, "Permission ID required for update", logger), nil
		}
		
	case http.MethodDelete:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// DELETE /permissions/{id} - Delete permission
			permissionID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid permission ID", logger), nil
			}
			return handleDeletePermission(ctx, permissionID, claims.OrgID), nil
		} else {
			return api.ErrorResponse(http.StatusBadRequest, "Permission ID required for deletion", logger), nil
		}
		
	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
	}
}

// handleCreatePermission handles POST /permissions
func handleCreatePermission(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreatePermissionRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create permission request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate required fields
	if createReq.PermissionName == "" || len(createReq.PermissionName) < 2 || len(createReq.PermissionName) > 100 {
		return api.ErrorResponse(http.StatusBadRequest, "Permission name must be between 2 and 100 characters", logger)
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
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create permission", logger)
	}

	return api.SuccessResponse(http.StatusCreated, createdPermission, logger)
}

// handleGetPermissions handles GET /permissions
func handleGetPermissions(ctx context.Context, orgID int64) events.APIGatewayProxyResponse {
	permissions, err := permissionRepository.GetPermissionsByOrg(ctx, orgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get permissions")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get permissions", logger)
	}

	response := models.PermissionListResponse{
		Permissions: permissions,
		Total:       len(permissions),
	}

	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleGetPermission handles GET /permissions/{id}
func handleGetPermission(ctx context.Context, permissionID, orgID int64) events.APIGatewayProxyResponse {
	permission, err := permissionRepository.GetPermissionByID(ctx, permissionID, orgID)
	if err != nil {
		if err.Error() == "permission not found" {
			return api.ErrorResponse(http.StatusNotFound, "Permission not found", logger)
		}
		logger.WithError(err).Error("Failed to get permission")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get permission", logger)
	}

	return api.SuccessResponse(http.StatusOK, permission, logger)
}

// handleUpdatePermission handles PUT /permissions/{id}
func handleUpdatePermission(ctx context.Context, permissionID, orgID int64, body string) events.APIGatewayProxyResponse {
	var updateReq models.UpdatePermissionRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update permission request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Create permission object with updates
	permission := &models.Permission{
		PermissionName: updateReq.PermissionName,
		Description:    updateReq.Description,
	}

	updatedPermission, err := permissionRepository.UpdatePermission(ctx, permissionID, orgID, permission)
	if err != nil {
		if err.Error() == "permission not found" {
			return api.ErrorResponse(http.StatusNotFound, "Permission not found", logger)
		}
		logger.WithError(err).Error("Failed to update permission")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update permission", logger)
	}

	return api.SuccessResponse(http.StatusOK, updatedPermission, logger)
}

// handleDeletePermission handles DELETE /permissions/{id}
func handleDeletePermission(ctx context.Context, permissionID, orgID int64) events.APIGatewayProxyResponse {
	err := permissionRepository.DeletePermission(ctx, permissionID, orgID)
	if err != nil {
		if err.Error() == "permission not found" {
			return api.ErrorResponse(http.StatusNotFound, "Permission not found", logger)
		}
		logger.WithError(err).Error("Failed to delete permission")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete permission", logger)
	}

	return api.SuccessResponse(http.StatusNoContent, nil, logger)
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