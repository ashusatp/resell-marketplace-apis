package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		lis = append(lis, l)
	}

	if err := rows.Err(); err != nil {
		lh.Logger.Error("GetListings rows error", "request_id", reqId, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// 3. Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		lh.Logger.Error("DeleteListing: invalid uuid", "id", id, "request_id", reqId, "err", err)
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	// 4. Use ExecContext with r.Context() to handle zombie query and connection
	result, err := lh.DB.ExecContext(ctx, `DELETE FROM listings WHERE id = $1`, id)
	if err != nil {
		lh.Logger.Error("DeleteListing exec error", "listing_id", id, "request_id", reqId, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		lh.Logger.Error("DeleteListing rowsAffected error", "listing_id", id, "request_id", reqId, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "listing not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
