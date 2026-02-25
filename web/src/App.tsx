import { useState } from 'react';
import type { ScanResult, AnalysisReport } from './types';
import Detect from './pages/Detect';
import Analyze from './pages/Analyze';
import Migrate from './pages/Migrate';
import Validate from './pages/Validate';

type Page = 'detect' | 'analyze' | 'migrate' | 'validate';

const navItems: { id: Page; label: string; icon: string; desc: string; step: number }[] = [
  { id: 'detect',   label: 'Detect',   icon: '🔍', desc: 'Scan cluster',          step: 1 },
  { id: 'analyze',  label: 'Analyze',  icon: '🔬', desc: 'Check compatibility',   step: 2 },
  { id: 'migrate',  label: 'Migrate',  icon: '🚀', desc: 'Generate files',        step: 3 },
  { id: 'validate', label: 'Validate', icon: '✅', desc: 'Verify migration',       step: 4 },
];

export default function App() {
  const [page, setPage] = useState<Page>('detect');
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [analysisReport, setAnalysisReport] = useState<AnalysisReport | null>(null);

  const pageIndex = navItems.findIndex(n => n.id === page);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      {/* Top header */}
      <header className="border-b border-slate-800 bg-slate-950/80 backdrop-blur-sm sticky top-0 z-20">
        <div className="max-w-screen-xl mx-auto px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center text-sm font-bold shadow-lg shadow-blue-900/40">
              ⇄
            </div>
            <div>
              <span className="font-bold text-white tracking-tight">ing-switch</span>
              <span className="text-slate-500 text-xs ml-2 hidden sm:inline">Kubernetes Ingress Migration</span>
            </div>
          </div>
          <div className="flex items-center gap-4 text-xs">
            <span className="text-slate-600 hidden md:inline">v1.0.0</span>
            <a
              href="https://github.com/saiyam1814/ing-switch"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-400 hover:text-blue-400 transition-colors flex items-center gap-1"
            >
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
              </svg>
              GitHub
            </a>
          </div>
        </div>
      </header>

      {/* Retirement warning banner */}
      <div className="bg-gradient-to-r from-red-900/60 via-red-800/60 to-red-900/60 border-b border-red-700/40 text-center py-2 px-4">
        <span className="text-red-200 text-xs font-medium">
          ⚠ Ingress NGINX retires <strong className="text-red-100">March 2026</strong>
          <span className="text-red-300 mx-2">·</span>
          Migrate now to avoid security vulnerabilities and loss of support
        </span>
      </div>

      <div className="max-w-screen-xl mx-auto flex" style={{ minHeight: 'calc(100vh - 88px)' }}>
        {/* Sidebar */}
        <aside className="w-56 flex-shrink-0 border-r border-slate-800 py-6 px-3 sticky top-14 self-start" style={{ height: 'calc(100vh - 88px)', overflowY: 'auto' }}>
          <nav className="space-y-1">
            {navItems.map((item, i) => {
              const isActive = item.id === page;
              const isCompleted = i < pageIndex;
              const isAccessible = i <= pageIndex + 1;
              return (
                <button
                  key={item.id}
                  onClick={() => isAccessible && setPage(item.id)}
                  disabled={!isAccessible}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-left transition-all group ${
                    isActive
                      ? 'bg-blue-500/15 border border-blue-500/30 text-blue-300 shadow-lg shadow-blue-900/20'
                      : isCompleted
                      ? 'text-emerald-400 hover:bg-slate-800/60 border border-transparent'
                      : isAccessible
                      ? 'text-slate-400 hover:bg-slate-800/60 hover:text-slate-200 border border-transparent'
                      : 'text-slate-700 cursor-not-allowed border border-transparent'
                  }`}
                >
                  <div className={`flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold ${
                    isActive ? 'bg-blue-500/20 text-blue-400' :
                    isCompleted ? 'bg-emerald-500/20 text-emerald-400' :
                    isAccessible ? 'bg-slate-700/60 text-slate-400 group-hover:bg-slate-700' :
                    'bg-slate-800/40 text-slate-700'
                  }`}>
                    {isCompleted && !isActive ? '✓' : item.step}
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-medium leading-none">{item.label}</div>
                    <div className={`text-xs mt-0.5 ${isActive ? 'text-blue-400/70' : 'text-slate-600'}`}>{item.desc}</div>
                  </div>
                  {isActive && (
                    <div className="ml-auto w-1.5 h-1.5 rounded-full bg-blue-400 flex-shrink-0" />
                  )}
                </button>
              );
            })}
          </nav>

          {/* Cluster info */}
          {scanResult && (
            <div className="mt-6 mx-1 p-3 rounded-xl bg-slate-800/60 border border-slate-700/40">
              <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Connected</div>
              <div className="text-xs text-slate-300 font-mono truncate">{scanResult.clusterName}</div>
              <div className="flex items-center justify-between mt-2">
                <span className="text-xs text-slate-500">{scanResult.ingresses.length} ingresses</span>
                {scanResult.controller.detected && (
                  <span className="text-xs text-red-400 bg-red-500/10 px-1.5 py-0.5 rounded">retiring</span>
                )}
              </div>
              {scanResult.controller.detected && (
                <div className="mt-2 text-xs font-mono text-slate-400 truncate">{scanResult.controller.type}</div>
              )}
            </div>
          )}

          {/* Progress bar */}
          <div className="mt-6 mx-1">
            <div className="flex justify-between text-xs text-slate-600 mb-1.5">
              <span>Progress</span>
              <span>{pageIndex + 1}/4</span>
            </div>
            <div className="h-1 bg-slate-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-blue-500 to-indigo-500 rounded-full transition-all duration-500"
                style={{ width: `${((pageIndex + 1) / 4) * 100}%` }}
              />
            </div>
          </div>
        </aside>

        {/* Main content */}
        <main className="flex-1 min-w-0 p-6 lg:p-8">
          {page === 'detect' && (
            <Detect
              onScanComplete={(r) => { setScanResult(r); setPage('analyze'); }}
              scanResult={scanResult}
            />
          )}
          {page === 'analyze' && (
            <Analyze
              scanResult={scanResult}
              onAnalysisComplete={(r) => { setAnalysisReport(r); setPage('migrate'); }}
              analysisReport={analysisReport}
            />
          )}
          {page === 'migrate' && (
            <Migrate scanResult={scanResult} analysisReport={analysisReport} />
          )}
          {page === 'validate' && (
            <Validate scanResult={scanResult} />
          )}
        </main>
      </div>
    </div>
  );
}
