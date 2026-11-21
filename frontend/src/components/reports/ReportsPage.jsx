import { useState, useEffect, useCallback, useRef } from 'react';
import { apiRequest, ApiError } from '../../api/client.js';
import { fmt, fmtDate } from '../../lib/format.js';
import { Card } from '../ui/Card.jsx';
import { Button } from '../ui/Button.jsx';
import { EmptyState } from '../ui/EmptyState.jsx';
import { Skeleton } from '../ui/Skeleton.jsx';
import { Markdown } from './Markdown.jsx';

const POLL_MS = 3000;

function monthLabel(month) {
  const [y, m] = month.split('-').map(Number);
  return new Date(y, m - 1, 1).toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

function Signed({ value, decimals = 2, className = '' }) {
  if (value == null) return <span className="text-muted">—</span>;
  const up = value >= 0;
  return (
    <span className={`font-mono ${up ? 'text-ok-bright' : 'text-bad-bright'} ${className}`}>
      {up ? '+' : ''}{fmt(value, decimals)}
    </span>
  );
}

function Totals({ data }) {
  const t = data.totals;
  const cells = [
    ['Invested', fmt(t.paid, 2)],
    ['Value', fmt(t.value, 2)],
    ['Total gain', <Signed key="g" value={t.gain} />],
    ['This month', <Signed key="m" value={t.month_gain} />],
  ];
  return (
    <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      {cells.map(([label, value]) => (
        <div key={label} className="rounded-lg border border-line bg-ink-sunken px-3 py-2.5">
          <dt className="stamp">{label}</dt>
          <dd className="mt-1 font-mono text-base font-semibold text-chalk">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function ItemsTable({ items }) {
  const th = 'stamp px-3 pb-2 pt-3 text-right';
  const thLeft = 'stamp px-3 pb-2 pt-3 text-left';
  return (
    <div className="-mx-5 overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr>
            <th scope="col" className={thLeft}>Item</th>
            <th scope="col" className={`${th} hidden sm:table-cell`}>Grams</th>
            <th scope="col" className={`${th} hidden sm:table-cell`}>Buy /g</th>
            <th scope="col" className={th}>Now /g</th>
            <th scope="col" className={th}>Gain</th>
            <th scope="col" className={th}>Month</th>
          </tr>
        </thead>
        <tbody>
          {items.map((it) => (
            <tr key={it.id} className="border-t border-line">
              <td className="px-3 py-2.5">
                <p className="text-chalk">{it.name}</p>
                <p className="text-xs text-muted">
                  {it.purity} · <span className="sm:hidden">{fmt(it.weight_grams, 2)} g @ {fmt(it.paid_per_gram, 3)}</span>
                  <span className="hidden sm:inline">{fmtDate(it.purchased_on)}</span>
                </p>
              </td>
              <td className="hidden px-3 py-2.5 text-right font-mono text-chalk sm:table-cell">{fmt(it.weight_grams, 2)}</td>
              <td className="hidden px-3 py-2.5 text-right font-mono text-muted sm:table-cell">{fmt(it.paid_per_gram, 3)}</td>
              <td className="px-3 py-2.5 text-right font-mono text-chalk">{it.now_per_gram != null ? fmt(it.now_per_gram, 3) : '—'}</td>
              <td className="px-3 py-2.5 text-right"><Signed value={it.gain} /></td>
              <td className="px-3 py-2.5 text-right"><Signed value={it.month_gain} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Report({ report, open, onToggle }) {
  const d = report.data;
  return (
    <Card
      title={monthLabel(report.month)}
      actions={
        <div className="flex items-center gap-2">
          {report.discord_sent_at ? (
            <span className="andon andon-ok">Sent</span>
          ) : (
            <span className="andon andon-idle">Not sent</span>
          )}
          <button
            onClick={onToggle}
            aria-expanded={open}
            className="min-h-[36px] rounded-chip px-2 font-display text-[11px] font-semibold uppercase tracking-stamp text-muted hover:text-chalk"
          >
            {open ? 'Hide' : 'Show'}
          </button>
        </div>
      }
    >
      <Totals data={d} />
      {open && (
        <div className="mt-5 space-y-5">
          {d.metals?.length > 0 && (
            <ul className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted">
              {d.metals.map((m) => (
                <li key={m.metal}>
                  <span className="capitalize text-chalk">{m.metal}</span> {fmt(m.open, 3)} → {fmt(m.close, 3)}{' '}
                  <Signed value={m.change_pct} />%
                </li>
              ))}
            </ul>
          )}
          <ItemsTable items={d.items || []} />
          {report.analysis ? (
            <div className="rounded-lg border border-line bg-ink-sunken p-4">
              <p className="stamp mb-2">AI analysis{report.model ? ` · ${report.model}` : ''}</p>
              <Markdown text={report.analysis} />
            </div>
          ) : null}
          {report.error && (
            <p className="rounded-chip border border-warn/30 border-l-4 border-l-warn bg-warn/10 px-3 py-2 text-xs text-warn-bright">
              {report.error}
            </p>
          )}
          <p className="text-xs text-muted">Generated {fmtDate(report.created_at)}</p>
        </div>
      )}
    </Card>
  );
}

export function ReportsPage() {
  const [reports, setReports] = useState(null);
  const [status, setStatus] = useState(null);
  const [error, setError] = useState(null);
  const [openMonth, setOpenMonth] = useState(null);
  const poll = useRef(null);

  const load = useCallback(async () => {
    try {
      const [list, st] = await Promise.all([apiRequest('/api/reports'), apiRequest('/api/reports/status')]);
      setReports(list);
      setStatus(st);
      return st;
    } catch (err) {
      setError(err.message);
      return null;
    }
  }, []);

  const startPolling = useCallback(() => {
    clearInterval(poll.current);
    poll.current = setInterval(async () => {
      const st = await load();
      if (st && !st.running) {
        clearInterval(poll.current);
        if (st.last_error) setError(st.last_error);
      }
    }, POLL_MS);
  }, [load]);

  useEffect(() => {
    load().then((st) => st?.running && startPolling());
    return () => clearInterval(poll.current);
  }, [load, startPolling]);

  const generate = async (month) => {
    setError(null);
    try {
      await apiRequest('/api/reports/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(month ? { month } : {}),
      });
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 409)) {
        setError(err.message);
        return;
      }
    }
    setStatus((s) => ({ ...s, running: true }));
    startPolling();
  };

  const running = status?.running;
  const thisMonth = new Date().toISOString().slice(0, 7);
  const shownOpen = openMonth ?? reports?.[0]?.month;

  return (
    <div className="mx-auto max-w-4xl space-y-5">
      <Card title="Monthly report">
        <p className="text-sm text-muted">
          A report for last month is generated on the 1st
          {status?.discord ? ' and posted to Discord' : ''}
          {status?.ai ? ', with an AI review and next steps' : ''}.
          {status && !status.discord && ' Set DISCORD_WEBHOOK_URL on the API to post it to Discord.'}
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <Button onClick={() => generate()} disabled={running}>
            {running ? 'Generating… (a few minutes)' : 'Generate last month'}
          </Button>
          <Button variant="secondary" onClick={() => generate(thisMonth)} disabled={running}>
            This month so far
          </Button>
        </div>
        {error && <p className="mt-3 text-xs text-bad-bright">{error}</p>}
      </Card>

      {reports === null ? (
        <Skeleton className="h-40 w-full" />
      ) : reports.length === 0 ? (
        <Card>
          <EmptyState title="No reports yet" description="Generate one now, or wait for the 1st of the month." />
        </Card>
      ) : (
        reports.map((r) => (
          <Report
            key={r.month}
            report={r}
            open={shownOpen === r.month}
            onToggle={() => setOpenMonth(shownOpen === r.month ? '' : r.month)}
          />
        ))
      )}
    </div>
  );
}
