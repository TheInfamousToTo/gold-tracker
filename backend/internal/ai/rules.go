package ai

import (
	"math"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// Actions the advisor can recommend.
const (
	ActionBuy     = "BUY"
	ActionHold    = "HOLD"
	ActionSell    = "SELL"
	ActionCutLoss = "CUT_LOSS"
)

// Every threshold the baseline uses, in one place. The profile is a
// long-term saver: buying is the default, size moves with how far price
// is from its long-run average, selling is rare and cutting a loss
// needs the long-term trend to have broken.
const (
	dipZ           = -1.5 // z200 at or below this is a deep dip
	stretchedZ     = 1.5  // z200 at or above this is stretched
	trimZ          = 2.5  // z200 needed before a trim is considered
	trimMomentum   = 40.0 // and a 252-day gain above this, in percent
	trimOverweight = 0.10 // and the metal this far above its target share
	dipMultiplier  = 1.5
	highMultiplier = 0.5
	dipReserveDraw = 0.25 // fraction of the reserve a deep dip may use
	maxDriftTilt   = 0.20 // how far allocation drift may move the split
	ratioTilt      = 0.20 // how far an extreme gold/silver ratio moves it
	ratioHighPct   = 0.80
	ratioLowPct    = 0.20
	minFeeMultiple = 10.0 // a buy under this many fees is batched instead
	buyMoreSigma   = 1.5  // buy_more_below sits this many σ under SMA200
)

// Baseline is the rules' answer for one metal, before the model sees
// news. The model may depart from it within the bounds clamp enforces.
type Baseline struct {
	Action      string   `json:"action"`
	AmountBHD   float64  `json:"amount_bhd"`
	AmountGrams float64  `json:"amount_grams"`
	Share       float64  `json:"share"`
	Multiplier  float64  `json:"multiplier"`
	ReserveDraw float64  `json:"reserve_draw"`
	ToReserve   float64  `json:"to_reserve"`
	Notes       []string `json:"notes"`

	BuyMoreBelow *float64 `json:"buy_more_below,omitempty"`
	CutLossBelow *float64 `json:"cut_loss_below,omitempty"`
}

// MetalState is everything known about one metal at decision time.
type MetalState struct {
	Metal      string
	Prices     []PriceHistoryPoint // oldest first
	Indicators Indicators
	Position   Position
	Baseline   Baseline
}

// Budget is the money the advisor may size against this month.
type Budget struct {
	Monthly   float64
	Spent     float64
	Remaining float64
	Reserve   float64
}

// splitShares divides this month's buying between the metals. It starts
// from the target allocation, leans toward whichever metal is under its
// target, then leans again on an extreme gold/silver ratio.
func splitShares(states []MetalState, settings model.AdvisorSettings, ratio RatioStats) map[string]float64 {
	priced := map[string]bool{}
	for _, s := range states {
		priced[s.Metal] = s.Indicators.Count > 0
	}
	if !priced[model.MetalSilver] {
		return map[string]float64{model.MetalGold: 1, model.MetalSilver: 0}
	}
	if !priced[model.MetalGold] {
		return map[string]float64{model.MetalGold: 0, model.MetalSilver: 1}
	}

	gold := settings.TargetShare(model.MetalGold)
	if anyHeld(states) {
		var held float64
		for _, s := range states {
			if s.Metal == model.MetalGold {
				held = s.Position.Share
			}
		}
		gold += math.Max(-maxDriftTilt, math.Min(maxDriftTilt, gold-held))
	}
	if ratio.Available {
		switch {
		case ratio.Percentile >= ratioHighPct:
			gold -= ratioTilt
		case ratio.Percentile <= ratioLowPct:
			gold += ratioTilt
		}
	}
	gold = math.Max(0, math.Min(1, gold))
	return map[string]float64{model.MetalGold: gold, model.MetalSilver: 1 - gold}
}

func anyHeld(states []MetalState) bool {
	for _, s := range states {
		if s.Position.Held {
			return true
		}
	}
	return false
}

// computeBaselines fills in each state's Baseline.
func computeBaselines(states []MetalState, settings model.AdvisorSettings, budget Budget, ratio RatioStats) {
	shares := splitShares(states, settings, ratio)
	var totalValue float64
	for _, s := range states {
		totalValue += s.Position.Value
	}
	for i := range states {
		states[i].Baseline = baselineFor(states[i], shares[states[i].Metal], settings, budget, totalValue)
	}
}

func baselineFor(s MetalState, share float64, settings model.AdvisorSettings, budget Budget, totalValue float64) Baseline {
	ind, pos := s.Indicators, s.Position
	spread := settings.SpreadFraction(s.Metal)
	stop := settings.StopLossPct
	b := Baseline{Action: ActionHold, Share: share, Multiplier: 1}

	if ind.Count == 0 {
		b.Notes = append(b.Notes, "no price data")
		return b
	}

	if ind.StdDev200 > 0 {
		b.BuyMoreBelow = round3(ind.SMA200 - buyMoreSigma*ind.StdDev200)
	}
	if pos.Held && pos.FineGrams > 0 && spread < 1 {
		b.CutLossBelow = round3(pos.Paid * (1 - stop/100) / (pos.FineGrams * (1 - spread)))
	}

	// Cutting a loss outranks everything: it needs the long-term trend
	// to have broken AND the position to be past the owner's own stop,
	// so a dip in an intact uptrend never triggers it.
	if pos.Held && !ind.Sparse() && ind.Latest < ind.SMA200 && ind.SMA50 < ind.SMA200 && pos.NetPLPct < -stop {
		b.Action = ActionCutLoss
		b.AmountBHD = round2v(pos.NetValue)
		b.AmountGrams = round3v(pos.FineGrams)
		b.Notes = append(b.Notes, "below the 200-day average in a downtrend and past the stop-loss")
		return b
	}

	target := settings.TargetShare(s.Metal)
	if pos.Held && ind.HasMom252 && ind.Z200 >= trimZ && ind.Mom252 > trimMomentum && pos.Share > target+trimOverweight {
		amount := (pos.Share - target) * totalValue * (1 - spread)
		b.Action = ActionSell
		b.AmountBHD = round2v(amount)
		b.AmountGrams = round3v(amount / (ind.Latest * (1 - spread)))
		b.Notes = append(b.Notes, "extremely stretched after a large yearly gain, and overweight: trim back to target")
		return b
	}

	switch {
	case ind.Z200 <= dipZ:
		b.Multiplier = dipMultiplier
		b.ReserveDraw = round2v(dipReserveDraw * budget.Reserve * share)
		b.Notes = append(b.Notes, "deep dip against the 200-day average: buy extra")
	case ind.Z200 >= stretchedZ:
		b.Multiplier = highMultiplier
		b.ToReserve = round2v((1 - highMultiplier) * share * budget.Remaining)
		b.Notes = append(b.Notes, "stretched above the 200-day average: buy half, park the rest in the reserve")
	}

	amount := share*budget.Remaining*b.Multiplier + b.ReserveDraw
	amount = math.Min(amount, budget.Remaining+budget.Reserve)
	switch {
	case budget.Monthly <= 0 && b.ReserveDraw <= 0:
		b.Notes = append(b.Notes, "no monthly budget is set")
		return b
	case amount <= 0:
		b.Notes = append(b.Notes, "this month's budget is already used")
		return b
	case settings.MinFeeBHD > 0 && amount < minFeeMultiple*settings.MinFeeBHD:
		b.Notes = append(b.Notes, "too small against the minimum fee: save it and batch next time")
		return b
	}
	b.Action = ActionBuy
	b.AmountBHD = round2v(amount)
	b.AmountGrams = round3v(amount / (ind.Latest * (1 + spread)))
	return b
}

func round2v(x float64) float64 { return math.Round(x*100) / 100 }
func round3v(x float64) float64 { return math.Round(x*1000) / 1000 }
func round3(x float64) *float64 {
	v := round3v(x)
	return &v
}
