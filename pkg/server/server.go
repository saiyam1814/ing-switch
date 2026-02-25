package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var embeddedUI embed.FS

// Server is the local web server that serves the React UI and REST API.
type Server struct {
	addr        string
	kubeconfig  string
	kubecontext string
}

// NewServer creates a new Server.
func NewServer(addr, kubeconfig, kubecontext string) *Server {
	return &Server{
		addr:        addr,
		kubeconfig:  kubeconfig,
		kubecontext: kubecontext,
	}
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register API handlers
	api := NewAPIHandler(s.kubeconfig, s.kubecontext)
	mux.HandleFunc("/api/scan", api.HandleScan)
	mux.HandleFunc("/api/analyze", api.HandleAnalyze)
	mux.HandleFunc("/api/migrate", api.HandleMigrate)
	mux.HandleFunc("/api/validate", api.HandleValidate)
	mux.HandleFunc("/api/download", api.HandleDownload)
	mux.HandleFunc("/api/apply", api.HandleApply)

	// Serve embedded React UI for all other paths
	uiFS, err := fs.Sub(embeddedUI, "dist")
	if err != nil {
		// If no dist folder, serve a placeholder
		mux.HandleFunc("/", servePlaceholder)
	} else {
		fileServer := http.FileServer(http.FS(uiFS))
		mux.Handle("/", SPAHandler{fileServer: fileServer, uiFS: uiFS})
	}

	return http.ListenAndServe(s.addr, mux)
}

// SPAHandler serves the React SPA, falling back to index.html for unknown routes.
type SPAHandler struct {
	fileServer http.Handler
	uiFS       fs.FS
}

func (h SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if the file exists
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	_, err := h.uiFS.Open(path[1:]) // Strip leading /
	if err != nil {
		// File not found — serve index.html for SPA routing
		r.URL.Path = "/"
	}
	h.fileServer.ServeHTTP(w, r)
}

func servePlaceholder(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>ing-switch UI</title></head>
<body>
<h1>ing-switch</h1>
<p>The UI is not built. Run <code>make build-ui</code> first.</p>
<p>For CLI usage: <code>ing-switch --help</code></p>
</body>
</html>`))
}
