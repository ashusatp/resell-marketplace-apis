package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
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
	DB *sql.DB
}

func NewListingHandler(db *sql.DB) *ListingHandler {
	return &ListingHandler{DB: db}
}

func (lh *ListingHandler) GetListings(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, description, price, city, image, created_at, updated_at
		FROM listings
		ORDER BY created_at DESC
		LIMIT 100;
	`

	// 1. Use QueryContext with r.Context()
	rows, err := lh.DB.QueryContext(r.Context(), query)
	if err != nil {
		log.Printf("GetListings query error: %v\n", err)
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
			log.Printf("GetListings scan error: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		lis = append(lis, l)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetListings rows error: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(lis); err != nil {
		log.Printf("GetListings encode error: %v\n", err)
	}
}

func (lh *ListingHandler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		log.Printf("DeleteListing: missing id\n")
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// 3. Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		log.Printf("DeleteListing: invalid uuid %q: %v\n", id, err)
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	// 4. Use ExecContext with r.Context()
	result, err := lh.DB.ExecContext(r.Context(), `DELETE FROM listings WHERE id = $1`, id)
	if err != nil {
		log.Printf("DeleteListing exec error: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("DeleteListing rowsAffected error: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		log.Printf("DeleteListing: listing not found: %s\n", id)
		http.Error(w, "listing not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
