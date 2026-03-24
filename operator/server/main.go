package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/server/handler"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dtmv1alpha1.AddToScheme(scheme))
}

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":8090", "HTTP listen address")
	flag.Parse()

	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	mux := http.NewServeMux()

	// API routes
	h := handler.New(k8sClient)
	mux.HandleFunc("GET /api/v1/policies", h.ListPolicies)
	mux.HandleFunc("GET /api/v1/policies/{namespace}/{name}", h.GetPolicy)
	mux.HandleFunc("POST /api/v1/policies", h.CreatePolicy)
	mux.HandleFunc("PUT /api/v1/policies/{namespace}/{name}", h.UpdatePolicy)
	mux.HandleFunc("DELETE /api/v1/policies/{namespace}/{name}", h.DeletePolicy)

	mux.HandleFunc("GET /api/v1/results", h.ListResults)
	mux.HandleFunc("GET /api/v1/results/{namespace}/{name}", h.GetResult)

	mux.HandleFunc("POST /api/v1/run/{namespace}/{name}", h.RunNow)

	mux.HandleFunc("GET /api/v1/settings", h.GetSettings)
	mux.HandleFunc("PUT /api/v1/settings", h.UpdateSettings)

	// Auth routes
	mux.HandleFunc("POST /api/v1/login", handler.Login)
	mux.HandleFunc("GET /api/v1/auth/check", handler.AuthCheck)

	// Health check — exempt from auth and used by k8s probes.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// Serve embedded UI
	uiHandler := handler.NewUIHandler()
	mux.Handle("/", uiHandler)

	// Middleware chain: CORS → JWT Auth → mux
	wrapped := corsMiddleware(handler.JWTAuthMiddleware(mux))

	fmt.Fprintf(os.Stdout, "DTM API Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, wrapped); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
