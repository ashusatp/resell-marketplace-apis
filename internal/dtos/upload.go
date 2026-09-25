package dtos

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxUploadFiles = 10
	MaxFileSize    = 5 * 1024 * 1024 // 5 MB
	UploadPrefix   = "uploads"
	PresignTTL     = 5 * time.Minute
)

var allowedContentTypes = []string{
	"image/jpeg",
	"image/png",
	"image/webp",
}

type PresignReq struct {
	Files []PresignFile `json:"files"`
}

type PresignFile struct {
	ContentType string `json:"content_type"`
	SizeByte    int64  `json:"size_byte"`
}

func (r *PresignReq) ValidateAndGetUploadKeys(userUUID string) ([]string, error) {
	if len(r.Files) == 0 {
		return nil, &ValidationError{Field: "files", Message: "files is required"}
	}

	if len(r.Files) > MaxUploadFiles {
		return nil, &ValidationError{
			Field:   "files",
			Message: fmt.Sprintf("cannot upload more than %d files at a time", MaxUploadFiles),
		}
	}

	keys := make([]string, len(r.Files))

	for i, file := range r.Files {
		fileType := strings.ToLower(strings.TrimSpace(file.ContentType))
		if fileType == "" {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("files[%d].content_type", i),
				Message: "content_type is required",
			}
		}

		if !isAllowedContentType(fileType) {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("files[%d].content_type", i),
				Message: "invalid content type, allowed types: image/jpeg, image/png, image/webp",
			}
		}

		if file.SizeByte <= 0 {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("files[%d].size_byte", i),
				Message: "size_byte must be greater than 0",
			}
		}

		if file.SizeByte > MaxFileSize {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("files[%d].size_byte", i),
				Message: fmt.Sprintf("file size exceeds maximum allowed size (%d MB)", MaxFileSize/(1024*1024)),
			}
		}

		keys[i] = mintUploadKey(userUUID, file.ContentType)

	}

	return keys, nil
}

func isAllowedContentType(contentType string) bool {
	return slices.Contains(allowedContentTypes, contentType)
}

type PresignedFile struct {
	UploadURL string `json:"upload_url"`
	Key       string `json:"key"`
}

type PresignedURLWithKey struct {
	UploadURL string    `json:"upload_url"`
	ObjectKey string    `json:"object_key"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PresignResponse struct {
	PresignedURLs []PresignedURLWithKey `json:"presigned_urls"`
}

// uploads/{user_id}/{uuid}.jpg
func mintUploadKey(userID string, contentType string) string {
	return fmt.Sprintf("%s/%s/%s.%s", UploadPrefix, userID, uuid.NewString(), getExtFromContentType(contentType))
}

func getExtFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	default:
		return "jpg"
	}
}

func GetContentType(key string) string {
	ext := filepath.Ext(key)
	switch ext {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
