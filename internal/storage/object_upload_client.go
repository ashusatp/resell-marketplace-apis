package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	Client    *s3.Client
	Presigner *s3.PresignClient
	Bucket    string
}

type R2Config struct {
	AccountID    string
	AccessKey    string
	AccessSecret string
	Bucket       string
	Region       string
}

func NewR2Client(ctx context.Context, cfg R2Config) (*Client, error) {

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	r2Cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.AccessSecret, ""),
		),
		config.WithRegion("auto"),
	)

	if err != nil {
		return nil, fmt.Errorf("storage: failed to load storage config: %v", err)
	}

	client := s3.NewFromConfig(r2Cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	presigner := s3.NewPresignClient(client)

	return &Client{
		Client:    client,
		Presigner: presigner,
		Bucket:    cfg.Bucket,
	}, nil
}
