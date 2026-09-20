package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type listing struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       int64     `json:"price"`
	City        string    `json:"city"`
	Image       *string   `json:"image"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func Listings(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// query all the listings
		rows, err := db.Query(
			`SELECT id, title, description, price, city, image, created_at, updated_at
			FROM listings
			ORDER BY created_at DESC
			LIMIT 100;`,
		)
		if err != nil {
			log.Printf("query: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// iterate over rows and scan into struct
		var lis []listing
		for rows.Next() {
			var l listing

			// scan row into struct also handle error for each field
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
				log.Printf("rows.Scan: %v\n", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			lis = append(lis, l)
		}

		// check if any error during iteration
		if err := rows.Err(); err != nil {
			log.Printf("rows.Err(): %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// convert to json
		// using json marshal
		_ = json.NewEncoder(w).Encode(lis)
	}
}
