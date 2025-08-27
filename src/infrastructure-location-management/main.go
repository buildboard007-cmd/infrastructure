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
	logger             *logrus.Logger
	isLocal            bool
	ssmRepository      data.SSMRepository
	ssmParams          map[string]string
	sqlDB              *sql.DB
	locationRepository data.LocationRepository
)

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.WithFields(logrus.Fields{
		"operation": "Handler",
		"method":    request.HTTPMethod,
		"path":      request.Path,
	}).Info("Location management request received")

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
		return util.CreateErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage locations"), nil
	}

	// Route based on HTTP method and path
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		// POST /locations - Create new location
		return handleCreateLocation(ctx, userID, orgID, request.Body), nil
		
	case http.MethodGet:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// GET /locations/{id} - Get specific location
			locationID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid location ID"), nil
			}
			return handleGetLocation(ctx, locationID, orgID), nil
		} else {
			// GET /locations - Get all locations for org
			return handleGetLocations(ctx, orgID), nil
		}
		
	case http.MethodPut:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// PUT /locations/{id} - Update location
			locationID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid location ID"), nil
			}
			return handleUpdateLocation(ctx, locationID, orgID, request.Body), nil
		} else {
			return util.CreateErrorResponse(http.StatusBadRequest, "Location ID required for update"), nil
		}
		
	case http.MethodDelete:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// DELETE /locations/{id} - Delete location
			locationID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
				return util.CreateErrorResponse(http.StatusBadRequest, "Invalid location ID"), nil
			}
			return handleDeleteLocation(ctx, locationID, orgID), nil
		} else {
			return util.CreateErrorResponse(http.StatusBadRequest, "Location ID required for deletion"), nil
		}
		
	default:
		return util.CreateErrorResponse(http.StatusMethodNotAllowed, "Method not allowed"), nil
	}
}

// handleCreateLocation handles POST /locations
func handleCreateLocation(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreateLocationRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create location request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if createReq.LocationName == "" || len(createReq.LocationName) < 2 || len(createReq.LocationName) > 100 {
		return util.CreateErrorResponse(http.StatusBadRequest, "Location name must be between 2 and 100 characters")
	}

	// Create location object
	location := &models.Location{
		LocationName: createReq.LocationName,
		Address:      createReq.Address,
	}

	// Create location (automatically assigns to creator with SuperAdmin role)
	createdLocation, err := locationRepository.CreateLocation(ctx, userID, orgID, location)
	if err != nil {
		logger.WithError(err).Error("Failed to create location")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to create location")
	}

	// Return success response
	responseBody, _ := json.Marshal(createdLocation)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleGetLocations handles GET /locations
func handleGetLocations(ctx context.Context, orgID int64) events.APIGatewayProxyResponse {
	locations, err := locationRepository.GetLocationsByOrg(ctx, orgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get locations")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to get locations")
	}

	response := models.LocationListResponse{
		Locations: locations,
		Total:     len(locations),
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

// handleGetLocation handles GET /locations/{id}
func handleGetLocation(ctx context.Context, locationID, orgID int64) events.APIGatewayProxyResponse {
	location, err := locationRepository.GetLocationByID(ctx, locationID, orgID)
	if err != nil {
		if err.Error() == "location not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Location not found")
		}
		logger.WithError(err).Error("Failed to get location")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to get location")
	}

	responseBody, _ := json.Marshal(location)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleUpdateLocation handles PUT /locations/{id}
func handleUpdateLocation(ctx context.Context, locationID, orgID int64, body string) events.APIGatewayProxyResponse {
	var updateReq models.UpdateLocationRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update location request")
		return util.CreateErrorResponse(http.StatusBadRequest, "Invalid request body")
	}

	// Create location object with updates
	location := &models.Location{
		LocationName: updateReq.LocationName,
		Address:      updateReq.Address,
	}

	updatedLocation, err := locationRepository.UpdateLocation(ctx, locationID, orgID, location)
	if err != nil {
		if err.Error() == "location not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Location not found")
		}
		logger.WithError(err).Error("Failed to update location")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to update location")
	}

	responseBody, _ := json.Marshal(updatedLocation)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// handleDeleteLocation handles DELETE /locations/{id}
func handleDeleteLocation(ctx context.Context, locationID, orgID int64) events.APIGatewayProxyResponse {
	err := locationRepository.DeleteLocation(ctx, locationID, orgID)
	if err != nil {
		if err.Error() == "location not found" {
			return util.CreateErrorResponse(http.StatusNotFound, "Location not found")
		}
		logger.WithError(err).Error("Failed to delete location")
		return util.CreateErrorResponse(http.StatusInternalServerError, "Failed to delete location")
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

	logger.WithField("operation", "init").Info("Location Management Lambda initialization completed successfully")
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

	// Initialize location repository
	locationRepository = &data.LocationDao{
		DB:     sqlDB,
		Logger: logger,
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.WithField("operation", "setupPostgresSQLClient").Debug("PostgreSQL client initialized successfully")
	}
	return nil
}