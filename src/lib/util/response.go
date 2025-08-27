package util

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
)

// ErrorResponse represents the error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateErrorResponse creates a standardized error response for API Gateway
func CreateErrorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	errorResp := ErrorResponse{
		Error: message,
	}
	
	body, _ := json.Marshal(errorResp)
	
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// CreateSuccessResponse creates a standardized success response for API Gateway
func CreateSuccessResponse(statusCode int, body interface{}) events.APIGatewayProxyResponse {
	responseBody, _ := json.Marshal(body)
	
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(responseBody),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}