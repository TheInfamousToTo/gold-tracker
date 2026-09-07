package repository

import (
	"context"
	"os"
	"testing"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise the migrations and the portfolio view against a
// real Postgres, because the valuation arithmetic lives in SQL and
// nothing else can check it. They are skipped unless
// GOLD_TEST_DATABASE_URL names a database they may freely write to:
//
//	GOLD_TEST_DATABASE_URL=postgres://user:pass@localhost:5432/gold_test go test ./internal/repository/
//
// The database is emptied between tests, so never point this at the
// real one.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("GOLD_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set GOLD_TEST_DATABASE_URL to run the schema tests")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := applyMigrations(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		"TRUNCATE gold_items, gold_prices, silver_prices, signals_log RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

func repoFor(t *testing.T) *PostgresRepository {
	return &PostgresRepository{Pool: testPool(t)}
}

// Re-running the migrations must be a no-op, since the API applies them
// on every boot. A statement that is not idempotent would take the API
// down on its second start rather than its first.
func TestMigrationsAreIdempotent(t *testing.T) {
	pool := testPool(t)
	for i := 0; i < 2; i++ {
		if err := applyMigrations(context.Background(), pool); err != nil {
			t.Fatalf("re-applying migrations (pass %d): %v", i+2, err)
		}
	}
}

func TestPortfolioValuesGoldAtItsKaratRate(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()

	if _, err := repo.CreatePrice(ctx, model.GoldPrice{
		PriceDate: "2026-08-01", PricePerGram24k: 24, Source: "test",
	}); err != nil {
		t.Fatalf("create price: %v", err)
	}

	karat := 21.0
	if _, err := repo.CreateItem(ctx, model.GoldItem{
		PurchaseDate: "2026-07-01", ItemName: "Chain", MetalType: model.MetalGold,
		PurityKarat: &karat, WeightGrams: 10, PricePaidTotal: 200,
	}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	summary, err := repo.GetPortfolioSummary(ctx)
	if err != nil {
		t.Fatalf("portfolio: %v", err)
	}
	if len(summary.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(summary.Items))
	}

	// 24 BHD/g at 24K is 21 BHD/g at 21K, so 10g is worth 210.
	item := summary.Items[0]
	if item.CurrentPricePerGram == nil || *item.CurrentPricePerGram != 21 {
		t.Errorf("price per gram = %v, want 21", item.CurrentPricePerGram)
	}
	if item.CurrentValue == nil || *item.CurrentValue != 210 {
		t.Errorf("current value = %v, want 210", item.CurrentValue)
	}
	if item.GainLoss == nil || *item.GainLoss != 10 {
		t.Errorf("gain = %v, want 10", item.GainLoss)
	}
}

// Silver is valued off its own table at its own fineness. Reading the
// gold price for a silver row is the failure this whole split exists
// to prevent, and it would be invisible in the UI: just a wrong number.
func TestPortfolioValuesSilverAtItsFineness(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()

	if _, err := repo.CreatePrice(ctx, model.GoldPrice{
		PriceDate: "2026-08-01", PricePerGram24k: 32.5, Source: "test",
	}); err != nil {
		t.Fatalf("create gold price: %v", err)
	}
	if _, err := repo.CreateSilverPrice(ctx, model.SilverPrice{
		PriceDate: "2026-08-01", PricePerGram999: 0.999, Source: "test",
	}); err != nil {
		t.Fatalf("create silver price: %v", err)
	}

	fineness := 925.0
	if _, err := repo.CreateItem(ctx, model.GoldItem{
		PurchaseDate: "2026-07-01", ItemName: "Tray", MetalType: model.MetalSilver,
		PurityFineness: &fineness, WeightGrams: 100, PricePaidTotal: 80,
	}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	summary, err := repo.GetPortfolioSummary(ctx)
	if err != nil {
		t.Fatalf("portfolio: %v", err)
	}
	item := summary.Items[0]

	if item.MetalType != model.MetalSilver {
		t.Errorf("metal = %q, want silver", item.MetalType)
	}
	// 0.999 per gram of 999 fine is 0.925 per gram of sterling.
	if item.CurrentPricePerGram == nil {
		t.Fatalf("silver item has no current price — it is not reading silver_prices")
	}
	if got := *item.CurrentPricePerGram; got < 0.9249 || got > 0.9251 {
		t.Errorf("price per gram = %v, want ~0.925", got)
	}
	if item.CurrentValue == nil {
		t.Fatal("silver item has no current value")
	}
	if got := *item.CurrentValue; got < 92.49 || got > 92.51 {
		t.Errorf("current value = %v, want ~92.50", got)
	}
}

// The two metals are priced from separate tables on the same day, and
// each row must pick up its own.
func TestPortfolioValuesBothMetalsInOneSummary(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()

	if _, err := repo.CreatePrice(ctx, model.GoldPrice{PriceDate: "2026-08-01", PricePerGram24k: 24}); err != nil {
		t.Fatalf("create gold price: %v", err)
	}
	if _, err := repo.CreateSilverPrice(ctx, model.SilverPrice{PriceDate: "2026-08-01", PricePerGram999: 0.999}); err != nil {
		t.Fatalf("create silver price: %v", err)
	}

	karat, fineness := 24.0, 999.0
	if _, err := repo.CreateItem(ctx, model.GoldItem{
		PurchaseDate: "2026-07-01", ItemName: "Bar", MetalType: model.MetalGold,
		PurityKarat: &karat, WeightGrams: 10, PricePaidTotal: 200,
	}); err != nil {
		t.Fatalf("create gold item: %v", err)
	}
	if _, err := repo.CreateItem(ctx, model.GoldItem{
		PurchaseDate: "2026-07-02", ItemName: "Round", MetalType: model.MetalSilver,
		PurityFineness: &fineness, WeightGrams: 100, PricePaidTotal: 80,
	}); err != nil {
		t.Fatalf("create silver item: %v", err)
	}

	summary, err := repo.GetPortfolioSummary(ctx)
	if err != nil {
		t.Fatalf("portfolio: %v", err)
	}

	byMetal := map[string]model.PortfolioItem{}
	for _, item := range summary.Items {
		byMetal[item.MetalType] = item
	}
	if v := byMetal[model.MetalGold].CurrentValue; v == nil || *v != 240 {
		t.Errorf("gold value = %v, want 240", v)
	}
	if v := byMetal[model.MetalSilver].CurrentValue; v == nil || *v < 99.8 || *v > 99.91 {
		t.Errorf("silver value = %v, want ~99.90", v)
	}
	// 280 paid in, ~339.90 back out — the totals must span both metals.
	if summary.Totals.TotalPaid != 280 {
		t.Errorf("total paid = %v, want 280", summary.Totals.TotalPaid)
	}
	if summary.Totals.TotalValue < 339.8 || summary.Totals.TotalValue > 339.95 {
		t.Errorf("total value = %v, want ~339.90", summary.Totals.TotalValue)
	}
}

// An item with no price behind it reports null rather than zero, so the
// UI can say "no price yet" instead of "worth nothing".
func TestPortfolioLeavesValueNullWithoutAPrice(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()

	karat := 21.0
	if _, err := repo.CreateItem(ctx, model.GoldItem{
		PurchaseDate: "2026-07-01", ItemName: "Chain", MetalType: model.MetalGold,
		PurityKarat: &karat, WeightGrams: 10, PricePaidTotal: 200,
	}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	summary, err := repo.GetPortfolioSummary(ctx)
	if err != nil {
		t.Fatalf("portfolio: %v", err)
	}
	if summary.Items[0].CurrentValue != nil {
		t.Errorf("current value = %v, want nil with no price recorded", *summary.Items[0].CurrentValue)
	}
	if summary.HasPriceData {
		t.Error("HasPriceData = true, want false with no price recorded")
	}
}

// The CHECK constraint is the backstop behind the handler's validation.
func TestSchemaRejectsSilverWithoutFineness(t *testing.T) {
	repo := repoFor(t)
	karat := 21.0

	_, err := repo.CreateItem(context.Background(), model.GoldItem{
		PurchaseDate: "2026-07-01", ItemName: "Tray", MetalType: model.MetalSilver,
		PurityKarat: &karat, WeightGrams: 100, PricePaidTotal: 80,
	})
	if err == nil {
		t.Fatal("expected the CHECK constraint to reject silver carrying only a karat")
	}
}

func TestSchemaRejectsAnUnknownMetal(t *testing.T) {
	repo := repoFor(t)
	karat := 21.0

	_, err := repo.CreateItem(context.Background(), model.GoldItem{
		PurchaseDate: "2026-07-01", ItemName: "Ring", MetalType: "platinum",
		PurityKarat: &karat, WeightGrams: 10, PricePaidTotal: 200,
	})
	if err == nil {
		t.Fatal("expected the CHECK constraint to reject an unknown metal")
	}
}

func TestSilverPricesDeriveTheLowerFineness(t *testing.T) {
	repo := repoFor(t)

	price, err := repo.CreateSilverPrice(context.Background(), model.SilverPrice{
		PriceDate: "2026-08-01", PricePerGram999: 0.999, Source: "test",
	})
	if err != nil {
		t.Fatalf("create silver price: %v", err)
	}
	if price.PricePerGram925 < 0.9249 || price.PricePerGram925 > 0.9251 {
		t.Errorf("925 rate = %v, want ~0.925", price.PricePerGram925)
	}
	if price.PricePerGram900 < 0.8999 || price.PricePerGram900 > 0.9001 {
		t.Errorf("900 rate = %v, want ~0.900", price.PricePerGram900)
	}
}

// Signals are recorded per metal, so the panel can show gold and silver
// disagreeing.
func TestSignalsRecordTheirMetal(t *testing.T) {
	repo := repoFor(t)
	ctx := context.Background()
	reasoning := "test"

	for _, metal := range []string{model.MetalGold, model.MetalSilver} {
		if _, err := repo.CreateSignal(ctx, model.SignalLog{
			Metal: metal, SignalType: "HOLD", Reasoning: &reasoning, Source: "manual",
		}); err != nil {
			t.Fatalf("create %s signal: %v", metal, err)
		}
	}

	signals, err := repo.GetSignals(ctx, 10)
	if err != nil {
		t.Fatalf("get signals: %v", err)
	}
	if len(signals) != 2 {
		t.Fatalf("got %d signals, want 2", len(signals))
	}
	seen := map[string]bool{}
	for _, s := range signals {
		seen[s.Metal] = true
	}
	if !seen[model.MetalGold] || !seen[model.MetalSilver] {
		t.Errorf("signals cover %v, want both metals", seen)
	}
}
