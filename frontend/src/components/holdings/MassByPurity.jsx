import { massByPurity } from '../../lib/holdings.js';
import { fmt } from '../../lib/format.js';
import { Card } from '../ui/Card.jsx';
import { Hallmark } from '../ui/Hallmark.jsx';
import { Skeleton } from '../ui/Skeleton.jsx';

/**
 * How much metal is held at each purity — the figure a dealer quotes
 * against, and the one the holdings table cannot answer because it
 * lists pieces rather than totals.
 *
 * Every column here is a fact, not a verdict: mass, cost, blended
 * entry rate, current worth. Nothing is coloured. Gain and loss are
 * the holdings table's job, and repeating the andon here would spend
 * it on a board that has no bad state to report.
 */
function Figure({ value, decimals = 2, muted = false }) {
  return (
    <td
      className={`px-5 py-2.5 text-right font-mono text-sm ${muted ? 'text-muted' : 'text-chalk'}`}
    >
      {value == null ? <span className="text-muted">—</span> : fmt(value, decimals)}
    </td>
  );
}

export function MassByPurity({ items, loading }) {
  const groups = massByPurity(items);

  if (loading) {
    return (
      <Card title="Mass by purity">
        <div className="space-y-3">
          {[0, 1].map((i) => <Skeleton key={i} className="h-8 w-full" />)}
        </div>
      </Card>
    );
  }

  // The holdings table already tells an owner with nothing that they
  // have nothing, and tells them what to do about it. A second empty
  // state under it would just be noise.
  if (groups.length === 0) return null;

  return (
    <Card title="Mass by purity" padded={false}>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr>
              <th scope="col" className="stamp px-5 pb-2 pt-4 text-left">Purity</th>
              <th scope="col" className="stamp px-5 pb-2 pt-4 text-right">Mass (g)</th>
              <th scope="col" className="stamp px-5 pb-2 pt-4 text-right">Paid</th>
              <th scope="col" className="stamp px-5 pb-2 pt-4 text-right">Entry BHD/g</th>
              <th scope="col" className="stamp px-5 pb-2 pt-4 text-right">Value</th>
            </tr>
          </thead>

          {groups.map((group) => (
            <tbody key={group.metal} className="border-t border-line">
              <tr>
                <th
                  scope="colgroup"
                  colSpan={5}
                  className="stamp bg-ink-sunken px-5 py-1.5 text-left"
                >
                  {group.metal}
                </th>
              </tr>

              {group.rows.map((row) => (
                <tr key={row.purity} className="border-t border-line">
                  <td className="px-5 py-2.5">
                    <Hallmark
                      metal={group.metal}
                      karat={group.metal === 'gold' ? row.purity : null}
                      fineness={group.metal === 'silver' ? row.purity : null}
                      size="sm"
                    />
                  </td>
                  <Figure value={row.grams} />
                  <Figure value={row.paid} muted />
                  <Figure value={row.avgEntry} decimals={3} muted />
                  <Figure value={row.value} />
                </tr>
              ))}

              {/* A subtotal for a metal held at one purity only would
                  repeat the row above it verbatim. */}
              {group.rows.length > 1 && (
                <tr className="border-t border-line-bright">
                  <td className="stamp px-5 py-2.5 text-left">Total {group.metal}</td>
                  <Figure value={group.subtotal.grams} />
                  <Figure value={group.subtotal.paid} muted />
                  <td />
                  <Figure value={group.subtotal.value} />
                </tr>
              )}
            </tbody>
          ))}
        </table>
      </div>
    </Card>
  );
}
