package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/tecisage/resell-apis/internal/dtos"
	"github.com/tecisage/resell-apis/internal/jobs"
	"github.com/tecisage/resell-apis/internal/storage"
)

type ListingService struct {
	Logger  *slog.Logger
	Db      *sql.DB
	Storage *storage.Client
}

func NewListingService(logger *slog.Logger, db *sql.DB, storage *storage.Client) *ListingService {
	return &ListingService{
		Logger:  logger,
		Db:      db,
		Storage: storage,
	}
}

func (ls *ListingService) CreateListing(ctx context.Context, reqId string, req dtos.CreateListingRequest, userId string, listingUploadKeys dtos.ListingUploadKeys) (dtos.CreateListingResponse, error) {
	// 1. Verify uploaded images with R2
	for _, uk := range listingUploadKeys.ImageObjectKeys {
		_, _, err := ls.Storage.HeadObject(ctx, uk.OriginalKey)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				return dtos.CreateListingResponse{}, &dtos.ValidationError{Field: "image_keys", Message: fmt.Sprintf("image not found: %s", uk.OriginalKey)}
			case errors.Is(err, storage.ErrZeroSizeObject):
				return dtos.CreateListingResponse{}, &dtos.ValidationError{Field: "image_keys", Message: "image has zero content length"}
			case errors.Is(err, storage.ErrFileSizeExceeded):
				return dtos.CreateListingResponse{}, &dtos.ValidationError{Field: "image_keys", Message: "image size exceeds limit"}
			case errors.Is(err, storage.ErrNoContentType):
				return dtos.CreateListingResponse{}, &dtos.ValidationError{Field: "image_keys", Message: "image has no content type"}
			case errors.Is(err, storage.ErrInvalidContentType):
				return dtos.CreateListingResponse{}, &dtos.ValidationError{Field: "image_keys", Message: "uploaded object has an unsupported content type"}
			default:
				ls.Logger.Error("Listing Service: failed to head object", "key", uk.OriginalKey, "request_id", reqId, "err", err)
				return dtos.CreateListingResponse{}, err
			}
		}
	}

	// 2. Start DB Transaction
	tx, err := ls.Db.BeginTx(ctx, nil)
	if err != nil {
		ls.Logger.Error("Listing Service: failed to begin transaction", "request_id", reqId, "err", err)
		return dtos.CreateListingResponse{}, err
	}
	defer tx.Rollback()

	var primaryImage *string
	if len(listingUploadKeys.ImageObjectKeys) > 0 {
		primaryImage = &listingUploadKeys.ImageObjectKeys[0].FinalObjectKey
	}

	query := `
		INSERT INTO listings (id, title, description, price, city, image, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, status, created_at;
	`

	// Insert listing into database
	var out dtos.CreateListingResponse
	err = tx.QueryRowContext(ctx, query, listingUploadKeys.ListingID, req.Title, req.Description, req.Price, req.City, primaryImage, userId).
		Scan(&out.ID, &out.Title, &out.Status, &out.Created)
	if err != nil {
		ls.Logger.Error("Listing Service: CreateListing insert error", "request_id", reqId, "err", err)
		return dtos.CreateListingResponse{}, err
	}

	// Insert images into database
	for i, s := range listingUploadKeys.ImageObjectKeys {
		query := `
			INSERT INTO images (listing_id, object_key, position)
			VALUES ($1, $2, $3);
		`

		imageUUID := uuid.MustParse(s.ImageID)
		if _, err := tx.ExecContext(ctx, query, listingUploadKeys.ListingID, dtos.MintImageKey(imageUUID), int16(i)); err != nil {
			ls.Logger.Error("Listing Service: CreateListing insert image error", "request_id", reqId, "err", err)
			return dtos.CreateListingResponse{}, err
		}
	}

	// Create job in jobs table for image processing
	payload, err := json.Marshal(ls.prepareJobPayload(listingUploadKeys))
	if err != nil {
		ls.Logger.Error("Listing Service: failed to marshal payload", "request_id", reqId, "err", err)
		return dtos.CreateListingResponse{}, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO jobs (kind, payload)
		VALUES ($1, $2);
	`, jobs.KindProcessingListingImages, payload)
	if err != nil {
		ls.Logger.Error("Listing Service: CreateListing insert job error", "request_id", reqId, "err", err)
		return dtos.CreateListingResponse{}, err
	}

	// 3. Commit DB Transaction
	if err := tx.Commit(); err != nil {
		ls.Logger.Error("Listing Service: failed to commit transaction", "request_id", reqId, "err", err)
		return dtos.CreateListingResponse{}, err
	}

	ls.Logger.Info("Listing created successfully", "id", out.ID, "request_id", reqId)
	return out, nil
}

func (ls *ListingService) DeleteListing(ctx context.Context, reqId string, listingId string, userId string) (int64, error) {
	result, err := ls.Db.ExecContext(ctx, `DELETE FROM listings WHERE id = $1 AND created_by = $2`, listingId, userId)
	if err != nil {
		ls.Logger.Error("Delete Listing Service: DeleteListing exec error", "listing_id", listingId, "request_id", reqId, "err", err)
		return -1, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		ls.Logger.Error("Listing Service: DeleteListing rowsAffected error", "listing_id", listingId, "request_id", reqId, "err", err)
		return -1, err
	}

	return rowsAffected, nil
}

func (ls *ListingService) prepareJobPayload(listingUploadKeys dtos.ListingUploadKeys) jobs.ProcessImagepayload {
	imageSources := make([]jobs.ImageSource, 0, len(listingUploadKeys.ImageObjectKeys))

	for _, uk := range listingUploadKeys.ImageObjectKeys {
		imageSources = append(imageSources, jobs.ImageSource{
			OriginalKey: uk.OriginalKey,
			FinalKey:    uk.FinalObjectKey,
		})
	}

	return jobs.ProcessImagepayload{
		ListingID:    listingUploadKeys.ListingID,
		ImageSources: imageSources,
	}
}
