package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// priceHistoryLimit is how many daily rows are read per metal: the whole
// ten-year history, which the indicators need for their long windows.
const priceHistoryLimit = 3700

// retryInstruction is appended when the first response fails schema
// validation.
const retryInstruction = "\n\nYour previous response could not be parsed as the exact JSON schema requested. Reply with only the JSON object and no other text."

// Status is what GET /api/signals/status returns.
type Status struct {
	Running         bool       `json:"running"`
	Kind            string     `json:"kind"`
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
	GetAdvisorSettings(ctx context.Context) (model.AdvisorSettings, error)
	CreateSignal(ctx context.Context, s model.SignalLog) (model.SignalLog, error)
	GetLatestSignal(ctx context.Context, source string) (*model.SignalLog, error)
}

// ErrAlreadyRunning means a generation is in flight; ErrCoolingDown
// means the manual cooldown has not elapsed. Callers map these to 409
// and 429 respectively.
var (
	ErrAlreadyRunning = errors.New("a signal generation is already running")
	ErrCoolingDown    = errors.New("please wait before generating another signal")
	ErrEmptyPlan      = errors.New("describe the plan to review")
)

type Service struct {
	repo   SignalRepo
	runner Runner
	cfg    Config
	now    func() time.Time

	mu           sync.Mutex
	status       Status
	lastManualAt time.Time
}

func NewService(repo SignalRepo, runner Runner, cfg Config) *Service {
	return &Service{repo: repo, runner: runner, cfg: cfg, now: time.Now}
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
// the caller may proceed. It has no side effects on refusal.
//
// Manual starts — Analyse and Review alike — additionally honour a
// cooldown, because both spend subscription quota shared with the
// owner's interactive Claude Code use. Automatic starts skip it: they
// already have the daily cap.
func (s *Service) TryStart(source string) error {
	return s.tryStart(source, model.SignalKindAnalysis)
}

func (s *Service) tryStart(source, kind string) error {
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
	s.status.Kind = kind
	s.status.StartedAt = &now
	s.status.LastError = ""
	return nil
}

// StartReview claims the slot for a plan review.
func (s *Service) StartReview(plan string) error {
	if strings.TrimSpace(plan) == "" {
		return ErrEmptyPlan
	}
	return s.tryStart("manual", model.SignalKindReview)
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

// RunOnce performs one analysis: gather, compute, prompt, run, parse,
// clamp, persist. It releases the single-flight slot on every exit path.
func (s *Service) RunOnce(ctx context.Context, source string) {
	in, err := s.gather(ctx)
	if err != nil {
		s.finish("could not read portfolio data: "+err.Error(), false)
		return
	}
	metals := in.metalNames()

	var advice map[string]Advice
	newsChecked, err := s.runWithFallback(ctx, &in, BuildAnalysisPrompt, func(raw string) error {
		var perr error
		advice, perr = ParseAnalysis(raw, metals)
		return perr
	})
	if err != nil {
		s.finish(err.Error(), false)
		return
	}

	clamped := make(map[string]*Clamped, len(in.Metals))
	latest := map[string]float64{}
	for _, m := range in.Metals {
		c := clampAdvice(advice[m.Metal], m, in.Settings, in.Budget)
		clamped[m.Metal] = &c
		latest[m.Metal] = m.Indicators.Latest
	}
	capTotalBuys(clamped, in.Budget, latest, in.Settings)

	// Written in prompt order rather than map order so the panel lists
	// the metals the same way on every run.
	for _, m := range in.Metals {
		c := clamped[m.Metal]
		sig := s.signalFor(m, c.Advice, source, newsChecked)
		sig.SignalType = c.Action
		sig.AmountBHD = optional(c.AmountBHD)
		sig.AmountGrams = optional(c.AmountGrams)
		sig.Deviates = c.Deviates
		if c.HorizonDays > 0 {
			h := c.HorizonDays
			sig.HorizonDays = &h
		}
		if _, err := s.repo.CreateSignal(ctx, sig); err != nil {
			s.finish("could not save the signal: "+err.Error(), false)
			return
		}
	}
	s.finish("", true)
}

// RunReview judges a plan the owner typed and saves the answer.
func (s *Service) RunReview(ctx context.Context, plan string) {
	in, err := s.gather(ctx)
	if err != nil {
		s.finish("could not read portfolio data: "+err.Error(), false)
		return
	}
	plan = sanitizePlan(plan)

	var review Review
	newsChecked, err := s.runWithFallback(ctx, &in, func(in AdvisorInput) string {
		return BuildReviewPrompt(in, plan)
	}, func(raw string) error {
		var perr error
		review, perr = ParseReview(raw)
		return perr
	})
	if err != nil {
		s.finish(err.Error(), false)
		return
	}

	sig := model.SignalLog{
		Kind:        model.SignalKindReview,
		SignalType:  review.Verdict,
		Reasoning:   &review.Reasoning,
		Confidence:  &review.Confidence,
		KeyFactors:  review.KeyFactors,
		News:        review.News,
		NewsChecked: newsChecked,
		Model:       &s.cfg.Model,
		Source:      "manual",
		PlanText:    &plan,
	}
	if sg := review.Suggested; sg != nil {
		for _, m := range in.Metals {
			if m.Metal != sg.Metal {
				continue
			}
			c := clampAdvice(Advice{Action: sg.Action, AmountBHD: sg.AmountBHD}, m, in.Settings, in.Budget)
			sig.Metal = m.Metal
			sig.SuggestedAction = &c.Action
			sig.AmountBHD = optional(c.AmountBHD)
			sig.AmountGrams = optional(c.AmountGrams)
			sig.PriceAtSignal = optional(m.Indicators.Latest)
			sig.Levels = levelsOf(m.Baseline)
			sig.Deviates = c.Deviates
			if raw, err := json.Marshal(m.Baseline); err == nil {
				sig.Baseline = raw
			}
		}
	}
	if _, err := s.repo.CreateSignal(ctx, sig); err != nil {
		s.finish("could not save the review: "+err.Error(), false)
		return
	}
	s.finish("", true)
}

func (s *Service) signalFor(m MetalState, a Advice, source string, newsChecked bool) model.SignalLog {
	sig := model.SignalLog{
		Kind:          model.SignalKindAnalysis,
		Metal:         m.Metal,
		Reasoning:     &a.Reasoning,
		Confidence:    &a.Confidence,
		KeyFactors:    a.KeyFactors,
		News:          a.News,
		NewsChecked:   newsChecked,
		PriceAtSignal: optional(m.Indicators.Latest),
		Levels:        levelsOf(m.Baseline),
		Model:         &s.cfg.Model,
		Source:        source,
	}
	if raw, err := json.Marshal(m.Baseline); err == nil {
		sig.Baseline = raw
	}
	return sig
}

func levelsOf(b Baseline) *model.Levels {
	if b.BuyMoreBelow == nil && b.CutLossBelow == nil {
		return nil
	}
	return &model.Levels{BuyMoreBelow: b.BuyMoreBelow, CutLossBelow: b.CutLossBelow}
}

func optional(v float64) *float64 {
	if v == 0 || math.IsNaN(v) {
		return nil
	}
	return &v
}

// runWithFallback runs the prompt with news tools when they are enabled,
// and once more without them if that run fails outright: a verdict on
// the data alone beats no verdict. A response that fails to parse gets
// one retry with the same tools. It reports whether news was checked.
func (s *Service) runWithFallback(ctx context.Context, in *AdvisorInput, build func(AdvisorInput) string, parse func(string) error) (bool, error) {
	attempt := func(tools []string) error {
		prompt := build(*in)
		raw, err := s.run(ctx, prompt, tools)
		if err != nil {
			return err
		}
		if perr := parse(raw); perr == nil {
			return nil
		}
		raw, err = s.run(ctx, prompt+retryInstruction, tools)
		if err != nil {
			return err
		}
		return parse(raw)
	}

	if in.News {
		err := attempt(NewsTools)
		if err == nil {
			return true, nil
		}
		if ctx.Err() != nil {
			return false, err
		}
		in.News = false
		if err2 := attempt(nil); err2 != nil {
			return false, fmt.Errorf("%v (and without news: %v)", err, err2)
		}
		return false, nil
	}
	return false, attempt(nil)
}

func (s *Service) run(ctx context.Context, prompt string, tools []string) (string, error) {
	result, err := s.runner.Run(ctx, prompt, s.cfg.Model, tools)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("claude reported an error: %s", result.Result)
	}
	return result.Result, nil
}

// gather reads everything the advisor needs and computes the figures
// and the baseline. A metal with neither prices nor holdings is left
// out: an owner who never touched silver gets a gold-only answer.
func (s *Service) gather(ctx context.Context) (AdvisorInput, error) {
	goldPrices, err := s.repo.GetPrices(ctx, priceHistoryLimit)
	if err != nil {
		return AdvisorInput{}, err
	}
	silverPrices, err := s.repo.GetSilverPrices(ctx, priceHistoryLimit)
	if err != nil {
		return AdvisorInput{}, err
	}
	portfolio, err := s.repo.GetPortfolioSummary(ctx)
	if err != nil {
		return AdvisorInput{}, err
	}
	settings, err := s.repo.GetAdvisorSettings(ctx)
	if err != nil {
		return AdvisorInput{}, err
	}

	gold := make([]PriceHistoryPoint, 0, len(goldPrices))
	for _, p := range goldPrices {
		gold = append(gold, PriceHistoryPoint{Date: p.PriceDate, PricePerGram: p.PricePerGram24k})
	}
	silver := make([]PriceHistoryPoint, 0, len(silverPrices))
	for _, p := range silverPrices {
		silver = append(silver, PriceHistoryPoint{Date: p.PriceDate, PricePerGram: p.PricePerGram999})
	}
	gold, silver = sortedOldestFirst(gold), sortedOldestFirst(silver)

	positions := computePositions(portfolio.Items, settings)
	states := []MetalState{{
		Metal: model.MetalGold, Prices: gold,
		Indicators: computeIndicators(gold), Position: positions[model.MetalGold],
	}}
	if len(silver) > 0 || positions[model.MetalSilver].Held {
		states = append(states, MetalState{
			Metal: model.MetalSilver, Prices: silver,
			Indicators: computeIndicators(silver), Position: positions[model.MetalSilver],
		})
	}

	now := s.now()
	spent := spentThisMonth(portfolio.Items, now.Format("2006-01"))
	budget := Budget{
		Monthly:   settings.MonthlyBudgetBHD,
		Spent:     spent,
		Remaining: math.Max(0, settings.MonthlyBudgetBHD-spent),
		Reserve:   settings.ReserveBHD,
	}
	ratio := computeRatio(gold, silver)
	computeBaselines(states, settings, budget, ratio)

	return AdvisorInput{
		Today:    now.Format("2006-01-02"),
		Metals:   states,
		Ratio:    ratio,
		Budget:   budget,
		Settings: settings,
		News:     settings.NewsEnabled,
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
