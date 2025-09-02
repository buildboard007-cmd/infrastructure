package main

import (
	"fmt"
	"infrastructure/lib/clients"
	"infrastructure/lib/constants"
	"infrastructure/lib/data"
	"infrastructure/lib/util"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

var (
	logger        *logrus.Logger
	isLocal       bool
	ssmRepository data.SSMRepository
	ssmParams     map[string]string
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	requestOrigin, ok := request.Headers["origin"]
	if !ok {
		fmt.Println("origin is not present in the request headers")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, nil
	}

	fmt.Println("origin from request header: ", request.Headers["origin"])
	fmt.Println("Allowed Origins: ", ssmParams[constants.ALLOWED_ORIGINS])

	allowedOrigins := strings.Split(ssmParams[constants.ALLOWED_ORIGINS], ",")

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == requestOrigin {
			return events.APIGatewayProxyResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"Access-Control-Allow-Origin":      requestOrigin,
					"Access-Control-Allow-Headers":     "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,geolocation,x-retry",
					"Access-Control-Allow-Methods":     "GET, PUT, DELETE, POST, OPTIONS, PATCH",
					"Access-Control-Allow-Credentials": "true",
				},
			}, nil
		}
	}

	fmt.Println("unauthorized origin from request header: " + requestOrigin)

	return events.APIGatewayProxyResponse{
		StatusCode: 400,
	}, nil
}

func main() {
	lambda.Start(handler)
}

func init() {
	isLocal, _ = strconv.ParseBool(os.Getenv("IS_LOCAL"))

	logger = logrus.New()
	util.SetLogLevel(logger, os.Getenv("LOG_LEVEL"))
	logger.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint: isLocal,
	})

	// Setup SSM client
	ssmClient := clients.NewSSMClient(isLocal)
	ssmRepository = &data.SSMDao{
		SSM:    ssmClient,
		Logger: logger,
	}

	// Get SSM parameters
	var err error
	ssmParams, err = ssmRepository.GetParameters()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Error while getting ssm params from param store")
	}
}
