package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tecisage/resell-apis/internal/dtos"
	"github.com/tecisage/resell-apis/internal/httpx"
	middleware "github.com/tecisage/resell-apis/internal/middlewares"
	"github.com/tecisage/resell-apis/internal/services"
	"github.com/tecisage/resell-apis/internal/storage"
)

type Listing struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int64     `json:"price"`
	City        string    `json:"city"`
	Image       *string   `json:"image"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListingHandler struct {
	DB             *sql.DB
	Logger         *slog.Logger
	ListingService *services.ListingService
	Storage        *storage.Client
}

func NewListingHandler(db *sql.DB, logger *slog.Logger, storage *storage.Client) *ListingHandler {
	listingService := services.NewListingService(logger, db)
	return &ListingHandler{
		DB:             db,
		Logger:         logger,
		ListingService: listingService,
		Storage:        storage,
	}
}

func (lh *ListingHandler) GetListings(w http.ResponseWriter, r *http.Request) {

	// request scoped context
	ctx := r.Context()
	// get request id from context
	reqId := middleware.GetRequestID(ctx)

	query := `
		SELECT id, title, description, price, city, image, created_at, updated_at
		FROM listings
		ORDER BY created_at DESC
		LIMIT 100;
	`

	// 1. Use QueryContext with r.Context() to handle zombie query and connection
	rows, err := lh.DB.QueryContext(ctx, query)
	if err != nil {
		lh.Logger.Error("GetListings query error", "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}
	defer rows.Close()

	// 2. Initialize slice to avoid JSON "null" on empty results
	lis := make([]Listing, 0)
	for rows.Next() {
		var l Listing
		if err := rows.Scan(
			&l.ID,
			&l.Title,
			&l.Description,
			&l.Price,
			&l.City,
			&l.Image,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			lh.Logger.Error("GetListings scan error", "request_id", reqId, "err", err)
			httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
			return
		}
		lis = append(lis, l)
	}

	if err := rows.Err(); err != nil {
		lh.Logger.Error("GetListings rows error", "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(lis); err != nil {
		lh.Logger.Error("GetListings encode error", "request_id", reqId, "err", err)
	}
}

func (lh *ListingHandler) DeleteListing(w http.ResponseWriter, r *http.Request) {

	// request scoped context
	ctx := r.Context()
	// get request id from context
	reqId := middleware.GetRequestID(ctx)

	id := r.PathValue("id")
	if id == "" {
		lh.Logger.Error("DeleteListing: missing id", "request_id", reqId)
		httpx.Error(w, http.StatusBadRequest, "missing id", httpx.CodeBadRequest)
		return
	}

	// 3. Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		lh.Logger.Error("DeleteListing: invalid uuid", "id", id, "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusBadRequest, "invalid id format", httpx.CodeInvalidID)
		return
	}

	claims, claimErr := middleware.GetAuthClaims(ctx)
	if claimErr != nil {
		lh.Logger.Error("DeleteListing: failed to get claims", "request_id", reqId, "err", claimErr)
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", httpx.CodeUnauthorized)
		return
	}

	rowsAffected, err := lh.ListingService.DeleteListing(ctx, reqId, id, claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}

	if rowsAffected == 0 {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", httpx.CodeUnauthorized)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (lh *ListingHandler) CreateListing(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)

	claims, claimErr := middleware.GetAuthClaims(ctx)
	if claimErr != nil {
		lh.Logger.Error("CreateListing: failed to get claims", "request_id", reqId, "err", claimErr)
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", httpx.CodeUnauthorized)
		return
	}

	var req dtos.CreateListingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lh.Logger.Error("CreateListing decode error", "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusBadRequest, "invalid request body", httpx.CodeBadRequest)
		return
	}

	listingUploadKeys, err := req.Validate(claims.UserID)
	if err != nil {
		var vError *dtos.ValidationError
		errors.As(err, &vError)
		lh.Logger.Error("CreateListing validate error", "request_id", reqId, "err", err)
		httpx.ValidationError(w, http.StatusUnprocessableEntity, err.Error(), httpx.CodeValidationFailed, vError.Field)
		return
	}

	for _, uk := range listingUploadKeys.ImageObjectKeys {
		_, _, err := lh.Storage.HeadObject(ctx, uk.OriginalKey)

		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				lh.Logger.Error("CreateListing: image not found", "request_id", reqId, "err", err)
				httpx.ValidationError(w, http.StatusUnprocessableEntity, "image not found", httpx.CodeValidationFailed, "image_upload")
				return
			case errors.Is(err, storage.ErrZeroSizeObject):
				lh.Logger.Error("CreateListing: image has zero content length", "request_id", reqId, "err", err)
				httpx.ValidationError(w, http.StatusUnprocessableEntity, "image has zero content length", httpx.CodeValidationFailed, "image_upload")
				return
			case errors.Is(err, storage.ErrFileSizeExceeded):
				lh.Logger.Error("CreateListing: image size exceeds limit", "request_id", reqId, "err", err)
				httpx.ValidationError(w, http.StatusUnprocessableEntity, "image size exceeds limit", httpx.CodeValidationFailed, "image_upload")
				return
			case errors.Is(err, storage.ErrNoContentType):
				lh.Logger.Error("CreateListing: image has no content type", "request_id", reqId, "err", err)
				httpx.ValidationError(w, http.StatusUnprocessableEntity, "image has no content type", httpx.CodeValidationFailed, "image_upload")
				return
			case errors.Is(err, storage.ErrInvalidContentType):
				lh.Logger.Error("CreateListing: image has invalid content type", "request_id", reqId, "err", err)
				httpx.ValidationError(w, http.StatusUnprocessableEntity, "uploaded object has an unsupported content type", httpx.CodeValidationFailed, "image_upload")
				return
			default:
				lh.Logger.Error("CreateListing: failed to head object", "request_id", reqId, "err", err)
				httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
				return
			}
		}

	}

	out, err := lh.ListingService.CreateListing(ctx, reqId, req, claims.UserID, listingUploadKeys)
	if err != nil {
		lh.Logger.Error("CreateListing service error", "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)

}
