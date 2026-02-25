import type { FC } from 'react';
import { useState } from 'react';
import type { IngressReport, AnnotationMapping } from '../types';

interface Props {
  reports: IngressReport[];
}

const statusConfig = {
  supported:   { icon: '✓', color: 'text-emerald-400', bg: 'bg-emerald-500/10', border: 'border-emerald-500/20' },
  partial:     { icon: '⚠', color: 'text-amber-400',   bg: 'bg-amber-500/10',   border: 'border-amber-500/20' },
  unsupported: { icon: '✗', color: 'text-red-400',     bg: 'bg-red-500/10',     border: 'border-red-500/20' },
};

const overallConfig = {
  ready:      { bg: 'bg-emerald-500/10', border: 'border-emerald-500/30', text: 'text-emerald-400', badge: 'bg-emerald-500/20 text-emerald-300', label: '✓ Ready' },
  workaround: { bg: 'bg-amber-500/10',   border: 'border-amber-500/30',   text: 'text-amber-400',   badge: 'bg-amber-500/20 text-amber-300',   label: '⚠ Workaround' },
  breaking:   { bg: 'bg-red-500/10',     border: 'border-red-500/30',     text: 'text-red-400',     badge: 'bg-red-500/20 text-red-300',     label: '✗ Breaking' },
};

const MappingRow: FC<{ mapping: AnnotationMapping; index: number }> = ({ mapping, index }) => {
  const s = statusConfig[mapping.status] ?? statusConfig.unsupported;
  return (
    <tr
      className="border-b border-slate-700/30 hover:bg-slate-700/20 transition-colors"
      style={{ animationDelay: `${index * 30}ms` }}
    >
      <td className="px-4 py-2.5">
        <code className="text-xs text-blue-300 font-mono bg-blue-500/10 px-1.5 py-0.5 rounded">
          {mapping.originalKey}
        </code>
      </td>
      <td className="px-4 py-2.5">
        <span className="text-xs text-slate-400 font-mono truncate max-w-[100px] block" title={mapping.originalValue}>
          {mapping.originalValue || '—'}
        </span>
      </td>
      <td className="px-4 py-2.5">
        <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border ${s.bg} ${s.color} ${s.border}`}>
          <span>{s.icon}</span>
          <span className="capitalize">{mapping.status}</span>
        </span>
      </td>
      <td className="px-4 py-2.5">
        <span className="text-xs text-slate-300">{mapping.targetResource || '—'}</span>
      </td>
      <td className="px-4 py-2.5">
        <span className="text-xs text-slate-500 italic">{mapping.note}</span>
      </td>
    </tr>
  );
};

const AnnotationMatrix: FC<Props> = ({ reports }) => {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  if (!reports?.length) {
    return (
      <div className="text-center py-12 text-slate-500">
        <p>No analysis data. Run analyze first.</p>
      </div>
    );
  }

  const sortOrder = { unsupported: 0, partial: 1, supported: 2 };

  return (
    <div className="space-y-3">
      {reports.map((report, i) => {
        const isOpen = openIndex === i;
        const cfg = overallConfig[report.overallStatus] ?? overallConfig.ready;
        const sorted = [...(report.mappings || [])].sort(
          (a, b) => (sortOrder[a.status] ?? 3) - (sortOrder[b.status] ?? 3)
        );
        const counts = {
          supported: sorted.filter(m => m.status === 'supported').length,
          partial: sorted.filter(m => m.status === 'partial').length,
          unsupported: sorted.filter(m => m.status === 'unsupported').length,
        };

        return (
          <div
            key={`${report.namespace}/${report.name}`}
            className={`rounded-xl border transition-all ${cfg.border} ${isOpen ? cfg.bg : 'bg-slate-800/40 border-slate-700/40 hover:border-slate-600/50'}`}
          >
            <button
              className="w-full flex items-center justify-between px-5 py-4 text-left"
              onClick={() => setOpenIndex(isOpen ? null : i)}
            >
              <div className="flex items-center gap-3 min-w-0">
                <div className={`flex-shrink-0 w-2 h-2 rounded-full ${
                  report.overallStatus === 'ready' ? 'bg-emerald-400' :
                  report.overallStatus === 'workaround' ? 'bg-amber-400' : 'bg-red-400'
                }`} />
                <div className="min-w-0">
                  <span className="font-mono text-sm">
                    <span className="text-slate-400">{report.namespace}/</span>
                    <span className="font-semibold text-white">{report.name}</span>
                  </span>
                </div>
                <span className={`flex-shrink-0 text-xs px-2 py-0.5 rounded-full font-medium ${cfg.badge}`}>
                  {cfg.label}
                </span>
              </div>
              <div className="flex items-center gap-3 ml-3 flex-shrink-0">
                {counts.unsupported > 0 && (
                  <span className="text-xs text-red-400 bg-red-500/10 px-2 py-0.5 rounded-full">
                    {counts.unsupported} ✗
                  </span>
                )}
                {counts.partial > 0 && (
                  <span className="text-xs text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-full">
                    {counts.partial} ⚠
                  </span>
                )}
                {counts.supported > 0 && (
                  <span className="text-xs text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full">
                    {counts.supported} ✓
                  </span>
                )}
                <span className="text-slate-500 text-sm ml-1">{isOpen ? '▲' : '▼'}</span>
              </div>
            </button>

            {isOpen && (
              <div className="border-t border-slate-700/40 animate-slide-up">
                {sorted.length === 0 ? (
                  <div className="px-5 py-8 text-center">
                    <div className="text-2xl mb-2">🎉</div>
                    <p className="text-emerald-400 font-medium text-sm">No nginx annotations — migrate as-is!</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="min-w-full">
                      <thead>
                        <tr className="bg-slate-900/40">
                          <th className="px-4 py-2 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Annotation</th>
                          <th className="px-4 py-2 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Value</th>
                          <th className="px-4 py-2 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Status</th>
                          <th className="px-4 py-2 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Target Resource</th>
                          <th className="px-4 py-2 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Notes</th>
                        </tr>
                      </thead>
                      <tbody>
                        {sorted.map((m, j) => <MappingRow key={j} mapping={m} index={j} />)}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};

export default AnnotationMatrix;
