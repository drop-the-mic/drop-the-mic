package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	settingsConfigMap = "dtm-settings"
	settingsNamespace = "dtm-system"
	settingsKey       = "settings.json"
)

// GetSettings returns the DTM settings from a ConfigMap.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var cm corev1.ConfigMap
	err := h.client.Get(r.Context(), client.ObjectKey{
		Namespace: settingsNamespace,
		Name:      settingsConfigMap,
	}, &cm)

	if errors.IsNotFound(err) {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting settings: %v", err))
		return
	}

	data, ok := cm.Data[settingsKey]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(data), &settings); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("parsing settings: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// UpdateSettings updates the DTM settings ConfigMap.
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	data, err := json.Marshal(settings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("marshaling settings: %v", err))
		return
	}

	var cm corev1.ConfigMap
	key := client.ObjectKey{Namespace: settingsNamespace, Name: settingsConfigMap}
	getErr := h.client.Get(r.Context(), key, &cm)

	if errors.IsNotFound(getErr) {
		cm = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      settingsConfigMap,
				Namespace: settingsNamespace,
			},
			Data: map[string]string{
				settingsKey: string(data),
			},
		}
		if err := h.client.Create(r.Context(), &cm); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating settings: %v", err))
			return
		}
	} else if getErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("getting settings: %v", getErr))
		return
	} else {
		cm.Data[settingsKey] = string(data)
		if err := h.client.Update(r.Context(), &cm); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating settings: %v", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, settings)
}
