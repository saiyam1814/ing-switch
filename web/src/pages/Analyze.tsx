import { useState } from 'react';
import { api } from '../api/client';
import type { ScanResult, AnalysisReport, Target } from '../types';
import AnnotationMatrix from '../components/AnnotationMatrix';
import DependencyGraph from '../components/DependencyGraph';

interface Props {
  scanResult: ScanResult | null;
  onAnalysisComplete: (report: AnalysisReport) => void;
  analysisReport: AnalysisReport | null;
}

const targetInfo: Record<Target, { icon: string; name: string; badge: string; badgeColor: string }> = {
  traefik: {
    icon: '🚀',
    name: 'Traefik v3',
    badge: 'Lowest friction',
    badgeColor: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20',
  },
  'gateway-api': {
    icon: '🌐',
    name: 'Gateway API',
    badge: 'Future-proof standard',
    badgeColor: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
  },
};

const Analyze = ({ scanResult, onAnalysisComplete, analysisReport }: Props) => {
  const [target, setTarget] = useState<Target>('traefik');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [view, setView] = useState<'matrix' | 'graph'>('matrix');

  const handleAnalyze = async () => {
    setLoading(true);
    setError(null);
    try {
      const report = await api.analyze(target);
      onAnalysisComplete(report);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Analysis failed');
    } finally {
      setLoading(false);
    }
  };

  if (!scanResult) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-slate-500">
        <div className="w-16 h-16 rounded-2xl bg-slate-800 flex items-center justify-center mb-4 text-3xl">🔍</div>
        <p className="text-base font-medium text-slate-400">No scan data</p>
        <p className="text-sm mt-1 text-slate-600">Run a cluster scan on the Detect tab first.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Analyze</h1>
        <p className="text-slate-400 mt-1 text-sm">
          See how each nginx annotation maps to your target controller and identify migration gaps.
        </p>
      </div>

      {/* Target selector */}
      <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-5">
        <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Target Controller</div>
        <div className="flex gap-3 flex-wrap items-start">
          <div className="flex gap-3 flex-wrap">
            {(['traefik', 'gateway-api'] as Target[]).map(t => {
              const info = targetInfo[t];
              const isSelected = target === t;
              return (
                <button
                  key={t}
                  onClick={() => setTarget(t)}
                  className={`group flex items-start gap-3 px-4 py-3 rounded-xl border-2 transition-all text-left ${
                    isSelected
                      ? 'border-blue-500/50 bg-blue-500/10'
                      : 'border-slate-700/50 bg-slate-900/50 hover:border-slate-600/60 hover:bg-slate-800/60'
                  }`}
                >
                  <span className="text-xl">{info.icon}</span>
                  <div>
                    <div className={`text-sm font-semibold ${isSelected ? 'text-blue-300' : 'text-slate-300'}`}>
                      {info.name}
                    </div>
                    <div className={`text-xs mt-0.5 px-1.5 py-0.5 rounded border inline-block ${info.badgeColor}`}>
                      {info.badge}
                    </div>
                  </div>
                  {isSelected && (
                    <div className="ml-auto flex-shrink-0 w-4 h-4 rounded-full border-2 border-blue-400 flex items-center justify-center">
                      <div className="w-2 h-2 rounded-full bg-blue-400" />
                    </div>
                  )}
                </button>
              );
            })}
          </div>

          <button
            onClick={handleAnalyze}
            disabled={loading}
            className="ml-auto px-5 py-2.5 bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-semibold rounded-lg transition-all shadow-lg shadow-purple-900/30 flex items-center gap-2 self-start"
          >
            {loading ? (
              <>
                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Analyzing...
              </>
            ) : '🔬 Analyze Compatibility'}
          </button>
        </div>

        {error && (
          <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300">
            {error}
          </div>
        )}
      </div>

      {analysisReport && analysisReport.target === target && (
        <>
          {/* Summary metric cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {[
              { label: 'Total',       value: analysisReport.summary.total,            color: 'slate',   icon: '📋' },
              { label: 'Ready',       value: analysisReport.summary.fullyCompatible,  color: 'emerald', icon: '✓' },
              { label: 'Workarounds', value: analysisReport.summary.needsWorkaround,  color: 'amber',   icon: '⚠' },
              { label: 'Breaking',    value: analysisReport.summary.hasUnsupported,   color: 'red',     icon: '✗' },
            ].map(({ label, value, color, icon }) => (
              <div
                key={label}
                className={`rounded-xl border p-4 ${
                  color === 'slate'   ? 'border-slate-700/50 bg-slate-800/40' :
                  color === 'emerald' ? 'border-emerald-500/20 bg-emerald-500/5' :
                  color === 'amber'   ? 'border-amber-500/20 bg-amber-500/5' :
                                        'border-red-500/20 bg-red-500/5'
                }`}
              >
                <div className={`text-3xl font-bold ${
                  color === 'slate'   ? 'text-slate-200' :
                  color === 'emerald' ? 'text-emerald-400' :
                  color === 'amber'   ? 'text-amber-400' :
                                        'text-red-400'
                }`}>{value}</div>
                <div className={`text-xs mt-1 flex items-center gap-1 ${
                  color === 'slate'   ? 'text-slate-500' :
                  color === 'emerald' ? 'text-emerald-500' :
                  color === 'amber'   ? 'text-amber-500' :
                                        'text-red-500'
                }`}>
                  <span>{icon}</span>
                  {label}
                </div>
              </div>
            ))}
          </div>

          {/* View toggle */}
          <div className="flex gap-1 bg-slate-800/60 border border-slate-700/50 rounded-lg p-1 w-fit">
            {(['matrix', 'graph'] as const).map(v => (
              <button
                key={v}
                onClick={() => setView(v)}
                className={`px-4 py-1.5 text-xs font-medium rounded-md transition-all ${
                  view === v
                    ? 'bg-slate-700 text-slate-200 shadow-sm'
                    : 'text-slate-500 hover:text-slate-300'
                }`}
              >
                {v === 'matrix' ? '📊 Annotation Matrix' : '🗺 Dependency Graph'}
              </button>
            ))}
          </div>

          {/* Content panel */}
          <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 overflow-hidden">
            <div className="px-5 py-3.5 border-b border-slate-700/50">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                {view === 'matrix' ? `Annotation Compatibility — ${target}` : 'Ingress Dependency Graph'}
              </span>
            </div>
            <div className="p-5">
              {view === 'matrix' ? (
                <AnnotationMatrix reports={analysisReport.ingressReports ?? []} />
              ) : (
                <DependencyGraph ingresses={scanResult.ingresses} />
              )}
            </div>
          </div>

          {/* Next step */}
          <div className="rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-center gap-3">
            <span className="text-blue-400 text-lg flex-shrink-0">→</span>
            <p className="text-sm text-blue-300">
              Analysis complete. Head to <strong className="text-blue-200">Migrate</strong> to generate migration files for{' '}
              <strong className="text-blue-200">{target}</strong>.
            </p>
          </div>
        </>
      )}
    </div>
  );
};

export default Analyze;
