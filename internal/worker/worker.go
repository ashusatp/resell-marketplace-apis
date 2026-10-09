package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/tecisage/resell-apis/internal/dtos"
	"github.com/tecisage/resell-apis/internal/jobs"
	"github.com/tecisage/resell-apis/internal/storage"
)

const maxAttempts = 4
const JobPollInterval = 3 * time.Second

var ErrPermanent = errors.New("permanent failure")

func permanentf(format string, args ...any) error {
	return fmt.Errorf("%w:%s", ErrPermanent, fmt.Sprintf(format, args...))
}

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

func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("Worker Started")
	for {
		isEmpty, err := w.claimAndProcess(ctx)
		if err != nil {
			w.logger.Error("Error claiming job", "error", err)
			continue
		}
		if isEmpty {
			w.logger.Info("No jobs found , will retry after ", "JobPollInterval", JobPollInterval)
			time.Sleep(JobPollInterval)
		}
	}
}

func (w *Worker) claimAndProcess(ctx context.Context) (bool, error) {

	var (
		id       int64
		kind     string
		payload  []byte
		attempts int
	)

	query := ` UPDATE jobs 
		SET attempts = attempts + 1 ,
		run_after = NOW() + LEAST(
			(INTERVAL '30 seconds') * POWER(2, attempts),
			INTERVAL '1 hour' )
		WHERE id = (
				SELECT id FROM jobs 
				WHERE run_after <= NOW()
				AND attempts < $1
				ORDER BY run_after, id
				FOR UPDATE SKIP LOCKED 
				LIMIT 1
			)
		RETURNING id, kind, payload ,attempts`

	err := w.db.QueryRowContext(ctx, query, maxAttempts).Scan(&id, &kind, &payload, &attempts)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, err
	}

	w.logger.Info("job processing started ", "id: ", id, "kind: ", kind, "attempts: ", attempts)

	//todo: remove this when worker is ready
	time.Sleep(1 * time.Second)

	return false, nil
}

func (w *Worker) process(ctx context.Context, jobID int64, kind string, payload []byte) error {

	switch kind {
	case jobs.KindProcessingListingImages:
		var p jobs.ProcessImagepayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return permanentf("invalid payload: %v", err)
		}
		return w.processImages(ctx, jobID, &p)

	default:
		return permanentf("unsupported job kind: %q", kind)
	}
}

func (w *Worker) processImages(ctx context.Context, jobID int64, payload *jobs.ProcessImagepayload) error {

	log := w.logger.With("job_id", jobID, "listing_id", payload.ListingID)

	//Step 1: Fetch listing images
	normalized := make([][]byte, 0, len(payload.ImageSources)) // tradeof - memory consumption
	for _, src := range payload.ImageSources {
		rc, err := w.storage.GetObject(ctx, src.OriginalKey)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("object not found in storage ,skipping", "originalKey", src.OriginalKey)
				return w.finishImageProcessing(ctx, jobID, payload.ListingID, dtos.ListingStatusRejected)
			}
			return fmt.Errorf("failed to get object %q: %w", src.OriginalKey, err)
		}

		imgBytes, err := io.ReadAll(rc)
		defer rc.Close()
		if err != nil {
			return fmt.Errorf("failed to read object %q: %w", src.OriginalKey, err)
		}

		normalized = append(normalized, imgBytes)
	}

	return nil
}

func (w *Worker) finishImageProcessing(ctx context.Context, jobID int64, listingID uuid.UUID, status dtos.ListingStatusType) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback() // if already commited then this is no-op
	}()

	// update the status of listing
	_, err = tx.ExecContext(ctx, "UPDATE listings SET status = $1, updated_at = NOW() WHERE id = $2", status, listingID)
	if err != nil {
		return fmt.Errorf("failed to update listing %q status %q: %w", listingID, status, err)
	}

	// deleter the job
	_, err = tx.ExecContext(ctx, "DELETE FROM jobs WHERE id = $1", jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job %d: %w", jobID, err)
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
