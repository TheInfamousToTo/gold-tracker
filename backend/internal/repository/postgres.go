package repository

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/TheInfamousToTo/gold-tracker/backend/migrations"
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

	// Bring existing installs up to date. Every migration is written to
	// be idempotent, so this is safe to run on every boot.
	if err := applyMigrations(context.Background(), pool); err != nil {
		return nil, err
	}

	return &PostgresRepository{Pool: pool}, nil
}

// applyMigrations runs every embedded migration in filename order.
//
// There is no ledger table: each file is written so that re-running it
// is a no-op, which keeps a fresh container and a long-lived one on the
// same path. The cost is that migrations must stay idempotent — a bare
// ALTER TABLE ADD COLUMN without IF NOT EXISTS would fail the second
// boot and take the API down with it.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		return fmt.Errorf("unable to list migrations: %v", err)
	}
	sort.Strings(entries)

	for _, name := range entries {
		statements, err := migrations.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("unable to read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(statements)); err != nil {
			return fmt.Errorf("unable to apply migration %s: %v", name, err)
		}
	}
	return nil
}

func (r *PostgresRepository) Close() {
	r.Pool.Close()
}
