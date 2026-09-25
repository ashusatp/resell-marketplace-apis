package dtos

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	ListingPrefix = "listings"
)

type CreateListingRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Price       int64    `json:"price" binding:"required"`
	City        string   `json:"city" binding:"required"`
	ImageKeys   []string `json:"image_keys"` //uploads/<user-id>/<image-uuid>.png
}

type ListingUploadKeys struct {
	ListingID       uuid.UUID                  `json:"listing_id"`
	ImageObjectKeys []ProcessedObjectUploadKey `json:"images"`
}

type ProcessedObjectUploadKey struct {
	OriginalKey    string
	ImageID        string
	Extension      string
	FinalObjectKey string
}

func (req *CreateListingRequest) Validate(userId string) (ListingUploadKeys, error) {
	if strings.TrimSpace(req.Title) == "" {
		return ListingUploadKeys{}, &ValidationError{Field: "title", Message: "title is required"}
	}
	if strings.TrimSpace(req.Description) == "" {
		return ListingUploadKeys{}, &ValidationError{Field: "description", Message: "description is required"}
	}
	if req.Price <= 0 {
		return ListingUploadKeys{}, &ValidationError{Field: "price", Message: "price must be greater than 0"}
	}
	if strings.TrimSpace(req.City) == "" {
		return ListingUploadKeys{}, &ValidationError{Field: "city", Message: "city is required"}
	}

	listingUUID := uuid.New()

	parsedUploadKeys := make([]ProcessedObjectUploadKey, 0, len(req.ImageKeys))
	if req.ImageKeys != nil {
		for _, key := range req.ImageKeys {
			parts := strings.Split(key, "/")
			if len(parts) != 3 {
				return ListingUploadKeys{}, &ValidationError{Field: "image_keys", Message: "invalid image key format"}
			}

			prefix := parts[0]
			userIDInKey := parts[1]
			fileName := parts[2]

			// Check prefix
			if prefix != "uploads" {
				return ListingUploadKeys{}, &ValidationError{Field: "image_keys", Message: "image key must start with 'uploads/'"}
			}

			// Validate UUID format
			if _, err := uuid.Parse(userIDInKey); err != nil {
				return ListingUploadKeys{}, &ValidationError{Field: "image_keys", Message: "invalid user ID in image key"}
			}

			ext := filepath.Ext(fileName)
			imageRawUUID := strings.TrimSuffix(fileName, ext)
			if _, err := uuid.Parse(imageRawUUID); err != nil {
				return ListingUploadKeys{}, &ValidationError{Field: "image_keys", Message: "invalid file name in image key"}
			}

			// Verify key belongs to the authenticated user
			if userIDInKey != userId {
				return ListingUploadKeys{}, &ValidationError{Field: "image_keys", Message: "image key does not belong to authenticated user"}
			}

			parsedUploadKeys = append(parsedUploadKeys, ProcessedObjectUploadKey{
				OriginalKey:    key,
				ImageID:        imageRawUUID,
				Extension:      ext,
				FinalObjectKey: mintFinalObjectKey(listingUUID, uuid.MustParse(imageRawUUID), ext),
			})
		}
	}
	return ListingUploadKeys{ImageObjectKeys: parsedUploadKeys, ListingID: listingUUID}, nil
}

type CreateListingResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Created string `json:"created_at"`
}

// {listingsPrefix}
// filepath.Ext() includes the leading dot (.jpg), formatting with "%s/%s/%s.%s" creates an extra dot:
func mintFinalObjectKey(listingID uuid.UUID, imageID uuid.UUID, ext string) string {
	return fmt.Sprintf("%s/%s/%s%s", ListingPrefix, listingID.String(), imageID.String(), ext)
}
