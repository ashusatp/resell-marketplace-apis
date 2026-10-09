package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tecisage/resell-apis/internal/config"
	"github.com/tecisage/resell-apis/internal/db"
	"github.com/tecisage/resell-apis/internal/handlers"
	middleware "github.com/tecisage/resell-apis/internal/middlewares"
	"github.com/tecisage/resell-apis/internal/storage"
	"github.com/tecisage/resell-apis/internal/worker"
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

	// Object Storage Initialisation
	r2Client, err := storage.NewR2Client(context.TODO(), storage.R2Config{
		AccountID:    cfg.R2_ACCOUNT_ID,
		AccessKey:    cfg.R2_ACCESS_KEY,
		AccessSecret: cfg.R2_ACCESS_SECRET,
		Bucket:       cfg.R2_BUCKET,
	})
	if err != nil {
		log.Fatalf("main.storage.NewR2Client: %v", err)
	}

	// worker
	worker := worker.NewWorker(db, r2Client, logger)
	go worker.Run(context.TODO())

	// create mux
	mux := http.NewServeMux()

	// health check
	mux.HandleFunc("GET /healthz", handlers.Healthz)

	// auth routes
	authHandler := handlers.NewAuthHandler(db, logger, cfg.JWTSecret)
	mux.HandleFunc("POST /auth/signup", authHandler.Signup)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	// 2. Protect routes (e.g. creating/deleting listings)
	requireAuth := middleware.AuthMiddleware(cfg.JWTSecret)

	// listing routes
	listingHandler := handlers.NewListingHandler(db, logger, r2Client)
	mux.HandleFunc("GET /listings", listingHandler.GetListings)
	mux.HandleFunc("GET /listings/{id}", listingHandler.GetListingByID)
	mux.Handle("DELETE /listings/{id}", requireAuth(http.HandlerFunc(listingHandler.DeleteListing)))
	mux.Handle("POST /listings", requireAuth(http.HandlerFunc(listingHandler.CreateListing)))

	// Upload routes
	uploadHandler := handlers.NewUploadHandler(logger, r2Client)
	mux.Handle("POST /uploads/presign", requireAuth(http.HandlerFunc(uploadHandler.Presign)))

	// middlewares
	corsMiddleware := middleware.CORSMiddleware(cfg.AllowedOrigins)
	rootHandler := middleware.RequestIDMiddleware(corsMiddleware(mux))

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
