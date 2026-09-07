# Silver support, mass by purity, and per-metal signals

Date: 2026-09-04

## Problem

Three requests, one design:

1. **Total grams per karat.** The holdings table shows every piece
   individually. Nothing answers "how much 21K do I actually own", which
   is the number you need when a dealer quotes you per gram.
2. **Track silver as well as gold.** `gold_items.metal_type` already
   exists and already defaults to `'gold'`, but nothing reads it: prices
   are 24K-only and `v_portfolio_summary` hardcodes the karat lookup, so
   a silver row would be valued at the gold price.
3. **Confirm the AI is deciding, not defaulting.** The signal panel has
   shown HOLD often enough to raise the question of whether BUY and SELL
   are reachable at all.

## Decisions

| Question | Decision |
|---|---|
| Silver price storage | A parallel `silver_prices` table; `gold_prices` untouched |
| Silver item purity | New nullable `purity_fineness` column on `gold_items`, with a CHECK tying it to `metal_type` |
| Signal scope | One CLI run returns a verdict per metal; `signals_log` gains `metal` |
| Gram totals | A "Mass by purity" card on Holdings; `StatGrid` stays one combined portfolio |
| Price API | Optional `metal` field, absent means gold, so the n8n feed is unchanged |

The rejected alternative worth recording: migrating every item to a
single millesimal `purity_fineness` column (21K gold stored as 875) would
give one code path for both metals, but it rewrites existing gold rows
and the `price_per_gram_paid` generated column with them. A parallel
table keeps the working gold path untouched at the cost of a `CASE` that
branches on metal.

## Schema

`migrations/0002_add_silver.sql`, applied at boot and by
`setup_gold_db.sh` (see Migration delivery below).

```sql
CREATE TABLE IF NOT EXISTS silver_prices (
    id SERIAL PRIMARY KEY,
    price_date DATE UNIQUE NOT NULL,
    price_per_gram_999 NUMERIC NOT NULL,
    price_per_gram_925 NUMERIC GENERATED ALWAYS AS (price_per_gram_999 * 925 / 999) STORED,
    price_per_gram_900 NUMERIC GENERATED ALWAYS AS (price_per_gram_999 * 900 / 999) STORED,
    source TEXT,
    created_at TIMESTAMP DEFAULT now()
);

ALTER TABLE gold_items ADD COLUMN IF NOT EXISTS purity_fineness NUMERIC;
ALTER TABLE gold_items ALTER COLUMN purity_karat DROP NOT NULL;
UPDATE gold_items SET metal_type = 'gold' WHERE metal_type IS NULL;
ALTER TABLE gold_items ALTER COLUMN metal_type SET NOT NULL;
ALTER TABLE gold_items DROP CONSTRAINT IF EXISTS chk_item_purity;
ALTER TABLE gold_items ADD CONSTRAINT chk_item_purity CHECK (
  (metal_type = 'gold'   AND purity_karat    IS NOT NULL) OR
  (metal_type = 'silver' AND purity_fineness IS NOT NULL));

ALTER TABLE signals_log ADD COLUMN IF NOT EXISTS metal TEXT NOT NULL DEFAULT 'gold';
```

A 925 piece contains 925 parts pure silver per thousand, so its per-gram
value is `price_per_gram_999 * 925 / 999` — the same shape as gold's
`price_per_gram_24k * 22 / 24`.

`v_portfolio_summary` is rewritten to join the latest row of whichever
price table the item's metal points at, and to compute the per-gram price
once in a `CROSS JOIN LATERAL` instead of repeating the purity `CASE`
four times as it does today. The view gains `metal_type` and
`purity_fineness`.

## Migration delivery

`repository.NewPostgresRepository` already applies the 0001 statements
inline on every boot, and `setup_gold_db.sh` builds a fresh schema. Both
have to learn about silver, and an inline copy of the DDL is a second
place for it to drift from the migration file.

So: embed `migrations/*.sql` with `go:embed` and execute them in filename
order at startup. Every statement stays idempotent, so repeated boots are
safe. `setup_gold_db.sh` gets `silver_prices`, the new item columns, and
the rewritten view, so a fresh install and a migrated install converge on
the same schema.

## Backend

- `GoldItem.PurityKarat` and the new `PurityFineness` become `*float64`;
  `MetalType` starts being read rather than defaulted. `PortfolioItem`
  gains all three.
- New `SilverPrice` model; `GetSilverPrices` / `CreateSilverPrice` in
  `repository/prices.go`.
- `GetPortfolioSummary` stops using `SELECT *`. A positional scan against
  a view whose column list is changing in this same commit is a bug
  waiting to happen.
- `GET /api/prices` reads `?metal` (default gold, response shape
  unchanged). `POST /api/prices` reads `metal` from the body; absent
  means gold, so the existing n8n payload keeps working untouched.
- `CreateItem` / `UpdateItem` validate that the metal is one of
  `gold`/`silver` and that the purity field matching that metal is set —
  the same rule as the CHECK constraint, reported as a 400 rather than a
  500.

## AI

`PromptInput` gains a silver series, and holdings aggregates become
metal-aware. When silver prices or silver holdings exist, the prompt asks
for a verdict per metal:

```json
{"gold":   {"signal": "BUY|SELL|HOLD", "confidence": 0.0, "reasoning": "...", "horizon_days": 30, "key_factors": ["..."]},
 "silver": {"signal": "BUY|SELL|HOLD", "confidence": 0.0, "reasoning": "...", "horizon_days": 30, "key_factors": ["..."]}}
```

`ParseVerdict` becomes `ParseVerdicts`, returning a verdict per metal and
applying the existing enum, confidence-range, and length validation to
each. A gold-only install gets a gold-only prompt and a gold-only schema,
so nothing changes for someone who never adds silver.

`RunOnce` writes one `signals_log` row per verdict, taking
`price_at_signal` from that metal's own series.

## Frontend

- `lib/karat.js` becomes `lib/purity.js`, holding per-metal purity option
  lists — silver as 999 Fine, 925 Sterling, 900 Coin. `Hallmark` takes a
  metal alongside the purity.
- `ItemForm` gains a metal select that drives the purity dropdown.
  `PriceForm` gains one that flips its label between 24K and 999.
- `useGoldData` fetches both price series. The masthead shows both spots.
  The Market tab gets a metal toggle for the chart and history list.
- New `MassByPurity` card on Holdings, computed client-side from
  `portfolio.items` via a pure `lib/holdings.js` aggregate function:

```
MASS BY PURITY        grams     paid    entry     value
GOLD   999 24K        10.00   325.00   32.500    340.20
       875 21K        42.30  1180.40   27.905   1250.10
       subtotal       52.30  1505.40            1590.30
SILVER 925           120.00    48.20    0.402      49.44
```

`StatGrid` stays a single combined portfolio. The masthead question is
"am I up", not "am I up in silver".

## Testing

The frontend test environment is `node`, not jsdom, so the existing suite
tests pure modules rather than rendering components. This design follows
that: the aggregation lives in `lib/holdings.js` and the purity tables in
`lib/purity.js`, both directly testable, with the components staying thin
enough to read.

Backend work is covered by the existing table-driven style in
`internal/ai` and `internal/api`. New cases: per-metal verdict parsing, a
gold-only prompt when no silver exists, metal routing in the price
handlers, and purity validation on item writes.

## Phases

Each phase leaves the app working.

1. `MassByPurity` + `lib/holdings.js`, gold-only. Answers request 1 with
   no API change.
2. Schema, migration runner, `setup_gold_db.sh`.
3. Backend: models, repository, handlers, validation.
4. Frontend silver: purity module, forms, price series, market toggle.
5. AI: per-metal prompt, parsing, and signal rows.

## Open question: is the AI actually deciding?

Reading the code settles half of it. There is no HOLD default anywhere:
`ParseVerdict` rejects any signal outside the BUY/SELL/HOLD enum, and
`RunOnce` retries once and then records an error, saving nothing. A HOLD
in the log was produced by the model.

The prompt does lean toward inaction, though. It instructs the model to
hedge below 14 observations and to keep confidence under 0.5 when
evidence is thin, and with a short price history that pressure is always
applied.

The probe: a `//go:build aiprobe` test that builds the real prompt over
five synthetic series — sustained rally, sustained slide, flat drift,
sparse history, and a sharp crash after a long rise — runs each through
the real CLI runner, and tallies the verdicts. One verdict everywhere
means the prompt is the problem. It spends subscription quota, so it is
opt-in behind the build tag and never runs in CI.

### Result, 2026-09-07

Five scenarios returned **3 HOLD and 2 SELL. No BUY.**

| Scenario | Verdict | Confidence |
|---|---|---|
| Sustained rally, price far above entry | HOLD | 0.42 |
| Sustained slide, price far below entry | SELL | 0.78 |
| Flat drift | HOLD | 0.35 |
| Sparse history | HOLD | 0.35 |
| Sharp crash after a long rise | SELL | 0.58 |

So HOLD is not a default — the model moves off it, with confidence
tracking how clear the picture is. But BUY was unreachable, and the two
cases that should most invite it produced the opposite call.

The cause is the prompt's own framing. It asks first for "where price
goes over the horizon", which makes the task a price forecast, and the
signal then follows the forecast's direction: falling price becomes
SELL, rising price becomes "stretched, don't enter" — HOLD. Under that
reading a dip is a reason to sell, which is backwards for someone
accumulating physical metal, where a dip is the entry.

The prompt never says what the three signals mean for this owner. Until
it does, the model supplies a momentum trader's definitions. Fixing it
means stating the owner's position — a long-horizon accumulator holding
physical metal, who cannot trade intraday and whose SELL is a rare
event — and defining BUY as an attractive entry against the owner's own
average, rather than a bet that price rises tomorrow.

That is a change to what a recommendation *means*, not a bug fix, so it
is left for the owner to decide rather than folded in here.
