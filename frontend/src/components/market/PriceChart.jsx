import { useMemo, useState } from 'react';
import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, ReferenceDot, ReferenceLine,
} from 'recharts';
import { fmt, fmtDate } from '../../lib/format.js';
import { availableRanges, rangeStats, sliceRange, RANGES } from '../../lib/chartRange.js';

// The series itself is data, not a verdict, so it is drawn in neutral
// white. Green and red inside the plot mean one thing only: whether a
// purchase is above or below the current spot.
const SERIES = '#E8ECEF';
const OK = '#16A34A';
const BAD = '#DC2626';
const LINE = '#232A31';
const LINE_BRIGHT = '#333C45';
const MUTED = '#8B949E';
const GROUND = '#161A1F';

const TICKER = { gold: 'XAU', silver: 'XAG' };

/** Dates arrive as "2026-08-24" or a full timestamp; compare the day. */
function dayOf(value) {
  return String(value ?? '').slice(0, 10);
}

function ChartTooltip({ active, payload, label, marksByDate }) {
  if (!active || !payload?.length) return null;
  const bought = marksByDate.get(dayOf(label)) || [];
  return (
    <div className="rounded-chip border border-line-bright bg-ink-sunken/95 px-3 py-2 shadow-lg backdrop-blur">
      <p className="stamp">{fmtDate(label)}</p>
      <p className="font-mono text-sm text-chalk">BHD {fmt(payload[0].value, 3)}</p>
      {bought.map((m, i) => (
        <p key={i} className={`font-mono text-xs ${m.up ? 'text-ok-bright' : 'text-bad-bright'}`}>
          Bought {fmt(m.pricePerGram, 3)}
          {m.exact ? '' : ` · ${fmtDate(m.date)}`}
        </p>
      ))}
    </div>
  );
}

/** Low–high track with a marker where the current price sits. */
function RangeBar({ low, high, value }) {
  const span = high - low;
  const pos = span > 0 ? Math.min(1, Math.max(0, (value - low) / span)) : 0.5;
  return (
    <div className="relative pt-2.5">
      <div
        className="absolute top-0 h-0 w-0 -translate-x-1/2 border-x-[6px] border-t-[7px] border-x-transparent border-t-chalk"
        style={{ left: `${pos * 100}%` }}
        aria-hidden
      />
      <div className="h-1.5 rounded-full bg-line-bright" />
    </div>
  );
}

/**
 * Spot price over a selectable window, with each purchase marked at the
 * price paid. A dot below the current spot is an entry in profit and is
 * drawn green; one above it is underwater and drawn red.
 *
 * The axis is categorical — a mark can only sit on a date the series
 * actually contains — so purchases made on days with no recorded price
 * (a weekend, or before the feed started) are snapped to the nearest
 * plotted day rather than dropped.
 */
export function PriceChart({ prices, purchases, metal = 'gold' }) {
  const data = useMemo(
    () =>
      [...prices]
        .sort((a, b) => new Date(a.price_date) - new Date(b.price_date))
        .map((p) => ({ date: dayOf(p.price_date), price: Number(p.rate) })),
    [prices],
  );

  const marks = useMemo(() => {
    if (data.length === 0) return [];
    const spot = data[data.length - 1].price;
    return purchases
      .filter((p) => p.date && Number.isFinite(Number(p.pricePerGram)))
      .map((p) => {
        const day = dayOf(p.date);
        // First plotted day on or after the purchase; failing that, the
        // last one — a purchase newer than the price feed still belongs
        // at the right-hand edge.
        let i = data.findIndex((d) => d.date >= day);
        if (i === -1) i = data.length - 1;
        return {
          ...p,
          pricePerGram: Number(p.pricePerGram),
          plottedDate: data[i].date,
          exact: data[i].date === day,
          up: Number(p.pricePerGram) <= spot,
        };
      });
  }, [data, purchases]);

  const marksByDate = useMemo(() => {
    const map = new Map();
    for (const m of marks) {
      const list = map.get(m.plottedDate) || [];
      list.push(m);
      map.set(m.plottedDate, list);
    }
    return map;
  }, [marks]);

  // Every window is shown so the options are always visible; ones the
  // recorded history cannot fill yet are disabled rather than hidden.
  const enabled = useMemo(() => new Set(availableRanges(data).map((r) => r.key)), [data]);
  const [picked, setPicked] = useState('1Y');
  // Switching metal can leave the picked window unavailable; fall back
  // to the widest one on offer rather than drawing an empty plot.
  const rangeKey = enabled.has(picked) ? picked : 'ALL';
  const caption = RANGES.find((r) => r.key === rangeKey).caption;

  const visible = useMemo(() => sliceRange(data, rangeKey), [data, rangeKey]);
  const stats = useMemo(() => rangeStats(visible), [visible]);
  const yearStats = useMemo(() => rangeStats(sliceRange(data, '1Y')), [data]);

  if (data.length === 0) {
    return (
      <p className="py-16 text-center text-xs text-muted">
        No price history yet. Record a spot price to start the series.
      </p>
    );
  }

  const visibleDates = new Set(visible.map((d) => d.date));
  const visibleMarks = marks.filter((m) => visibleDates.has(m.plottedDate));
  const snapped = visibleMarks.filter((m) => !m.exact).length;

  // The scale follows the price, padded so the line never rides an
  // edge. Widening it for a purchase far outside the window flattened
  // the series into a ruler, so such a purchase is pinned to the edge
  // it is beyond and drawn hollow instead; the tooltip keeps its price.
  const pad = (stats.high - stats.low) * 0.08 || stats.high * 0.02 || 1;
  const domain = [Math.max(0, stats.low - pad), stats.high + pad];
  const pinned = visibleMarks.map((m) => ({
    ...m,
    y: Math.min(domain[1], Math.max(domain[0], m.pricePerGram)),
    offScale: m.pricePerGram < domain[0] || m.pricePerGram > domain[1],
  }));
  const offScale = pinned.filter((m) => m.offScale).length;

  const rising = stats.change >= 0;
  const tone = rising ? 'text-ok-bright' : 'text-bad-bright';

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm text-muted">
          Current price <span className="px-1 text-line-bright">•</span>
          <span className="text-chalk/90">{TICKER[metal] || TICKER.gold}/BHD</span>
          <span className="pl-1 text-muted/70">per gram</span>
        </p>
        <p className="mt-1 text-3xl font-semibold tabular-nums tracking-tight text-chalk sm:text-4xl">
          BHD {fmt(stats.close, 3)}
        </p>
        <p className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          <span className={`flex items-center gap-1.5 font-semibold tabular-nums ${tone}`}>
            <svg width="10" height="7" viewBox="0 0 10 7" aria-hidden className={rising ? '' : 'rotate-180'}>
              <path d="M5 0 10 7H0z" fill="currentColor" />
            </svg>
            BHD {fmt(Math.abs(stats.change), 3)} ({rising ? '+' : '−'}
            {fmt(Math.abs(stats.pct), 2)}%)
          </span>
          <span className="flex items-center gap-1.5 text-muted">
            <svg width="13" height="13" viewBox="0 0 16 16" aria-hidden>
              <circle cx="8" cy="8" r="7" fill="currentColor" opacity="0.85" />
              <path d="M8 4v4.3l2.6 1.6" stroke={GROUND} strokeWidth="1.6" fill="none" strokeLinecap="round" />
            </svg>
            {caption}
          </span>
        </p>
      </div>

      <div className="h-64 border-b border-line sm:h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={visible} margin={{ top: 8, right: 0, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="price-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={SERIES} stopOpacity={0.28} />
                <stop offset="100%" stopColor={SERIES} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="date" hide />
            <YAxis
              orientation="right"
              domain={domain}
              tickCount={4}
              tickFormatter={(v) => fmt(v, v >= 100 ? 0 : v >= 10 ? 1 : 3)}
              tick={{ fontSize: 10, fill: MUTED }}
              tickLine={false}
              axisLine={false}
              width={44}
              mirror
            />
            <ReferenceLine y={stats.close} stroke={LINE_BRIGHT} strokeDasharray="2 4" />
            <Tooltip
              content={<ChartTooltip marksByDate={marksByDate} />}
              cursor={{ stroke: MUTED, strokeWidth: 1, strokeDasharray: '3 3' }}
            />
            <Area
              type="monotone"
              dataKey="price"
              stroke={SERIES}
              strokeWidth={1.75}
              fill="url(#price-fill)"
              dot={false}
              activeDot={{ r: 4, fill: SERIES, stroke: GROUND, strokeWidth: 2 }}
              animationDuration={450}
            />
            {pinned.map((m, i) => (
              <ReferenceDot key={i} x={m.plottedDate} y={m.y} r={4} isFront
                fill={m.offScale ? GROUND : m.up ? OK : BAD}
                stroke={m.offScale ? (m.up ? OK : BAD) : GROUND} strokeWidth={2} />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div
        className="grid grid-cols-5 gap-1 rounded-lg border border-line bg-ink-sunken p-1 sm:flex"
        role="group"
        aria-label="Time range"
      >
        {RANGES.map((r) => {
          const active = r.key === rangeKey;
          const disabled = !enabled.has(r.key);
          return (
            <button
              key={r.key}
              type="button"
              onClick={() => setPicked(r.key)}
              aria-pressed={active}
              disabled={disabled}
              title={disabled ? `History starts ${fmtDate(data[0].date)}, not enough for ${r.label}` : undefined}
              className={`flex-1 rounded-md py-1.5 font-display text-xs font-semibold tracking-wide transition-colors ${
                active
                  ? 'bg-ink-raised text-chalk ring-1 ring-chalk/80'
                  : disabled
                    ? 'cursor-not-allowed text-muted/35'
                    : 'text-muted hover:bg-line/50 hover:text-chalk'
              }`}
            >
              {r.label}
            </button>
          );
        })}
      </div>

      <div className="rounded-lg border border-line px-4 py-4">
        <RangeBar low={stats.low} high={stats.high} value={stats.close} />
        <div className="mt-2.5 flex justify-between text-sm font-semibold tabular-nums text-chalk">
          <span><span className="stamp mr-2">Low</span>BHD {fmt(stats.low, 3)}</span>
          <span>BHD {fmt(stats.high, 3)}<span className="stamp ml-2">High</span></span>
        </div>
        <div className="mt-3 flex flex-wrap justify-between gap-2 border-t border-line pt-3 text-xs text-muted">
          <span>52-week range (low–high)</span>
          <span className="tabular-nums">
            BHD {fmt(yearStats.low, 3)} – {fmt(yearStats.high, 3)}
          </span>
        </div>
      </div>

      {(visibleMarks.length > 0 || snapped > 0) && (
        <div className="flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-muted">
          <span>
            <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ background: OK }} />
            Purchase below spot
          </span>
          <span>
            <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ background: BAD }} />
            Purchase above spot
          </span>
          {offScale > 0 && (
            <span className="text-muted/70">
              Hollow: {offScale} purchase{offScale === 1 ? '' : 's'} beyond this range, pinned to the edge.
            </span>
          )}
          {snapped > 0 && (
            <span className="text-muted/70">
              {snapped} purchase{snapped === 1 ? '' : 's'} shown on the nearest recorded day.
            </span>
          )}
        </div>
      )}
    </div>
  );
}
