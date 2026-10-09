package worker

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/tecisage/resell-apis/internal/storage"
)

type Worker struct {
	db      *sql.DB
	storage *storage.Client
	logger  *slog.Logger
}

func NewWorker(db *sql.DB, storage *storage.Client, logger *slog.Logger) *Worker {
	return &Worker{
		db:      db,
		storage: storage,
		logger:  logger,
	}
}

func (w *Worker) Run() {
	w.logger.Info("Worker Started")

	for {
		// fetch jobs from db
		w.logger.Info("Fetching jobs from db")

		// process jobs
		w.logger.Info("Processing jobs")

		// update db
		w.logger.Info("Updating db")

		time.Sleep(5 * time.Second)
	}
}
