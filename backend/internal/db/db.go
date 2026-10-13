package db

import (
	"context"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaFS embed.FS

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	var lastErr error
	for i := 0; i < 30; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = pool.Ping(pingCtx)
		cancel()
		if lastErr == nil {
			break
		}
		log.Printf("db: waiting for postgres (%d/30): %v", i+1, lastErr)
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", lastErr)
	}
	if err := Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := pool.Exec(ctx, string(schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	// TimescaleDB / Tiger Data: turn sales into a hypertable when available so
	// time-range aggregates get chunk exclusion. Plain Postgres skips this.
	var hasTS bool
	err = pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')`).Scan(&hasTS)
	if err == nil && hasTS {
		if _, err := pool.Exec(ctx, `SELECT create_hypertable('sales', 'sold_at', if_not_exists => TRUE, migrate_data => TRUE)`); err != nil {
			log.Printf("db: hypertable setup skipped: %v", err)
		} else {
			log.Printf("db: timescaledb detected, sales stored as hypertable")
		}
	}
	return nil
}
