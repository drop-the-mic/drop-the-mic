package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
)

// ListPolicies returns all ChecklistPolicies.
func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	var list dtmv1alpha1.ChecklistPolicyList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := h.client.List(r.Context(), &list, opts...); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing policies: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, list.Items)
}

// GetPolicy returns a single ChecklistPolicy.
func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	var policy dtmv1alpha1.ChecklistPolicy
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, &policy); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("policy not found: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, policy)
}

// CreatePolicy creates a new ChecklistPolicy.
func (h *Handler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var policy dtmv1alpha1.ChecklistPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if err := h.client.Create(r.Context(), &policy); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating policy: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, policy)
}

// UpdatePolicy updates an existing ChecklistPolicy.
func (h *Handler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	var existing dtmv1alpha1.ChecklistPolicy
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, &existing); err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("policy not found: %v", err))
		return
	}

	var updated dtmv1alpha1.ChecklistPolicySpec
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	existing.Spec = updated
	if err := h.client.Update(r.Context(), &existing); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating policy: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// DeletePolicy deletes a ChecklistPolicy.
func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	policy := &dtmv1alpha1.ChecklistPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	if err := h.client.Delete(r.Context(), policy); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting policy: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
