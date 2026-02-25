export interface ControllerInfo {
  detected: boolean;
  type: string;
  version: string;
  namespace: string;
  podName: string;
}

export interface PathInfo {
  host: string;
  path: string;
  pathType: string;
  serviceName: string;
  servicePort: number;
}

export interface ServiceRef {
  namespace: string;
  name: string;
  port: number;
}

export interface IngressInfo {
  namespace: string;
  name: string;
  ingressClass: string;
  hosts: string[];
  paths: PathInfo[];
  tlsEnabled: boolean;
  tlsSecrets: string[];
  annotations: Record<string, string>;
  nginxAnnotations: Record<string, string>;
  services: ServiceRef[];
  complexity: 'simple' | 'complex' | 'unsupported';
}

export interface ScanResult {
  clusterName: string;
  controller: ControllerInfo;
  ingresses: IngressInfo[];
  namespaces: string[];
}

export interface AnnotationMapping {
  originalKey: string;
  originalValue: string;
  status: 'supported' | 'partial' | 'unsupported';
  targetResource: string;
  generatedYaml?: string;
  note: string;
}

export interface IngressReport {
  namespace: string;
  name: string;
  mappings: AnnotationMapping[];
  overallStatus: 'ready' | 'workaround' | 'breaking';
}

export interface Summary {
  total: number;
  fullyCompatible: number;
  needsWorkaround: number;
  hasUnsupported: number;
}

export interface AnalysisReport {
  target: string;
  ingressReports: IngressReport[];
  summary: Summary;
}

export interface GeneratedFile {
  relPath: string;
  content: string;
  description: string;
  category: string;
}

/** A single annotation that needs attention during migration */
export interface AnnotationIssue {
  key: string;              // e.g. "proxy-body-size"
  value: string;            // value set on the ingress
  status: 'partial' | 'unsupported';
  targetResource: string;   // e.g. "Middleware (RateLimit)"
  note: string;             // one-line description
  what: string;             // what the annotation does
  fix: string;              // step-by-step fix instructions
  example: string;          // YAML/command example
  docsLink: string;         // upstream docs URL
  fileCategory?: string;    // category of generated file that handles this (for navigation)
  consequence?: string;     // what happens if not migrated
  issueUrl?: string;        // upstream GitHub issue URL for tracking
}

/** Per-ingress migration breakdown returned by /api/migrate */
export interface IngressMigrationSummary {
  namespace: string;
  name: string;
  overallStatus: 'ready' | 'workaround' | 'breaking';
  issues: AnnotationIssue[];
}

export interface MigrateResponse {
  files: GeneratedFile[];
  summary: string;
  ingressCount: number;
  perIngress: IngressMigrationSummary[];
}

export interface ValidationCheck {
  name: string;
  status: 'pass' | 'warn' | 'fail';
  message: string;
}

/** Rich validation result from /api/validate */
export interface ValidationResult {
  target: string;
  phase: 'pre-migration' | 'migrating' | 'post-migration';
  phaseDesc: string;
  checks: ValidationCheck[];
  overall: 'pass' | 'warn' | 'fail';
  nextSteps: string[];
}

/** Apply endpoint request */
export interface ApplyRequest {
  target: Target;
  category: string;
  dryRun: boolean;
  namespace?: string;
}

/** Apply endpoint response */
export interface ApplyResponse {
  success: boolean;
  output: string;
  dryRun: boolean;
  applied: string[];
  error?: string;
}

export type Target = 'traefik' | 'gateway-api';
