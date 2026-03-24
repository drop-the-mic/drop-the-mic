package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type getLogsParams struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container,omitempty"`
	TailLines int64  `json:"tailLines,omitempty"`
	Previous  bool   `json:"previous,omitempty"`
}

func RegisterLogs(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "get_logs",
		Description: "Get logs from a specific pod container. Returns the last N lines of logs.",
		Parameters: []Parameter{
			{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
			{Name: "pod", Type: "string", Description: "Pod name", Required: true},
			{Name: "container", Type: "string", Description: "Container name (required if pod has multiple containers)", Required: false},
			{Name: "tailLines", Type: "integer", Description: "Number of lines from the end to return (default 100)", Required: false},
			{Name: "previous", Type: "boolean", Description: "Return logs from previous terminated container", Required: false},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p getLogsParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		tailLines := int64(100)
		if p.TailLines > 0 {
			tailLines = p.TailLines
		}

		opts := &corev1.PodLogOptions{
			TailLines: &tailLines,
			Previous:  p.Previous,
		}
		if p.Container != "" {
			opts.Container = p.Container
		}

		req := client.CoreV1().Pods(p.Namespace).GetLogs(p.Pod, opts)
		stream, err := req.Stream(ctx)
		if err != nil {
			return "", fmt.Errorf("getting logs: %w", err)
		}
		defer stream.Close()

		logs, err := io.ReadAll(stream)
		if err != nil {
			return "", fmt.Errorf("reading logs: %w", err)
		}

		if len(logs) == 0 {
			return "No logs available.", nil
		}

		return string(logs), nil
	})
}
