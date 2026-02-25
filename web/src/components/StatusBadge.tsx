import type { FC } from 'react';

type Status = string;

interface Props {
  status: Status;
  size?: 'xs' | 'sm' | 'md';
}

const config: Record<string, { bg: string; text: string; border: string; dot: string; label: string }> = {
  simple:      { bg: 'bg-emerald-500/10', text: 'text-emerald-400', border: 'border-emerald-500/20', dot: 'bg-emerald-400', label: 'Simple' },
  complex:     { bg: 'bg-amber-500/10',   text: 'text-amber-400',   border: 'border-amber-500/20',   dot: 'bg-amber-400',   label: 'Complex' },
  unsupported: { bg: 'bg-red-500/10',     text: 'text-red-400',     border: 'border-red-500/20',     dot: 'bg-red-400',     label: 'Unsupported' },
  supported:   { bg: 'bg-emerald-500/10', text: 'text-emerald-400', border: 'border-emerald-500/20', dot: 'bg-emerald-400', label: '✓ Supported' },
  partial:     { bg: 'bg-amber-500/10',   text: 'text-amber-400',   border: 'border-amber-500/20',   dot: 'bg-amber-400',   label: '⚠ Partial' },
  ready:       { bg: 'bg-emerald-500/10', text: 'text-emerald-400', border: 'border-emerald-500/20', dot: 'bg-emerald-400', label: '✓ Ready' },
  workaround:  { bg: 'bg-amber-500/10',   text: 'text-amber-400',   border: 'border-amber-500/20',   dot: 'bg-amber-400',   label: '⚠ Workaround' },
  breaking:    { bg: 'bg-red-500/10',     text: 'text-red-400',     border: 'border-red-500/20',     dot: 'bg-red-400',     label: '✗ Breaking' },
  pass:        { bg: 'bg-emerald-500/10', text: 'text-emerald-400', border: 'border-emerald-500/20', dot: 'bg-emerald-400', label: '✓ Pass' },
  warn:        { bg: 'bg-amber-500/10',   text: 'text-amber-400',   border: 'border-amber-500/20',   dot: 'bg-amber-400',   label: '⚠ Warn' },
  fail:        { bg: 'bg-red-500/10',     text: 'text-red-400',     border: 'border-red-500/20',     dot: 'bg-red-400',     label: '✗ Fail' },
};

const StatusBadge: FC<Props> = ({ status, size = 'sm' }) => {
  const c = config[status] ?? config['complex'];
  const padding = size === 'xs' ? 'px-1.5 py-0.5 text-xs' : size === 'sm' ? 'px-2.5 py-1 text-xs' : 'px-3 py-1.5 text-sm';
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full font-medium border ${padding} ${c.bg} ${c.text} ${c.border}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot} flex-shrink-0`} />
      {c.label}
    </span>
  );
};

export default StatusBadge;
