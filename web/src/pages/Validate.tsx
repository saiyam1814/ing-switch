import { useState } from 'react';
import { api } from '../api/client';
import type { ScanResult, ValidationResult, Target } from '../types';

interface Props {
  scanResult: ScanResult | null;
}

const manualChecklist = [
  'Run verify.sh from the migration output directory',
  'Test each host with: curl -kv https://<host> --resolve <host>:443:<new-ip>',
  'Verify TLS certificates are served correctly on all HTTPS hosts',
  'Test authentication (auth-url / ForwardAuth) if configured',
  'Validate rate limiting is enforced (use a load test tool)',
  'Verify canary routing percentages if any canary ingresses exist',
  'Monitor logs for 5xx errors: kubectl logs -n <controller-ns> -l app=<controller>',
  'Monitor metrics for 24+ hours after DNS cutover before removing NGINX',
];

const phaseConfig = {
  'pre-migration': {
    icon: '🔴',
    label: 'Pre-Migration',
    desc: 'NGINX is handling all traffic.',
    color: 'text-slate-400',
    bg: 'bg-slate-700/30',
    border: 'border-slate-600/40',
    step: 1,
  },
  'migrating': {
    icon: '🟡',
    label: 'Parallel Migration',
    desc: 'Both controllers running. Traffic safe.',
    color: 'text-amber-400',
    bg: 'bg-amber-500/8',
    border: 'border-amber-500/25',
    step: 2,
  },
  'post-migration': {
    icon: '🟢',
    label: 'Migration Complete',
    desc: 'New controller active. NGINX removed.',
    color: 'text-emerald-400',
    bg: 'bg-emerald-500/8',
    border: 'border-emerald-500/25',
    step: 3,
  },
};

const checkCfg: Record<string, { border: string; bg: string; text: string; icon: string }> = {
  pass: { border: 'border-emerald-500/20', bg: 'bg-emerald-500/5',  text: 'text-emerald-400', icon: '✓' },
  warn: { border: 'border-amber-500/20',   bg: 'bg-amber-500/5',    text: 'text-amber-400',   icon: '⚠' },
  fail: { border: 'border-red-500/20',     bg: 'bg-red-500/5',      text: 'text-red-400',     icon: '✗' },
};

export default function Validate({ scanResult }: Props) {
  const [target, setTarget] = useState<Target>('traefik');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ValidationResult | null>(null);
  const [checkedItems, setCheckedItems] = useState<Set<number>>(new Set());

  const handleValidate = async () => {
    setLoading(true);
    setError(null);
    try {
      const r = await api.validate(target);
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Validation failed');
    } finally {
      setLoading(false);
    }
  };

  const toggleCheck = (i: number) => {
    setCheckedItems(prev => {
      const next = new Set(prev);
      if (next.has(i)) next.delete(i); else next.add(i);
      return next;
    });
  };

  const overallBanner = {
    pass: { border: 'border-emerald-500/30', bg: 'bg-emerald-500/8',  text: 'text-emerald-300', icon: '✓', label: 'Migration Looking Good' },
    warn: { border: 'border-amber-500/30',   bg: 'bg-amber-500/8',    text: 'text-amber-300',   icon: '⚠', label: 'Needs Attention' },
    fail: { border: 'border-red-500/30',     bg: 'bg-red-500/8',      text: 'text-red-300',     icon: '✗', label: 'Action Required' },
  };

  return (
    <div className="space-y-5 animate-fade-in">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Validate</h1>
        <p className="text-slate-400 mt-1 text-sm">
          Run live checks against the cluster to confirm your migration is progressing correctly.
        </p>
      </div>

      {/* Controls */}
      <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-5">
        <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Run Validation</div>
        <div className="flex gap-4 items-end flex-wrap">
          <div>
            <label className="block text-xs font-medium text-slate-500 mb-1.5">Target Controller</label>
            <div className="flex gap-2">
              {(['traefik', 'gateway-api'] as Target[]).map(t => (
                <button key={t} onClick={() => setTarget(t)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-all border ${
                    target === t ? 'bg-blue-500/20 text-blue-300 border-blue-500/40' : 'text-slate-400 border-slate-700 hover:border-slate-600 hover:text-slate-300 bg-slate-900/50'
                  }`}>
                  {t === 'traefik' ? '🚀 Traefik v3' : '🌐 Gateway API (Envoy)'}
                </button>
              ))}
            </div>
          </div>
          <button onClick={handleValidate} disabled={loading || !scanResult}
            className="px-5 py-2 bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-semibold rounded-lg transition-all shadow-lg shadow-purple-900/30 flex items-center gap-2">
            {loading ? (
              <><svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>Checking cluster...</>
            ) : '🔎 Run Validation'}
          </button>
        </div>

        {!scanResult && <p className="mt-3 text-xs text-slate-600">Complete the Detect step first to enable validation.</p>}

        {error && <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300">{error}</div>}
      </div>

      {result && (
        <>
          {/* Migration phase tracker */}
          <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-5">
            <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Migration Phase</div>
            <div className="flex items-center gap-0">
              {(['pre-migration', 'migrating', 'post-migration'] as const).map((phase, i) => {
                const cfg = phaseConfig[phase];
                const isActive = result.phase === phase;
                const isPast = phaseConfig[result.phase].step > cfg.step;
                return (
                  <div key={phase} className="flex items-center flex-1">
                    <div className={`flex-1 flex flex-col items-center p-3 rounded-xl transition-all ${
                      isActive ? `${cfg.bg} border ${cfg.border}` : 'opacity-40'
                    }`}>
                      <div className="text-xl mb-1">{isPast ? '✓' : cfg.icon}</div>
                      <div className={`text-xs font-semibold ${isActive ? cfg.color : 'text-slate-600'}`}>{cfg.label}</div>
                      <div className="text-xs text-slate-600 text-center mt-0.5 hidden sm:block">{cfg.desc}</div>
                    </div>
                    {i < 2 && (
                      <div className={`w-8 h-px flex-shrink-0 ${phaseConfig[result.phase].step > i + 1 ? 'bg-emerald-500' : 'bg-slate-700'}`} />
                    )}
                  </div>
                );
              })}
            </div>
            <p className="text-sm text-slate-400 mt-3 text-center">{result.phaseDesc}</p>
          </div>

          {/* Overall status + check counts */}
          {(() => {
            const cfg = overallBanner[result.overall] ?? overallBanner.warn;
            const passCt = result.checks.filter(c => c.status === 'pass').length;
            const warnCt = result.checks.filter(c => c.status === 'warn').length;
            const failCt = result.checks.filter(c => c.status === 'fail').length;
            return (
              <div className={`rounded-xl border p-4 flex items-center gap-4 ${cfg.border} ${cfg.bg}`}>
                <div className={`text-2xl font-bold w-10 h-10 rounded-xl flex items-center justify-center border flex-shrink-0 ${cfg.border} ${cfg.bg} ${cfg.text}`}>
                  {cfg.icon}
                </div>
                <div className="flex-1">
                  <div className={`font-bold text-base ${cfg.text}`}>{cfg.label}</div>
                  <div className="text-xs text-slate-500 mt-0.5">{result.checks.length} checks · target: {result.target}</div>
                </div>
                <div className="flex gap-2 flex-shrink-0">
                  {passCt > 0 && <span className="text-xs px-2 py-1 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">{passCt} ✓</span>}
                  {warnCt > 0 && <span className="text-xs px-2 py-1 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-400">{warnCt} ⚠</span>}
                  {failCt > 0 && <span className="text-xs px-2 py-1 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400">{failCt} ✗</span>}
                </div>
              </div>
            );
          })()}

          {/* Detailed checks */}
          <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 overflow-hidden">
            <div className="px-5 py-3.5 border-b border-slate-700/50">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Cluster Check Results</span>
            </div>
            <div className="p-4 space-y-2">
              {result.checks.map((check, i) => {
                const cfg = checkCfg[check.status] ?? checkCfg.warn;
                return (
                  <div key={i} className={`flex items-start gap-3 p-3.5 rounded-lg border ${cfg.bg} ${cfg.border}`}>
                    <div className={`flex-shrink-0 w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold border flex-shrink-0 ${cfg.border} ${cfg.bg} ${cfg.text}`}>
                      {cfg.icon}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-slate-200">{check.name}</div>
                      <div className="text-xs text-slate-500 mt-0.5 break-all">{check.message}</div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Next steps */}
          {result.nextSteps?.length > 0 && (
            <div className="rounded-xl border border-blue-500/20 bg-blue-500/5 overflow-hidden">
              <div className="px-5 py-3.5 border-b border-blue-500/15">
                <span className="text-xs font-semibold text-blue-400 uppercase tracking-wider">→ Next Steps</span>
              </div>
              <div className="p-4 space-y-2">
                {result.nextSteps.map((step, i) => (
                  <div key={i} className="flex items-start gap-2.5 text-sm text-blue-300/80">
                    <span className="flex-shrink-0 text-blue-500 text-xs mt-1">›</span>
                    {step}
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}

      {/* Manual verification checklist — always visible */}
      <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 overflow-hidden">
        <div className="px-5 py-3.5 border-b border-slate-700/50 flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Manual Verification Checklist</span>
          <span className="text-xs text-slate-600">{checkedItems.size}/{manualChecklist.length}</span>
        </div>
        <div className="h-0.5 bg-slate-800">
          <div className="h-full bg-gradient-to-r from-emerald-500 to-teal-500 transition-all duration-500"
            style={{ width: `${(checkedItems.size / manualChecklist.length) * 100}%` }} />
        </div>
        <div className="p-4 space-y-1.5">
          {manualChecklist.map((item, i) => {
            const checked = checkedItems.has(i);
            return (
              <div key={i} onClick={() => toggleCheck(i)}
                className={`flex items-start gap-3 p-3 rounded-lg cursor-pointer transition-all ${
                  checked ? 'bg-emerald-500/8 border border-emerald-500/20' : 'hover:bg-slate-700/30 border border-transparent'
                }`}>
                <div className={`flex-shrink-0 w-4 h-4 rounded border-2 flex items-center justify-center mt-0.5 transition-all ${
                  checked ? 'bg-emerald-500 border-emerald-500' : 'border-slate-600 hover:border-slate-500'
                }`}>
                  {checked && <span className="text-white text-xs leading-none">✓</span>}
                </div>
                <span className={`text-sm ${checked ? 'text-emerald-400/70 line-through decoration-emerald-700' : 'text-slate-400'}`}>
                  {item}
                </span>
              </div>
            );
          })}
        </div>
      </div>

      {!result && (
        <div className="rounded-xl border border-slate-700/40 bg-slate-800/20 p-8 text-center">
          <div className="text-4xl mb-3">🔎</div>
          <p className="text-sm font-medium text-slate-400">No validation results yet</p>
          <p className="text-xs mt-1 text-slate-600">
            Run validation at any stage to see your migration phase and what's left to do.
          </p>
        </div>
      )}
    </div>
  );
}
