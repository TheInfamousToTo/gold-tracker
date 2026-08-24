import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceDot,
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

function ChartTooltip({ active, payload, label }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-chip border border-line-bright bg-ink-sunken px-3 py-2">
      <p className="stamp">{fmtDate(label)}</p>
      <p className="font-mono text-sm text-chalk">{fmt(payload[0].value, 3)} BHD/g</p>
    </div>
  );
}

/**
 * Spot price over time, with each purchase marked at the price paid.
 * A dot below the current spot is an entry in profit and is drawn
 * green; one above it is underwater and drawn red. That is the question
 * this chart exists to answer at a glance, so it is the only thing on
 * it that carries colour.
 */
export function PriceChart({ prices, purchases }) {
  const data = [...prices]
    .sort((a, b) => new Date(a.price_date) - new Date(b.price_date))
    .map((p) => ({ date: p.price_date, price: p.price_per_gram_24k }));

  if (data.length === 0) {
    return (
      <p className="py-16 text-center text-xs text-muted">
        No price history yet. Record a spot price to start the series.
      </p>
    );
  }

  const spot = Number(data[data.length - 1].price);

  // Only mark purchases that fall inside the plotted range; a dot at an
  // unplotted date would float against the axis.
  const plotted = new Set(data.map((d) => d.date));
  const marks = purchases
    .filter((p) => plotted.has(p.date))
    .map((p) => ({ ...p, up: Number(p.pricePerGram) <= spot }));

  return (
    <div className="h-72">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: -8 }}>
          <CartesianGrid stroke={LINE} vertical={false} />
          <XAxis dataKey="date" tickFormatter={fmtDate} stroke={MUTED}
            tick={{ fontSize: 10, fill: MUTED }} tickLine={false} axisLine={{ stroke: LINE }} minTickGap={40} />
          <YAxis stroke={MUTED} tick={{ fontSize: 10, fill: MUTED }} tickLine={false}
            axisLine={false} domain={['auto', 'auto']} width={52} />
          <Tooltip content={<ChartTooltip />} cursor={{ stroke: MUTED, strokeDasharray: '3 3' }} />
          <Line type="monotone" dataKey="price" stroke={SERIES} strokeWidth={2} dot={false}
            activeDot={{ r: 4, fill: SERIES }} isAnimationActive={false} />
          {marks.map((m, i) => (
            <ReferenceDot key={i} x={m.date} y={m.pricePerGram} r={4} fill={m.up ? OK : BAD}
              stroke="#0F1215" strokeWidth={2} isFront />
          ))}
        </LineChart>
      </ResponsiveContainer>
      {marks.length > 0 && (
        <div className="flex flex-wrap gap-x-5 gap-y-1 pt-2 text-xs text-muted">
          <span>
            <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ background: OK }} />
            Purchase below spot
          </span>
          <span>
            <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ background: BAD }} />
            Purchase above spot
          </span>
        </div>
      )}
    </div>
  );
}
