package ai

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

type fakeRepo struct {
	mu          sync.Mutex
	prices      []model.GoldPrice
	portfolio   model.PortfolioSummary
	created     []model.SignalLog
	latestBySrc map[string]*model.SignalLog
	createErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{latestBySrc: map[string]*model.SignalLog{}}
}

func (f *fakeRepo) GetPrices(ctx context.Context, limit int) ([]model.GoldPrice, error) {
	return f.prices, nil
}

func (f *fakeRepo) GetPortfolioSummary(ctx context.Context) (model.PortfolioSummary, error) {
	return f.portfolio, nil
}

func (f *fakeRepo) CreateSignal(ctx context.Context, s model.SignalLog) (model.SignalLog, error) {
	if f.createErr != nil {
		return model.SignalLog{}, f.createErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s.ID = len(f.created) + 1
	s.SignalDate = time.Now()
	f.created = append(f.created, s)
	cp := s
	f.latestBySrc[s.Source] = &cp
	return s, nil
}

func (f *fakeRepo) GetLatestSignal(ctx context.Context, source string) (*model.SignalLog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.latestBySrc[source], nil
}

func (f *fakeRepo) createdCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.created)
}

type fakeRunner struct {
	mu    sync.Mutex
	calls int
	fn    func(call int) (RunResult, error)
}

func (f *fakeRunner) Run(ctx context.Context, prompt string, model string) (RunResult, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.mu.Unlock()
	return f.fn(call)
}

func (f *fakeRunner) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func testConfig() Config {
	return Config{Enabled: true, Model: "claude-opus-5", Timeout: 5 * time.Second, AutoMinHours: 24}
}

const goodVerdict = `{"signal":"BUY","confidence":0.6,"reasoning":"test","horizon_days":30,"key_factors":[]}`

func waitUntilIdle(t *testing.T, svc *Service) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if !svc.GetStatus().Running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("service never returned to idle")
}

func TestRunOnceSuccessPersistsSignal(t *testing.T) {
	repo := newFakeRepo()
	repo.prices = []model.GoldPrice{
		{PriceDate: "2026-08-02", PricePerGram24k: 46},
		{PriceDate: "2026-08-01", PricePerGram24k: 45},
	}
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	if !svc.TryStart("manual") {
		t.Fatal("TryStart should succeed when nothing is running")
	}
	svc.RunOnce(context.Background(), "manual")

	st := svc.GetStatus()
	if st.Running {
		t.Error("Running should be false once RunOnce returns")
	}
	if st.LastError != "" {
		t.Errorf("LastError = %q, want empty", st.LastError)
	}
	if st.LastGeneratedAt == nil {
		t.Fatal("LastGeneratedAt should be set")
	}
	if !st.Enabled {
		t.Error("Enabled should mirror the configured value")
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 persisted signal, got %d", len(repo.created))
	}

	got := repo.created[0]
	if got.SignalType != "BUY" {
		t.Errorf("SignalType = %q, want BUY", got.SignalType)
	}
	if got.Source != "manual" {
		t.Errorf("Source = %q, want manual", got.Source)
	}
	if got.Model == nil || *got.Model != "claude-opus-5" {
		t.Errorf("Model = %v, want claude-opus-5", got.Model)
	}
	// GetPrices returns newest first; the signal should record the
	// newest price, not the oldest.
	if got.PriceAtSignal == nil || *got.PriceAtSignal != 46 {
		t.Errorf("PriceAtSignal = %v, want 46 (the most recent price)", got.PriceAtSignal)
	}
}

func TestRunOnceRetriesOnceThenSucceeds(t *testing.T) {
	repo := newFakeRepo()
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		if call == 1 {
			return RunResult{Result: "not json"}, nil
		}
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.TryStart("manual")
	svc.RunOnce(context.Background(), "manual")

	if runner.callCount() != 2 {
		t.Fatalf("expected exactly 2 runner calls (one retry), got %d", runner.callCount())
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 persisted signal after the retry succeeded, got %d", len(repo.created))
	}
	if st := svc.GetStatus(); st.LastError != "" {
		t.Errorf("LastError = %q, want empty after a successful retry", st.LastError)
	}
}

func TestRunOnceFailsTwicePersistsNothing(t *testing.T) {
	repo := newFakeRepo()
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: "still not json"}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.TryStart("manual")
	svc.RunOnce(context.Background(), "manual")

	if runner.callCount() != 2 {
		t.Fatalf("expected exactly 2 runner calls and no more, got %d", runner.callCount())
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected nothing persisted, got %d signals", len(repo.created))
	}
	if st := svc.GetStatus(); st.LastError == "" {
		t.Error("LastError should be set after two failed parses")
	}
}

func TestRunOnceReportsCLIError(t *testing.T) {
	repo := newFakeRepo()
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{IsError: true, Result: "credit balance too low"}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.TryStart("manual")
	svc.RunOnce(context.Background(), "manual")

	st := svc.GetStatus()
	if st.LastError == "" {
		t.Fatal("LastError should be set when the CLI reports an error")
	}
	if repo.createdCount() != 0 {
		t.Error("nothing should be persisted when the CLI reports an error")
	}
}

func TestRunOnceReportsTransportError(t *testing.T) {
	repo := newFakeRepo()
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{}, errors.New("claude CLI timed out after 3m0s")
	}}
	svc := NewService(repo, runner, testConfig())

	svc.TryStart("manual")
	svc.RunOnce(context.Background(), "manual")

	if st := svc.GetStatus(); st.LastError == "" {
		t.Fatal("LastError should be set on a transport failure")
	}
}

func TestRunOnceReportsPersistFailure(t *testing.T) {
	repo := newFakeRepo()
	repo.createErr = errors.New("connection refused")
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.TryStart("manual")
	svc.RunOnce(context.Background(), "manual")

	st := svc.GetStatus()
	if st.LastError == "" {
		t.Fatal("LastError should be set when persisting fails")
	}
	if st.LastGeneratedAt != nil {
		t.Error("LastGeneratedAt should stay unset when the signal was never stored")
	}
}

func TestTryStartBlocksConcurrentRun(t *testing.T) {
	repo := newFakeRepo()
	release := make(chan struct{})
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		<-release
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	if !svc.TryStart("manual") {
		t.Fatal("first TryStart should succeed")
	}
	go svc.RunOnce(context.Background(), "manual")

	if svc.TryStart("manual") {
		t.Fatal("second TryStart should fail while a run is in flight")
	}

	close(release)
	waitUntilIdle(t, svc)

	if svc.callCountForTest() != 1 {
		t.Errorf("expected the blocked caller not to have started a second run")
	}
}

func (s *Service) callCountForTest() int {
	if r, ok := s.runner.(*fakeRunner); ok {
		return r.callCount()
	}
	return -1
}

func TestMaybeAutoGenerateSkipsWhenRecentAutoSignalExists(t *testing.T) {
	repo := newFakeRepo()
	repo.latestBySrc["auto"] = &model.SignalLog{SignalDate: time.Now().Add(-1 * time.Hour), Source: "auto"}
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.MaybeAutoGenerate(context.Background())
	time.Sleep(50 * time.Millisecond)

	if runner.callCount() != 0 {
		t.Fatalf("runner should not run while the cap is unexpired, got %d calls", runner.callCount())
	}
}

func TestMaybeAutoGenerateRunsWhenCapExpired(t *testing.T) {
	repo := newFakeRepo()
	repo.latestBySrc["auto"] = &model.SignalLog{SignalDate: time.Now().Add(-25 * time.Hour), Source: "auto"}
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.MaybeAutoGenerate(context.Background())
	waitUntilIdle(t, svc)

	if repo.createdCount() != 1 {
		t.Fatalf("expected 1 auto signal once the cap expired, got %d", repo.createdCount())
	}
	if repo.created[0].Source != "auto" {
		t.Errorf("Source = %q, want auto", repo.created[0].Source)
	}
}

func TestMaybeAutoGenerateRunsWhenNoAutoSignalExists(t *testing.T) {
	repo := newFakeRepo()
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.MaybeAutoGenerate(context.Background())
	waitUntilIdle(t, svc)

	if repo.createdCount() != 1 {
		t.Fatalf("expected the first auto signal to generate, got %d", repo.createdCount())
	}
}

func TestMaybeAutoGenerateDoesNothingWhenDisabled(t *testing.T) {
	repo := newFakeRepo()
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		return RunResult{Result: goodVerdict}, nil
	}}
	cfg := testConfig()
	cfg.Enabled = false
	svc := NewService(repo, runner, cfg)

	svc.MaybeAutoGenerate(context.Background())
	time.Sleep(50 * time.Millisecond)

	if runner.callCount() != 0 {
		t.Fatalf("runner should never run while AI is disabled, got %d calls", runner.callCount())
	}
	if svc.Enabled() {
		t.Error("Enabled() should report false")
	}
}

// An auto trigger must not preempt or duplicate a manual run already
// in flight — the subscription's rate limit is shared with the owner's
// interactive use.
func TestMaybeAutoGenerateSkipsWhileManualRunInFlight(t *testing.T) {
	repo := newFakeRepo()
	release := make(chan struct{})
	runner := &fakeRunner{fn: func(call int) (RunResult, error) {
		<-release
		return RunResult{Result: goodVerdict}, nil
	}}
	svc := NewService(repo, runner, testConfig())

	svc.TryStart("manual")
	go svc.RunOnce(context.Background(), "manual")

	svc.MaybeAutoGenerate(context.Background())

	close(release)
	waitUntilIdle(t, svc)

	if runner.callCount() != 1 {
		t.Fatalf("expected only the manual run, got %d runner calls", runner.callCount())
	}
}

func TestGatherAggregatesHoldingsByKarat(t *testing.T) {
	repo := newFakeRepo()
	repo.portfolio = model.PortfolioSummary{
		Items: []model.PortfolioItem{
			{PurityKarat: 21, WeightGrams: 10, PricePaidTotal: 400},
			{PurityKarat: 21, WeightGrams: 30, PricePaidTotal: 1200},
			{PurityKarat: 24, WeightGrams: 5, PricePaidTotal: 250},
		},
	}
	svc := NewService(repo, &fakeRunner{}, testConfig())

	in, err := svc.gather(context.Background())
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if len(in.Holdings) != 2 {
		t.Fatalf("expected 2 karat groups, got %d", len(in.Holdings))
	}

	byKarat := map[float64]HoldingsAggregate{}
	for _, h := range in.Holdings {
		byKarat[h.Karat] = h
	}
	k21 := byKarat[21]
	if k21.TotalWeightGrams != 40 || k21.TotalPaid != 1600 {
		t.Errorf("21K = %+v, want 40g / 1600 paid", k21)
	}
	if k21.AvgPricePerGram != 40 {
		t.Errorf("21K average = %v, want 40", k21.AvgPricePerGram)
	}
}

func TestGatherOrdersPricesOldestFirst(t *testing.T) {
	repo := newFakeRepo()
	// The repository returns newest first.
	repo.prices = []model.GoldPrice{
		{PriceDate: "2026-08-03", PricePerGram24k: 47},
		{PriceDate: "2026-08-02", PricePerGram24k: 46},
		{PriceDate: "2026-08-01", PricePerGram24k: 45},
	}
	svc := NewService(repo, &fakeRunner{}, testConfig())

	in, err := svc.gather(context.Background())
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if in.Prices[0].Date != "2026-08-01" {
		t.Errorf("first price = %s, want the oldest (2026-08-01)", in.Prices[0].Date)
	}
	if in.Prices[len(in.Prices)-1].Date != "2026-08-03" {
		t.Errorf("last price = %s, want the newest (2026-08-03)", in.Prices[len(in.Prices)-1].Date)
	}
}
