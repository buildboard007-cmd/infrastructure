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
	logger                     *logrus.Logger
	isLocal                    bool
	ssmRepository              data.SSMRepository
	ssmParams                  map[string]string
	sqlDB                      *sql.DB
	roleRepository             data.RoleRepository
	rolePermissionRepository   data.RolePermissionRepository
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "Handler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
	}).Info("Roles management request received")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	if !claims.IsSuperAdmin {
		logger.WithField("user_id", claims.UserID).Warn("User is not a super admin")
		return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage roles", logger), nil
	}

	// Route based on HTTP method and path
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		if len(pathSegments) >= 3 && pathSegments[2] == "permissions" {
			// POST /roles/{id}/permissions - Assign permission to role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid role ID", logger), nil
			}
			return handleAssignPermission(ctx, roleID, claims.OrgID, request.Body), nil
		} else {
			// POST /roles - Create new role
			return handleCreateRole(ctx, claims.UserID, claims.OrgID, request.Body), nil
		}
		
	case http.MethodGet:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// GET /roles/{id} - Get specific role with permissions
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid role ID", logger), nil
			}
			return handleGetRoleWithPermissions(ctx, roleID, claims.OrgID), nil
		} else {
			// GET /roles - Get all roles for org
			return handleGetRoles(ctx, claims.OrgID), nil
		}
		
	case http.MethodPut:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// PUT /roles/{id} - Update role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid role ID", logger), nil
			}
			return handleUpdateRole(ctx, roleID, claims.OrgID, request.Body), nil
		} else {
			return api.ErrorResponse(http.StatusBadRequest, "Role ID required for update", logger), nil
		}
		
	case http.MethodDelete:
		if len(pathSegments) >= 3 && pathSegments[2] == "permissions" {
			// DELETE /roles/{id}/permissions - Unassign permission from role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid role ID", logger), nil
			}
			return handleUnassignPermission(ctx, roleID, claims.OrgID, request.Body), nil
		} else if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// DELETE /roles/{id} - Delete role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return api.ErrorResponse(http.StatusBadRequest, "Invalid role ID", logger), nil
			}
			return handleDeleteRole(ctx, roleID, claims.OrgID), nil
		} else {
			return api.ErrorResponse(http.StatusBadRequest, "Role ID required for deletion", logger), nil
		}
		
	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
	}
}

// handleCreateRole handles POST /roles
func handleCreateRole(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreateRoleRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create role request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Validate required fields
	if createReq.RoleName == "" || len(createReq.RoleName) < 2 || len(createReq.RoleName) > 100 {
		return api.ErrorResponse(http.StatusBadRequest, "Role name must be between 2 and 100 characters", logger)
	}

	// Create role object
	role := &models.Role{
		RoleName:    createReq.RoleName,
		Description: createReq.Description,
	}

	// Create role
	createdRole, err := roleRepository.CreateRole(ctx, orgID, role)
	if err != nil {
		logger.WithError(err).Error("Failed to create role")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create role", logger)
	}

	return api.SuccessResponse(http.StatusCreated, createdRole, logger)
}

// handleGetRoles handles GET /roles
func handleGetRoles(ctx context.Context, orgID int64) events.APIGatewayProxyResponse {
	roles, err := roleRepository.GetRolesByOrg(ctx, orgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get roles")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get roles", logger)
	}

	response := models.RoleListResponse{
		Roles: roles,
		Total: len(roles),
	}

	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleGetRoleWithPermissions handles GET /roles/{id}
func handleGetRoleWithPermissions(ctx context.Context, roleID, orgID int64) events.APIGatewayProxyResponse {
	roleWithPermissions, err := roleRepository.GetRoleWithPermissions(ctx, roleID, orgID)
	if err != nil {
		if err.Error() == "role not found" {
			return api.ErrorResponse(http.StatusNotFound, "Role not found", logger)
		}
		logger.WithError(err).Error("Failed to get role with permissions")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get role", logger)
	}

	return api.SuccessResponse(http.StatusOK, roleWithPermissions, logger)
}

// handleUpdateRole handles PUT /roles/{id}
func handleUpdateRole(ctx context.Context, roleID, orgID int64, body string) events.APIGatewayProxyResponse {
	var updateReq models.UpdateRoleRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update role request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Create role object with updates
	role := &models.Role{
		RoleName:    updateReq.RoleName,
		Description: updateReq.Description,
	}

	updatedRole, err := roleRepository.UpdateRole(ctx, roleID, orgID, role)
	if err != nil {
		if err.Error() == "role not found" {
			return api.ErrorResponse(http.StatusNotFound, "Role not found", logger)
		}
		logger.WithError(err).Error("Failed to update role")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update role", logger)
	}

	return api.SuccessResponse(http.StatusOK, updatedRole, logger)
}

// handleDeleteRole handles DELETE /roles/{id}
func handleDeleteRole(ctx context.Context, roleID, orgID int64) events.APIGatewayProxyResponse {
	err := roleRepository.DeleteRole(ctx, roleID, orgID)
	if err != nil {
		if err.Error() == "role not found" {
			return api.ErrorResponse(http.StatusNotFound, "Role not found", logger)
		}
		logger.WithError(err).Error("Failed to delete role")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete role", logger)
	}

	return api.SuccessResponse(http.StatusNoContent, nil, logger)
}

// handleAssignPermission handles POST /roles/{id}/permissions
func handleAssignPermission(ctx context.Context, roleID, orgID int64, body string) events.APIGatewayProxyResponse {
	var assignReq models.AssignPermissionRequest
	if err := json.Unmarshal([]byte(body), &assignReq); err != nil {
		logger.WithError(err).Error("Failed to parse assign permission request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	err := rolePermissionRepository.AssignPermissionToRole(ctx, roleID, assignReq.PermissionID, orgID)
	if err != nil {
		if err.Error() == "role not found" || err.Error() == "permission not found" {
			return api.ErrorResponse(http.StatusNotFound, "Role or permission not found", logger)
		}
		logger.WithError(err).Error("Failed to assign permission to role")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to assign permission", logger)
	}

	response := map[string]string{"message": "Permission assigned successfully"}
	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleUnassignPermission handles DELETE /roles/{id}/permissions
func handleUnassignPermission(ctx context.Context, roleID, orgID int64, body string) events.APIGatewayProxyResponse {
	var unassignReq models.UnassignPermissionRequest
	if err := json.Unmarshal([]byte(body), &unassignReq); err != nil {
		logger.WithError(err).Error("Failed to parse unassign permission request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	err := rolePermissionRepository.UnassignPermissionFromRole(ctx, roleID, unassignReq.PermissionID, orgID)
	if err != nil {
		if err.Error() == "role or permission not found" || err.Error() == "permission not assigned to role" {
			return api.ErrorResponse(http.StatusNotFound, "Role, permission not found or permission not assigned", logger)
		}
		logger.WithError(err).Error("Failed to unassign permission from role")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to unassign permission", logger)
	}

	response := map[string]string{"message": "Permission unassigned successfully"}
	return api.SuccessResponse(http.StatusOK, response, logger)
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

	logger.WithField("operation", "init").Info("Roles Management Lambda initialization completed successfully")
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

	// Initialize repositories
	roleRepository = &data.RoleDao{
		DB:     sqlDB,
		Logger: logger,
	}
	
	rolePermissionRepository = &data.RolePermissionDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}