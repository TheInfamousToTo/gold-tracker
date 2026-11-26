package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// Sure posts each metal's daily value to an account in Sure
// (https://github.com/we-promise/sure), so it counts in net worth there.
type Sure struct {
	URL      string
	Key      string
	Accounts map[string]string // metal -> Sure account ID
}

// SureFromEnv returns nil unless SURE_URL and SURE_API_KEY are set.
func SureFromEnv() *Sure {
	s := &Sure{
		URL: strings.TrimRight(os.Getenv("SURE_URL"), "/"),
		Key: os.Getenv("SURE_API_KEY"),
		Accounts: map[string]string{
			model.MetalGold:   os.Getenv("SURE_GOLD_ACCOUNT_ID"),
			model.MetalSilver: os.Getenv("SURE_SILVER_ACCOUNT_ID"),
		},
	}
	if s.URL == "" || s.Key == "" {
		return nil
	}
	return s
}

// recentDays are posted daily; older history one point a week, which is
// enough for Sure's charts and keeps the backfill to ~100 requests.
const recentDays = 30

// sureDates picks the price dates to post: those after `after` (all
// dates from `from` when after is ""), thinned to the last date of each
// week when older than recentDays, plus always the latest date so a new
// purchase shows up the same day.
func sureDates(s []point, from, after string) []string {
	if len(s) == 0 {
		return nil
	}
	latest := s[len(s)-1].Date
	recent := mustDate(latest).AddDate(0, 0, -recentDays).Format("2006-01-02")
	var out []string
	for i, p := range s {
		if p.Date < from || (after != "" && p.Date <= after && p.Date != latest) {
			continue
		}
		if p.Date < recent && i+1 < len(s) {
			y1, w1 := mustDate(p.Date).ISOWeek()
			y2, w2 := mustDate(s[i+1].Date).ISOWeek()
			if y1 == y2 && w1 == w2 {
				continue // not the last date of its week
			}
		}
		out = append(out, p.Date)
	}
	return out
}

// metalValue is what the metal's holdings were worth on date: items
// bought by then, at that day's price adjusted to each item's purity.
func metalValue(items []model.PortfolioItem, metal string, s []point, date string) float64 {
	price, ok := onOrBefore(s, date)
	if !ok {
		return 0
	}
	latest := s[len(s)-1].Price
	var v float64
	for _, it := range items {
		if metalOf(it) == metal && it.PurchaseDate <= date {
			v += it.WeightGrams * price * purityFactor(it, latest)
		}
	}
	return round(v, 3)
}

func (c *Sure) post(ctx context.Context, client *http.Client, account, date string, amount float64) error {
	body, _ := json.Marshal(map[string]any{"valuation": map[string]any{
		"account_id": account, "date": date, "amount": amount, "notes": "Synced from gold-tracker",
	}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/api/v1/valuations", bytes.NewReader(body))
	if err != nil {
		return errors.New("SURE_URL is not a valid URL")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.Key)
	resp, err := client.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("sure request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sure returned %s for %s", resp.Status, date)
	}
	return nil
}

// SyncSure posts any metal values Sure hasn't got yet.
func (s *Service) SyncSure(ctx context.Context) {
	if s.Sure == nil {
		return
	}
	fail := func(err error) { log.Printf("sure sync: %v", err) }
	portfolio, err := s.repo.GetPortfolioSummary(ctx)
	if err != nil {
		fail(err)
		return
	}
	var in Inputs
	if in.Gold, err = s.repo.GetPrices(ctx, 3700); err != nil {
		fail(err)
		return
	}
	if in.Silver, err = s.repo.GetSilverPrices(ctx, 3700); err != nil {
		fail(err)
		return
	}
	for metal, prices := range series(in) {
		account := s.Sure.Accounts[metal]
		first := ""
		for _, it := range portfolio.Items {
			if metalOf(it) == metal && (first == "" || it.PurchaseDate < first) {
				first = it.PurchaseDate
			}
		}
		if account == "" || first == "" {
			continue
		}
		after, _, err := s.repo.SureSyncedThrough(ctx, metal)
		if err != nil {
			fail(err)
			continue
		}
		dates := sureDates(prices, first, after)
		for _, d := range dates {
			if err := s.Sure.post(ctx, s.client, account, d, metalValue(portfolio.Items, metal, prices, d)); err != nil {
				fail(err)
				break
			}
			if err := s.repo.SaveSureSynced(ctx, metal, d); err != nil {
				fail(err)
				break
			}
		}
		if len(dates) > 1 {
			log.Printf("sure sync: %s, %d values posted", metal, len(dates))
		}
	}
}
