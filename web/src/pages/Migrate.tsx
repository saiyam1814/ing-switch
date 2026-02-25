import { useState } from 'react';
import { api } from '../api/client';
import type { ScanResult, AnalysisReport, MigrateResponse, ApplyResponse, Target } from '../types';
import FileViewer from '../components/FileViewer';
import MigrationGaps from '../components/MigrationGaps';

interface Props {
  scanResult: ScanResult | null;
  analysisReport: AnalysisReport | null;
}

type StepStatus = 'idle' | 'dry-running' | 'dry-run-ok' | 'applying' | 'done' | 'error';

interface StepDef {
  label: string;
  desc: string;
  icon: string;
  category: string;
  danger?: boolean;   // destructive step
  manual?: boolean;   // can't auto-apply (Helm, DNS, etc.)
  traffic?: boolean;  // affects production traffic
  filePath?: string;  // relPath of the file to navigate to in file viewer
}

const stepDefs: Record<Target, StepDef[]> = {
  traefik: [
    { label: 'Review Migration Report',  desc: '00-migration-report.md — full summary of what changes',               icon: '📖', category: 'guide',      manual: true, filePath: '00-migration-report.md' },
    { label: 'Install Traefik v3',        desc: 'helm-install.sh — installs Traefik alongside NGINX (safe)',          icon: '📦', category: 'install',    manual: true, filePath: '01-install-traefik/helm-install.sh' },
    { label: 'Apply Middlewares',         desc: '02-middlewares/ — creates Middleware CRDs (auth, rate-limit, CORS)', icon: '🔧', category: 'middleware' },
    { label: 'Apply Updated Ingresses',   desc: '03-ingresses/ — attaches Traefik middleware annotations',            icon: '🔀', category: 'ingress' },
    { label: 'Verify Traefik',            desc: '04-verify.sh — test Traefik before touching DNS',                    icon: '✅', category: 'verify',     manual: true, filePath: '04-verify.sh' },
    { label: 'DNS Cutover',               desc: 'Update DNS to Traefik LoadBalancer IP — shifts live traffic',        icon: '🌐', category: 'guide',      manual: true, traffic: true, filePath: '05-dns-migration.md' },
    { label: 'Remove NGINX',             desc: '06-cleanup/ — run remove-nginx.sh only after DNS propagated',        icon: '🗑️', category: 'cleanup',    manual: true, danger: true, filePath: '06-cleanup/02-remove-nginx.sh' },
  ],
  'gateway-api': [
    { label: 'Review Migration Report',   desc: '00-migration-report.md — full summary of what changes',             icon: '📖', category: 'guide',      manual: true, filePath: '00-migration-report.md' },
    { label: 'Install Gateway API CRDs',  desc: 'install.sh — registers API types (non-breaking)',                   icon: '📦', category: 'install',    manual: true, filePath: '01-install-gateway-api-crds/install.sh' },
    { label: 'Install Envoy Gateway',     desc: 'helm-install.sh — installs Envoy Gateway alongside NGINX',         icon: '📦', category: 'install',    manual: true, filePath: '02-install-envoy-gateway/helm-install.sh' },
    { label: 'Apply Gateway Resources',   desc: '03-gateway/ — GatewayClass + Gateway (Envoy listener)',            icon: '🌐', category: 'gateway' },
    { label: 'Apply HTTPRoutes',          desc: '04-httproutes/ — converted from Ingress objects',                  icon: '🛣️', category: 'httproute' },
    { label: 'Apply Policies',            desc: '05-policies/ — rate limits, auth policies, IP filters',            icon: '🛡️', category: 'policy' },
    { label: 'Verify & DNS Cutover',      desc: '06-verify.sh, then update DNS — shifts live traffic',              icon: '✅', category: 'verify',     manual: true, traffic: true, filePath: '06-verify.sh' },
    { label: 'Remove NGINX',             desc: '07-cleanup/remove-nginx.sh — only after DNS fully propagated',     icon: '🗑️', category: 'cleanup',    manual: true, danger: true, filePath: '07-cleanup/remove-nginx.sh' },
  ],
};

export default function Migrate({ scanResult }: Props) {
  const [target, setTarget] = useState<Target>('traefik');
  const [outputDir, setOutputDir] = useState('./migration');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [migrateResult, setMigrateResult] = useState<MigrateResponse | null>(null);
  const [completedSteps, setCompletedSteps] = useState<Set<number>>(new Set());
  const [stepStatuses, setStepStatuses] = useState<Record<number, StepStatus>>({});
  const [stepOutputs, setStepOutputs] = useState<Record<number, ApplyResponse | null>>({});
  const [expandedStep, setExpandedStep] = useState<number | null>(null);
  const [activeTab, setActiveTab] = useState<'checklist' | 'files' | 'gaps'>('checklist');
  const [selectedFilePath, setSelectedFilePath] = useState<string | null>(null);

  const handleMigrate = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.migrate(target, outputDir);
      setMigrateResult(result);
      setCompletedSteps(new Set());
      setStepStatuses({});
      setStepOutputs({});
      setActiveTab('checklist');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Migration generation failed');
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async (stepIdx: number, dryRun: boolean) => {
    const step = stepDefs[target][stepIdx];
    setStepStatuses(p => ({ ...p, [stepIdx]: dryRun ? 'dry-running' : 'applying' }));
    try {
      const resp = await api.apply({ target, category: step.category, dryRun });
      setStepOutputs(p => ({ ...p, [stepIdx]: resp }));
      if (resp.success) {
        setStepStatuses(p => ({ ...p, [stepIdx]: dryRun ? 'dry-run-ok' : 'done' }));
        if (!dryRun) {
          setCompletedSteps(p => new Set([...p, stepIdx]));
        }
      } else {
        setStepStatuses(p => ({ ...p, [stepIdx]: 'error' }));
      }
      setExpandedStep(stepIdx);
    } catch (e) {
      setStepOutputs(p => ({ ...p, [stepIdx]: { success: false, output: '', dryRun, applied: [], error: String(e) } }));
      setStepStatuses(p => ({ ...p, [stepIdx]: 'error' }));
      setExpandedStep(stepIdx);
    }
  };

  const toggleStep = (i: number) => {
    setCompletedSteps(prev => {
      const next = new Set(prev);
      if (next.has(i)) next.delete(i); else next.add(i);
      return next;
    });
  };

  const handleDownloadAll = () => { window.location.href = api.downloadUrl(target); };

  if (!scanResult) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-slate-500">
        <div className="w-16 h-16 rounded-2xl bg-slate-800 flex items-center justify-center mb-4 text-3xl">🚀</div>
        <p className="text-base font-medium text-slate-400">No scan data</p>
        <p className="text-sm mt-1 text-slate-600">Complete Detect and Analyze steps first.</p>
      </div>
    );
  }

  const steps = stepDefs[target];
  const doneCount = completedSteps.size;

  // Migration gaps from analysis
  const perIngress = migrateResult?.perIngress ?? [];
  const breakingCount = perIngress.filter(p => p.overallStatus === 'breaking').length;
  const workaroundCount = perIngress.filter(p => p.overallStatus === 'workaround').length;
  const readyCount = perIngress.filter(p => p.overallStatus === 'ready').length;
  const totalGapCount = breakingCount + workaroundCount;

  const navigateToFile = (relPath: string) => {
    setSelectedFilePath(relPath);
    setActiveTab('files');
  };

  const navigateToCategory = (category: string) => {
    // Find first file in that category
    const file = migrateResult?.files.find(f => f.category === category);
    if (file) {
      setSelectedFilePath(file.relPath);
      setActiveTab('files');
    } else {
      setActiveTab('files');
    }
  };

  return (
    <div className="space-y-5 animate-fade-in">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Migrate</h1>
        <p className="text-slate-400 mt-1 text-sm">
          Generate migration files, dry-run to preview, then apply to cluster step-by-step.
        </p>
      </div>

      {/* Safety banner */}
      <div className="rounded-xl border border-blue-500/20 bg-blue-500/5 p-3.5 flex items-start gap-3">
        <span className="text-blue-400 text-base flex-shrink-0 mt-0.5">🛡️</span>
        <div className="text-sm text-blue-300/90">
          <strong className="text-blue-200">Zero-downtime migration:</strong> Install the new controller alongside NGINX first.
          DNS still points to NGINX — production is unaffected until the{' '}
          <span className="text-amber-300 font-medium">DNS Cutover</span> step.
          Every apply step supports <strong className="text-blue-200">dry-run mode</strong> so you preview changes first.
        </div>
      </div>

      {/* Config card */}
      <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 p-5">
        <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Generate Migration Files</div>
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
          <div className="flex-1 min-w-[180px] max-w-xs">
            <label className="block text-xs font-medium text-slate-500 mb-1.5">Output Directory</label>
            <input type="text" value={outputDir} onChange={e => setOutputDir(e.target.value)}
              className="w-full bg-slate-900 border border-slate-700 text-slate-200 rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-colors" />
          </div>
          <button onClick={handleMigrate} disabled={loading}
            className="px-5 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-semibold rounded-lg transition-all shadow-lg shadow-emerald-900/30 flex items-center gap-2">
            {loading ? (
              <><svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>Generating...</>
            ) : '⚡ Generate Migration Files'}
          </button>
        </div>

        {error && <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-sm text-red-300">{error}</div>}

        {migrateResult && (
          <div className="mt-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-between flex-wrap gap-2">
            <span className="text-sm text-emerald-300">✓ {migrateResult.summary}</span>
            <div className="flex gap-3 text-xs">
              {readyCount > 0 && <span className="text-emerald-400">{readyCount} ready</span>}
              {workaroundCount > 0 && <span className="text-amber-400">{workaroundCount} workaround</span>}
              {breakingCount > 0 && <span className="text-red-400">{breakingCount} need manual work</span>}
            </div>
          </div>
        )}
      </div>

      {/* Tabs */}
      {migrateResult && (
        <div className="flex gap-1 bg-slate-800/60 border border-slate-700/50 rounded-lg p-1 w-fit">
          {([
            { id: 'checklist', label: '📋 Migration Steps' },
            { id: 'files',     label: `📁 Files (${migrateResult.files.length})` },
            { id: 'gaps',      label: `⚠ Gaps${totalGapCount > 0 ? ` (${totalGapCount})` : ''}` },
          ] as const).map(tab => (
            <button key={tab.id} onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-1.5 text-xs font-medium rounded-md transition-all ${
                activeTab === tab.id ? 'bg-slate-700 text-slate-200 shadow-sm' : 'text-slate-500 hover:text-slate-300'
              }`}>
              {tab.label}
            </button>
          ))}
        </div>
      )}

      {/* Tab: Checklist */}
      {(!migrateResult || activeTab === 'checklist') && (
        <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 overflow-hidden">
          <div className="px-5 py-3.5 border-b border-slate-700/50 flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
              Migration Checklist — {target === 'traefik' ? 'Traefik v3' : 'Gateway API (Envoy Gateway v1.3)'}
            </span>
            <span className="text-xs text-slate-600">{doneCount}/{steps.length} completed</span>
          </div>
          <div className="h-0.5 bg-slate-800">
            <div className="h-full bg-gradient-to-r from-emerald-500 to-teal-500 transition-all duration-500"
              style={{ width: steps.length > 0 ? `${(doneCount / steps.length) * 100}%` : '0%' }} />
          </div>
          <div className="divide-y divide-slate-800/60">
            {steps.map((step, i) => {
              const done = completedSteps.has(i);
              const status = stepStatuses[i] ?? 'idle';
              const output = stepOutputs[i];
              const isExpanded = expandedStep === i;
              const canApply = migrateResult && !step.manual;

              return (
                <div key={i} className={`transition-colors ${done ? 'bg-emerald-500/5' : ''}`}>
                  <div className="flex items-center gap-3 p-3.5">
                    {/* Checkbox */}
                    <div onClick={() => toggleStep(i)}
                      className={`flex-shrink-0 w-5 h-5 rounded-full border-2 flex items-center justify-center cursor-pointer transition-all ${
                        done ? 'bg-emerald-500 border-emerald-500' : 'border-slate-600 hover:border-slate-500'
                      }`}>
                      {done && <span className="text-white text-xs leading-none">✓</span>}
                    </div>

                    {/* Icon + label */}
                    <span className="text-base flex-shrink-0">{step.icon}</span>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className={`text-sm font-medium ${done ? 'text-emerald-400/70 line-through decoration-emerald-700' : 'text-slate-200'}`}>
                          <span className="text-slate-600 mr-1.5 text-xs">#{i + 1}</span>{step.label}
                        </span>
                        {step.traffic && (
                          <span className="text-xs px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-400 border border-amber-500/25">
                            ⚡ affects traffic
                          </span>
                        )}
                        {step.danger && (
                          <span className="text-xs px-1.5 py-0.5 rounded bg-red-500/15 text-red-400 border border-red-500/25">
                            ⚠ destructive
                          </span>
                        )}
                        {status === 'done' && (
                          <span className="text-xs text-emerald-400">✓ applied</span>
                        )}
                        {status === 'dry-run-ok' && (
                          <span className="text-xs text-blue-400">✓ dry-run passed</span>
                        )}
                        {status === 'error' && (
                          <span className="text-xs text-red-400">✗ error</span>
                        )}
                      </div>
                      <div className="text-xs text-slate-500 mt-0.5 font-mono truncate">{step.desc}</div>
                    </div>

                    {/* Action buttons */}
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {output && (
                        <button onClick={() => setExpandedStep(isExpanded ? null : i)}
                          className="text-xs text-slate-500 hover:text-slate-300 px-2 py-1 rounded border border-slate-700 hover:border-slate-600 transition-colors">
                          {isExpanded ? '▲ hide' : '▼ output'}
                        </button>
                      )}

                      {canApply && (
                        <>
                          <button
                            onClick={() => handleApply(i, true)}
                            disabled={status === 'dry-running' || status === 'applying'}
                            className="text-xs px-3 py-1.5 rounded-lg border border-blue-500/30 bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 disabled:opacity-40 disabled:cursor-not-allowed transition-all flex items-center gap-1">
                            {status === 'dry-running' ? (
                              <><svg className="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                              </svg>testing...</>
                            ) : '🔍 Dry-run'}
                          </button>
                          <button
                            onClick={() => handleApply(i, false)}
                            disabled={status === 'dry-running' || status === 'applying' || status === 'done'}
                            className={`text-xs px-3 py-1.5 rounded-lg border transition-all flex items-center gap-1 ${
                              status === 'done'
                                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400 cursor-default'
                                : 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 disabled:opacity-40 disabled:cursor-not-allowed'
                            }`}>
                            {status === 'applying' ? (
                              <><svg className="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                              </svg>applying...</>
                            ) : status === 'done' ? '✓ applied' : '▶ Apply'}
                          </button>
                        </>
                      )}

                      {step.manual && migrateResult && (
                        <>
                          {step.filePath && (
                            <button
                              onClick={() => navigateToFile(step.filePath!)}
                              className="text-xs px-2.5 py-1 rounded border border-slate-600/60 bg-slate-800/60 text-slate-400 hover:text-slate-200 hover:border-slate-500 transition-all"
                            >
                              📄 View File
                            </button>
                          )}
                          {!step.filePath && (
                            <span className="text-xs text-slate-600 italic">manual step</span>
                          )}
                        </>
                      )}
                    </div>
                  </div>

                  {/* Output panel */}
                  {isExpanded && output && (
                    <div className="mx-3.5 mb-3 rounded-lg border border-slate-700/60 overflow-hidden">
                      <div className={`px-3 py-2 text-xs font-medium border-b border-slate-700/50 flex items-center justify-between ${
                        output.success ? 'bg-emerald-500/8 text-emerald-400' : 'bg-red-500/8 text-red-400'
                      }`}>
                        <span>{output.dryRun ? '🔍 Dry-run output' : '▶ Apply output'} — {output.success ? '✓ success' : '✗ failed'}</span>
                        {output.applied?.length > 0 && (
                          <span className="text-slate-500">{output.applied.length} files</span>
                        )}
                      </div>
                      <pre className="text-xs font-mono p-3 bg-slate-950 text-slate-300 overflow-x-auto max-h-48 overflow-y-auto leading-5 whitespace-pre-wrap">
                        {output.output || output.error || 'No output'}
                      </pre>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
          <div className="px-5 py-2.5 border-t border-slate-800/60 text-xs text-slate-600">
            Click checkboxes to mark steps done manually. Use Dry-run to preview before applying.
          </div>
        </div>
      )}

      {/* Tab: Files */}
      {migrateResult && activeTab === 'files' && (
        <div className="rounded-xl border border-slate-700/50 bg-slate-800/40 overflow-hidden">
          <div className="px-5 py-3.5 border-b border-slate-700/50">
            <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Generated Files</span>
          </div>
          <div className="p-4">
            <FileViewer files={migrateResult.files} onDownloadAll={handleDownloadAll} initialPath={selectedFilePath ?? undefined} />
          </div>
        </div>
      )}

      {/* Tab: Migration Gaps */}
      {migrateResult && activeTab === 'gaps' && (
        <MigrationGaps
          perIngress={migrateResult.perIngress ?? []}
          target={target}
          onNavigateToCategory={navigateToCategory}
        />
      )}

      {/* No files yet hint */}
      {!migrateResult && (
        <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 flex items-start gap-3">
          <span className="text-amber-400 text-base flex-shrink-0">💡</span>
          <p className="text-sm text-amber-300/80">
            Click <strong className="text-amber-300">Generate Migration Files</strong> to produce all YAML manifests,
            Middleware CRDs, and install scripts for <strong className="text-amber-300">{target === 'traefik' ? 'Traefik v3' : 'Envoy Gateway + Gateway API'}</strong>.
            Files are also written to <code className="font-mono text-amber-200 bg-amber-500/10 px-1 rounded">{outputDir}</code>.
          </p>
        </div>
      )}
    </div>
  );
}
