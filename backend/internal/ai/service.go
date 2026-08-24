package ai

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// priceHistoryLimit is how many price rows the model is shown.
const priceHistoryLimit = 90

// retryInstruction is appended when the first response fails schema
// validation.
const retryInstruction = "\n\nYour previous response could not be parsed as the exact JSON schema requested. Reply with only the JSON object and no other text."

// Status is what GET /api/signals/status returns.
type Status struct {
	Running         bool       `json:"running"`
	StartedAt       *time.Time `json:"started_at"`
	LastError       string     `json:"last_error"`
	LastGeneratedAt *time.Time `json:"last_generated_at"`
	Enabled         bool       `json:"enabled"`
}

// SignalRepo is the slice of repository behavior the service needs.
// *repository.PostgresRepository satisfies it structurally.
type SignalRepo interface {
	GetPrices(ctx context.Context, limit int) ([]model.GoldPrice, error)
	GetPortfolioSummary(ctx context.Context) (model.PortfolioSummary, error)
	CreateSignal(ctx context.Context, s model.SignalLog) (model.SignalLog, error)
	GetLatestSignal(ctx context.Context, source string) (*model.SignalLog, error)
}

// ErrAlreadyRunning means a generation is in flight; ErrCoolingDown
// means the manual cooldown has not elapsed. Callers map these to 409
// and 429 respectively.
var (
	ErrAlreadyRunning = errors.New("a signal generation is already running")
	ErrCoolingDown    = errors.New("please wait before generating another signal")
)

type Service struct {
	repo   SignalRepo
	runner Runner
	cfg    Config

	mu           sync.Mutex
	status       Status
	lastManualAt time.Time
}

func NewService(repo SignalRepo, runner Runner, cfg Config) *Service {
	return &Service{repo: repo, runner: runner, cfg: cfg}
}

func (s *Service) Enabled() bool {
	return s.cfg.Enabled
}

func (s *Service) GetStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	st.Enabled = s.cfg.Enabled
	return st
}

// TryStart atomically claims the single-flight slot, returning nil when
// the caller may proceed to RunOnce. It has no side effects on refusal.
//
// Manual starts additionally honour a cooldown. The API has no
// authentication, so without one anything that can reach it could spend
// subscription quota shared with the owner's interactive Claude Code
// use. Automatic starts skip it: they already have the daily cap.
func (s *Service) TryStart(source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status.Running {
		return ErrAlreadyRunning
	}
	now := time.Now()
	if source == "manual" && !s.lastManualAt.IsZero() && now.Sub(s.lastManualAt) < s.cfg.ManualCooldown {
		return ErrCoolingDown
	}
	if source == "manual" {
		s.lastManualAt = now
	}

	s.status.Running = true
	s.status.StartedAt = &now
	s.status.LastError = ""
	return nil
}

func (s *Service) finish(errMsg string, generated bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Running = false
	s.status.LastError = errMsg
	if generated {
		now := time.Now()
		s.status.LastGeneratedAt = &now
	}
}

// RunOnce performs one generate cycle: gather, prompt, run, parse with
// a single retry, persist. It releases the single-flight slot on every
// exit path.
func (s *Service) RunOnce(ctx context.Context, source string) {
	input, err := s.gather(ctx)
	if err != nil {
		s.finish("could not read portfolio data: "+err.Error(), false)
		return
	}

	prompt := BuildPrompt(input)
	verdict, err := s.runAndParse(ctx, prompt)
	if err != nil {
		verdict, err = s.runAndParse(ctx, prompt+retryInstruction)
	}
	if err != nil {
		s.finish(err.Error(), false)
		return
	}

	signal := model.SignalLog{
		SignalType:    verdict.Signal,
		Reasoning:     &verdict.Reasoning,
		PriceAtSignal: latestPrice(input.Prices),
		Model:         &s.cfg.Model,
		Source:        source,
	}
	if _, err := s.repo.CreateSignal(ctx, signal); err != nil {
		s.finish("could not save the signal: "+err.Error(), false)
		return
	}

	s.finish("", true)
}

func (s *Service) runAndParse(ctx context.Context, prompt string) (Verdict, error) {
	result, err := s.runner.Run(ctx, prompt, s.cfg.Model)
	if err != nil {
		return Verdict{}, err
	}
	if result.IsError {
		return Verdict{}, fmt.Errorf("claude reported an error: %s", result.Result)
	}
	return ParseVerdict(result.Result)
}

// latestPrice returns the newest observation, which gather has placed
// last. Nil when there is no price history at all.
func latestPrice(prices []PriceHistoryPoint) *float64 {
	if len(prices) == 0 {
		return nil
	}
	p := prices[len(prices)-1].PricePerGram24k
	return &p
}

// gather reads the data the prompt needs and reduces holdings to
// per-karat aggregates, so no per-item free text leaves the database.
func (s *Service) gather(ctx context.Context) (PromptInput, error) {
	prices, err := s.repo.GetPrices(ctx, priceHistoryLimit)
	if err != nil {
		return PromptInput{}, err
	}
	portfolio, err := s.repo.GetPortfolioSummary(ctx)
	if err != nil {
		return PromptInput{}, err
	}

	// GetPrices returns newest first; the model reads a series better
	// oldest first.
	points := make([]PriceHistoryPoint, 0, len(prices))
	for i := len(prices) - 1; i >= 0; i-- {
		points = append(points, PriceHistoryPoint{
			Date:            prices[i].PriceDate,
			PricePerGram24k: prices[i].PricePerGram24k,
		})
	}

	byKarat := map[float64]*HoldingsAggregate{}
	karats := []float64{}
	for _, item := range portfolio.Items {
		agg, ok := byKarat[item.PurityKarat]
		if !ok {
			agg = &HoldingsAggregate{Karat: item.PurityKarat}
			byKarat[item.PurityKarat] = agg
			karats = append(karats, item.PurityKarat)
		}
		agg.TotalWeightGrams += item.WeightGrams
		agg.TotalPaid += item.PricePaidTotal
	}

	holdings := make([]HoldingsAggregate, 0, len(karats))
	for _, k := range karats {
		agg := byKarat[k]
		if agg.TotalWeightGrams > 0 {
			agg.AvgPricePerGram = agg.TotalPaid / agg.TotalWeightGrams
		}
		holdings = append(holdings, *agg)
	}

	return PromptInput{
		Prices:           points,
		Holdings:         holdings,
		TotalPaid:        portfolio.Totals.TotalPaid,
		TotalValue:       portfolio.Totals.TotalValue,
		TotalGainLossPct: portfolio.Totals.TotalGainLossPct,
	}, nil
}

// MaybeAutoGenerate starts a background run if AI is enabled, nothing
// is already running, and the newest auto signal is older than the
// configured cap. It never blocks and never returns an error: it is
// called from the price-write path, which must not fail because of AI.
func (s *Service) MaybeAutoGenerate(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}
	latest, err := s.repo.GetLatestSignal(ctx, "auto")
	if err != nil {
		return
	}
	if latest != nil && time.Since(latest.SignalDate).Hours() < s.cfg.AutoMinHours {
		return
	}
	if err := s.TryStart("auto"); err != nil {
		return
	}
	// Detached from the request context so the run survives the HTTP
	// response that triggered it.
	go s.RunOnce(context.Background(), "auto")
}
