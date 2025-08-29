package auth

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
)

// Claims represents the JWT claims extracted from the API Gateway authorizer context
type Claims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	CognitoID    string `json:"sub"`
	OrgID        int64  `json:"org_id"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
}

// ExtractClaimsFromRequest extracts and parses JWT claims from API Gateway request
func ExtractClaimsFromRequest(request events.APIGatewayProxyRequest) (*Claims, error) {
	// Get claims from authorizer context
	var claimsMap map[string]interface{}
	var ok bool

	// Try different possible claim locations in the authorizer context
	if authClaims, exists := request.RequestContext.Authorizer["claims"]; exists {
		claimsMap, ok = authClaims.(map[string]interface{})
	}

	// If claims not found, try direct access to authorizer (some API Gateway configurations)
	if !ok {
		claimsMap = request.RequestContext.Authorizer
		ok = (claimsMap != nil)
	}

	if !ok || claimsMap == nil {
		return nil, fmt.Errorf("claims not found in authorizer context")
	}

	// Extract and parse user_id
	var userID int64
	if userIDValue, exists := claimsMap["user_id"]; exists {
		// Try as string first
		if userIDStr, ok := userIDValue.(string); ok {
			var err error
			userID, err = strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse user_id string: %w", err)
			}
		} else if userIDFloat, ok := userIDValue.(float64); ok {
			// Try as float64 (JSON numbers are parsed as float64)
			userID = int64(userIDFloat)
		} else {
			return nil, fmt.Errorf("user_id has unexpected type")
		}
	} else {
		return nil, fmt.Errorf("user_id not found in claims")
	}

	// Extract and parse org_id
	var orgID int64
	if orgIDValue, exists := claimsMap["org_id"]; exists {
		// Try as string first
		if orgIDStr, ok := orgIDValue.(string); ok {
			var err error
			orgID, err = strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse org_id string: %w", err)
			}
		} else if orgIDFloat, ok := orgIDValue.(float64); ok {
			// Try as float64 (JSON numbers are parsed as float64)
			orgID = int64(orgIDFloat)
		} else {
			return nil, fmt.Errorf("org_id has unexpected type")
		}
	} else {
		return nil, fmt.Errorf("org_id not found in claims")
	}

	// Extract email
	email, ok := claimsMap["email"].(string)
	if !ok {
		return nil, fmt.Errorf("email not found or invalid in claims")
	}

	// Extract Cognito ID (sub)
	cognitoID, ok := claimsMap["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("sub not found or invalid in claims")
	}

	// Extract isSuperAdmin
	var isSuperAdmin bool
	if superAdminValue, exists := claimsMap["isSuperAdmin"]; exists {
		if isSuperAdmin, ok = superAdminValue.(bool); !ok {
			// Try as string "true"/"false"
			if superAdminStr, ok := superAdminValue.(string); ok && superAdminStr == "true" {
				isSuperAdmin = true
			}
		}
	}

	return &Claims{
		UserID:       userID,
		Email:        email,
		CognitoID:    cognitoID,
		OrgID:        orgID,
		IsSuperAdmin: isSuperAdmin,
	}, nil
}

// ToJSON converts claims to JSON string for logging
func (c *Claims) ToJSON() string {
	data, _ := json.Marshal(c)
	return string(data)
}