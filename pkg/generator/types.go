package generator

// GeneratedFile represents a single file produced by the migration engine.
type GeneratedFile struct {
	// Path relative to the output directory (e.g., "02-middlewares/ssl-redirect.yaml")
	RelPath string `json:"relPath"`
	// Content is the file contents (YAML, shell script, or markdown)
	Content string `json:"content"`
	// Description is a human-readable explanation of this file
	Description string `json:"description"`
	// Category groups files for UI display (e.g., "middleware", "ingress", "install")
	Category string `json:"category"`
}
