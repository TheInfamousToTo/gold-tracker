package repository

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	Pool *pgxpool.Pool
}

func NewPostgresRepository() (*PostgresRepository, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	// Bring existing installs up to date. Both statements are
	// idempotent, so this is safe to run on every boot.
	if _, err := pool.Exec(context.Background(), `
		ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS model TEXT;
		ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'manual';
	`); err != nil {
		return nil, fmt.Errorf("unable to apply signals_log migration: %v", err)
	}

	return &PostgresRepository{Pool: pool}, nil
}

func (r *PostgresRepository) Close() {
	r.Pool.Close()
}
