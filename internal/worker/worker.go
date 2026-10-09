package worker

import (
	"database/sql"
	"log/slog"

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
		w.claimAndProcess()
	}
}

func (w *Worker) claimAndProcess() {
	w.logger.Info("Claiming and processing jobs....")

}
