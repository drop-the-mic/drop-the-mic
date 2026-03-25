package handler

import (
	"fmt"
	"net/http"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListNamespaces returns all namespace names in the cluster.
func (h *Handler) ListNamespaces(w http.ResponseWriter, r *http.Request) {
	var list corev1.NamespaceList
	if err := h.client.List(r.Context(), &list); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing namespaces: %v", err))
		return
	}

	names := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		names = append(names, ns.Name)
	}
	sort.Strings(names)

	writeJSON(w, http.StatusOK, names)
}

type resourceSummary struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Ready string `json:"ready"`
}

// ListResources returns a summary of workload resources in a namespace.
func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("ns")

	var resources []resourceSummary
	opts := []client.ListOption{}
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}

	// Deployments
	var deps appsv1.DeploymentList
	if err := h.client.List(r.Context(), &deps, opts...); err == nil {
		for _, d := range deps.Items {
			resources = append(resources, resourceSummary{
				Kind:  "Deployment",
				Name:  d.Name,
				Ready: fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas),
			})
		}
	}

	// StatefulSets
	var stss appsv1.StatefulSetList
	if err := h.client.List(r.Context(), &stss, opts...); err == nil {
		for _, s := range stss.Items {
			resources = append(resources, resourceSummary{
				Kind:  "StatefulSet",
				Name:  s.Name,
				Ready: fmt.Sprintf("%d/%d", s.Status.ReadyReplicas, s.Status.Replicas),
			})
		}
	}

	// DaemonSets
	var dss appsv1.DaemonSetList
	if err := h.client.List(r.Context(), &dss, opts...); err == nil {
		for _, d := range dss.Items {
			resources = append(resources, resourceSummary{
				Kind:  "DaemonSet",
				Name:  d.Name,
				Ready: fmt.Sprintf("%d/%d", d.Status.NumberReady, d.Status.DesiredNumberScheduled),
			})
		}
	}

	if resources == nil {
		resources = []resourceSummary{}
	}

	writeJSON(w, http.StatusOK, resources)
}
