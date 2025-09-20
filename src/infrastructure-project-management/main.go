package main

import (
	"context"
	"database/sql"
	"fmt"
	"infrastructure/lib/api"
	"infrastructure/lib/auth"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/models"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

var (
	logger            *logrus.Logger
	isLocal           bool
	ssmRepository     data.SSMRepository
	ssmParams         map[string]string
	sqlDB             *sql.DB
	projectRepository data.ProjectRepository
)

// Handler processes API Gateway requests for project management operations
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"method":      request.HTTPMethod,
		"path":        request.Path,
		"resource":    request.Resource,
		"path_params": request.PathParameters,
		"operation":   "Handler",
	}).Debug("Processing project management request")

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"operation": "Handler",
		}).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	logger.WithFields(logrus.Fields{
		"user_id":    claims.UserID,
		"org_id":     claims.OrgID,
		"email":      claims.Email,
		"operation":  "Handler",
	}).Debug("User authenticated successfully")

	// Route the request based on path and method
	switch {
	// Project CRUD operations
	case request.Resource == "/projects" && request.HTTPMethod == "POST":
		return handleCreateProject(ctx, request, claims)
	case request.Resource == "/projects" && request.HTTPMethod == "GET":
		return handleGetProjects(ctx, request, claims)
	case request.Resource == "/projects/{projectId}" && request.HTTPMethod == "GET":
		return handleGetProject(ctx, request, claims)
	case request.Resource == "/projects/{projectId}" && request.HTTPMethod == "PUT":
		return handleUpdateProject(ctx, request, claims)


	// Project Attachment operations
	case request.Resource == "/projects/{projectId}/attachments" && request.HTTPMethod == "POST":
		return handleCreateProjectAttachment(ctx, request, claims)
	case request.Resource == "/projects/{projectId}/attachments" && request.HTTPMethod == "GET":
		return handleGetProjectAttachments(ctx, request, claims)
	case request.Resource == "/projects/{projectId}/attachments/{attachmentId}" && request.HTTPMethod == "GET":
		return handleGetProjectAttachment(ctx, request, claims)
	case request.Resource == "/projects/{projectId}/attachments/{attachmentId}" && request.HTTPMethod == "DELETE":
		return handleDeleteProjectAttachment(ctx, request, claims)

	// Project User Role operations
	case request.Resource == "/projects/{projectId}/users" && request.HTTPMethod == "POST":
		return handleAssignUserToProject(ctx, request, claims)
	case request.Resource == "/projects/{projectId}/users" && request.HTTPMethod == "GET":
		return handleGetProjectUserRoles(ctx, request, claims)
	case request.Resource == "/projects/{projectId}/users/{assignmentId}" && request.HTTPMethod == "PUT":
		return handleUpdateProjectUserRole(ctx, request, claims)
	case request.Resource == "/projects/{projectId}/users/{assignmentId}" && request.HTTPMethod == "DELETE":
		return handleRemoveUserFromProject(ctx, request, claims)

	default:
		logger.WithFields(logrus.Fields{
			"method":    request.HTTPMethod,
			"resource":  request.Resource,
			"operation": "Handler",
		}).Warn("Endpoint not found")
		return api.ErrorResponse(http.StatusNotFound, "Endpoint not found", logger), nil
	}
}

// handleCreateProject handles POST /projects
func handleCreateProject(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	var createRequest models.CreateProjectRequest
	if err := api.ParseJSONBody(request.Body, &createRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for create project")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID

	orgID := claims.OrgID

	response, err := projectRepository.CreateProject(ctx, orgID, &createRequest, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to create project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create project", logger), nil
	}

	// Check if project creation was successful
	if !response.Success {
		logger.WithField("message", response.Message).Error("Project creation failed")
		if response.Errors != nil && len(response.Errors) > 0 {
			// Flatten the validation errors for the response
			var validationErrors []string
			for field, fieldErrors := range response.Errors {
				for _, fieldError := range fieldErrors {
					validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", field, fieldError))
				}
			}
			return api.ValidationErrorResponse(response.Message, validationErrors, logger), nil
		}
		return api.ErrorResponse(http.StatusInternalServerError, response.Message, logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, response, logger), nil
}

// handleGetProjects handles GET /projects with optional location_id query parameter
func handleGetProjects(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	orgID := claims.OrgID

	// Check if location_id query parameter is provided
	locationIDStr, hasLocationID := request.QueryStringParameters["location_id"]
	
	var projects []models.Project
	var err error
	
	if hasLocationID && locationIDStr != "" {
		// Get projects by location ID
		locationID, parseErr := strconv.ParseInt(locationIDStr, 10, 64)
		if parseErr != nil {
			logger.WithError(parseErr).Error("Invalid location_id parameter")
			return api.ErrorResponse(http.StatusBadRequest, "Invalid location_id parameter", logger), nil
		}
		
		projects, err = projectRepository.GetProjectsByLocationID(ctx, locationID, orgID)
		if err != nil {
			logger.WithError(err).Error("Failed to get projects by location")
			return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects by location", logger), nil
		}
	} else {
		// Get all projects for organization
		projects, err = projectRepository.GetProjectsByOrg(ctx, orgID)
		if err != nil {
			logger.WithError(err).Error("Failed to get projects")
			return api.ErrorResponse(http.StatusInternalServerError, "Failed to get projects", logger), nil
		}
	}

	response := models.ProjectListResponse{
		Projects: projects,
		Total:    len(projects),
	}

	return api.SuccessResponse(http.StatusOK, response, logger), nil
}

// handleGetProject handles GET /projects/{projectId}
func handleGetProject(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	orgID := claims.OrgID

	project, err := projectRepository.GetProjectByID(ctx, projectID, orgID)
	if err != nil {
		if err.Error() == "project not found" {
			return api.ErrorResponse(http.StatusNotFound, "Project not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get project", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, project, logger), nil
}

// handleUpdateProject handles PUT /projects/{projectId}
func handleUpdateProject(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	var updateRequest models.UpdateProjectRequest
	if err := api.ParseJSONBody(request.Body, &updateRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for update project")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID

	orgID := claims.OrgID

	project, err := projectRepository.UpdateProject(ctx, projectID, orgID, &updateRequest, userID)
	if err != nil {
		if err.Error() == "project not found" {
			return api.ErrorResponse(http.StatusNotFound, "Project not found", logger), nil
		}
		logger.WithError(err).Error("Failed to update project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update project", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, project, logger), nil
}



// handleCreateProjectAttachment handles POST /projects/{projectId}/attachments
func handleCreateProjectAttachment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	var createRequest models.CreateProjectAttachmentRequest
	if err := api.ParseJSONBody(request.Body, &createRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for create project attachment")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID

	attachment, err := projectRepository.CreateProjectAttachment(ctx, projectID, &createRequest, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to create project attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create project attachment", logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, attachment, logger), nil
}

// handleGetProjectAttachments handles GET /projects/{projectId}/attachments
func handleGetProjectAttachments(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	attachments, err := projectRepository.GetProjectAttachmentsByProject(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("Failed to get project attachments")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get project attachments", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, attachments, logger), nil
}

// handleGetProjectAttachment handles GET /projects/{projectId}/attachments/{attachmentId}
func handleGetProjectAttachment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	attachmentID, err := strconv.ParseInt(request.PathParameters["attachmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid attachment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid attachment ID", logger), nil
	}

	attachment, err := projectRepository.GetProjectAttachmentByID(ctx, attachmentID, projectID)
	if err != nil {
		if err.Error() == "project attachment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Project attachment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to get project attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get project attachment", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, attachment, logger), nil
}

// handleDeleteProjectAttachment handles DELETE /projects/{projectId}/attachments/{attachmentId}
func handleDeleteProjectAttachment(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	attachmentID, err := strconv.ParseInt(request.PathParameters["attachmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid attachment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid attachment ID", logger), nil
	}

	userID := claims.UserID

	err = projectRepository.DeleteProjectAttachment(ctx, attachmentID, projectID, userID)
	if err != nil {
		if err.Error() == "project attachment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Project attachment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to delete project attachment")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete project attachment", logger), nil
	}

	return api.SuccessResponse(http.StatusNoContent, nil, logger), nil
}

// handleAssignUserToProject handles POST /projects/{projectId}/users
func handleAssignUserToProject(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	var createRequest models.CreateProjectUserRoleRequest
	if err := api.ParseJSONBody(request.Body, &createRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for assign user to project")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID

	assignment, err := projectRepository.AssignUserToProject(ctx, projectID, &createRequest, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to assign user to project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to assign user to project", logger), nil
	}

	return api.SuccessResponse(http.StatusCreated, assignment, logger), nil
}

// handleGetProjectUserRoles handles GET /projects/{projectId}/users
func handleGetProjectUserRoles(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	assignments, err := projectRepository.GetProjectUserRoles(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("Failed to get project user roles")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get project user roles", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, assignments, logger), nil
}

// handleUpdateProjectUserRole handles PUT /projects/{projectId}/users/{assignmentId}
func handleUpdateProjectUserRole(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	assignmentID, err := strconv.ParseInt(request.PathParameters["assignmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid assignment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid assignment ID", logger), nil
	}

	var updateRequest models.UpdateProjectUserRoleRequest
	if err := api.ParseJSONBody(request.Body, &updateRequest); err != nil {
		logger.WithError(err).Error("Invalid request body for update project user role")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger), nil
	}

	userID := claims.UserID

	assignment, err := projectRepository.UpdateProjectUserRole(ctx, assignmentID, projectID, &updateRequest, userID)
	if err != nil {
		if err.Error() == "project user role assignment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Project user role assignment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to update project user role")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update project user role", logger), nil
	}

	return api.SuccessResponse(http.StatusOK, assignment, logger), nil
}

// handleRemoveUserFromProject handles DELETE /projects/{projectId}/users/{assignmentId}
func handleRemoveUserFromProject(ctx context.Context, request events.APIGatewayProxyRequest, claims *auth.Claims) (events.APIGatewayProxyResponse, error) {
	projectID, err := strconv.ParseInt(request.PathParameters["projectId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid project ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid project ID", logger), nil
	}

	assignmentID, err := strconv.ParseInt(request.PathParameters["assignmentId"], 10, 64)
	if err != nil {
		logger.WithError(err).Error("Invalid assignment ID")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid assignment ID", logger), nil
	}

	userID := claims.UserID

	err = projectRepository.RemoveUserFromProject(ctx, assignmentID, projectID, userID)
	if err != nil {
		if err.Error() == "project user role assignment not found" {
			return api.ErrorResponse(http.StatusNotFound, "Project user role assignment not found", logger), nil
		}
		logger.WithError(err).Error("Failed to remove user from project")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to remove user from project", logger), nil
	}

	return api.SuccessResponse(http.StatusNoContent, nil, logger), nil
}

// setupPostgresSQLClient initializes the PostgreSQL database connection
func setupPostgresSQLClient(ssmParams map[string]string) error {
	var err error

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

	// Setup logging
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.ErrorLevel)
	}

	logger.WithField("operation", "init").Error("Initializing Project Management Lambda")

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

	// Setup PostgreSQL client
	err = setupPostgresSQLClient(ssmParams)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"operation": "init",
			"error":     err.Error(),
		}).Fatal("Error setting up PostgreSQL client")
	}

	// Initialize project repository
	projectRepository = data.NewProjectRepository(sqlDB)

	logger.WithField("operation", "init").Error("Project Management Lambda initialization completed successfully")
}