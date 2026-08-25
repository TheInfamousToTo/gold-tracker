import { useMemo, useState } from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceDot, Brush,
} from 'recharts';
import { fmt, fmtDate } from '../../lib/format.js';

// The series itself is data, not a verdict, so it is drawn in neutral
// white. Green and red on this chart mean one thing only: whether a
// purchase is above or below the current spot.
const SERIES = '#E8ECEF';
const OK = '#16A34A';
const BAD = '#DC2626';
const LINE = '#232A31';
const MUTED = '#8B949E';
const GROUND = '#0F1215';

/** Dates arrive as "2026-08-24" or a full timestamp; compare the day. */
function dayOf(value) {
  return String(value ?? '').slice(0, 10);
}

function ChartTooltip({ active, payload, label, marksByDate }) {
  if (!active || !payload?.length) return null;
  const bought = marksByDate.get(dayOf(label)) || [];
  return (
    <div className="rounded-chip border border-line-bright bg-ink-sunken px-3 py-2">
      <p className="stamp">{fmtDate(label)}</p>
      <p className="font-mono text-sm text-chalk">{fmt(payload[0].value, 3)} BHD/g</p>
      {bought.map((m, i) => (
        <p key={i} className={`font-mono text-xs ${m.up ? 'text-ok-bright' : 'text-bad-bright'}`}>
          Bought {fmt(m.pricePerGram, 3)}
          {m.exact ? '' : ` · ${fmtDate(m.date)}`}
        </p>
      ))}
    </div>
  );
}

/**
 * Spot price over time, with each purchase marked at the price paid.
 * A dot below the current spot is an entry in profit and is drawn
 * green; one above it is underwater and drawn red. That is the question
 * this chart exists to answer at a glance, so it is the only thing on
 * it that carries colour.
 *
 * The axis is categorical — a mark can only sit on a date the series
 * actually contains — so purchases made on days with no recorded price
 * (a weekend, or before the feed started) are snapped to the nearest
 * plotted day rather than dropped. Dropping them was the old behaviour
 * and it silently hid every purchase.
 */
export function PriceChart({ prices, purchases }) {
  const data = useMemo(
    () =>
      [...prices]
        .sort((a, b) => new Date(a.price_date) - new Date(b.price_date))
        .map((p) => ({ date: dayOf(p.price_date), price: p.price_per_gram_24k })),
    [prices],
  );

  const marks = useMemo(() => {
    if (data.length === 0) return [];
    const spot = Number(data[data.length - 1].price);
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

  // Brush indices are held here so the range survives a re-render and
  // so "Reset" has something to clear.
  const [range, setRange] = useState(null);
  const zoomed =
    range && (range.startIndex > 0 || range.endIndex < Math.max(data.length - 1, 0));

  if (data.length === 0) {
    return (
      <p className="py-16 text-center text-xs text-muted">
        No price history yet. Record a spot price to start the series.
      </p>
    );
  }

  const visible = range ? data.slice(range.startIndex, range.endIndex + 1) : data;
  const visibleDates = new Set(visible.map((d) => d.date));
  const visibleMarks = marks.filter((m) => visibleDates.has(m.plottedDate));
  const snapped = visibleMarks.filter((m) => !m.exact).length;

  return (
    <div>
      <div className="h-72">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: -8 }}>
            <CartesianGrid stroke={LINE} vertical={false} />
            <XAxis dataKey="date" tickFormatter={fmtDate} stroke={MUTED}
              tick={{ fontSize: 10, fill: MUTED }} tickLine={false} axisLine={{ stroke: LINE }} minTickGap={40} />
            <YAxis stroke={MUTED} tick={{ fontSize: 10, fill: MUTED }} tickLine={false}
              axisLine={false} domain={['auto', 'auto']} width={52} />
            <Tooltip content={<ChartTooltip marksByDate={marksByDate} />}
              cursor={{ stroke: MUTED, strokeDasharray: '3 3' }} />
            <Line type="monotone" dataKey="price" stroke={SERIES} strokeWidth={2} dot={false}
              activeDot={{ r: 4, fill: SERIES }} isAnimationActive={false} />
            {marks.map((m, i) => (
              <ReferenceDot key={i} x={m.plottedDate} y={m.pricePerGram} r={4} fill={m.up ? OK : BAD}
                stroke={GROUND} strokeWidth={2} isFront />
            ))}
            {/* Drag the handles to zoom, drag the middle to pan. */}
            <Brush
              dataKey="date"
              height={22}
              travellerWidth={8}
              stroke={LINE}
              fill={GROUND}
              tickFormatter={fmtDate}
              startIndex={range?.startIndex}
              endIndex={range?.endIndex}
              onChange={(r) => setRange({ startIndex: r.startIndex, endIndex: r.endIndex })}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="flex flex-wrap items-center gap-x-5 gap-y-1 pt-2 text-xs text-muted">
        {marks.length > 0 && (
          <>
            <span>
              <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ background: OK }} />
              Purchase below spot
            </span>
            <span>
              <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ background: BAD }} />
              Purchase above spot
            </span>
          </>
        )}
        <span className="text-muted/70">Drag the bar below the chart to zoom or pan.</span>
        {snapped > 0 && (
          <span className="text-muted/70">
            {snapped} purchase{snapped === 1 ? '' : 's'} shown on the nearest recorded day.
          </span>
        )}
        {zoomed && (
          <button
            onClick={() => setRange(null)}
            className="stamp ml-auto rounded-chip border border-line-bright px-2 py-0.5 hover:text-chalk"
          >
            Reset range
          </button>
        )}
      </div>
    </div>
  );
}
