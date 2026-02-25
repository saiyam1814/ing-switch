import type { FC } from 'react';
import type { IngressInfo } from '../types';

interface Props {
  ingresses: IngressInfo[];
}

const complexityConfig = {
  simple:      { bg: 'bg-emerald-500/20', border: 'border-emerald-500/40', text: 'text-emerald-300', dot: 'bg-emerald-400' },
  complex:     { bg: 'bg-amber-500/20',   border: 'border-amber-500/40',   text: 'text-amber-300',   dot: 'bg-amber-400' },
  unsupported: { bg: 'bg-red-500/20',     border: 'border-red-500/40',     text: 'text-red-300',     dot: 'bg-red-400' },
};

const DependencyGraph: FC<Props> = ({ ingresses }) => {
  if (!ingresses?.length) {
    return <div className="text-slate-500 text-sm p-4">No ingresses to visualize.</div>;
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-slate-500 flex items-center gap-2">
        <span className="w-3 h-3 rounded-sm bg-emerald-500/40 inline-block"/>Simple
        <span className="w-3 h-3 rounded-sm bg-amber-500/40 inline-block ml-2"/>Complex
        <span className="w-3 h-3 rounded-sm bg-red-500/40 inline-block ml-2"/>Unsupported
      </p>

      {ingresses.map((ing) => {
        const cfg = complexityConfig[ing.complexity] ?? complexityConfig.complex;
        const annotCount = Object.keys(ing.nginxAnnotations || {}).length;
        return (
          <div
            key={`${ing.namespace}/${ing.name}`}
            className={`rounded-xl border p-4 ${cfg.bg} ${cfg.border} animate-slide-up`}
          >
            <div className="flex items-start gap-4">
              {/* Ingress Node */}
              <div className="flex-shrink-0">
                <div className={`rounded-lg border px-3 py-2.5 min-w-[150px] text-center ${cfg.bg} ${cfg.border}`}>
                  <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-0.5">Ingress</div>
                  <div className={`text-sm font-bold ${cfg.text}`}>{ing.name}</div>
                  <div className="text-xs text-slate-500 mt-0.5 font-mono">{ing.namespace}</div>
                  {ing.tlsEnabled && (
                    <div className="mt-1.5 text-xs text-emerald-400 flex items-center justify-center gap-1">
                      <span>🔒</span> TLS
                    </div>
                  )}
                </div>
                {annotCount > 0 && (
                  <div className="mt-2 text-center">
                    <span className={`text-xs px-2 py-0.5 rounded-full ${
                      ing.complexity === 'unsupported' ? 'bg-red-500/20 text-red-400' :
                      ing.complexity === 'complex' ? 'bg-amber-500/20 text-amber-400' :
                      'bg-emerald-500/20 text-emerald-400'
                    }`}>
                      {annotCount} annotations
                    </span>
                  </div>
                )}
              </div>

              {/* Arrow */}
              <div className="flex-shrink-0 flex flex-col items-center justify-center pt-6 text-slate-500">
                <div className="w-8 h-px bg-slate-600"/>
                <div className="text-slate-500 -mr-1">▶</div>
              </div>

              {/* Routes */}
              <div className="flex-1 space-y-1.5">
                {ing.paths.slice(0, 5).map((path, i) => (
                  <div key={i} className="flex items-center gap-2 text-xs">
                    <div className="flex-shrink-0 bg-slate-800 border border-slate-700 rounded px-2 py-1 font-mono text-blue-300 min-w-0">
                      <span className="text-slate-500">{path.host || '*'}</span>
                      <span className="text-blue-400">{path.path || '/'}</span>
                    </div>
                    <span className="text-slate-600 flex-shrink-0">→</span>
                    <div className="flex-shrink-0 bg-purple-500/10 border border-purple-500/30 rounded px-2 py-1 font-mono text-purple-300">
                      {path.serviceName}
                      <span className="text-purple-500">:{path.servicePort || 80}</span>
                    </div>
                  </div>
                ))}
                {ing.paths.length > 5 && (
                  <p className="text-xs text-slate-500 italic">+{ing.paths.length - 5} more paths</p>
                )}

                {/* Annotation tags */}
                {annotCount > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1 pt-1 border-t border-slate-700/40">
                    {Object.keys(ing.nginxAnnotations).slice(0, 4).map(k => (
                      <span key={k} className="text-xs bg-slate-800 border border-slate-700 text-slate-400 rounded px-1.5 py-0.5 font-mono">
                        {k}
                      </span>
                    ))}
                    {annotCount > 4 && (
                      <span className="text-xs text-slate-500 py-0.5">+{annotCount - 4} more</span>
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default DependencyGraph;
