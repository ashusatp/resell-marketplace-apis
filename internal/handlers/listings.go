package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tecisage/resell-apis/internal/httpx"
	middleware "github.com/tecisage/resell-apis/internal/middlewares"
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
	DB     *sql.DB
	Logger *slog.Logger
}

func NewListingHandler(db *sql.DB, logger *slog.Logger) *ListingHandler {
	return &ListingHandler{
		DB:     db,
		Logger: logger,
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

	// 4. Use ExecContext with r.Context() to handle zombie query and connection
	result, err := lh.DB.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)
	if err != nil {
		lh.Logger.Error("DeleteListing exec error", "listing_id", id, "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		lh.Logger.Error("DeleteListing rowsAffected error", "listing_id", id, "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}

	if rowsAffected == 0 {
		httpx.Error(w, http.StatusNotFound, "listing not found", httpx.CodeNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type createListingRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	City        string `json:"city"`
}

func (lh *ListingHandler) CreateListing(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)

	var req createListingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lh.Logger.Error("CreateListing decode error", "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusBadRequest, "invalid request body", httpx.CodeBadRequest)
		return
	}

	query := `
		INSERT INTO listings (title, description, price, city)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`

	var id string
	err := lh.DB.QueryRowContext(ctx, query, req.Title, req.Description, req.Price, req.City).
		Scan(&id)
	if err != nil {
		lh.Logger.Error("CreateListing query error", "request_id", reqId, "err", err)
		httpx.Error(w, http.StatusInternalServerError, "internal server error", httpx.CodeInternalError)
		return
	}

	lh.Logger.Info("Listing created successfully", "id", id, "request_id", reqId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "message": "Listing created successfully"})

}
