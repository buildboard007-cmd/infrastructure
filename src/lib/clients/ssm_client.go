package clients

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func NewSSMClient(isLocal bool) *ssm.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-2"),
	)
	if err != nil {
		panic(err)
	}

	if isLocal {
		cfg.BaseEndpoint = aws.String("http://docker.for.mac.host.internal:4566")
	}

	return ssm.NewFromConfig(cfg)
}
