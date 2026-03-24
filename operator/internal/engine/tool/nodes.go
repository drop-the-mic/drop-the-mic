package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type listNodesParams struct {
	LabelSelector string `json:"labelSelector,omitempty"`
}

func RegisterNodes(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "list_nodes",
		Description: "List cluster nodes with their status, conditions, capacity, and allocatable resources.",
		Parameters: []Parameter{
			{Name: "labelSelector", Type: "string", Description: "Label selector to filter nodes", Required: false},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p listNodesParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			LabelSelector: p.LabelSelector,
		})
		if err != nil {
			return "", fmt.Errorf("listing nodes: %w", err)
		}

		if len(nodes.Items) == 0 {
			return "No nodes found.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d nodes:\n", len(nodes.Items)))
		for _, node := range nodes.Items {
			ready := "Unknown"
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady {
					if cond.Status == corev1.ConditionTrue {
						ready = "Ready"
					} else {
						ready = fmt.Sprintf("NotReady (%s)", cond.Message)
					}
				}
			}
			cpu := node.Status.Allocatable.Cpu().String()
			mem := node.Status.Allocatable.Memory().String()
			sb.WriteString(fmt.Sprintf("- %s | %s | CPU: %s | Memory: %s\n",
				node.Name, ready, cpu, mem))

			for _, cond := range node.Status.Conditions {
				if cond.Type != corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					sb.WriteString(fmt.Sprintf("  Condition %s=True: %s\n", cond.Type, cond.Message))
				}
			}

			for _, taint := range node.Spec.Taints {
				sb.WriteString(fmt.Sprintf("  Taint: %s=%s:%s\n", taint.Key, taint.Value, taint.Effect))
			}
		}
		return sb.String(), nil
	})
}
