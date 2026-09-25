package dtos

import (
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type CreateListingRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Price       int64    `json:"price" binding:"required"`
	City        string   `json:"city" binding:"required"`
	ImageKeys   []string `json:"image_keys"` //uploads/<user-id>/<image-uuid>.png
}

func (req *CreateListingRequest) Validate(userId string) ([]string, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, &ValidationError{Field: "title", Message: "title is required"}
	}
	if strings.TrimSpace(req.Description) == "" {
		return nil, &ValidationError{Field: "description", Message: "description is required"}
	}
	if req.Price <= 0 {
		return nil, &ValidationError{Field: "price", Message: "price must be greater than 0"}
	}
	if strings.TrimSpace(req.City) == "" {
		return nil, &ValidationError{Field: "city", Message: "city is required"}
	}

	parsedUploadKeys := make([]string, 0, len(req.ImageKeys))
	if req.ImageKeys != nil {
		for _, key := range req.ImageKeys {
			parts := strings.Split(key, "/")
			if len(parts) != 3 {
				return nil, &ValidationError{Field: "image_keys", Message: "invalid image key format"}
			}

			prefix := parts[0]
			userIDInKey := parts[1]
			fileName := parts[2]

			// Check prefix
			if prefix != "uploads" {
				return nil, &ValidationError{Field: "image_keys", Message: "image key must start with 'uploads/'"}
			}

			// Check if key contains subdirectories (e.g. uploads/user/123/abc.jpg)
			if strings.Contains(fileName, "/") {
				return nil, &ValidationError{Field: "image_keys", Message: "no subdirectories allowed in image key"}
			}

			// Validate UUID format
			if _, err := uuid.Parse(userIDInKey); err != nil {
				return nil, &ValidationError{Field: "image_keys", Message: "invalid user ID in image key"}
			}

			ext := filepath.Ext(fileName)
			rawUUID := strings.TrimSuffix(fileName, ext)
			if _, err := uuid.Parse(rawUUID); err != nil {
				return nil, &ValidationError{Field: "image_keys", Message: "invalid file name in image key"}
			}

			// Verify key belongs to the authenticated user
			if userIDInKey != userId {
				return nil, &ValidationError{Field: "image_keys", Message: "image key does not belong to authenticated user"}
			}

			parsedUploadKeys = append(parsedUploadKeys, fileName)
		}
	}
	return parsedUploadKeys, nil
}

type CreateListingResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Created string `json:"created_at"`
}
