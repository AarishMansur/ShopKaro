package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/AarishMansur/ShopKaro/backend/internal/models"
)

func (s *Store) SaveReport(ctx context.Context, kind, lang string, start, end time.Time, content, meta string) (*models.Report, error) {
	var r models.Report
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reports (kind, period_start, period_end, lang, content, meta)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, kind, period_start, period_end, lang, content, meta, created_at`,
		kind, start, end, lang, content, meta,
	).Scan(&r.ID, &r.Kind, &r.PeriodStart, &r.PeriodEnd, &r.Lang, &r.Content, &r.Meta, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) LatestReport(ctx context.Context, kind string) (*models.Report, error) {
	var r models.Report
	err := s.pool.QueryRow(ctx, `
		SELECT id, kind, period_start, period_end, lang, content, meta, created_at
		FROM reports WHERE kind = $1 ORDER BY created_at DESC LIMIT 1`, kind,
	).Scan(&r.ID, &r.Kind, &r.PeriodStart, &r.PeriodEnd, &r.Lang, &r.Content, &r.Meta, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) JobRun(ctx context.Context, name string) (models.JobRun, error) {
	var run models.JobRun
	var lastRun *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT last_run, last_ok, last_error FROM job_runs WHERE job_name = $1`, name,
	).Scan(&lastRun, &run.LastOK, &run.Error)
	if err == pgx.ErrNoRows {
		return models.JobRun{}, nil
	}
	if err != nil {
		return run, err
	}
	run.LastRun = lastRun
	return run, nil
}

func (s *Store) RecordJobRun(ctx context.Context, name string, runErr error) error {
	ok := runErr == nil
	msg := ""
	if runErr != nil {
		msg = runErr.Error()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_runs (job_name, last_run, last_ok, last_error, updated_at)
		VALUES ($1, now(), $2, $3, now())
		ON CONFLICT (job_name) DO UPDATE
		SET last_run = now(), last_ok = $2, last_error = $3, updated_at = now()`,
		name, ok, msg)
	return err
}
