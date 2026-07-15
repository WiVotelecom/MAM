// Thin fetch wrappers around the NetInsight API.

async function json(path, opts) {
  const res = await fetch(path, opts);
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body.error) msg = body.error;
    } catch {
      // ignore parse failure
    }
    throw new Error(msg);
  }
  return res.json();
}

export const api = {
  info: () => json('/api/info'),
  network: () => json('/api/network'),
  browser: () => json('/api/browser'),
  health: () => json('/api/health'),
  connectivity: () => json('/api/connectivity'),
  certificate: (host, port) =>
    json(`/api/certificate?host=${encodeURIComponent(host)}&port=${port || ''}`),
  listTargets: () => json('/api/targets'),
  addTarget: (t) =>
    json('/api/targets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(t),
    }),
  deleteTarget: (id) => fetch(`/api/targets/${id}`, { method: 'DELETE' }),
  reportURL: (format) => `/api/report?format=${format}`,
  qrURL: (content) => `/api/qr?content=${encodeURIComponent(content)}`,
};

// Measure download throughput by fetching a payload of the given size (bytes).
export async function measureDownload(bytes = 8 * 1024 * 1024) {
  const start = performance.now();
  const res = await fetch(`/api/speed/download?size=${bytes}`);
  const buf = await res.arrayBuffer();
  const seconds = (performance.now() - start) / 1000;
  return (buf.byteLength * 8) / seconds / 1e6;
}

// Measure upload throughput by posting a payload of the given size (bytes).
export async function measureUpload(bytes = 8 * 1024 * 1024) {
  const payload = new Uint8Array(bytes);
  const res = await json('/api/speed/upload', { method: 'POST', body: payload });
  return res.mbps;
}
