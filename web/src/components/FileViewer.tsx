import { useState, useEffect, type FC } from 'react';
import type { GeneratedFile } from '../types';

interface Props {
  files: GeneratedFile[];
  onDownloadAll?: () => void;
  initialPath?: string; // file relPath to select on mount or when changed
}

const categoryOrder = ['install', 'gateway', 'middleware', 'ingress', 'httproute', 'policy', 'verify', 'guide', 'cleanup'];

const categoryConfig: Record<string, { label: string; icon: string; color: string }> = {
  install:    { label: 'Installation',   icon: '📦', color: 'text-blue-400' },
  gateway:    { label: 'Gateway',        icon: '🌐', color: 'text-purple-400' },
  middleware: { label: 'Middlewares',    icon: '🔧', color: 'text-cyan-400' },
  ingress:    { label: 'Ingresses',      icon: '🔀', color: 'text-emerald-400' },
  httproute:  { label: 'HTTPRoutes',     icon: '🛣️',  color: 'text-indigo-400' },
  policy:     { label: 'Policies',       icon: '🛡️',  color: 'text-rose-400' },
  verify:     { label: 'Verification',   icon: '✅', color: 'text-emerald-400' },
  guide:      { label: 'Guides',         icon: '📖', color: 'text-amber-400' },
  cleanup:    { label: 'Cleanup',        icon: '🗑️',  color: 'text-red-400' },
};

const FileViewer: FC<Props> = ({ files, onDownloadAll, initialPath }) => {
  const [activeFile, setActiveFile] = useState<GeneratedFile | null>(
    initialPath ? (files.find(f => f.relPath === initialPath) ?? files[0] ?? null) : (files[0] ?? null)
  );
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (initialPath) {
      const found = files.find(f => f.relPath === initialPath);
      if (found) setActiveFile(found);
    }
  }, [initialPath, files]);

  const grouped = files.reduce<Record<string, GeneratedFile[]>>((acc, f) => {
    (acc[f.category] ??= []).push(f);
    return acc;
  }, {});

  const handleCopy = async () => {
    if (!activeFile) return;
    await navigator.clipboard.writeText(activeFile.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownloadFile = () => {
    if (!activeFile) return;
    const blob = new Blob([activeFile.content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = activeFile.relPath.split('/').pop() ?? 'file';
    a.click();
    URL.revokeObjectURL(url);
  };

  const isYaml = activeFile?.relPath.endsWith('.yaml') || activeFile?.relPath.endsWith('.yml');
  const isShell = activeFile?.relPath.endsWith('.sh');
  const isMd = activeFile?.relPath.endsWith('.md');

  return (
    <div className="flex border border-slate-700/50 rounded-xl overflow-hidden bg-slate-900" style={{ height: 580 }}>
      {/* Sidebar */}
      <div className="w-64 border-r border-slate-700/50 overflow-y-auto flex-shrink-0 bg-slate-900">
        {onDownloadAll && (
          <div className="p-3 border-b border-slate-700/50">
            <button
              onClick={onDownloadAll}
              className="w-full py-2 px-3 bg-gradient-to-r from-blue-600 to-blue-500 hover:from-blue-500 hover:to-blue-400 text-white text-sm font-semibold rounded-lg transition-all shadow-lg shadow-blue-900/30 flex items-center justify-center gap-2"
            >
              <span>⬇</span> Download All (ZIP)
            </button>
          </div>
        )}
        <div className="py-2">
          {categoryOrder.map(cat => {
            const catFiles = grouped[cat];
            if (!catFiles?.length) return null;
            const cfg = categoryConfig[cat] ?? { label: cat, icon: '📄', color: 'text-slate-400' };
            return (
              <div key={cat} className="mb-1">
                <div className="flex items-center gap-2 px-3 py-2">
                  <span className="text-sm">{cfg.icon}</span>
                  <span className={`text-xs font-semibold uppercase tracking-wider ${cfg.color}`}>{cfg.label}</span>
                  <span className="ml-auto text-xs text-slate-600">{catFiles.length}</span>
                </div>
                {catFiles.map(f => {
                  const filename = f.relPath.split('/').pop() ?? f.relPath;
                  const isActive = activeFile?.relPath === f.relPath;
                  return (
                    <button
                      key={f.relPath}
                      onClick={() => setActiveFile(f)}
                      className={`w-full text-left px-3 py-2 flex items-start gap-2 transition-all ${
                        isActive
                          ? 'bg-blue-500/20 border-r-2 border-blue-500'
                          : 'hover:bg-slate-800/60'
                      }`}
                    >
                      <span className="text-xs mt-0.5 flex-shrink-0">
                        {filename.endsWith('.yaml') || filename.endsWith('.yml') ? '📄' :
                         filename.endsWith('.sh') ? '⚙️' :
                         filename.endsWith('.md') ? '📝' : '📄'}
                      </span>
                      <div className="min-w-0">
                        <div className={`text-xs font-mono truncate ${isActive ? 'text-blue-300' : 'text-slate-300'}`}>
                          {filename}
                        </div>
                        <div className="text-xs text-slate-600 truncate mt-0.5">{f.description}</div>
                      </div>
                    </button>
                  );
                })}
              </div>
            );
          })}
        </div>
      </div>

      {/* Content panel */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {activeFile ? (
          <>
            {/* File header */}
            <div className="flex items-center justify-between px-4 py-3 bg-slate-800/80 border-b border-slate-700/50 flex-shrink-0">
              <div className="flex items-center gap-3 min-w-0">
                <span className="text-sm">
                  {isYaml ? '📄' : isShell ? '⚙️' : isMd ? '📝' : '📄'}
                </span>
                <div>
                  <div className="text-sm font-mono text-slate-200">{activeFile.relPath.split('/').pop()}</div>
                  <div className="text-xs text-slate-500 mt-0.5 truncate">{activeFile.relPath}</div>
                </div>
                <span className={`text-xs px-2 py-0.5 rounded font-mono ${
                  isYaml ? 'bg-blue-500/20 text-blue-400' :
                  isShell ? 'bg-emerald-500/20 text-emerald-400' :
                  'bg-slate-700 text-slate-400'
                }`}>
                  {isYaml ? 'YAML' : isShell ? 'Shell' : isMd ? 'Markdown' : 'Text'}
                </span>
              </div>
              <div className="flex gap-2 flex-shrink-0">
                <button
                  onClick={handleCopy}
                  className={`px-3 py-1.5 text-xs rounded-lg font-medium transition-all flex items-center gap-1.5 ${
                    copied
                      ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                      : 'bg-slate-700 hover:bg-slate-600 text-slate-300 border border-slate-600'
                  }`}
                >
                  {copied ? '✓ Copied' : '📋 Copy'}
                </button>
                <button
                  onClick={handleDownloadFile}
                  className="px-3 py-1.5 text-xs rounded-lg font-medium bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 border border-blue-500/30 transition-all flex items-center gap-1.5"
                >
                  ⬇ Download
                </button>
              </div>
            </div>

            {/* File content */}
            <div className="flex-1 overflow-auto bg-[#0d1117]">
              <pre className="text-xs font-mono p-5 leading-6 text-slate-300 min-h-full">
                {activeFile.content}
              </pre>
            </div>

            {/* Footer */}
            <div className="px-4 py-1.5 bg-slate-800/50 border-t border-slate-700/50 flex items-center gap-4 text-xs text-slate-600 flex-shrink-0">
              <span>{activeFile.content.split('\n').length} lines</span>
              <span>{(activeFile.content.length / 1024).toFixed(1)} KB</span>
              <span className="text-slate-700">·</span>
              <span>{activeFile.description}</span>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-slate-600">
            <div className="text-center">
              <div className="text-4xl mb-3">👈</div>
              <p>Select a file to view</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default FileViewer;
