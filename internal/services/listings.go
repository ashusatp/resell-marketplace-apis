package services

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/tecisage/resell-apis/internal/dtos"
)

type ListingService struct {
	Logger *slog.Logger
	Db     *sql.DB
}

func NewListingService(logger *slog.Logger, db *sql.DB) *ListingService {
	return &ListingService{
		Logger: logger,
		Db:     db,
	}
}

func (ls *ListingService) CreateListing(ctx context.Context, reqId string, req dtos.CreateListingRequest, userId string, parsedUploadKeys []string) (dtos.CreateListingResponse, error) {
	query := `
		INSERT INTO listings (title, description, price, city, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, created_at;
	`

	var out dtos.CreateListingResponse
	err := ls.Db.QueryRowContext(ctx, query, req.Title, req.Description, req.Price, req.City, userId).
		Scan(&out.ID, &out.Title, &out.Created)
	if err != nil {
		ls.Logger.Error("Listing Service: CreateListing query error", "request_id", reqId, "err", err)
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
