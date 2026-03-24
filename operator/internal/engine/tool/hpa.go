package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type listHPAParams struct {
	Namespace string `json:"namespace"`
}

func RegisterHPA(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "list_hpas",
		Description: "List HorizontalPodAutoscalers in a namespace with current/desired replicas and scaling metrics.",
		Parameters: []Parameter{
			{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p listHPAParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		hpas, err := client.AutoscalingV2().HorizontalPodAutoscalers(p.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("listing HPAs: %w", err)
		}

		if len(hpas.Items) == 0 {
			return "No HorizontalPodAutoscalers found.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d HPAs:\n", len(hpas.Items)))
		for _, hpa := range hpas.Items {
			sb.WriteString(fmt.Sprintf("- %s | Target: %s/%s | Min: %d | Max: %d | Current: %d | Desired: %d\n",
				hpa.Name,
				hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name,
				*hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas,
				hpa.Status.CurrentReplicas, hpa.Status.DesiredReplicas))

			for _, metric := range hpa.Status.CurrentMetrics {
				if metric.Resource != nil {
					current := "N/A"
					if metric.Resource.Current.AverageUtilization != nil {
						current = fmt.Sprintf("%d%%", *metric.Resource.Current.AverageUtilization)
					}
					sb.WriteString(fmt.Sprintf("  Metric %s: %s\n", metric.Resource.Name, current))
				}
			}

			for _, cond := range hpa.Status.Conditions {
				if cond.Status == "False" {
					sb.WriteString(fmt.Sprintf("  Condition %s=False: %s\n", cond.Type, cond.Message))
				}
			}
		}
		return sb.String(), nil
	})
}
