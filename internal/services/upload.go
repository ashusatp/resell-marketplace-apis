package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tecisage/resell-apis/internal/dtos"
	"github.com/tecisage/resell-apis/internal/storage"
)

type UploadService struct {
	Logger   *slog.Logger
	R2Client *storage.Client
}

func NewUploadService(logger *slog.Logger, r2Client *storage.Client) *UploadService {
	return &UploadService{
		Logger:   logger,
		R2Client: r2Client,
	}
}

func (us *UploadService) GetPresignedURLs(keys []string, expiry time.Duration) (dtos.PresignResponse, error) {
	presignedURLs := make([]dtos.PresignedURLWithKey, len(keys))
	for i, key := range keys {
		presignResult, err := us.R2Client.Presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(us.R2Client.Bucket),
			Key:         aws.String(key),
			ContentType: aws.String(dtos.GetContentType(key)),
		}, s3.WithPresignExpires(expiry))
		if err != nil {
			return dtos.PresignResponse{}, err
		}
		presignedURLs[i] = dtos.PresignedURLWithKey{
			UploadURL: presignResult.URL,
			ObjectKey: key,
			ExpiresAt: time.Now().Add(expiry).UTC(),
		}
	}
	return dtos.PresignResponse{PresignedURLs: presignedURLs}, nil
}
