package handler

import (
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
)

// RunNow triggers an immediate scan by patching the run-now annotation.
func (h *Handler) RunNow(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	var policy dtmv1alpha1.ChecklistPolicy
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, &policy); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("policy not found: %v", err))
		return
	}

	if policy.Annotations == nil {
		policy.Annotations = make(map[string]string)
	}
	policy.Annotations["dtm.io/run-now"] = time.Now().UTC().Format(time.RFC3339)

	if err := h.client.Update(r.Context(), &policy); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("triggering run: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "triggered"})
}
