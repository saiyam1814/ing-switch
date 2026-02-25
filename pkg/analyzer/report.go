package analyzer

import "github.com/saiyam1814/ing-switch/pkg/scanner"

// Analyzer performs compatibility analysis for a given target controller.
type Analyzer struct {
	target string
}

// NewAnalyzer creates an Analyzer for the given target ("traefik" or "gateway-api").
func NewAnalyzer(target string) *Analyzer {
	return &Analyzer{target: target}
}

// AnalysisReport is the full output of an analysis run.
type AnalysisReport struct {
	Target        string          `json:"target"`
	IngressReports []IngressReport `json:"ingressReports"`
	Summary       Summary         `json:"summary"`
}

// IngressReport is the analysis of a single Ingress resource.
type IngressReport struct {
	Namespace string              `json:"namespace"`
	Name      string              `json:"name"`
	Mappings  []AnnotationMapping `json:"mappings"`
	OverallStatus string          `json:"overallStatus"` // "ready" | "workaround" | "breaking"
}

// Summary aggregates across all ingresses.
type Summary struct {
	Total           int `json:"total"`
	FullyCompatible int `json:"fullyCompatible"`
	NeedsWorkaround int `json:"needsWorkaround"`
	HasUnsupported  int `json:"hasUnsupported"`
}

// Analyze performs compatibility analysis on all ingresses in the scan result.
func (a *Analyzer) Analyze(scan *scanner.ScanResult) *AnalysisReport {
	report := &AnalysisReport{
		Target: a.target,
	}

	for _, ing := range scan.Ingresses {
		ir := a.analyzeIngress(ing)
		report.IngressReports = append(report.IngressReports, ir)

		report.Summary.Total++
		switch ir.OverallStatus {
		case "ready":
			report.Summary.FullyCompatible++
		case "workaround":
			report.Summary.NeedsWorkaround++
		case "breaking":
			report.Summary.HasUnsupported++
		}
	}

	return report
}

func (a *Analyzer) analyzeIngress(ing scanner.IngressInfo) IngressReport {
	ir := IngressReport{
		Namespace: ing.Namespace,
		Name:      ing.Name,
	}

	hasUnsupported := false
	hasPartial := false

	for key, value := range ing.NginxAnnotations {
		mapping := MapAnnotation(key, value, a.target)
		ir.Mappings = append(ir.Mappings, mapping)

		switch mapping.Status {
		case StatusUnsupported:
			hasUnsupported = true
		case StatusPartial:
			hasPartial = true
		}
	}

	switch {
	case hasUnsupported:
		ir.OverallStatus = "breaking"
	case hasPartial:
		ir.OverallStatus = "workaround"
	default:
		ir.OverallStatus = "ready"
	}

	return ir
}
