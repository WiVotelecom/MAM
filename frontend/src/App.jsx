import React from 'react';
import {
  InfoPage,
  NetworkPage,
  ConnectivityPage,
  CertificatePage,
  SpeedPage,
  HealthPage,
  ReportPage,
} from './pages.jsx';

const TABS = [
  { id: 'info', label: 'Info', component: InfoPage },
  { id: 'network', label: 'Network', component: NetworkPage },
  { id: 'connectivity', label: 'Connectivity', component: ConnectivityPage },
  { id: 'certificate', label: 'Certificate', component: CertificatePage },
  { id: 'speed', label: 'Speed', component: SpeedPage },
  { id: 'health', label: 'Health', component: HealthPage },
  { id: 'report', label: 'Report / QR', component: ReportPage },
];

export default function App() {
  const [active, setActive] = React.useState('info');
  const Active = TABS.find((t) => t.id === active).component;

  return (
    <div className="min-h-screen">
      <header className="bg-brand-dark text-white">
        <div className="max-w-5xl mx-auto px-4 py-4 flex items-center gap-3">
          <div className="text-2xl font-bold tracking-tight">NetInsight</div>
          <span className="text-blue-200 text-sm">Agentless network & enterprise diagnostics</span>
        </div>
        <nav className="max-w-5xl mx-auto px-4 flex gap-1 overflow-x-auto">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setActive(t.id)}
              className={`px-3 py-2 text-sm font-medium border-b-2 whitespace-nowrap ${
                active === t.id ? 'border-white text-white' : 'border-transparent text-blue-200 hover:text-white'
              }`}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>
      <main className="max-w-5xl mx-auto px-4 py-6">
        <Active />
      </main>
      <footer className="max-w-5xl mx-auto px-4 py-6 text-center text-slate-400 text-xs">
        NetInsight — works fully offline / air-gapped
      </footer>
    </div>
  );
}
