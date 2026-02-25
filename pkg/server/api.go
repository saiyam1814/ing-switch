package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/migrator/gatewayapi"
	"github.com/saiyam1814/ing-switch/pkg/migrator/traefik"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// APIHandler handles all /api/* requests.
type APIHandler struct {
	kubeconfig  string
	kubecontext string
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(kubeconfig, kubecontext string) *APIHandler {
	return &APIHandler{kubeconfig: kubeconfig, kubecontext: kubecontext}
}

func (h *APIHandler) HandleScan(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	kubeconfig := r.URL.Query().Get("kubeconfig")
	if kubeconfig == "" {
		kubeconfig = h.kubeconfig
	}
	kubecontext := r.URL.Query().Get("context")
	if kubecontext == "" {
		kubecontext = h.kubecontext
	}
	ns := r.URL.Query().Get("namespace")

	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Cannot connect to cluster: %v", err))
		return
	}

	result, err := s.Scan(ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
		return
	}

	writeJSON(w, result)
}

func (h *APIHandler) HandleAnalyze(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	target := r.URL.Query().Get("target")
	if target == "" {
		writeError(w, http.StatusBadRequest, "target parameter required (traefik or gateway-api)")
		return
	}

	ns := r.URL.Query().Get("namespace")

	s, err := scanner.NewScanner(h.kubeconfig, h.kubecontext)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Cannot connect to cluster: %v", err))
		return
	}

	scanResult, err := s.Scan(ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
		return
	}

	a := analyzer.NewAnalyzer(target)
	report := a.Analyze(scanResult)

	writeJSON(w, report)
}

type migrateRequest struct {
	Target    string `json:"target"`
	OutputDir string `json:"outputDir"`
	Namespace string `json:"namespace"`
}

// AnnotationIssue is a single annotation that needs attention during migration.
type AnnotationIssue struct {
	Key            string `json:"key"`                      // short key, e.g. "proxy-body-size"
	Value          string `json:"value"`                    // annotation value set by the user
	Status         string `json:"status"`                   // "partial" | "unsupported"
	TargetResource string `json:"targetResource"`           // e.g. "Middleware (RateLimit)"
	FileCategory   string `json:"fileCategory,omitempty"`   // category of the generated file that handles this (for navigation)
	Note           string `json:"note"`                     // one-line description
	What           string `json:"what"`                     // what the annotation does
	Fix            string `json:"fix"`                      // step-by-step fix instructions
	Example        string `json:"example"`                  // YAML/command example
	DocsLink       string `json:"docsLink"`                 // upstream docs URL
	Consequence    string `json:"consequence,omitempty"`    // what happens if not migrated
	IssueUrl       string `json:"issueUrl,omitempty"`       // upstream GitHub issue URL
}

// IngressMigrationSummary describes per-ingress migration status.
type IngressMigrationSummary struct {
	Namespace     string            `json:"namespace"`
	Name          string            `json:"name"`
	OverallStatus string            `json:"overallStatus"` // "ready" | "workaround" | "breaking"
	Issues        []AnnotationIssue `json:"issues"`        // partial + unsupported annotations
}

type migrateResponse struct {
	Files        []generator.GeneratedFile `json:"files"`
	Summary      string                    `json:"summary"`
	IngressCount int                       `json:"ingressCount"`
	PerIngress   []IngressMigrationSummary `json:"perIngress"`
}

func (h *APIHandler) HandleMigrate(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	var req migrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Target = r.URL.Query().Get("target")
		req.Namespace = r.URL.Query().Get("namespace")
	}

	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target required (traefik or gateway-api)")
		return
	}

	s, err := scanner.NewScanner(h.kubeconfig, h.kubecontext)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Cannot connect to cluster: %v", err))
		return
	}

	scanResult, err := s.Scan(req.Namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
		return
	}

	a := analyzer.NewAnalyzer(req.Target)
	report := a.Analyze(scanResult)

	var files []generator.GeneratedFile
	switch req.Target {
	case "traefik":
		m := traefik.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	case "gateway-api":
		m := gatewayapi.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	default:
		writeError(w, http.StatusBadRequest, "unknown target")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Migration failed: %v", err))
		return
	}

	if req.OutputDir != "" {
		gen := generator.NewOutputGenerator(req.OutputDir)
		if err := gen.Write(files, report); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Writing files: %v", err))
			return
		}
	}

	// Prepend the migration report as the first file in the response
	reportContent := generator.GenerateMigrationReport(files, report)
	reportFile := generator.GeneratedFile{
		RelPath:     "00-migration-report.md",
		Content:     reportContent,
		Description: "Full migration summary and annotation analysis",
		Category:    "guide",
	}
	allFiles := append([]generator.GeneratedFile{reportFile}, files...)

	// Build per-ingress summaries
	perIngress := buildPerIngressSummaries(report, req.Target)

	writeJSON(w, migrateResponse{
		Files:        allFiles,
		Summary:      fmt.Sprintf("Generated %d migration files for %s across %d ingresses", len(allFiles), req.Target, len(scanResult.Ingresses)),
		IngressCount: len(scanResult.Ingresses),
		PerIngress:   perIngress,
	})
}

// targetResourceToFileCategory maps a target resource description to the file category
// it belongs to (for UI navigation to the right generated file).
func targetResourceToFileCategory(targetResource string) string {
	tr := strings.ToLower(targetResource)
	switch {
	case strings.Contains(tr, "middleware") || strings.Contains(tr, "serversTransport") || strings.Contains(tr, "serverstransport"):
		return "middleware"
	case strings.Contains(tr, "httproute") || strings.Contains(tr, "urlrewrite") || strings.Contains(tr, "requestredirect") || strings.Contains(tr, "grpcroute") || strings.Contains(tr, "tlsroute"):
		return "httproute"
	case strings.Contains(tr, "policy") || strings.Contains(tr, "backendtrafficpolicy") || strings.Contains(tr, "securitypolicy") || strings.Contains(tr, "backendlbpolicy") || strings.Contains(tr, "backendtlspolicy"):
		return "policy"
	case strings.Contains(tr, "gateway") || strings.Contains(tr, "gatewayclass"):
		return "gateway"
	case strings.Contains(tr, "service") || strings.Contains(tr, "sticky") || strings.Contains(tr, "nativelb") || strings.Contains(tr, "router") || strings.Contains(tr, "ingress"):
		return "ingress"
	default:
		return ""
	}
}

func buildPerIngressSummaries(report *analyzer.AnalysisReport, target string) []IngressMigrationSummary {
	summaries := make([]IngressMigrationSummary, 0, len(report.IngressReports))
	for _, ir := range report.IngressReports {
		sum := IngressMigrationSummary{
			Namespace:     ir.Namespace,
			Name:          ir.Name,
			OverallStatus: ir.OverallStatus,
			Issues:        []AnnotationIssue{}, // always a non-nil slice
		}
		for _, m := range ir.Mappings {
			if m.Status != analyzer.StatusPartial && m.Status != analyzer.StatusUnsupported {
				continue
			}
			shortKey := strings.TrimPrefix(m.OriginalKey, "nginx.ingress.kubernetes.io/")
			guide := GetAnnotationGuide(target, shortKey)
			issue := AnnotationIssue{
				Key:            shortKey,
				Value:          m.OriginalValue,
				Status:         string(m.Status),
				TargetResource: m.TargetResource,
				FileCategory:   targetResourceToFileCategory(m.TargetResource),
				Note:           m.Note,
				What:           guide.What,
				Fix:            guide.Fix,
				Example:        guide.Example,
				DocsLink:       guide.DocsLink,
				Consequence:    guide.Consequence,
				IssueUrl:       guide.IssueUrl,
			}
			// Fall back to the mapping note if no dedicated guide exists
			if issue.What == "" {
				issue.What = m.Note
			}
			if issue.Fix == "" && m.Note != "" {
				issue.Fix = m.Note
			}
			sum.Issues = append(sum.Issues, issue)
		}
		summaries = append(summaries, sum)
	}
	return summaries
}

// --- Apply endpoint ---

type applyRequest struct {
	Target    string `json:"target"`
	Category  string `json:"category"` // "middleware" | "ingress" | "gateway" | "httproute" | "policy"
	DryRun    bool   `json:"dryRun"`
	Namespace string `json:"namespace"`
}

type applyResponse struct {
	Success  bool     `json:"success"`
	Output   string   `json:"output"`
	DryRun   bool     `json:"dryRun"`
	Applied  []string `json:"applied"`
	Error    string   `json:"error,omitempty"`
}

// applyableCategories are categories whose YAML files can be kubectl-applied directly.
var applyableCategories = map[string]bool{
	"middleware": true,
	"ingress":    true,
	"gateway":    true,
	"httproute":  true,
	"policy":     true,
}

func (h *APIHandler) HandleApply(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	var req applyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Target == "" || req.Category == "" {
		writeError(w, http.StatusBadRequest, "target and category required")
		return
	}

	// Non-kubectl categories: return instructions
	if !applyableCategories[req.Category] {
		msg := categoryInstructions(req.Category, req.Target)
		writeJSON(w, applyResponse{Success: true, Output: msg, DryRun: req.DryRun})
		return
	}

	// Generate migration files
	s, err := scanner.NewScanner(h.kubeconfig, h.kubecontext)
	if err != nil {
		writeJSON(w, applyResponse{Success: false, Error: fmt.Sprintf("Cannot connect to cluster: %v", err)})
		return
	}

	scanResult, err := s.Scan(req.Namespace)
	if err != nil {
		writeJSON(w, applyResponse{Success: false, Error: fmt.Sprintf("Scan failed: %v", err)})
		return
	}

	a := analyzer.NewAnalyzer(req.Target)
	report := a.Analyze(scanResult)

	var files []generator.GeneratedFile
	switch req.Target {
	case "traefik":
		m := traefik.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	case "gateway-api":
		m := gatewayapi.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	default:
		writeJSON(w, applyResponse{Success: false, Error: "unknown target"})
		return
	}
	if err != nil {
		writeJSON(w, applyResponse{Success: false, Error: fmt.Sprintf("Migration failed: %v", err)})
		return
	}

	// Filter to the requested category and collect YAML files
	tmpDir, err := os.MkdirTemp("", "ing-switch-apply-*")
	if err != nil {
		writeJSON(w, applyResponse{Success: false, Error: "Cannot create temp dir"})
		return
	}
	defer os.RemoveAll(tmpDir)

	var applied []string
	for _, f := range files {
		if f.Category != req.Category {
			continue
		}
		if !strings.HasSuffix(f.RelPath, ".yaml") && !strings.HasSuffix(f.RelPath, ".yml") {
			continue
		}
		fname := filepath.Base(f.RelPath)
		dest := filepath.Join(tmpDir, fname)
		if err := os.WriteFile(dest, []byte(f.Content), 0644); err != nil {
			continue
		}
		applied = append(applied, fname)
	}

	if len(applied) == 0 {
		writeJSON(w, applyResponse{
			Success: true,
			Output:  fmt.Sprintf("No YAML files found for category '%s'. This may be a shell script or guide — download it from the file viewer.", req.Category),
			DryRun:  req.DryRun,
		})
		return
	}

	// Build kubectl command
	args := buildKubectlArgs(h.kubeconfig, h.kubecontext, req.DryRun, tmpDir)
	cmd := exec.Command("kubectl", args...)
	output, cmdErr := cmd.CombinedOutput()

	resp := applyResponse{
		Success: cmdErr == nil,
		Output:  string(output),
		DryRun:  req.DryRun,
		Applied: applied,
	}
	if cmdErr != nil {
		resp.Error = cmdErr.Error()
	}
	writeJSON(w, resp)
}

func buildKubectlArgs(kubeconfig, kubecontext string, dryRun bool, dir string) []string {
	args := []string{}
	if kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	if kubecontext != "" {
		args = append(args, "--context", kubecontext)
	}
	args = append(args, "apply", "-f", dir)
	if dryRun {
		args = append(args, "--dry-run=server")
	}
	return args
}

func categoryInstructions(category, target string) string {
	switch category {
	case "install":
		if target == "traefik" {
			return "Helm install required — run the generated helm-install.sh script:\n\n" +
				"  helm repo add traefik https://traefik.github.io/charts\n" +
				"  helm repo update\n" +
				"  helm install traefik traefik/traefik -n traefik --create-namespace -f values.yaml\n\n" +
				"This installs Traefik alongside NGINX without affecting production traffic."
		}
		return "Install via kubectl + Helm — run the generated scripts:\n\n" +
			"  # Gateway API CRDs:\n" +
			"  kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml\n\n" +
			"  # Envoy Gateway:\n" +
			"  helm install eg oci://docker.io/envoyproxy/gateway-helm --version v1.3.0 -n envoy-gateway-system --create-namespace"
	case "verify":
		return "Download and run verify.sh from the file viewer:\n\n" +
			"  chmod +x verify.sh && ./verify.sh\n\n" +
			"This tests connectivity to each host via the new controller's IP without affecting DNS."
	case "guide":
		return "This is a documentation file. Open it from the file viewer to read the DNS migration guide."
	case "cleanup":
		return "⚠️  CAUTION: Cleanup removes NGINX. Only run after DNS has been updated and traffic confirmed on the new controller.\n\n" +
			"  Review 06-cleanup/remove-nginx.sh before running.\n" +
			"  Backup your NGINX Helm values first: helm get values ingress-nginx -n ingress-nginx > nginx-values-backup.yaml"
	}
	return "Download the files from the file viewer and apply manually."
}

// ValidationCheck is a single validation result.
type ValidationCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass" | "warn" | "fail"
	Message string `json:"message"`
}

func (h *APIHandler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	target := r.URL.Query().Get("target")
	ns := r.URL.Query().Get("namespace")

	result, err := runRichValidation(h.kubeconfig, h.kubecontext, target, ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, result)
}

func (h *APIHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	target := r.URL.Query().Get("target")
	if target == "" {
		writeError(w, http.StatusBadRequest, "target required")
		return
	}

	ns := r.URL.Query().Get("namespace")

	s, err := scanner.NewScanner(h.kubeconfig, h.kubecontext)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Cannot connect to cluster: %v", err))
		return
	}

	scanResult, err := s.Scan(ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
		return
	}

	a := analyzer.NewAnalyzer(target)
	report := a.Analyze(scanResult)

	var files []generator.GeneratedFile
	switch target {
	case "traefik":
		m := traefik.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	case "gateway-api":
		m := gatewayapi.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	default:
		writeError(w, http.StatusBadRequest, "unknown target")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	zipData, err := generator.CreateZip(files, report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Creating ZIP: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="migration-%s.zip"`, target))
	w.Write(zipData)
}

// Helpers

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func init() {
	_ = os.MkdirAll(filepath.Join("pkg", "server", "dist"), 0755)
}
