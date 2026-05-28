package router

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/dist
var embeddedFiles embed.FS

// RegisterStaticRoutes registers handlers to serve frontend assets from memory
func RegisterStaticRoutes(mux *http.ServeMux) {
	fsys, err := fs.Sub(embeddedFiles, "web/dist")
	if err != nil {
		// Fallback to empty server if frontend files are missing
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<h1>NexusAI-Gateway Static Dashboard (Placeholder)</h1><p>Frontend assets not built.</p>"))
		})
		return
	}
	mux.Handle("/", http.FileServer(http.FS(fsys)))
}
