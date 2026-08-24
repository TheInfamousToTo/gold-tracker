import { HoldingRow } from './HoldingRow.jsx';
import { Card } from '../ui/Card.jsx';
import { EmptyState } from '../ui/EmptyState.jsx';
import { Skeleton } from '../ui/Skeleton.jsx';

const COLUMNS = [
  { label: 'Item', align: 'left' },
  { label: 'Purity', align: 'left' },
  { label: 'Mass (g)', align: 'right' },
  { label: 'Entry BHD/g', align: 'right' },
  { label: 'Paid', align: 'right' },
  { label: 'Value', align: 'right' },
  { label: 'Gain / loss', align: 'right' },
  { label: '', align: 'right' },
];

export function HoldingsTable({ items, loading, onEdit, onDelete, onAddFirst }) {
  return (
    <Card title="Holdings" padded={false}>
      {loading ? (
        <div className="space-y-3 p-5">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          title="No holdings yet"
          description="Record a purchase and this table will track it against the current spot price."
          action={{ label: 'Add a purchase', onClick: onAddFirst }}
        />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr>
                {COLUMNS.map((col, i) => (
                  <th
                    key={i}
                    scope="col"
                    className={`stamp px-5 pb-2 pt-4 ${col.align === 'right' ? 'text-right' : 'text-left'}`}
                  >
                    {col.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <HoldingRow key={item.id} item={item} onEdit={onEdit} onDelete={onDelete} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
