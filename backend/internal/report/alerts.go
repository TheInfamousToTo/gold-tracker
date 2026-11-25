package report

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// dropWindowDays is how far back the drop is measured from: the highest
// price in the last week against the latest one.
const dropWindowDays = 7

// dropThresholdPct is a "huge" drop per metal. Silver swings roughly
// twice as hard as gold, so it needs a bigger move to count.
var dropThresholdPct = map[string]float64{model.MetalGold: 5, model.MetalSilver: 8}

type drop struct {
	Peak, Latest point
	Pct          float64 // negative for a fall
}

// weekDrop compares the latest price with the highest of the week before.
// s must be sorted by date.
func weekDrop(s []point) (drop, bool) {
	if len(s) < 2 {
		return drop{}, false
	}
	latest := s[len(s)-1]
	cutoff := mustDate(latest.Date).AddDate(0, 0, -dropWindowDays).Format("2006-01-02")
	var peak point
	for _, p := range s[:len(s)-1] {
		if p.Date >= cutoff && p.Price > peak.Price {
			peak = p
		}
	}
	if peak.Price <= 0 {
		return drop{}, false
	}
	return drop{Peak: peak, Latest: latest, Pct: (latest.Price - peak.Price) / peak.Price * 100}, true
}

// shouldAlert fires once per slide: again only after a further full
// threshold fall below the last alert, or a week after it.
func shouldAlert(d drop, threshold float64, lastDate string, lastPrice float64, alerted bool) bool {
	switch {
	case d.Pct > -threshold:
		return false
	case !alerted:
		return true
	case lastDate >= d.Latest.Date:
		return false
	case d.Latest.Price <= lastPrice*(1-threshold/100):
		return true
	default:
		return !mustDate(lastDate).AddDate(0, 0, dropWindowDays).After(mustDate(d.Latest.Date))
	}
}

func dropMessage(metal string, d drop) string {
	unit := "24K"
	if metal == model.MetalSilver {
		unit = "999"
	}
	return fmt.Sprintf("⚠️ **%s dropped %.1f%% in a week**\n%.3f → %.3f BHD/g (%s), from the %s high to %s.",
		strings.ToUpper(metal[:1])+metal[1:], -d.Pct, d.Peak.Price, d.Latest.Price, unit, d.Peak.Date, d.Latest.Date)
}

// CheckDrops posts a Discord alert when a metal has fallen hard.
func (s *Service) CheckDrops(ctx context.Context) {
	if s.webhook == "" {
		return
	}
	var in Inputs
	var err error
	if in.Gold, err = s.repo.GetPrices(ctx, 30); err != nil {
		log.Printf("price alert: %v", err)
		return
	}
	if in.Silver, err = s.repo.GetSilverPrices(ctx, 30); err != nil {
		log.Printf("price alert: %v", err)
		return
	}
	for metal, prices := range series(in) {
		d, ok := weekDrop(prices)
		if !ok {
			continue
		}
		lastDate, lastPrice, alerted, err := s.repo.LastPriceAlert(ctx, metal)
		if err != nil {
			log.Printf("price alert: %v", err)
			continue
		}
		if !shouldAlert(d, dropThresholdPct[metal], lastDate, lastPrice, alerted) {
			continue
		}
		if err := Send(ctx, s.client, s.webhook, []string{dropMessage(metal, d)}); err != nil {
			log.Printf("price alert: %v", err)
			continue
		}
		if err := s.repo.SavePriceAlert(ctx, metal, d.Latest.Date, d.Latest.Price); err != nil {
			log.Printf("price alert: %v", err)
		}
		log.Printf("price alert: %s %.1f%% sent", metal, d.Pct)
	}
}
