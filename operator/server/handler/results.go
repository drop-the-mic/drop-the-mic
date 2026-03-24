package handler

import (
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
)

// ListResults returns all ChecklistResults.
func (h *Handler) ListResults(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	policyRef := r.URL.Query().Get("policy")

	var list dtmv1alpha1.ChecklistResultList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if policyRef != "" {
		opts = append(opts, client.MatchingLabels{"dtm.io/policy": policyRef})
	}

	if err := h.client.List(r.Context(), &list, opts...); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing results: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, list.Items)
}

// GetResult returns a single ChecklistResult.
func (h *Handler) GetResult(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	var result dtmv1alpha1.ChecklistResult
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, &result); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("result not found: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, result)
}
