package handler

import (
	"crypto/subtle"
	"net/http"
	"os"
)

// BasicAuthMiddleware returns an HTTP middleware that enforces Basic Auth.
// Credentials are read from DTM_AUTH_USER and DTM_AUTH_PASSWORD env vars.
// If both are empty or unset, the middleware is a passthrough.
// The /healthz endpoint is always exempt from authentication.
func BasicAuthMiddleware(next http.Handler) http.Handler {
	user := os.Getenv("DTM_AUTH_USER")
	pass := os.Getenv("DTM_AUTH_PASSWORD")

	// If credentials are not configured, skip auth entirely.
	if user == "" && pass == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check endpoint is always public (used by k8s probes).
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		givenUser, givenPass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="DTM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userMatch := subtle.ConstantTimeCompare([]byte(givenUser), []byte(user)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(givenPass), []byte(pass)) == 1

		if !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="DTM"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
