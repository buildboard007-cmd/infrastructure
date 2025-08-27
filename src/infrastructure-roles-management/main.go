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
		return util.CreateErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage roles"), nil
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
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid role ID"), nil
			}
			return handleAssignPermission(ctx, roleID, orgID, request.Body), nil
		} else {
			// POST /roles - Create new role
			return handleCreateRole(ctx, userID, orgID, request.Body), nil
		}
		
	case http.MethodGet:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// GET /roles/{id} - Get specific role with permissions
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid role ID"), nil
			}
			return handleGetRoleWithPermissions(ctx, roleID, orgID), nil
		} else {
			// GET /roles - Get all roles for org
			return handleGetRoles(ctx, orgID), nil
		}
		
	case http.MethodPut:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// PUT /roles/{id} - Update role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid role ID"), nil
			}
			return handleUpdateRole(ctx, roleID, orgID, request.Body), nil
		} else {
			return util.CreateErrorResponse(http.StatusBadRequest, "Role ID required for update"), nil
		}
		
	case http.MethodDelete:
		if len(pathSegments) >= 3 && pathSegments[2] == "permissions" {
			// DELETE /roles/{id}/permissions - Unassign permission from role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid role ID"), nil
			}
			return handleUnassignPermission(ctx, roleID, orgID, request.Body), nil
		} else if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// DELETE /roles/{id} - Delete role
			roleID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid role ID"), nil
			}
			return handleDeleteRole(ctx, roleID, orgID), nil
		} else {
			return util.CreateErrorResponse(http.StatusBadRequest, "Role ID required for deletion"), nil
		}
		
	default:
		return util.CreateErrorResponse(http.StatusMethodNotAllowed, "Method not allowed"), nil
	}
}

// handleCreateRole handles POST /roles
func handleCreateRole(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreateRoleRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create role request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if createReq.RoleName == "" || len(createReq.RoleName) < 2 || len(createReq.RoleName) > 100 {
		return util.CreateErrorResponse(http.StatusBadRequest, "Role name must be between 2 and 100 characters")
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
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to create role")
	}

	// Return success response
	responseBody, _ := json.Marshal(createdRole)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleGetRoles handles GET /roles
func handleGetRoles(ctx context.Context, orgID int64) events.APIGatewayProxyResponse {
	roles, err := roleRepository.GetRolesByOrg(ctx, orgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get roles")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to get roles")
	}

	response := models.RoleListResponse{
		Roles: roles,
		Total: len(roles),
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

// handleGetRoleWithPermissions handles GET /roles/{id}
func handleGetRoleWithPermissions(ctx context.Context, roleID, orgID int64) events.APIGatewayProxyResponse {
	roleWithPermissions, err := roleRepository.GetRoleWithPermissions(ctx, roleID, orgID)
	if err != nil {
		if err.Error() == "role not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Role not found")
		}
		logger.WithError(err).Error("Failed to get role with permissions")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to get role")
	}

	responseBody, _ := json.Marshal(roleWithPermissions)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleUpdateRole handles PUT /roles/{id}
func handleUpdateRole(ctx context.Context, roleID, orgID int64, body string) events.APIGatewayProxyResponse {
	var updateReq models.UpdateRoleRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update role request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Create role object with updates
	role := &models.Role{
		RoleName:    updateReq.RoleName,
		Description: updateReq.Description,
	}

	updatedRole, err := roleRepository.UpdateRole(ctx, roleID, orgID, role)
	if err != nil {
		if err.Error() == "role not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Role not found")
		}
		logger.WithError(err).Error("Failed to update role")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to update role")
	}

	responseBody, _ := json.Marshal(updatedRole)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleDeleteRole handles DELETE /roles/{id}
func handleDeleteRole(ctx context.Context, roleID, orgID int64) events.APIGatewayProxyResponse {
	err := roleRepository.DeleteRole(ctx, roleID, orgID)
	if err != nil {
		if err.Error() == "role not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Role not found")
		}
		logger.WithError(err).Error("Failed to delete role")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to delete role")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNoContent,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleAssignPermission handles POST /roles/{id}/permissions
func handleAssignPermission(ctx context.Context, roleID, orgID int64, body string) events.APIGatewayProxyResponse {
	var assignReq models.AssignPermissionRequest
	if err := json.Unmarshal([]byte(body), &assignReq); err != nil {
		logger.WithError(err).Error("Failed to parse assign permission request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	err := rolePermissionRepository.AssignPermissionToRole(ctx, roleID, assignReq.PermissionID, orgID)
	if err != nil {
		if err.Error() == "role not found" || err.Error() == "permission not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Role or permission not found")
		}
		logger.WithError(err).Error("Failed to assign permission to role")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to assign permission")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       `{"message":"Permission assigned successfully"}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleUnassignPermission handles DELETE /roles/{id}/permissions
func handleUnassignPermission(ctx context.Context, roleID, orgID int64, body string) events.APIGatewayProxyResponse {
	var unassignReq models.UnassignPermissionRequest
	if err := json.Unmarshal([]byte(body), &unassignReq); err != nil {
		logger.WithError(err).Error("Failed to parse unassign permission request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	err := rolePermissionRepository.UnassignPermissionFromRole(ctx, roleID, unassignReq.PermissionID, orgID)
	if err != nil {
		if err.Error() == "role or permission not found" || err.Error() == "permission not assigned to role" {
			return util.CreateErrorResponse(http.StatusNotFound, "Role, permission not found or permission not assigned")
		}
		logger.WithError(err).Error("Failed to unassign permission from role")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to unassign permission")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       `{"message":"Permission unassigned successfully"}`,
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