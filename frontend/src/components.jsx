import React from 'react';

export function Card({ title, children, actions }) {
  return (
    <section className="bg-white rounded-xl shadow-sm border border-slate-200 p-5 mb-5">
      {(title || actions) && (
        <div className="flex items-center justify-between mb-3">
          {title && <h2 className="text-lg font-semibold">{title}</h2>}
          {actions}
        </div>
      )}
      {children}
    </section>
  );
}

export function KeyValues({ data }) {
  const entries = Object.entries(data || {});
  return (
    <table className="w-full text-sm">
      <tbody>
        {entries.map(([k, v]) => (
          <tr key={k} className="border-b border-slate-100 last:border-0">
            <td className="py-1.5 pr-4 font-medium text-slate-500 align-top w-56">{k}</td>
            <td className="py-1.5 break-all">{render(v)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function render(v) {
  if (v === null || v === undefined || v === '') return <span className="text-slate-400">—</span>;
  if (Array.isArray(v)) return v.length ? v.join(', ') : <span className="text-slate-400">—</span>;
  if (typeof v === 'boolean') return v ? 'Yes' : 'No';
  if (typeof v === 'object') return JSON.stringify(v);
  return String(v);
}

export function StatusBadge({ ok }) {
  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold ${
        ok ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'
      }`}
    >
      <span className={`w-2 h-2 rounded-full ${ok ? 'bg-emerald-500' : 'bg-rose-500'}`} />
      {ok ? 'UP' : 'DOWN'}
    </span>
  );
}

export function Loading() {
  return <div className="text-slate-400 text-sm">Loading…</div>;
}

export function ErrorBox({ message }) {
  return <div className="text-rose-600 text-sm">Error: {message}</div>;
}

// Generic hook-free async loader used by simple read-only pages.
export function useAsync(fn, deps = []) {
  const [state, setState] = React.useState({ loading: true, error: null, data: null });
  React.useEffect(() => {
    let alive = true;
    setState({ loading: true, error: null, data: null });
    fn()
      .then((data) => alive && setState({ loading: false, error: null, data }))
      .catch((err) => alive && setState({ loading: false, error: err.message, data: null }));
    return () => {
      alive = false;
    };
  }, deps);
  return state;
}
