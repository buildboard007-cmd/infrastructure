package clients

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ClientInterface defines the interface for S3 operations
type S3ClientInterface interface {
	GenerateUploadURL(key string, expiry time.Duration) (string, error)
	GenerateDownloadURL(key string, expiry time.Duration) (string, error)
	DeleteObject(key string) error
	ObjectExists(key string) (bool, error)
}

// S3Client wraps the AWS S3 client with our custom methods
type S3Client struct {
	svc           *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

// NewS3Client creates a new S3 client instance
func NewS3Client(isLocal bool, bucket string) S3ClientInterface {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-2"),
	)
	if err != nil {
		panic("failed to load AWS configuration: " + err.Error())
	}

	// Create S3 client with custom options
	var svc *s3.Client
	if isLocal {
		// LocalStack configuration
		svc = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String("http://docker.for.mac.host.internal:4566")
			o.UsePathStyle = true
		})
	} else {
		svc = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	// Create presign client for generating presigned URLs
	presignClient := s3.NewPresignClient(svc)

	return &S3Client{
		svc:           svc,
		presignClient: presignClient,
		bucket:        bucket,
	}
}

// GenerateUploadURL creates a presigned URL for uploading a file to S3
func (client *S3Client) GenerateUploadURL(key string, expiry time.Duration) (string, error) {
	ctx := context.Background()

	presignResult, err := client.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(client.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", err
	}

	return presignResult.URL, nil
}

// GenerateDownloadURL creates a presigned URL for downloading a file from S3
func (client *S3Client) GenerateDownloadURL(key string, expiry time.Duration) (string, error) {
	ctx := context.Background()

	presignResult, err := client.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(client.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", err
	}

	return presignResult.URL, nil
}

// DeleteObject deletes an object from S3
func (client *S3Client) DeleteObject(key string) error {
	ctx := context.Background()

	_, err := client.svc.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(client.bucket),
		Key:    aws.String(key),
	})

	return err
}

// ObjectExists checks if an object exists in S3
func (client *S3Client) ObjectExists(key string) (bool, error) {
	ctx := context.Background()

	_, err := client.svc.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(client.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return false, nil // Object doesn't exist or other error
	}

	return true, nil
}