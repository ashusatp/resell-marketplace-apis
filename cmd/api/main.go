package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tecisage/resell-apis/internal/config"
	"github.com/tecisage/resell-apis/internal/db"
	"github.com/tecisage/resell-apis/internal/handlers"
)

func main() {

	// load config
	cfg := config.MustLoad()

	// DB connect
	db, err := db.Connect(cfg.DB_URL)
	if err != nil {
		log.Fatalf("main.db.Connect: %v", err)
	}
	defer db.Close()

	// create mux
	mux := http.NewServeMux()

	// health check
	mux.HandleFunc("GET /healthz", handlers.Healthz)

	// listing routes
	listingHandler := handlers.NewListingHandler(db)
	mux.HandleFunc("GET /listings", listingHandler.GetListings)
	mux.HandleFunc("DELETE /listings/{id}", listingHandler.DeleteListing)

	// create http server
	log.Println("Starting server on :" + cfg.Port)
	srv := http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// start server
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("main.srv.ListenAndServe: %v", err)
	}
}
