package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tecisage/resell-apis/internal/config"
	"github.com/tecisage/resell-apis/internal/db"
	"github.com/tecisage/resell-apis/internal/handlers"
	middleware "github.com/tecisage/resell-apis/internal/middlewares"
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

	// logging
	loggerHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})
	logger := slog.New(loggerHandler)
	slog.SetDefault(logger)

	// create mux
	mux := http.NewServeMux()

	// health check
	mux.HandleFunc("GET /healthz", handlers.Healthz)

	// listing routes
	listingHandler := handlers.NewListingHandler(db, logger)
	mux.HandleFunc("GET /listings", listingHandler.GetListings)
	mux.HandleFunc("DELETE /listings/{id}", listingHandler.DeleteListing)
	mux.HandleFunc("POST /listings", listingHandler.CreateListing)

	// auth routes
	authHandler := handlers.NewAuthHandler(db, logger)
	mux.HandleFunc("POST /auth/signup", authHandler.Signup)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	// request id middleware
	rootHandler := middleware.RequestIDMiddleware(mux)

	// create http server
	log.Println("Starting server on :" + cfg.Port)
	srv := http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           rootHandler,
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
