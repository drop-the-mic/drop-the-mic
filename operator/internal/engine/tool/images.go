package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type listImagesParams struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector,omitempty"`
}

func RegisterImages(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "list_images",
		Description: "List container images used by pods in a namespace. Shows image name, tag, pull policy, and which pod uses it.",
		Parameters: []Parameter{
			{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
			{Name: "labelSelector", Type: "string", Description: "Label selector to filter pods", Required: false},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p listImagesParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		pods, err := client.CoreV1().Pods(p.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: p.LabelSelector,
		})
		if err != nil {
			return "", fmt.Errorf("listing pods: %w", err)
		}

		if len(pods.Items) == 0 {
			return "No pods found.", nil
		}

		type imageInfo struct {
			pods       []string
			pullPolicy string
		}
		images := make(map[string]*imageInfo)

		for _, pod := range pods.Items {
			for _, c := range pod.Spec.Containers {
				if info, ok := images[c.Image]; ok {
					info.pods = append(info.pods, pod.Name)
				} else {
					images[c.Image] = &imageInfo{
						pods:       []string{pod.Name},
						pullPolicy: string(c.ImagePullPolicy),
					}
				}
			}
			for _, c := range pod.Spec.InitContainers {
				if info, ok := images[c.Image]; ok {
					info.pods = append(info.pods, pod.Name+"(init)")
				} else {
					images[c.Image] = &imageInfo{
						pods:       []string{pod.Name + "(init)"},
						pullPolicy: string(c.ImagePullPolicy),
					}
				}
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d unique images across %d pods:\n", len(images), len(pods.Items)))
		for image, info := range images {
			sb.WriteString(fmt.Sprintf("- %s | PullPolicy: %s | UsedBy: %s\n",
				image, info.pullPolicy, strings.Join(info.pods, ", ")))
		}
		return sb.String(), nil
	})
}
