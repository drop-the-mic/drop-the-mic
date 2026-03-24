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

type listPodsParams struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty"`
}

func RegisterPods(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "list_pods",
		Description: "List pods in a namespace with optional label and field selectors. Returns pod name, status, restarts, and age.",
		Parameters: []Parameter{
			{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
			{Name: "labelSelector", Type: "string", Description: "Label selector (e.g. app=nginx)", Required: false},
			{Name: "fieldSelector", Type: "string", Description: "Field selector (e.g. status.phase=Running)", Required: false},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p listPodsParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		pods, err := client.CoreV1().Pods(p.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: p.LabelSelector,
			FieldSelector: p.FieldSelector,
		})
		if err != nil {
			return "", fmt.Errorf("listing pods: %w", err)
		}

		if len(pods.Items) == 0 {
			return "No pods found.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d pods:\n", len(pods.Items)))
		for _, pod := range pods.Items {
			restarts := int32(0)
			for _, cs := range pod.Status.ContainerStatuses {
				restarts += cs.RestartCount
			}
			sb.WriteString(fmt.Sprintf("- %s | Phase: %s | Restarts: %d | Node: %s\n",
				pod.Name, pod.Status.Phase, restarts, pod.Spec.NodeName))

			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					sb.WriteString(fmt.Sprintf("  Container %s: Waiting (%s: %s)\n",
						cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message))
				}
				if cs.State.Terminated != nil {
					sb.WriteString(fmt.Sprintf("  Container %s: Terminated (%s, exit=%d)\n",
						cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode))
				}
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Status == corev1.ConditionFalse {
					sb.WriteString(fmt.Sprintf("  Condition %s=False: %s\n", cond.Type, cond.Message))
				}
			}
		}
		return sb.String(), nil
	})
}
