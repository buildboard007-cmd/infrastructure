package api

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

// SuccessResponse creates a successful API Gateway response
func SuccessResponse(statusCode int, data interface{}, logger *logrus.Logger) events.APIGatewayProxyResponse {
	body, err := json.Marshal(data)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal response data")
		return ErrorResponse(http.StatusInternalServerError, "Internal server error", logger)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
			"Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS",
		},
	}
}

// ErrorResponse creates an error API Gateway response
func ErrorResponse(statusCode int, message string, logger *logrus.Logger) events.APIGatewayProxyResponse {
	errorData := map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	}

	body, err := json.Marshal(errorData)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal error response")
		body = []byte(`{"error":true,"message":"Internal server error","status":500}`)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
			"Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS",
		},
	}
}

// ValidationErrorResponse creates a validation error response
func ValidationErrorResponse(message string, errors []string, logger *logrus.Logger) events.APIGatewayProxyResponse {
	errorData := map[string]interface{}{
		"error":      true,
		"message":    message,
		"status":     http.StatusBadRequest,
		"validation": errors,
	}

	body, err := json.Marshal(errorData)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal validation error response")
		return ErrorResponse(http.StatusInternalServerError, "Internal server error", logger)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusBadRequest,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
			"Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS",
		},
	}
}