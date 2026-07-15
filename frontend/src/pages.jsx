import React from 'react';
import { api, measureDownload, measureUpload } from './api.js';
import { Card, KeyValues, StatusBadge, Loading, ErrorBox, useAsync } from './components.jsx';

export function InfoPage() {
  const { loading, error, data } = useAsync(api.info);
  if (loading) return <Loading />;
  if (error) return <ErrorBox message={error} />;
  return (
    <Card title="Client & Session">
      <KeyValues
        data={{
          'Public IP': data.public_ip,
          'Remote IP': data.remote_ip,
          Hostname: data.server_hostname,
          Browser: data.browser?.browser,
          OS: data.browser?.os,
          Mobile: data.browser?.mobile,
          Time: data.time,
          Timezone: data.timezone,
          Language: data.language,
          Protocol: data.protocol,
          HTTPS: data.https,
          'TLS Version': data.tls_version,
          'User Agent': data.user_agent,
        }}
      />
    </Card>
  );
}

export function NetworkPage() {
  const { loading, error, data } = useAsync(api.network);
  if (loading) return <Loading />;
  if (error) return <ErrorBox message={error} />;
  return (
    <>
      <Card title="Network">
        <KeyValues
          data={{
            Hostname: data.hostname,
            'Private IPv4': data.private_ipv4,
            'Private IPv6': data.private_ipv6,
            'DNS Servers': data.dns_servers,
            'DNS Suffix': data.dns_suffix,
            'Default Gateway': data.default_gateway,
            'Dual Stack': data.dual_stack,
          }}
        />
      </Card>
      <Card title="Interfaces (NICs)">
        {(data.nics || []).map((nic) => (
          <div key={nic.name} className="mb-3 last:mb-0">
            <div className="font-medium">{nic.name}</div>
            <KeyValues
              data={{
                MAC: nic.mac,
                MTU: nic.mtu,
                Up: nic.up,
                Addresses: nic.addresses,
              }}
            />
          </div>
        ))}
      </Card>
    </>
  );
}

export function ConnectivityPage() {
  const [targets, setTargets] = React.useState([]);
  const [results, setResults] = React.useState([]);
  const [error, setError] = React.useState(null);
  const [form, setForm] = React.useState({ name: '', host: '', port: '', type: 'tcp' });

  const refresh = React.useCallback(() => {
    api.listTargets().then(setTargets).catch((e) => setError(e.message));
  }, []);
  React.useEffect(refresh, [refresh]);

  const runProbes = () => {
    api.connectivity().then((r) => setResults(r.results || [])).catch((e) => setError(e.message));
  };

  const add = (e) => {
    e.preventDefault();
    api
      .addTarget({ ...form, port: form.port ? Number(form.port) : 0 })
      .then(() => {
        setForm({ name: '', host: '', port: '', type: 'tcp' });
        refresh();
      })
      .catch((err) => setError(err.message));
  };

  const remove = (id) => api.deleteTarget(id).then(refresh);

  const types = ['tcp', 'dns', 'https', 'ldap', 'ldaps', 'ssh', 'rdp', 'winrm', 'smtp', 'imap', 'pop3', 'mssql', 'oracle', 'postgres', 'smb', 'kerberos', 'ntp'];

  return (
    <>
      {error && <ErrorBox message={error} />}
      <Card
        title="Connectivity Targets"
        actions={
          <button onClick={runProbes} className="bg-brand text-white px-3 py-1.5 rounded-lg text-sm hover:bg-brand-dark">
            Run tests
          </button>
        }
      >
        <form onSubmit={add} className="grid grid-cols-2 md:grid-cols-5 gap-2 mb-4">
          <input required placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="border rounded px-2 py-1 text-sm" />
          <input required placeholder="Host" value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} className="border rounded px-2 py-1 text-sm" />
          <input placeholder="Port" value={form.port} onChange={(e) => setForm({ ...form, port: e.target.value })} className="border rounded px-2 py-1 text-sm" />
          <select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })} className="border rounded px-2 py-1 text-sm">
            {types.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          <button className="bg-slate-800 text-white rounded px-3 py-1 text-sm">Add</button>
        </form>
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-slate-500 border-b">
              <th className="py-1">Name</th><th>Host</th><th>Port</th><th>Type</th><th>Status</th><th></th>
            </tr>
          </thead>
          <tbody>
            {targets.map((t) => {
              const res = results.find((r) => r.name === t.name && r.host === t.host);
              return (
                <tr key={t.id} className="border-b border-slate-100">
                  <td className="py-1.5">{t.name}</td>
                  <td>{t.host}</td>
                  <td>{t.port || '—'}</td>
                  <td>{t.type}</td>
                  <td>{res ? <StatusBadge ok={res.ok} /> : <span className="text-slate-400">—</span>}</td>
                  <td className="text-right"><button onClick={() => remove(t.id)} className="text-rose-600 hover:underline">Delete</button></td>
                </tr>
              );
            })}
            {targets.length === 0 && (
              <tr><td colSpan="6" className="py-3 text-slate-400">No targets yet. Add DC01, Exchange, FileServer, ERP…</td></tr>
            )}
          </tbody>
        </table>
      </Card>
    </>
  );
}

export function CertificatePage() {
  const [host, setHost] = React.useState('');
  const [port, setPort] = React.useState('443');
  const [report, setReport] = React.useState(null);
  const [error, setError] = React.useState(null);

  const inspect = (e) => {
    e.preventDefault();
    setError(null);
    setReport(null);
    api.certificate(host, port).then(setReport).catch((err) => setError(err.message));
  };

  return (
    <Card title="Certificate Inspector">
      <form onSubmit={inspect} className="flex gap-2 mb-4">
        <input required placeholder="host (e.g. mail.corp)" value={host} onChange={(e) => setHost(e.target.value)} className="border rounded px-2 py-1 text-sm flex-1" />
        <input placeholder="port" value={port} onChange={(e) => setPort(e.target.value)} className="border rounded px-2 py-1 text-sm w-24" />
        <button className="bg-brand text-white px-3 py-1.5 rounded-lg text-sm">Inspect</button>
      </form>
      {error && <ErrorBox message={error} />}
      {report && report.error && <ErrorBox message={report.error} />}
      {report && report.ok && (
        <>
          <KeyValues data={{ Host: report.host, Port: report.port, 'TLS Version': report.tls_version, Cipher: report.cipher }} />
          {report.chain.map((c, i) => (
            <div key={i} className="mt-3 border-t pt-2">
              <KeyValues data={{ Subject: c.subject, Issuer: c.issuer, SAN: c.san, 'Valid To': c.not_after, 'Days Left': c.days_left, 'Is CA': c.is_ca }} />
            </div>
          ))}
        </>
      )}
    </Card>
  );
}

export function SpeedPage() {
  const [down, setDown] = React.useState(null);
  const [up, setUp] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState(null);

  const run = async () => {
    setBusy(true);
    setError(null);
    try {
      setDown(await measureDownload());
      setUp(await measureUpload());
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card
      title="Speed Test (server-side, no internet)"
      actions={
        <button onClick={run} disabled={busy} className="bg-brand text-white px-3 py-1.5 rounded-lg text-sm disabled:opacity-50">
          {busy ? 'Running…' : 'Start'}
        </button>
      }
    >
      {error && <ErrorBox message={error} />}
      <div className="grid grid-cols-2 gap-4">
        <Metric label="Download" value={down} />
        <Metric label="Upload" value={up} />
      </div>
    </Card>
  );
}

function Metric({ label, value }) {
  return (
    <div className="bg-slate-50 rounded-lg p-4 text-center">
      <div className="text-slate-500 text-sm">{label}</div>
      <div className="text-3xl font-bold text-brand">{value != null ? value.toFixed(1) : '—'}</div>
      <div className="text-slate-400 text-xs">Mbps</div>
    </div>
  );
}

export function HealthPage() {
  const { loading, error, data } = useAsync(api.health);
  if (loading) return <Loading />;
  if (error) return <ErrorBox message={error} />;
  const color = data.status === 'ok' ? 'text-emerald-600' : data.status === 'down' ? 'text-rose-600' : 'text-amber-600';
  return (
    <Card title="Health Dashboard">
      <div className="mb-4">
        <span className="text-slate-500 text-sm">Overall status: </span>
        <span className={`font-bold uppercase ${color}`}>{data.status}</span>
        <span className="text-slate-400 text-sm"> ({data.up}/{data.total} up)</span>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
        {(data.checks || []).map((c, i) => (
          <div key={i} className="flex items-center justify-between bg-slate-50 rounded-lg px-3 py-2">
            <span className="truncate">{c.name}</span>
            <StatusBadge ok={c.ok} />
          </div>
        ))}
        {(!data.checks || data.checks.length === 0) && (
          <div className="text-slate-400 text-sm col-span-3">No checks configured — add targets on the Connectivity page.</div>
        )}
      </div>
    </Card>
  );
}

export function ReportPage() {
  const qr = api.qrURL(`${window.location.origin}/api/report?format=html`);
  return (
    <>
      <Card title="Report">
        <p className="text-sm text-slate-500 mb-3">Generate a full diagnostics report in your preferred format.</p>
        <div className="flex gap-2">
          <a href={api.reportURL('json')} target="_blank" rel="noreferrer" className="bg-slate-800 text-white px-3 py-1.5 rounded-lg text-sm">JSON</a>
          <a href={api.reportURL('html')} target="_blank" rel="noreferrer" className="bg-slate-800 text-white px-3 py-1.5 rounded-lg text-sm">HTML</a>
          <a href={api.reportURL('pdf')} target="_blank" rel="noreferrer" className="bg-brand text-white px-3 py-1.5 rounded-lg text-sm">PDF</a>
        </div>
      </Card>
      <Card title="QR Code">
        <p className="text-sm text-slate-500 mb-3">Scan to open the HTML report on another device.</p>
        <img src={qr} alt="Report QR code" width="200" height="200" className="border rounded-lg" />
      </Card>
    </>
  );
}
