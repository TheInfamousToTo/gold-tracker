package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/ai"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// Repo is the slice of the repository the report needs.
type Repo interface {
	GetPortfolioSummary(ctx context.Context) (model.PortfolioSummary, error)
	GetPrices(ctx context.Context, limit int) ([]model.GoldPrice, error)
	GetSilverPrices(ctx context.Context, limit int) ([]model.SilverPrice, error)
	GetAdvisorSettings(ctx context.Context) (model.AdvisorSettings, error)
	ListStockPositions(ctx context.Context) ([]model.StockPosition, error)
	GetSignals(ctx context.Context, limit int) ([]model.SignalLog, error)
	ReportExists(ctx context.Context, month string) (bool, error)
	SaveReport(ctx context.Context, rep model.MonthlyReport) error
}

var ErrRunning = errors.New("a report is already being generated")

type Service struct {
	repo    Repo
	runner  ai.Runner
	aiCfg   ai.Config
	webhook string
	client  *http.Client

	mu        sync.Mutex
	running   bool
	lastError string
}

func NewService(repo Repo, runner ai.Runner, aiCfg ai.Config, webhook string) *Service {
	return &Service{repo: repo, runner: runner, aiCfg: aiCfg, webhook: webhook, client: &http.Client{Timeout: 20 * time.Second}}
}

type Status struct {
	Running   bool   `json:"running"`
	LastError string `json:"last_error"`
	Discord   bool   `json:"discord"`
	AI        bool   `json:"ai"`
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Status{Running: s.running, LastError: s.lastError, Discord: s.webhook != "", AI: s.aiCfg.Enabled}
}

// Start claims the single-flight slot and generates the month's report
// in the background.
func (s *Service) Start(month string) error {
	if _, _, err := monthBounds(month); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return ErrRunning
	}
	s.running = true
	s.lastError = ""
	go func() {
		err := s.Generate(context.Background(), month)
		s.mu.Lock()
		s.running = false
		if err != nil {
			s.lastError = err.Error()
		}
		s.mu.Unlock()
	}()
	return nil
}

// Generate builds, analyses, saves and posts one month's report. An AI
// or Discord failure is recorded on the report rather than losing it.
func (s *Service) Generate(ctx context.Context, month string) error {
	now := time.Now()
	in := Inputs{Month: month, Now: now}
	portfolio, err := s.repo.GetPortfolioSummary(ctx)
	if err != nil {
		return err
	}
	in.Items = portfolio.Items
	if in.Gold, err = s.repo.GetPrices(ctx, 3700); err != nil {
		return err
	}
	if in.Silver, err = s.repo.GetSilverPrices(ctx, 3700); err != nil {
		return err
	}
	if in.Settings, err = s.repo.GetAdvisorSettings(ctx); err != nil {
		return err
	}
	if in.Stocks, err = s.repo.ListStockPositions(ctx); err != nil {
		return err
	}
	data, err := Build(in)
	if err != nil {
		return err
	}

	rep := model.MonthlyReport{Month: month, Data: data}
	if s.aiCfg.Enabled {
		signals, _ := s.repo.GetSignals(ctx, 10)
		analysis, aerr := s.analyse(ctx, data, in.Settings, signals)
		if aerr != nil {
			msg := aerr.Error()
			rep.Error = &msg
		} else {
			rep.Analysis = &analysis
			rep.Model = &s.aiCfg.Model
		}
	}

	if s.webhook != "" {
		if derr := Send(ctx, s.client, s.webhook, Messages(rep)); derr != nil {
			msg := "discord: " + derr.Error()
			if rep.Error != nil {
				msg = *rep.Error + "; " + msg
			}
			rep.Error = &msg
		} else {
			sent := time.Now()
			rep.DiscordSentAt = &sent
		}
	}
	return s.repo.SaveReport(ctx, rep)
}

// analyse asks the model for the written review, with web search when
// news is enabled and once more without it if that run fails.
func (s *Service) analyse(ctx context.Context, d model.ReportData, settings model.AdvisorSettings, signals []model.SignalLog) (string, error) {
	tools := []string(nil)
	if settings.NewsEnabled {
		tools = ai.NewsTools
	}
	prompt := BuildPrompt(d, settings, signals, tools != nil)
	out, err := s.run(ctx, prompt, tools)
	if err != nil && tools != nil && ctx.Err() == nil {
		out, err = s.run(ctx, BuildPrompt(d, settings, signals, false), nil)
	}
	return out, err
}

func (s *Service) run(ctx context.Context, prompt string, tools []string) (string, error) {
	res, err := s.runner.Run(ctx, prompt, s.aiCfg.Model, tools)
	if err != nil {
		return "", err
	}
	if res.IsError {
		return "", fmt.Errorf("claude reported an error: %s", res.Result)
	}
	out := strings.TrimSpace(res.Result)
	if out == "" {
		return "", errors.New("claude returned an empty analysis")
	}
	return out, nil
}

// Schedule checks hourly and generates last month's report once it is
// missing — on the 1st, or on the first boot after a missed 1st.
func (s *Service) Schedule(ctx context.Context) {
	check := func() {
		month := PreviousMonth(time.Now())
		exists, err := s.repo.ReportExists(ctx, month)
		if err != nil || exists {
			return
		}
		if err := s.Start(month); err == nil {
			log.Printf("monthly report: generating %s", month)
		}
	}
	check()
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			check()
		}
	}
}

// anonymised replaces item and asset names with neutral labels: item
// names, vendors and notes never leave the database (only figures do).
func anonymised(d model.ReportData) model.ReportData {
	label := func(items []model.ReportItem) []model.ReportItem {
		out := make([]model.ReportItem, len(items))
		for i, it := range items {
			it.Name = fmt.Sprintf("Item %d (%s %s)", it.ID, it.Purity, it.Metal)
			out[i] = it
		}
		return out
	}
	d.Items, d.Purchases = label(d.Items), label(d.Purchases)
	stocks := make([]model.StockPosition, len(d.Stocks))
	for i, p := range d.Stocks {
		p.AssetName = ""
		stocks[i] = p
	}
	d.Stocks = stocks
	return d
}

// BuildPrompt writes the analysis request. The figures go in as JSON so
// the model reads exact numbers rather than prose.
func BuildPrompt(d model.ReportData, settings model.AdvisorSettings, signals []model.SignalLog, news bool) string {
	var b strings.Builder
	b.WriteString(`You are a careful personal investment analyst writing a monthly review for one private investor in Bahrain who holds physical gold and silver (priced in BHD per gram) and possibly a few stocks/ETFs.

Write the review of month ` + d.Month + ` in Discord-flavoured Markdown (## headings, - bullets, **bold**; no tables, no code blocks). Keep it under 3,000 characters. Use only the figures below; never invent numbers.

Sections, in this order:
## Summary — two or three sentences: how the portfolio did this month and overall.
## Performance — which holdings drove the month, best and worst by gain, cost basis versus today's price.
## Market — what gold and silver did this month (open, close, range) and why it matters for this portfolio.
## What to do next — 2 to 4 concrete, numbered actions for next month (buy / hold / wait for a price / rebalance), each with an amount in BHD or grams that fits the monthly budget and keeps the reserve, and the price level that would change the call.
## Risks — the one or two things that would make you wrong.

Physical metal is bought and sold with a dealer spread (gold `)
	fmt.Fprintf(&b, "%.1f%%, silver %.1f%%), so short-term trading usually loses money; weigh that. Target allocation: %.0f%% gold. Stop-loss threshold: %.0f%%. Monthly budget %.2f BHD, reserve %.2f BHD.\n",
		settings.SpreadPctGold, settings.SpreadPctSilver, settings.TargetGoldPct, settings.StopLossPct, settings.MonthlyBudgetBHD, settings.ReserveBHD)
	if news {
		b.WriteString("\nSearch the web for this month's main gold and silver market drivers (central banks, US dollar, rates, geopolitics) and the outlook for next month. Mention the two or three that matter, each with its source name. Do not use any other tools.\n")
	} else {
		b.WriteString("\nYou have no tools; base the review on the data alone.\n")
	}

	raw, _ := json.MarshalIndent(anonymised(d), "", " ")
	b.WriteString("\nReport data (BHD; *_per_gram are BHD/g for that item's purity; month_gain is the value change during the month):\n")
	b.Write(raw)

	if len(signals) > 0 {
		b.WriteString("\n\nRecent calls from the daily advisor (newest first), for continuity:\n")
		for _, sig := range signals {
			reason := ""
			if sig.Reasoning != nil {
				reason = clip(*sig.Reasoning, 240)
			}
			fmt.Fprintf(&b, "- %s %s %s: %s\n", sig.SignalDate.Format("2006-01-02"), sig.Metal, sig.SignalType, reason)
		}
	}
	b.WriteString("\nReply with the review only — no preamble.")
	return b.String()
}
