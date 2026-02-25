import { useState, type FC } from 'react';
import type { IngressMigrationSummary, AnnotationIssue, Target } from '../types';
import { api } from '../api/client';

interface Props {
  perIngress: IngressMigrationSummary[];
  target: Target;
  onNavigateToCategory?: (category: string) => void;
}

const categoryLabel: Record<string, string> = {
  middleware: 'Middlewares',
  policy:     'Policies',
  httproute:  'HTTPRoutes',
  gateway:    'Gateway',
  ingress:    'Ingresses',
};

// ─── IssueCard ───────────────────────────────────────────────────────────────

interface IssueCardProps {
  issue: AnnotationIssue;
  target: Target;
  onViewFiles?: (category: string) => void;
}

const IssueCard: FC<IssueCardProps> = ({ issue, target, onViewFiles }) => {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [accepted, setAccepted] = useState(false);   // for unsupported "I understand" flow
  const [applying, setApplying] = useState(false);
  const [applyResult, setApplyResult] = useState<{ ok: boolean; msg: string } | null>(null);

  const isPartial = issue.status === 'partial';

  const handleCopy = async () => {
    if (!issue.example) return;
    await navigator.clipboard.writeText(issue.example);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleApply = async (dryRun: boolean) => {
    if (!issue.fileCategory) return;
    setApplying(true);
    setApplyResult(null);
    try {
      const resp = await api.apply({ target, category: issue.fileCategory, dryRun });
      setApplyResult({
        ok: resp.success,
        msg: resp.success
          ? `${dryRun ? 'Dry-run' : 'Applied'} ${issue.fileCategory} files successfully.`
          : (resp.error ?? resp.output ?? 'Apply failed'),
      });
    } catch (e) {
      setApplyResult({ ok: false, msg: String(e) });
    } finally {
      setApplying(false);
    }
  };

  return (
    <div className={`rounded-lg border transition-all ${
      open
        ? isPartial ? 'border-amber-500/30 bg-amber-500/5' : 'border-red-500/30 bg-red-500/5'
        : 'border-slate-700/50 bg-slate-800/40 hover:border-slate-600/60'
    }`}>
      {/* Header row */}
      <button
        className="w-full flex items-center gap-3 px-4 py-3 text-left"
        onClick={() => setOpen(o => !o)}
      >
        {/* Status icon */}
        <span className={`flex-shrink-0 w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold border ${
          isPartial
            ? 'bg-amber-500/15 text-amber-400 border-amber-500/25'
            : 'bg-red-500/15 text-red-400 border-red-500/25'
        }`}>
          {isPartial ? '⚠' : '✗'}
        </span>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <code className="text-xs font-mono text-slate-200 bg-slate-700/60 px-1.5 py-0.5 rounded">
              {issue.key}
            </code>
            {issue.value && (
              <code className="text-xs font-mono text-slate-500">= {issue.value}</code>
            )}
            {/* Status badge */}
            <span className={`text-xs px-1.5 py-0.5 rounded border font-medium ${
              isPartial
                ? 'bg-amber-500/15 text-amber-400 border-amber-500/25'
                : 'bg-red-500/15 text-red-400 border-red-500/25'
            }`}>
              {isPartial ? 'Workaround' : 'No equivalent'}
            </span>
            {/* Auto-generated badge for partial */}
            {isPartial && issue.fileCategory && (
              <span className="text-xs px-1.5 py-0.5 rounded border bg-emerald-500/10 text-emerald-400 border-emerald-500/20 font-medium">
                ✓ Auto-generated
              </span>
            )}
            {issue.targetResource && (
              <span className="text-xs text-slate-500">→ {issue.targetResource}</span>
            )}
          </div>
          {!open && issue.what && (
            <div className="text-xs text-slate-500 mt-0.5 truncate">{issue.what}</div>
          )}
        </div>
        <span className="text-slate-600 text-xs flex-shrink-0">{open ? '▲' : '▼'}</span>
      </button>

      {/* Expanded content */}
      {open && (
        <div className="px-4 pb-4 space-y-3 border-t border-slate-700/40 pt-3">

          {/* What it does */}
          {issue.what && (
            <div>
              <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">What it does</div>
              <p className="text-sm text-slate-300">{issue.what}</p>
            </div>
          )}

          {/* === PARTIAL (workaround available) === */}
          {isPartial && (
            <>
              {/* Auto-generated callout */}
              {issue.fileCategory && (
                <div className="rounded-lg border border-emerald-500/25 bg-emerald-500/8 px-3 py-2.5">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-emerald-400 text-sm">✓</span>
                    <span className="text-xs font-semibold text-emerald-300">Configuration auto-generated</span>
                  </div>
                  <p className="text-xs text-slate-300">
                    This has been automatically configured in the{' '}
                    <strong className="text-emerald-300">{categoryLabel[issue.fileCategory] ?? issue.fileCategory}</strong>{' '}
                    files. Apply those files to activate this change — no manual YAML editing required.
                  </p>
                  {/* Action buttons */}
                  <div className="flex items-center gap-2 mt-2.5 flex-wrap">
                    {onViewFiles && (
                      <button
                        onClick={() => onViewFiles(issue.fileCategory!)}
                        className="text-xs px-2.5 py-1 rounded border border-slate-600 bg-slate-800 text-slate-300 hover:border-slate-500 hover:text-white transition-all"
                      >
                        📄 View Generated Files
                      </button>
                    )}
                    <button
                      onClick={() => handleApply(true)}
                      disabled={applying}
                      className="text-xs px-2.5 py-1 rounded border border-blue-500/40 bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 disabled:opacity-50 transition-all flex items-center gap-1"
                    >
                      {applying ? (
                        <><svg className="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                        </svg>testing...</>
                      ) : '🔍 Dry-run'}
                    </button>
                    <button
                      onClick={() => handleApply(false)}
                      disabled={applying}
                      className="text-xs px-2.5 py-1 rounded border border-emerald-500/40 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 disabled:opacity-50 transition-all flex items-center gap-1"
                    >
                      {applying ? (
                        <><svg className="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                        </svg>applying...</>
                      ) : '▶ Apply Now'}
                    </button>
                  </div>
                  {applyResult && (
                    <div className={`mt-2 text-xs px-2 py-1 rounded ${applyResult.ok ? 'text-emerald-400' : 'text-red-400'}`}>
                      {applyResult.ok ? '✓' : '✗'} {applyResult.msg}
                    </div>
                  )}
                </div>
              )}

              {/* How to configure (for reference or manual steps) */}
              {issue.fix && (
                <div>
                  <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
                    → Configuration details
                  </div>
                  <div className="text-sm text-slate-300 whitespace-pre-line leading-relaxed">{issue.fix}</div>
                </div>
              )}
            </>
          )}

          {/* === UNSUPPORTED (no automatic fix) === */}
          {!isPartial && (
            <>
              {/* Consequence */}
              {issue.consequence && (
                <div className="rounded-lg border border-red-500/25 bg-red-500/8 px-3 py-2">
                  <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">⚡ Impact if not migrated</div>
                  <p className="text-xs text-slate-300">{issue.consequence}</p>
                </div>
              )}

              {/* Best available alternative */}
              {issue.fix && (
                <div>
                  <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1.5">
                    🔧 Best available alternative
                  </div>
                  <div className="text-sm text-slate-300 whitespace-pre-line leading-relaxed">{issue.fix}</div>
                </div>
              )}

              {/* Accept impact */}
              <div className={`rounded-lg border px-3 py-2.5 transition-all ${
                accepted
                  ? 'border-slate-600/40 bg-slate-700/20'
                  : 'border-amber-500/20 bg-amber-500/5'
              }`}>
                <label className="flex items-start gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={accepted}
                    onChange={e => setAccepted(e.target.checked)}
                    className="mt-0.5 accent-amber-400 flex-shrink-0"
                  />
                  <div>
                    <span className={`text-xs font-medium ${accepted ? 'text-slate-400 line-through' : 'text-amber-300'}`}>
                      I understand this limitation and accept the impact described above
                    </span>
                    {accepted && (
                      <div className="text-xs text-slate-500 mt-0.5 no-underline">✓ Acknowledged — proceed with the best available alternative above.</div>
                    )}
                  </div>
                </label>
              </div>
            </>
          )}

          {/* Example YAML */}
          {issue.example && (
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                  {isPartial ? 'Generated Config Example' : 'Recommended Config'}
                </div>
                <button
                  onClick={handleCopy}
                  className={`text-xs px-2 py-0.5 rounded border transition-all ${
                    copied
                      ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/25'
                      : 'bg-slate-700/60 text-slate-400 border-slate-600 hover:border-slate-500'
                  }`}
                >
                  {copied ? '✓ Copied' : '📋 Copy'}
                </button>
              </div>
              <pre className="text-xs font-mono bg-slate-950 border border-slate-700/50 rounded-lg p-3 text-slate-300 overflow-x-auto whitespace-pre leading-5">
                {issue.example}
              </pre>
            </div>
          )}

          {/* Links row */}
          <div className="flex items-center gap-3 flex-wrap text-xs pt-1">
            {issue.docsLink && (
              <a
                href={issue.docsLink}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-blue-400 hover:text-blue-300 hover:underline transition-colors"
              >
                📚 Official Docs
              </a>
            )}
            {issue.issueUrl && (
              <a
                href={issue.issueUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-orange-400 hover:text-orange-300 hover:underline transition-colors"
              >
                🐛 Track on GitHub
              </a>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

// ─── IngressGapCard ───────────────────────────────────────────────────────────

interface IngressGapCardProps {
  ing: IngressMigrationSummary;
  isOpen: boolean;
  onToggle: () => void;
  accent: 'red' | 'amber';
  target: Target;
  onViewFiles?: (category: string) => void;
}

const IngressGapCard: FC<IngressGapCardProps> = ({ ing, isOpen, onToggle, accent, target, onViewFiles }) => {
  const unsupportedCount = ing.issues.filter(i => i.status === 'unsupported').length;
  const partialCount = ing.issues.filter(i => i.status === 'partial').length;

  const borderCls = isOpen
    ? accent === 'red' ? 'border-red-500/30' : 'border-amber-500/30'
    : 'border-slate-700/50';
  const bgCls = isOpen
    ? accent === 'red' ? 'bg-red-500/5' : 'bg-amber-500/5'
    : 'bg-slate-800/40';

  return (
    <div className={`rounded-xl border transition-all ${borderCls} ${bgCls}`}>
      <button
        className="w-full flex items-center gap-3 px-4 py-3.5 text-left"
        onClick={onToggle}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-mono text-sm">
              <span className="text-slate-500">{ing.namespace}/</span>
              <span className={`font-semibold ${accent === 'red' ? 'text-red-300' : 'text-amber-300'}`}>
                {ing.name}
              </span>
            </span>
          </div>
          <div className="flex items-center gap-2 mt-1">
            {unsupportedCount > 0 && (
              <span className="text-xs text-red-400 bg-red-500/10 border border-red-500/20 px-1.5 py-0.5 rounded">
                {unsupportedCount} no equivalent
              </span>
            )}
            {partialCount > 0 && (
              <span className="text-xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-1.5 py-0.5 rounded">
                {partialCount} auto-generated
              </span>
            )}
            <span className="text-xs text-slate-600">{ing.issues.length} annotation{ing.issues.length !== 1 ? 's' : ''} to review</span>
          </div>
        </div>
        <span className="text-slate-500 text-sm flex-shrink-0">{isOpen ? '▲' : '▼'}</span>
      </button>

      {isOpen && (
        <div className="px-4 pb-4 space-y-2 border-t border-slate-700/30 pt-3">
          {/* Unsupported first */}
          {ing.issues.filter(i => i.status === 'unsupported').map((issue, idx) => (
            <IssueCard key={idx} issue={issue} target={target} onViewFiles={onViewFiles} />
          ))}
          {/* Then partial */}
          {ing.issues.filter(i => i.status === 'partial').map((issue, idx) => (
            <IssueCard key={idx} issue={issue} target={target} onViewFiles={onViewFiles} />
          ))}
        </div>
      )}
    </div>
  );
};

// ─── MigrationGaps (main) ────────────────────────────────────────────────────

const MigrationGaps: FC<Props> = ({ perIngress, target, onNavigateToCategory }) => {
  const [expandedIngress, setExpandedIngress] = useState<string | null>(null);

  const breaking   = perIngress.filter(i => i.overallStatus === 'breaking');
  const workaround = perIngress.filter(i => i.overallStatus === 'workaround');
  const ready      = perIngress.filter(i => i.overallStatus === 'ready');

  const totalUnsupported = perIngress.reduce((n, i) => n + i.issues.filter(x => x.status === 'unsupported').length, 0);
  const totalPartial     = perIngress.reduce((n, i) => n + i.issues.filter(x => x.status === 'partial').length, 0);

  if (breaking.length === 0 && workaround.length === 0) {
    return (
      <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-8 text-center">
        <div className="text-3xl mb-2">🎉</div>
        <p className="text-emerald-400 font-semibold">All {perIngress.length} ingresses are fully compatible!</p>
        <p className="text-sm text-slate-500 mt-1">No manual steps needed. Proceed with the migration checklist.</p>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      {/* Summary cards */}
      <div className="grid grid-cols-3 gap-3">
        <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-3.5 text-center">
          <div className="text-2xl font-bold text-emerald-400">{ready.length}</div>
          <div className="text-xs text-emerald-500 mt-0.5">Fully Compatible</div>
          <div className="text-xs text-slate-600 mt-0.5">No action needed</div>
        </div>
        <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-3.5 text-center">
          <div className="text-2xl font-bold text-amber-400">{workaround.length}</div>
          <div className="text-xs text-amber-500 mt-0.5">Auto-Generated</div>
          <div className="text-xs text-slate-600 mt-0.5">{totalPartial} annotations handled</div>
        </div>
        <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-3.5 text-center">
          <div className="text-2xl font-bold text-red-400">{breaking.length}</div>
          <div className="text-xs text-red-500 mt-0.5">Needs Review</div>
          <div className="text-xs text-slate-600 mt-0.5">{totalUnsupported} no equivalent</div>
        </div>
      </div>

      {/* Legend */}
      <div className="text-xs text-slate-500 space-y-1 px-1">
        <p>
          <span className="text-emerald-400">✓ Auto-generated</span> — configuration has been created automatically in the generated files. Just apply to activate.
        </p>
        <p>
          <span className="text-red-400">✗ No equivalent</span> — this feature has no direct equivalent in {target === 'traefik' ? 'Traefik' : 'core Gateway API'}. The best available alternative is shown with impact details.
        </p>
      </div>

      {/* Needs Review (unsupported) — shown first as highest priority */}
      {breaking.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-red-400 flex-shrink-0" />
            <h3 className="text-sm font-semibold text-red-300">
              Needs Review — {breaking.length} ingress{breaking.length > 1 ? 'es' : ''}
            </h3>
            <div className="flex-1 h-px bg-red-500/15" />
          </div>
          {breaking.map(ing => {
            const key = `${ing.namespace}/${ing.name}`;
            const isOpen = expandedIngress === key;
            return (
              <IngressGapCard
                key={key}
                ing={ing}
                isOpen={isOpen}
                onToggle={() => setExpandedIngress(isOpen ? null : key)}
                accent="red"
                target={target}
                onViewFiles={onNavigateToCategory}
              />
            );
          })}
        </div>
      )}

      {/* Auto-generated workarounds */}
      {workaround.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-amber-400 flex-shrink-0" />
            <h3 className="text-sm font-semibold text-amber-300">
              Auto-Generated Workarounds — {workaround.length} ingress{workaround.length > 1 ? 'es' : ''}
            </h3>
            <div className="flex-1 h-px bg-amber-500/15" />
          </div>
          {workaround.map(ing => {
            const key = `${ing.namespace}/${ing.name}`;
            const isOpen = expandedIngress === key;
            return (
              <IngressGapCard
                key={key}
                ing={ing}
                isOpen={isOpen}
                onToggle={() => setExpandedIngress(isOpen ? null : key)}
                accent="amber"
                target={target}
                onViewFiles={onNavigateToCategory}
              />
            );
          })}
        </div>
      )}
    </div>
  );
};

export default MigrationGaps;
