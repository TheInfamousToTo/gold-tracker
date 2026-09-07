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
	GetSilverPrices(ctx context.Context, limit int) ([]model.SilverPrice, error)
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
	metals := input.metalNames()
	verdicts, err := s.runAndParse(ctx, prompt, metals)
	if err != nil {
		verdicts, err = s.runAndParse(ctx, prompt+retryInstruction, metals)
	}
	if err != nil {
		s.finish(err.Error(), false)
		return
	}

	// Written in prompt order rather than map order so the panel lists
	// the metals the same way on every run.
	for _, m := range input.Metals {
		verdict, ok := verdicts[m.Metal]
		if !ok {
			continue
		}
		signal := model.SignalLog{
			Metal:         m.Metal,
			SignalType:    verdict.Signal,
			Reasoning:     &verdict.Reasoning,
			PriceAtSignal: latestPrice(m.Prices),
			Model:         &s.cfg.Model,
			Source:        source,
		}
		if _, err := s.repo.CreateSignal(ctx, signal); err != nil {
			s.finish("could not save the signal: "+err.Error(), false)
			return
		}
	}

	s.finish("", true)
}

func (s *Service) runAndParse(ctx context.Context, prompt string, metals []string) (map[string]Verdict, error) {
	result, err := s.runner.Run(ctx, prompt, s.cfg.Model)
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, fmt.Errorf("claude reported an error: %s", result.Result)
	}
	return ParseVerdicts(result.Result, metals)
}

// latestPrice returns the newest observation, which gather has placed
// last. Nil when there is no price history at all.
func latestPrice(prices []PriceHistoryPoint) *float64 {
	if len(prices) == 0 {
		return nil
	}
	p := prices[len(prices)-1].PricePerGram
	return &p
}

// gather reads the data the prompt needs and reduces holdings to
// per-purity aggregates, so no per-item free text leaves the database.
//
// A metal with neither prices nor holdings is left out entirely: an
// owner who has never touched silver gets the same single-metal prompt
// and single-metal response schema as before.
func (s *Service) gather(ctx context.Context) (PromptInput, error) {
	goldPrices, err := s.repo.GetPrices(ctx, priceHistoryLimit)
	if err != nil {
		return PromptInput{}, err
	}
	silverPrices, err := s.repo.GetSilverPrices(ctx, priceHistoryLimit)
	if err != nil {
		return PromptInput{}, err
	}
	portfolio, err := s.repo.GetPortfolioSummary(ctx)
	if err != nil {
		return PromptInput{}, err
	}

	goldPoints := make([]PriceHistoryPoint, 0, len(goldPrices))
	for i := len(goldPrices) - 1; i >= 0; i-- {
		goldPoints = append(goldPoints, PriceHistoryPoint{
			Date:         goldPrices[i].PriceDate,
			PricePerGram: goldPrices[i].PricePerGram24k,
		})
	}
	silverPoints := make([]PriceHistoryPoint, 0, len(silverPrices))
	for i := len(silverPrices) - 1; i >= 0; i-- {
		silverPoints = append(silverPoints, PriceHistoryPoint{
			Date:         silverPrices[i].PriceDate,
			PricePerGram: silverPrices[i].PricePerGram999,
		})
	}

	holdings := aggregateHoldings(portfolio.Items)

	metals := []MetalData{{
		Metal:    model.MetalGold,
		Prices:   goldPoints,
		Holdings: holdings[model.MetalGold],
	}}
	if len(silverPoints) > 0 || len(holdings[model.MetalSilver]) > 0 {
		metals = append(metals, MetalData{
			Metal:    model.MetalSilver,
			Prices:   silverPoints,
			Holdings: holdings[model.MetalSilver],
		})
	}

	return PromptInput{
		Metals:           metals,
		TotalPaid:        portfolio.Totals.TotalPaid,
		TotalValue:       portfolio.Totals.TotalValue,
		TotalGainLossPct: portfolio.Totals.TotalGainLossPct,
	}, nil
}

// aggregateHoldings groups items by metal and purity, preserving the
// order each purity was first seen so the prompt is stable across runs.
func aggregateHoldings(items []model.PortfolioItem) map[string][]HoldingsAggregate {
	type key struct {
		metal  string
		purity string
	}
	byKey := map[key]*HoldingsAggregate{}
	order := map[string][]key{}

	for _, item := range items {
		metal := item.MetalType
		if metal == "" {
			metal = model.MetalGold
		}
		k := key{metal: metal, purity: purityLabel(metal, item)}

		agg, ok := byKey[k]
		if !ok {
			agg = &HoldingsAggregate{PurityLabel: k.purity}
			byKey[k] = agg
			order[metal] = append(order[metal], k)
		}
		agg.TotalWeightGrams += item.WeightGrams
		agg.TotalPaid += item.PricePaidTotal
	}

	out := map[string][]HoldingsAggregate{}
	for metal, keys := range order {
		for _, k := range keys {
			agg := byKey[k]
			if agg.TotalWeightGrams > 0 {
				agg.AvgPricePerGram = agg.TotalPaid / agg.TotalWeightGrams
			}
			out[metal] = append(out[metal], *agg)
		}
	}
	return out
}

// purityLabel writes a purity the way its metal is traded: gold in
// karat, silver in millesimal fineness, which is the only notation
// silver has.
func purityLabel(metal string, item model.PortfolioItem) string {
	if metal == model.MetalSilver {
		if item.PurityFineness == nil {
			return "unmarked"
		}
		return fmt.Sprintf("%.0f", *item.PurityFineness)
	}
	if item.PurityKarat == nil {
		return "unmarked"
	}
	return fmt.Sprintf("%.0fK", *item.PurityKarat)
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
