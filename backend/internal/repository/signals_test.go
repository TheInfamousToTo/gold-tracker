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

func TestSignalRoundTripsTheAdvisorFields(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()

	conf, amount, grams, buyBelow := 0.7, 40.0, 0.79, 47.5
	horizon := 180
	action := "BUY"
	plan := "buy 40 BHD of gold"
	in := model.SignalLog{
		Kind: model.SignalKindReview, Metal: "gold", SignalType: "ADJUST", Source: "manual",
		Confidence: &conf, HorizonDays: &horizon, KeyFactors: []string{"trend up", "Fed on hold"},
		AmountBHD: &amount, AmountGrams: &grams, Levels: &model.Levels{BuyMoreBelow: &buyBelow},
		News:        []model.NewsItem{{Title: "Fed holds", Source: "Reuters", Date: "2026-10-08", URL: "https://reuters.com/a", Impact: "bullish"}},
		NewsChecked: true, Deviates: true, SuggestedAction: &action, PlanText: &plan,
		Baseline: []byte(`{"action":"BUY","amount_bhd":35}`),
	}
	if _, err := repo.CreateSignal(ctx, in); err != nil {
		t.Fatalf("create: %v", err)
	}
	// A legacy row with none of the new columns must still read.
	if _, err := repo.Pool.Exec(ctx, `INSERT INTO signals_log (signal_type, reasoning, source) VALUES ('HOLD', 'old', 'n8n')`); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetSignals(ctx, 10)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d signals, want 2", len(got))
	}
	var s, legacy model.SignalLog
	for _, g := range got {
		if g.Kind == model.SignalKindReview {
			s = g
		} else {
			legacy = g
		}
	}
	if s.Confidence == nil || *s.Confidence != 0.7 || len(s.KeyFactors) != 2 || len(s.News) != 1 ||
		s.Levels == nil || *s.Levels.BuyMoreBelow != 47.5 || !s.NewsChecked || !s.Deviates ||
		*s.SuggestedAction != "BUY" || *s.PlanText != plan || len(s.Baseline) == 0 {
		t.Errorf("review did not round-trip: %+v", s)
	}
	if legacy.Kind != model.SignalKindAnalysis || legacy.KeyFactors != nil || legacy.Levels != nil {
		t.Errorf("legacy row read as %+v", legacy)
	}

	// Reviews never count as the latest automatic analysis.
	if latest, err := repo.GetLatestSignal(ctx, "manual"); err != nil || latest != nil {
		t.Errorf("GetLatestSignal(manual) = %+v, %v; want nil (only a review exists)", latest, err)
	}
}

func TestAdvisorSettingsDefaultAndUpdate(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()
	if _, err := repo.Pool.Exec(ctx, `UPDATE ai_settings SET monthly_budget_bhd = 0, reserve_bhd = 0,
		target_gold_pct = 80, spread_pct_gold = 1, spread_pct_silver = 1, min_fee_bhd = 0,
		stop_loss_pct = 15, news_enabled = true`); err != nil {
		t.Fatal(err)
	}

	s, err := repo.GetAdvisorSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.TargetGoldPct != 80 || s.StopLossPct != 15 || !s.NewsEnabled {
		t.Errorf("defaults = %+v", s)
	}

	s.MonthlyBudgetBHD, s.TargetGoldPct, s.NewsEnabled = 60, 70, false
	saved, err := repo.UpdateAdvisorSettings(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := repo.GetAdvisorSettings(ctx)
	if saved.MonthlyBudgetBHD != 60 || again.TargetGoldPct != 70 || again.NewsEnabled {
		t.Errorf("update did not stick: saved %+v, read %+v", saved, again)
	}
	var rows int
	_ = repo.Pool.QueryRow(ctx, "SELECT count(*) FROM ai_settings").Scan(&rows)
	if rows != 1 {
		t.Errorf("ai_settings has %d rows, want 1", rows)
	}
}
