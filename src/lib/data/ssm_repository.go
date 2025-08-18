package data

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/sirupsen/logrus"
)

type SSMRepository interface {
	GetParameters() (map[string]string, error)
}

type SSMClientInterface interface {
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

type SSMDao struct {
	SSM    SSMClientInterface
	Logger *logrus.Logger
}

func (client *SSMDao) GetParameters() (map[string]string, error) {
	params := map[string]string{}
	ssmClient := client.SSM
	input := &ssm.GetParametersByPathInput{
		Path:           aws.String("/infrastructure"),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}

	for {
		output, err := ssmClient.GetParametersByPath(context.TODO(), input)
		if err != nil {
			return nil, err
		}

		// Store parameters in map
		for _, param := range output.Parameters {
			params[*param.Name] = *param.Value
		}

		// If there's no NextToken, we've got all parameters
		if output.NextToken == nil {
			break
		}

		// Set the NextToken for the next request
		input.NextToken = output.NextToken
	}
	return params, nil
}
