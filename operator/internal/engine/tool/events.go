package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type listEventsParams struct {
	Namespace     string `json:"namespace"`
	FieldSelector string `json:"fieldSelector,omitempty"`
	Limit         int64  `json:"limit,omitempty"`
}

func RegisterEvents(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "list_events",
		Description: "List Kubernetes events in a namespace. Can filter by involved object using fieldSelector (e.g. involvedObject.name=my-pod).",
		Parameters: []Parameter{
			{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
			{Name: "fieldSelector", Type: "string", Description: "Field selector to filter events", Required: false},
			{Name: "limit", Type: "integer", Description: "Max number of events to return (default 50)", Required: false},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p listEventsParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		limit := int64(50)
		if p.Limit > 0 {
			limit = p.Limit
		}

		events, err := client.CoreV1().Events(p.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: p.FieldSelector,
			Limit:         limit,
		})
		if err != nil {
			return "", fmt.Errorf("listing events: %w", err)
		}

		if len(events.Items) == 0 {
			return "No events found.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d events:\n", len(events.Items)))
		for _, ev := range events.Items {
			sb.WriteString(fmt.Sprintf("- [%s] %s/%s: %s (count=%d, reason=%s)\n",
				ev.Type, ev.InvolvedObject.Kind, ev.InvolvedObject.Name,
				ev.Message, ev.Count, ev.Reason))
		}
		return sb.String(), nil
	})
}
