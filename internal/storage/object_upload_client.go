package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/tecisage/resell-apis/internal/dtos"
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

var ErrNotFound = errors.New("storage: object not found")
var ErrZeroSizeObject = errors.New("storage: object has zero content length")
var ErrNoContentType = errors.New("storage: object has no content type")
var ErrFileSizeExceeded = errors.New("storage: object size exceeds limit")
var ErrInvalidContentType = errors.New("storage: invalid content type")

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

func (c *Client) HeadObject(ctx context.Context, key string) (contentLength int64, contentType string, err error) {
	out, err := c.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		if isNotFound(err) {
			return 0, "", ErrNotFound
		}
		return 0, "", fmt.Errorf("storage: failed to head object: %q: %w", key, err)
	}

	if out.ContentLength != nil {
		contentLength = *out.ContentLength
	}

	if out.ContentType != nil {
		contentType = *out.ContentType
	}

	if contentLength == 0 {
		return 0, "", ErrZeroSizeObject
	}
	if contentLength > dtos.MaxFileSize {
		return 0, "", ErrFileSizeExceeded
	}

	if contentType == "" {
		return 0, "", ErrNoContentType
	}

	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if !dtos.IsAllowedContentType(mediaType) {
		return 0, "", ErrInvalidContentType
	}

	return contentLength, contentType, nil
}

func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := c.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("storage: failed to get object: %q: %w", key, err)
	}

	return out.Body, nil
}

func isNotFound(err error) bool {
	var se smithy.APIError
	if errors.As(err, &se) {
		switch se.ErrorCode() {
		case "NotFound", "NoSuchKey", "404":
			return true
		}
	}
	return false
}

func (c *Client) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	copySource := fmt.Sprintf("%s/%s", c.Bucket, srcKey)
	_, err := c.Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.Bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstKey),
	})
	if err != nil {
		return fmt.Errorf("storage: failed to copy object from %q to %q: %w", srcKey, dstKey, err)
	}
	return nil
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: failed to delete object %q: %w", key, err)
	}
	return nil
}
