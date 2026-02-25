import type {
  ScanResult,
  AnalysisReport,
  MigrateResponse,
  ValidationResult,
  ApplyRequest,
  ApplyResponse,
  Target,
} from '../types';

const BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export const api = {
  scan: (namespace?: string, kubeconfig?: string): Promise<ScanResult> => {
    const params = new URLSearchParams();
    if (namespace) params.set('namespace', namespace);
    if (kubeconfig) params.set('kubeconfig', kubeconfig);
    return get<ScanResult>(`/api/scan?${params}`);
  },

  analyze: (target: Target, namespace?: string): Promise<AnalysisReport> => {
    const params = new URLSearchParams({ target });
    if (namespace) params.set('namespace', namespace);
    return get<AnalysisReport>(`/api/analyze?${params}`);
  },

  migrate: (target: Target, outputDir?: string, namespace?: string): Promise<MigrateResponse> => {
    return post<MigrateResponse>('/api/migrate', { target, outputDir, namespace });
  },

  apply: (req: ApplyRequest): Promise<ApplyResponse> => {
    return post<ApplyResponse>('/api/apply', req);
  },

  validate: (target: Target, namespace?: string): Promise<ValidationResult> => {
    const params = new URLSearchParams({ target });
    if (namespace) params.set('namespace', namespace);
    return get<ValidationResult>(`/api/validate?${params}`);
  },

  downloadUrl: (target: Target): string => {
    return `${BASE}/api/download?target=${target}`;
  },
};
