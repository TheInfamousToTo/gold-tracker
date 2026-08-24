package repository

import (
	"context"
	"os"
	"testing"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

func testRepo(t *testing.T) *PostgresRepository {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set, skipping integration test")
	}
	repo, err := NewPostgresRepository()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(repo.Close)
	return repo
}

func TestCreateSignalAndGetLatestSignal(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	reasoning := "integration test signal"
	m := "claude-opus-5"
	created, err := repo.CreateSignal(ctx, model.SignalLog{
		SignalType: "HOLD",
		Reasoning:  &reasoning,
		Model:      &m,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("CreateSignal: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected non-zero ID")
	}
	if created.Source != "test" {
		t.Errorf("Source = %q, want test", created.Source)
	}
	if created.Model == nil || *created.Model != m {
		t.Errorf("Model = %v, want %q", created.Model, m)
	}

	latest, err := repo.GetLatestSignal(ctx, "test")
	if err != nil {
		t.Fatalf("GetLatestSignal: %v", err)
	}
	if latest == nil {
		t.Fatalf("expected a latest signal, got nil")
	}
	if latest.ID != created.ID {
		t.Errorf("GetLatestSignal returned ID %d, want %d (most recent)", latest.ID, created.ID)
	}

	// Clean up so repeat runs don't accumulate rows in the owner's data.
	if _, err := repo.Pool.Exec(ctx, "DELETE FROM signals_log WHERE id = $1", created.ID); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

func TestGetLatestSignalNoneReturnsNilNoError(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	latest, err := repo.GetLatestSignal(ctx, "source-that-does-not-exist")
	if err != nil {
		t.Fatalf("GetLatestSignal: %v", err)
	}
	if latest != nil {
		t.Fatalf("expected nil for a source with no rows, got %+v", latest)
	}
}
