import type { FC } from 'react';
import type { IngressInfo } from '../types';
import StatusBadge from './StatusBadge';

interface Props {
  ingresses: IngressInfo[];
  onSelect?: (ing: IngressInfo) => void;
  selected?: string;
}

const IngressTable: FC<Props> = ({ ingresses, onSelect, selected }) => {
  if (ingresses.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-slate-500">
        <div className="w-16 h-16 rounded-2xl bg-slate-800 flex items-center justify-center mb-4 text-3xl">📭</div>
        <p className="text-base font-medium text-slate-300">No Ingress resources found</p>
        <p className="text-sm mt-1">Click "Scan Cluster" to discover Ingress objects</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full">
        <thead>
          <tr className="border-b border-slate-700/50">
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">Namespace</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">Name</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">Hosts</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">Class</th>
            <th className="px-4 py-3 text-center text-xs font-semibold text-slate-400 uppercase tracking-wider">TLS</th>
            <th className="px-4 py-3 text-center text-xs font-semibold text-slate-400 uppercase tracking-wider">Annotations</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-slate-400 uppercase tracking-wider">Complexity</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/30">
          {ingresses.map((ing) => {
            const key = `${ing.namespace}/${ing.name}`;
            const isSelected = selected === key;
            const annotCount = Object.keys(ing.nginxAnnotations || {}).length;
            return (
              <tr
                key={key}
                onClick={() => onSelect?.(ing)}
                className={`transition-all group ${onSelect ? 'cursor-pointer' : ''} ${
                  isSelected
                    ? 'bg-blue-500/10 border-l-2 border-blue-500'
                    : 'hover:bg-slate-700/30'
                }`}
              >
                <td className="px-4 py-3">
                  <span className="inline-flex items-center px-2 py-0.5 rounded-md bg-slate-700/50 text-slate-300 text-xs font-mono">
                    {ing.namespace}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className="text-sm font-semibold text-white group-hover:text-blue-400 transition-colors">
                    {ing.name}
                  </span>
                </td>
                <td className="px-4 py-3">
                  {ing.hosts.slice(0, 2).map(h => (
                    <div key={h} className="text-xs font-mono text-slate-300">{h}</div>
                  ))}
                  {ing.hosts.length > 2 && (
                    <span className="text-xs text-slate-500">+{ing.hosts.length - 2} more</span>
                  )}
                  {ing.hosts.length === 0 && <span className="text-xs text-slate-500">*</span>}
                </td>
                <td className="px-4 py-3">
                  <span className="text-xs font-mono text-slate-400">{ing.ingressClass || '—'}</span>
                </td>
                <td className="px-4 py-3 text-center">
                  {ing.tlsEnabled
                    ? <span className="text-emerald-400 text-sm">🔒</span>
                    : <span className="text-slate-600 text-sm">—</span>
                  }
                </td>
                <td className="px-4 py-3 text-center">
                  <span className={`inline-flex items-center justify-center w-7 h-7 rounded-lg text-xs font-bold ${
                    annotCount > 8 ? 'bg-red-500/20 text-red-400' :
                    annotCount > 3 ? 'bg-amber-500/20 text-amber-400' :
                    'bg-slate-700 text-slate-300'
                  }`}>
                    {annotCount}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status={ing.complexity} size="xs" />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

export default IngressTable;
