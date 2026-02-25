import { useState } from 'react';
import { api } from '../api/client';
import type { ScanResult } from '../types';
import IngressTable from '../components/IngressTable';

interface Props {
  onScanComplete: (result: ScanResult) => void;
  scanResult: ScanResult | null;
}

const Detect = ({ onScanComplete, scanResult }: Props) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [namespace, setNamespace] = useState('');

  const handleScan = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.scan(namespace || undefined);
      onScanComplete(result);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Scan failed');
    } finally {
      setLoading(false);
    }
  };

  const summary = scanResult
    ? {
        simple: scanResult.ingresses.filter(i => i.complexity === 'simple').length,
        complex: scanResult.ingresses.filter(i => i.complexity === 'complex').length,
        unsupported: scanResult.ingresses.filter(i => i.complexity === 'unsupported').length,
      }
    : null;

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Detect</h1>
        <p className="text-slate-400 mt-1 text-sm">
          Scan your Kubernetes cluster to discover Ingress resources and identify the current controller.
        </p>
      </div>

      {/* Scan card */}
      <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-5">
        <div className="flex items-center gap-2 mb-4">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Cluster Scan</span>
        </div>
        <div className="flex gap-3 items-end flex-wrap">
          <div className="flex-1 min-w-[180px] max-w-xs">
            <label className="block text-xs font-medium text-slate-500 mb-1.5">Namespace <span className="text-slate-600">(optional)</span></label>
            <input
              type="text"
              value={namespace}
              onChange={e => setNamespace(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && !loading && handleScan()}
              placeholder="All namespaces"
              className="w-full bg-slate-900 border border-slate-700 text-slate-200 placeholder-slate-600 rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-colors"
            />
          </div>
          <button
            onClick={handleScan}
            disabled={loading}
            className="px-5 py-2 bg-gradient-to-r from-blue-600 to-blue-500 hover:from-blue-500 hover:to-blue-400 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-semibold rounded-lg transition-all shadow-lg shadow-blue-900/30 flex items-center gap-2"
          >
            {loading ? (
              <>
                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Scanning...
              </>
            ) : (
              <>🔍 Scan Cluster</>
            )}
          </button>
        </div>

        {error && (
          <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300">
            <div className="font-medium">Error: {error}</div>
            <div className="mt-1 text-xs text-red-400/70">
              Make sure ing-switch has cluster access via --kubeconfig flag or KUBECONFIG env var.
            </div>
          </div>
        )}
      </div>

      {/* Results */}
      {scanResult && (
        <>
          {/* Info cards */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {/* Cluster */}
            <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-4">
              <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1.5">Cluster</div>
              <div className="text-base font-bold text-white truncate">{scanResult.clusterName}</div>
            </div>

            {/* Controller */}
            <div className={`rounded-xl border p-4 ${
              scanResult.controller.detected && scanResult.controller.type === 'ingress-nginx'
                ? 'border-red-500/30 bg-red-500/5'
                : 'border-slate-700/50 bg-slate-800/40'
            }`}>
              <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1.5">Ingress Controller</div>
              {scanResult.controller.detected ? (
                <>
                  <div className="text-base font-bold text-white">{scanResult.controller.type}</div>
                  <div className="text-xs text-slate-500 mt-0.5 font-mono">
                    v{scanResult.controller.version} · {scanResult.controller.namespace}
                  </div>
                  {scanResult.controller.type === 'ingress-nginx' && (
                    <div className="mt-2 flex items-center gap-1.5">
                      <span className="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />
                      <span className="text-xs text-red-400 font-medium">Retiring March 2026</span>
                    </div>
                  )}
                </>
              ) : (
                <div className="text-slate-500 text-sm">Not detected</div>
              )}
            </div>

            {/* Namespaces */}
            <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-4">
              <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1.5">Namespaces</div>
              <div className="text-base font-bold text-white">{scanResult.namespaces.length}</div>
              <div className="text-xs text-slate-500 mt-0.5 truncate">
                {scanResult.namespaces.slice(0, 3).join(', ')}
                {scanResult.namespaces.length > 3 && ` +${scanResult.namespaces.length - 3}`}
              </div>
            </div>
          </div>

          {/* Complexity summary */}
          {summary && (
            <div className="grid grid-cols-3 gap-4">
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4">
                <div className="text-3xl font-bold text-emerald-400">{summary.simple}</div>
                <div className="text-sm font-medium text-emerald-400 mt-0.5">Simple</div>
                <div className="text-xs text-slate-500 mt-0.5">Ready to migrate</div>
              </div>
              <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4">
                <div className="text-3xl font-bold text-amber-400">{summary.complex}</div>
                <div className="text-sm font-medium text-amber-400 mt-0.5">Complex</div>
                <div className="text-xs text-slate-500 mt-0.5">Needs middleware / policy</div>
              </div>
              <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-4">
                <div className="text-3xl font-bold text-red-400">{summary.unsupported}</div>
                <div className="text-sm font-medium text-red-400 mt-0.5">Unsupported</div>
                <div className="text-xs text-slate-500 mt-0.5">Manual intervention needed</div>
              </div>
            </div>
          )}

          {/* Ingress table */}
          <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 overflow-hidden">
            <div className="px-5 py-3.5 border-b border-slate-700/50 flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                Ingress Resources
              </span>
              <span className="text-xs text-slate-600">{scanResult.ingresses.length} found</span>
            </div>
            <IngressTable ingresses={scanResult.ingresses} />
          </div>

          {/* Next step */}
          <div className="rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-center gap-3">
            <span className="text-blue-400 text-lg flex-shrink-0">→</span>
            <p className="text-sm text-blue-300">
              Scan complete. Head to <strong className="text-blue-200">Analyze</strong> to check annotation compatibility with your target controller.
            </p>
          </div>
        </>
      )}
    </div>
  );
};

export default Detect;
