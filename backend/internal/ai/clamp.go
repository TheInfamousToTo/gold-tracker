package ai

import (
	"fmt"
	"math"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

const (
	// newsBuyStretch is how far news may push a BUY above the rules'
	// amount: the prompt's "±50%".
	newsBuyStretch = 1.5
	// contraryConfidenceCap bounds the confidence of a SELL or CUT_LOSS
	// that the rules did not call for: going against a long-term
	// saver's rules on the strength of headlines is never a sure thing.
	contraryConfidenceCap = 0.6
	// deviationTolerance is how many BHD an amount may differ from the
	// baseline before the verdict is marked as departing from it.
	deviationTolerance = 0.5
)

// Clamped is an Advice after Go has bounded it: what is stored.
type Clamped struct {
	Advice
	AmountGrams float64
	Deviates    bool
	Notes       []string
}

// clampAdvice bounds the model's numbers to what the money and the
// holdings allow. The model never chooses grams; they are derived from
// the amount at the price the owner would actually pay or receive.
func clampAdvice(a Advice, s MetalState, settings model.AdvisorSettings, budget Budget) Clamped {
	b := s.Baseline
	c := Clamped{Advice: a}
	spread := settings.SpreadFraction(s.Metal)
	latest := s.Indicators.Latest

	switch a.Action {
	case ActionBuy:
		limit := b.Share * budget.Remaining
		if b.Action == ActionBuy {
			limit = newsBuyStretch * b.AmountBHD
		}
		limit = math.Min(limit, budget.Remaining+budget.Reserve)
		if c.AmountBHD <= 0 && b.Action == ActionBuy {
			c.AmountBHD = b.AmountBHD
		}
		if c.AmountBHD > limit {
			c.Notes = append(c.Notes, fmt.Sprintf("amount capped at %.2f BHD", limit))
			c.AmountBHD = limit
		}
		if c.AmountBHD <= 0 || latest <= 0 {
			c.toHold("no budget left to buy with")
			break
		}
		c.AmountGrams = c.AmountBHD / (latest * (1 + spread))

	case ActionSell, ActionCutLoss:
		if !s.Position.Held || s.Position.NetValue <= 0 || latest <= 0 {
			c.toHold("nothing held to sell")
			break
		}
		if c.AmountBHD <= 0 {
			if a.Action == ActionSell {
				if b.Action != ActionSell {
					c.toHold("a sell needs an amount")
					break
				}
				c.AmountBHD = b.AmountBHD
			} else {
				c.AmountBHD = s.Position.NetValue
			}
		}
		if c.AmountBHD > s.Position.NetValue {
			c.AmountBHD = s.Position.NetValue
		}
		c.AmountGrams = c.AmountBHD / (latest * (1 - spread))
		if c.AmountGrams > s.Position.FineGrams {
			c.AmountGrams = s.Position.FineGrams
		}
		if b.Action != a.Action && c.Confidence > contraryConfidenceCap {
			c.Confidence = contraryConfidenceCap
		}

	default:
		c.AmountBHD = 0
	}

	c.AmountBHD = round2v(c.AmountBHD)
	c.AmountGrams = round3v(c.AmountGrams)
	c.Deviates = c.Action != b.Action || math.Abs(c.AmountBHD-b.AmountBHD) > deviationTolerance
	return c
}

func (c *Clamped) toHold(why string) {
	c.Action = ActionHold
	c.AmountBHD = 0
	c.AmountGrams = 0
	c.Notes = append(c.Notes, why)
}

// capTotalBuys scales every BUY down together when, across metals,
// they would spend more than the budget left plus the reserve. Each
// metal's own cap cannot see the other's.
func capTotalBuys(advice map[string]*Clamped, budget Budget, latest map[string]float64, settings model.AdvisorSettings) {
	var total float64
	for _, c := range advice {
		if c.Action == ActionBuy {
			total += c.AmountBHD
		}
	}
	limit := budget.Remaining + budget.Reserve
	if total <= limit || total == 0 {
		return
	}
	scale := limit / total
	for metal, c := range advice {
		if c.Action != ActionBuy {
			continue
		}
		c.AmountBHD = round2v(c.AmountBHD * scale)
		if p := latest[metal]; p > 0 {
			c.AmountGrams = round3v(c.AmountBHD / (p * (1 + settings.SpreadFraction(metal))))
		}
		c.Notes = append(c.Notes, "scaled down to fit the budget across both metals")
	}
}
