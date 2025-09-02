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

	// Extract claims from JWT token via API Gateway authorizer
	claims, err := auth.ExtractClaimsFromRequest(request)
	if err != nil {
		logger.WithError(err).Error("Authentication failed")
		return api.ErrorResponse(http.StatusUnauthorized, "Authentication failed", logger), nil
	}

	if !claims.IsSuperAdmin {
		logger.WithField("user_id", claims.UserID).Warn("User is not a super admin")
		return api.ErrorResponse(http.StatusForbidden, "Forbidden: Only super admins can manage locations", logger), nil
	}

	// Route based on HTTP method and path
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Handle different routes
	switch request.HTTPMethod {
	case http.MethodPost:
		// POST /locations - Create new location
		return handleCreateLocation(ctx, claims.UserID, claims.OrgID, request.Body), nil
		
	case http.MethodGet:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// GET /locations/{id} - Get specific location
			locationID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
					return api.ErrorResponse(http.StatusBadRequest, "Invalid location ID", logger), nil
			}
			return handleGetLocation(ctx, locationID, claims.OrgID), nil
		} else {
			// GET /locations - Get all locations for org
			return handleGetLocations(ctx, claims.OrgID), nil
		}
		
	case http.MethodPut:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// PUT /locations/{id} - Update location
			locationID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
					return api.ErrorResponse(http.StatusBadRequest, "Invalid location ID", logger), nil
			}
			return handleUpdateLocation(ctx, locationID, claims.OrgID, claims.UserID, request.Body), nil
		} else {
			return api.ErrorResponse(http.StatusBadRequest, "Location ID required for update", logger), nil
		}
		
	case http.MethodDelete:
		if len(pathSegments) >= 2 && pathSegments[1] != "" {
			// DELETE /locations/{id} - Delete location
			locationID, err := strconv.ParseInt(pathSegments[1], 10, 64)
			if err != nil {
					return api.ErrorResponse(http.StatusBadRequest, "Invalid location ID", logger), nil
			}
			return handleDeleteLocation(ctx, locationID, claims.OrgID, claims.UserID), nil
		} else {
			return api.ErrorResponse(http.StatusBadRequest, "Location ID required for deletion", logger), nil
		}
		
	default:
		return api.ErrorResponse(http.StatusMethodNotAllowed, "Method not allowed", logger), nil
	}
}

// handleCreateLocation handles POST /locations
func handleCreateLocation(ctx context.Context, userID, orgID int64, body string) events.APIGatewayProxyResponse {
	var createReq models.CreateLocationRequest
	if err := json.Unmarshal([]byte(body), &createReq); err != nil {
		logger.WithError(err).Error("Failed to parse create location request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	// Create location object
	location := &models.Location{
		Name:         createReq.Name,
		LocationType: createReq.LocationType,
		Address:      createReq.Address,
		City:         createReq.City,
		State:        createReq.State,
		ZipCode:      createReq.ZipCode,
		Country:      createReq.Country,
		Status:       createReq.Status,
	}

	// Create location (automatically assigns to creator with SuperAdmin role)
	createdLocation, err := locationRepository.CreateLocation(ctx, userID, orgID, location)
	if err != nil {
		logger.WithError(err).Error("Failed to create location")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to create location", logger)
	}

	return api.SuccessResponse(http.StatusCreated, createdLocation, logger)
}

// handleGetLocations handles GET /locations
func handleGetLocations(ctx context.Context, orgID int64) events.APIGatewayProxyResponse {
	locations, err := locationRepository.GetLocationsByOrg(ctx, orgID)
	if err != nil {
		logger.WithError(err).Error("Failed to get locations")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get locations", logger)
	}

	response := models.LocationListResponse{
		Locations: locations,
		Total:     len(locations),
	}

	return api.SuccessResponse(http.StatusOK, response, logger)
}

// handleGetLocation handles GET /locations/{id}
func handleGetLocation(ctx context.Context, locationID, orgID int64) events.APIGatewayProxyResponse {
	location, err := locationRepository.GetLocationByID(ctx, locationID, orgID)
	if err != nil {
		if err.Error() == "location not found" {
			return api.ErrorResponse(http.StatusNotFound, "Location not found", logger)
		}
		logger.WithError(err).Error("Failed to get location")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to get location", logger)
	}

	return api.SuccessResponse(http.StatusOK, location, logger)
}

// handleUpdateLocation handles PUT /locations/{id}
func handleUpdateLocation(ctx context.Context, locationID, orgID, userID int64, body string) events.APIGatewayProxyResponse {
	var updateReq models.UpdateLocationRequest
	if err := json.Unmarshal([]byte(body), &updateReq); err != nil {
		logger.WithError(err).Error("Failed to parse update location request")
		return api.ErrorResponse(http.StatusBadRequest, "Invalid request body", logger)
	}

	updatedLocation, err := locationRepository.UpdateLocation(ctx, locationID, orgID, &updateReq, userID)
	if err != nil {
		if err.Error() == "location not found" {
			return api.ErrorResponse(http.StatusNotFound, "Location not found", logger)
		}
		logger.WithError(err).Error("Failed to update location")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to update location", logger)
	}

	return api.SuccessResponse(http.StatusOK, updatedLocation, logger)
}

// handleDeleteLocation handles DELETE /locations/{id}
func handleDeleteLocation(ctx context.Context, locationID, orgID, userID int64) events.APIGatewayProxyResponse {
	err := locationRepository.DeleteLocation(ctx, locationID, orgID, userID)
	if err != nil {
		if err.Error() == "location not found" {
			return api.ErrorResponse(http.StatusNotFound, "Location not found", logger)
		}
		logger.WithError(err).Error("Failed to delete location")
		return api.ErrorResponse(http.StatusInternalServerError, "Failed to delete location", logger)
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