package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListImages_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterImages(r, client)

	out, err := r.Call(context.Background(), "list_images", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No pods found." {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestListImages_WithPods(t *testing.T) {
	pods := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "web", Image: "nginx:1.25", ImagePullPolicy: corev1.PullAlways},
				},
				InitContainers: []corev1.Container{
					{Name: "init", Image: "busybox:latest", ImagePullPolicy: corev1.PullIfNotPresent},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-2", Namespace: "default"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "web", Image: "nginx:1.25", ImagePullPolicy: corev1.PullAlways},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pods...)
	r := NewRegistry()
	RegisterImages(r, client)

	out, err := r.Call(context.Background(), "list_images", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "nginx:1.25") {
		t.Fatalf("expected nginx image, got %q", out)
	}
	if !strings.Contains(out, "busybox:latest") {
		t.Fatalf("expected busybox image, got %q", out)
	}
	if !strings.Contains(out, "2 unique images") {
		t.Fatalf("expected 2 unique images count, got %q", out)
	}
}
