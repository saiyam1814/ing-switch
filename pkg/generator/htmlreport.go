package generator

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// GenerateHTMLReport produces a self-contained HTML report of the migration analysis.
func GenerateHTMLReport(scan *scanner.ScanResult, report *analyzer.AnalysisReport) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ing-switch Migration Report</title>
<style>
  :root {
    --green: #22c55e; --yellow: #eab308; --red: #ef4444;
    --bg: #0f172a; --surface: #1e293b; --border: #334155;
    --text: #e2e8f0; --muted: #94a3b8;
  }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: var(--bg); color: var(--text); line-height: 1.6; padding: 2rem; }
  .container { max-width: 1100px; margin: 0 auto; }
  h1 { font-size: 1.8rem; margin-bottom: 0.5rem; }
  h2 { font-size: 1.3rem; margin: 2rem 0 1rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
  h3 { font-size: 1.1rem; margin: 1.5rem 0 0.5rem; }
  .meta { color: var(--muted); font-size: 0.9rem; margin-bottom: 2rem; }
  .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin: 1rem 0; }
  .card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.2rem; }
  .card .label { color: var(--muted); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .card .value { font-size: 2rem; font-weight: 700; margin-top: 0.3rem; }
  .card .value.green { color: var(--green); }
  .card .value.yellow { color: var(--yellow); }
  .card .value.red { color: var(--red); }
  table { width: 100%; border-collapse: collapse; margin: 1rem 0; font-size: 0.9rem; }
  th, td { padding: 0.6rem 0.8rem; text-align: left; border-bottom: 1px solid var(--border); }
  th { background: var(--surface); color: var(--muted); font-weight: 600; font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.04em; }
  tr:hover { background: rgba(255,255,255,0.03); }
  .badge { display: inline-block; padding: 0.15rem 0.6rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600; }
  .badge-green { background: rgba(34,197,94,0.15); color: var(--green); }
  .badge-yellow { background: rgba(234,179,8,0.15); color: var(--yellow); }
  .badge-red { background: rgba(239,68,68,0.15); color: var(--red); }
  .ingress-section { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.2rem; margin: 1rem 0; }
  .bar-container { display: flex; height: 8px; border-radius: 4px; overflow: hidden; margin: 0.5rem 0; }
  .bar-green { background: var(--green); }
  .bar-yellow { background: var(--yellow); }
  .bar-red { background: var(--red); }
  .cluster-info { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1.2rem; margin: 1rem 0; }
  .cluster-info dt { color: var(--muted); font-size: 0.8rem; }
  .cluster-info dd { margin-bottom: 0.8rem; }
  code { background: rgba(255,255,255,0.08); padding: 0.15rem 0.4rem; border-radius: 4px; font-size: 0.85em; }
  footer { margin-top: 3rem; padding-top: 1rem; border-top: 1px solid var(--border); color: var(--muted); font-size: 0.8rem; text-align: center; }
</style>
</head>
<body>
<div class="container">
`)

	sb.WriteString(`<h1>ing-switch Migration Report</h1>`)
	sb.WriteString(fmt.Sprintf(`<p class="meta">Target: <strong>%s</strong> &middot; Generated: %s</p>`,
		html.EscapeString(report.Target), time.Now().Format("2006-01-02 15:04 MST")))

	// Cluster info
	if scan != nil {
		sb.WriteString(`<div class="cluster-info"><dl>`)
		sb.WriteString(fmt.Sprintf(`<dt>Cluster</dt><dd>%s</dd>`, html.EscapeString(scan.ClusterName)))
		if scan.Controller.Detected {
			sb.WriteString(fmt.Sprintf(`<dt>Controller</dt><dd>%s %s (%s)</dd>`,
				html.EscapeString(scan.Controller.Type),
				html.EscapeString(scan.Controller.Version),
				html.EscapeString(scan.Controller.Namespace)))
		}
		sb.WriteString(fmt.Sprintf(`<dt>Namespaces</dt><dd>%s</dd>`, html.EscapeString(strings.Join(scan.Namespaces, ", "))))
		sb.WriteString(`</dl></div>`)
	}

	// Summary cards
	total := report.Summary.Total
	sb.WriteString(`<h2>Summary</h2><div class="cards">`)
	sb.WriteString(fmt.Sprintf(`<div class="card"><div class="label">Total Ingresses</div><div class="value">%d</div></div>`, total))
	sb.WriteString(fmt.Sprintf(`<div class="card"><div class="label">Fully Compatible</div><div class="value green">%d</div></div>`, report.Summary.FullyCompatible))
	sb.WriteString(fmt.Sprintf(`<div class="card"><div class="label">Needs Workaround</div><div class="value yellow">%d</div></div>`, report.Summary.NeedsWorkaround))
	sb.WriteString(fmt.Sprintf(`<div class="card"><div class="label">Has Unsupported</div><div class="value red">%d</div></div>`, report.Summary.HasUnsupported))
	sb.WriteString(`</div>`)

	// Progress bar
	if total > 0 {
		greenPct := report.Summary.FullyCompatible * 100 / total
		yellowPct := report.Summary.NeedsWorkaround * 100 / total
		redPct := 100 - greenPct - yellowPct
		sb.WriteString(`<div class="bar-container">`)
		sb.WriteString(fmt.Sprintf(`<div class="bar-green" style="width:%d%%"></div>`, greenPct))
		sb.WriteString(fmt.Sprintf(`<div class="bar-yellow" style="width:%d%%"></div>`, yellowPct))
		sb.WriteString(fmt.Sprintf(`<div class="bar-red" style="width:%d%%"></div>`, redPct))
		sb.WriteString(`</div>`)
	}

	// Per-ingress detail
	sb.WriteString(`<h2>Ingress Analysis</h2>`)
	for _, ir := range report.IngressReports {
		badgeClass := "badge-green"
		badgeLabel := "Ready"
		switch ir.OverallStatus {
		case "workaround":
			badgeClass = "badge-yellow"
			badgeLabel = "Workaround"
		case "breaking":
			badgeClass = "badge-red"
			badgeLabel = "Breaking"
		}

		sb.WriteString(`<div class="ingress-section">`)
		sb.WriteString(fmt.Sprintf(`<h3>%s/%s <span class="badge %s">%s</span></h3>`,
			html.EscapeString(ir.Namespace), html.EscapeString(ir.Name), badgeClass, badgeLabel))

		if len(ir.Mappings) == 0 {
			sb.WriteString(`<p>No annotations — ready to migrate as-is.</p>`)
		} else {
			sb.WriteString(`<table><thead><tr><th>Annotation</th><th>Value</th><th>Status</th><th>Target Resource</th><th>Notes</th></tr></thead><tbody>`)
			for _, m := range ir.Mappings {
				statusBadge := `<span class="badge badge-green">supported</span>`
				switch m.Status {
				case analyzer.StatusPartial:
					statusBadge = `<span class="badge badge-yellow">partial</span>`
				case analyzer.StatusUnsupported:
					statusBadge = `<span class="badge badge-red">unsupported</span>`
				}
				val := m.OriginalValue
				if len(val) > 60 {
					val = val[:60] + "..."
				}
				sb.WriteString(fmt.Sprintf(`<tr><td><code>%s</code></td><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td></tr>`,
					html.EscapeString(m.OriginalKey),
					html.EscapeString(val),
					statusBadge,
					html.EscapeString(m.TargetResource),
					html.EscapeString(m.Note)))
			}
			sb.WriteString(`</tbody></table>`)
		}
		sb.WriteString(`</div>`)
	}

	// Readiness score
	if total > 0 {
		score := ((report.Summary.FullyCompatible * 100) + (report.Summary.NeedsWorkaround * 70)) / total
		sb.WriteString(`<h2>Migration Readiness</h2>`)
		sb.WriteString(fmt.Sprintf(`<div class="cards"><div class="card"><div class="label">Readiness Score</div><div class="value green">%d%%</div></div></div>`, score))
	}

	sb.WriteString(fmt.Sprintf(`<footer>Generated by <strong>ing-switch</strong> &middot; %s</footer>`, time.Now().Format("2006-01-02")))
	sb.WriteString(`</div></body></html>`)

	return sb.String()
}
