// Package rewrap implements the offline, resumable AES key rewrap campaign.
package rewrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/djalmajr/tenancit/server/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Runner) Run(ctx context.Context, config Config) (summary Summary, runErr error) {
	if r.DB == nil || r.Cryptor == nil {
		return summary, fmt.Errorf("%w: database and cryptor are required", ErrInvalidConfig)
	}
	now := r.Now
	if now == nil {
		now = time.Now
	}
	if config.JobID == "" {
		config.JobID = uuid.NewString()
	}
	if _, err := uuid.Parse(config.JobID); err != nil {
		return summary, fmt.Errorf("%w: job ID must be a UUID", ErrInvalidConfig)
	}
	if config.TargetVersion <= 0 || config.TargetVersion != r.Cryptor.CurrentVersion() || config.BatchSize < 1 || config.BatchSize > 1000 || config.MaxDuration <= 0 || (config.DryRun && config.ConfirmedWrite) {
		return summary, ErrInvalidConfig
	}
	if !config.DryRun && !config.ConfirmedWrite {
		return summary, fmt.Errorf("%w: write confirmation required", ErrInvalidConfig)
	}
	if config.NoProgressTimeout <= 0 {
		config.NoProgressTimeout = 30 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 250 * time.Millisecond
	}
	summary = Summary{JobID: config.JobID, TargetVersion: config.TargetVersion, DryRun: config.DryRun, StartedAt: now().UTC()}
	defer func() {
		if runErr != nil {
			telemetry.RecordRewrapFailure(context.Background(), metricFailureReason(runErr))
			telemetry.RecordRewrapCompletion(context.Background(), "error", summary.Rewrapped, summary.Remaining, now().UTC().Sub(summary.StartedAt))
		}
	}()
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}

	campaignContext, cancel := context.WithTimeout(ctx, config.MaxDuration)
	defer cancel()
	lockConn, err := r.DB.Acquire(campaignContext)
	if err != nil {
		return summary, fmt.Errorf("acquire campaign connection: %w", err)
	}
	defer lockConn.Release()
	var locked bool
	if err := lockConn.QueryRow(campaignContext, `SELECT pg_try_advisory_lock($1)`, advisoryLockID).Scan(&locked); err != nil {
		return summary, fmt.Errorf("acquire campaign lock: %w", err)
	}
	if !locked {
		return summary, ErrCampaignLocked
	}
	defer func() {
		unlockContext, unlockCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer unlockCancel()
		_, _ = lockConn.Exec(unlockContext, `SELECT pg_advisory_unlock($1)`, advisoryLockID)
	}()

	preflightTx, err := r.DB.BeginTx(campaignContext, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return summary, fmt.Errorf("begin rewrap preflight: %w", err)
	}
	defer preflightTx.Rollback(campaignContext)
	summary.Inventory, err = r.inventory(campaignContext, preflightTx)
	if err != nil {
		return summary, err
	}
	for _, item := range summary.Inventory {
		if item.KeyVersion == nil || *item.KeyVersion <= 0 {
			return summary, ErrMalformedCiphertext
		}
		version := int(*item.KeyVersion)
		if version > config.TargetVersion {
			return summary, fmt.Errorf("%w: database version is newer than target", ErrInvalidConfig)
		}
		if !r.Cryptor.HasVersion(version) {
			return summary, ErrMissingKeyVersion
		}
		telemetry.RecordRewrapRemaining(campaignContext, version, item.Rows)
	}
	if err := r.authenticateAll(campaignContext, preflightTx, config.BatchSize, &summary); err != nil {
		return summary, err
	}
	if err := preflightTx.Commit(campaignContext); err != nil {
		return summary, fmt.Errorf("commit rewrap preflight: %w", err)
	}
	if config.DryRun {
		summary.Remaining, err = r.remaining(campaignContext, config.TargetVersion)
		if err != nil {
			return summary, err
		}
		summary.CompletedAt = now().UTC()
		telemetry.RecordRewrapCompletion(campaignContext, "dry_run", summary.Scanned, summary.Remaining, summary.CompletedAt.Sub(summary.StartedAt))
		logger.Info("rewrap dry-run complete", "job_id", summary.JobID, "target_version", summary.TargetVersion, "rows_scanned", summary.Scanned, "rows_remaining", summary.Remaining)
		return summary, nil
	}
	if ok, evidenceErr := r.hasSafetyEvidence(campaignContext); evidenceErr != nil {
		return summary, evidenceErr
	} else if !ok {
		return summary, ErrSafetyEvidence
	}

	lastProgress := now()
	for {
		batchStarted := now()
		processed, batchErr := r.processBatch(campaignContext, config.TargetVersion, config.BatchSize)
		if batchErr != nil {
			telemetry.RecordRewrapBatch(campaignContext, "error", 0, now().Sub(batchStarted))
			return summary, batchErr
		}
		if processed > 0 {
			summary.Batches++
			summary.Rewrapped += int64(processed)
			lastProgress = now()
			telemetry.RecordRewrapBatch(campaignContext, "success", processed, now().Sub(batchStarted))
		}
		summary.Remaining, err = r.remaining(campaignContext, config.TargetVersion)
		if err != nil {
			return summary, err
		}
		if summary.Remaining == 0 {
			break
		}
		if processed == 0 {
			summary.LockedRetries++
			telemetry.RecordRewrapBatch(campaignContext, "locked", 0, now().Sub(batchStarted))
			if now().Sub(lastProgress) >= config.NoProgressTimeout {
				return summary, ErrNoProgress
			}
			select {
			case <-campaignContext.Done():
				return summary, fmt.Errorf("rewrap canceled: %w", campaignContext.Err())
			case <-time.After(config.PollInterval):
			}
		}
	}
	summary.CompletedAt = now().UTC()
	telemetry.RecordRewrapCompletion(campaignContext, "success", summary.Rewrapped, 0, summary.CompletedAt.Sub(summary.StartedAt))
	logger.Info("rewrap complete", "job_id", summary.JobID, "target_version", summary.TargetVersion, "rows_rewrapped", summary.Rewrapped, "batches", summary.Batches)
	if r.Reporter != nil {
		if err := r.Reporter.Report(campaignContext, summary, "healthy"); err != nil {
			return summary, fmt.Errorf("publish rewrap report: %w", err)
		}
	}
	return summary, nil
}
