package handler

import (
	"io/fs"
	"net/http"

	"github.com/drop-the-mic/operator/server/embed"
)

// NewUIHandler returns an HTTP handler that serves the embedded UI files.
func NewUIHandler() http.Handler {
	subFS, err := fs.Sub(embed.UIAssets, "dist")
	if err != nil {
		// Fallback: serve a simple message if UI is not embedded
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body><h1>DTM UI not available</h1><p>Build the UI with 'make ui-build' first.</p></body></html>"))
		})
	}

	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file; if it doesn't exist, serve index.html for SPA routing
		f, err := subFS.Open(r.URL.Path[1:]) // strip leading /
		if err != nil {
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
}
