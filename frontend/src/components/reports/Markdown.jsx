/**
 * Renders the small Markdown subset the report prompt asks for:
 * ## headings, - or 1. list items, **bold**, and paragraphs. Text is
 * rendered as React children, never as HTML.
 */
function inline(text) {
  return text.split(/(\*\*[^*]+\*\*)/g).map((part, i) =>
    part.startsWith('**') && part.endsWith('**') ? (
      <strong key={i} className="font-semibold text-chalk">{part.slice(2, -2)}</strong>
    ) : (
      part
    ),
  );
}

export function Markdown({ text }) {
  const blocks = [];
  let list = null;
  text.split('\n').forEach((raw, i) => {
    const line = raw.trim();
    const item = line.match(/^(?:[-*]|\d+[.)])\s+(.*)$/);
    if (item) {
      if (!list) blocks.push((list = { type: 'list', items: [], key: i }));
      list.items.push(item[1]);
      return;
    }
    list = null;
    if (!line) return;
    const heading = line.match(/^#{1,4}\s+(.*)$/);
    blocks.push(heading ? { type: 'h', text: heading[1], key: i } : { type: 'p', text: line, key: i });
  });

  return (
    <div className="space-y-2 text-sm leading-relaxed text-chalk/90">
      {blocks.map((b) =>
        b.type === 'h' ? (
          <h3 key={b.key} className="stamp pt-2 text-chalk">{inline(b.text)}</h3>
        ) : b.type === 'list' ? (
          <ul key={b.key} className="list-disc space-y-1 pl-5">
            {b.items.map((t, j) => <li key={j}>{inline(t)}</li>)}
          </ul>
        ) : (
          <p key={b.key}>{inline(b.text)}</p>
        ),
      )}
    </div>
  );
}
