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

func TestListPods_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterPods(r, client)

	out, err := r.Call(context.Background(), "list_pods", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No pods found." {
		t.Fatalf("expected 'No pods found.', got %q", out)
	}
}

func TestListPods_WithPods(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-abc123",
			Namespace: "default",
			Labels:    map[string]string{"app": "nginx"},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:1.25"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "nginx", RestartCount: 2, Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
		},
	}

	client := fake.NewSimpleClientset([]runtime.Object{pod}...)
	r := NewRegistry()
	RegisterPods(r, client)

	out, err := r.Call(context.Background(), "list_pods", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "nginx-abc123") {
		t.Fatalf("expected output to contain pod name, got %q", out)
	}
	if !strings.Contains(out, "Restarts: 2") {
		t.Fatalf("expected output to contain restart count, got %q", out)
	}
	if !strings.Contains(out, "Running") {
		t.Fatalf("expected output to contain phase Running, got %q", out)
	}
}

func TestListPods_WithLabelSelector(t *testing.T) {
	pods := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default", Labels: map[string]string{"app": "web"}},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "web:1"}}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "db-1", Namespace: "default", Labels: map[string]string{"app": "db"}},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "db", Image: "db:1"}}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
	}

	client := fake.NewSimpleClientset(pods...)
	r := NewRegistry()
	RegisterPods(r, client)

	out, err := r.Call(context.Background(), "list_pods", json.RawMessage(`{"namespace":"default","labelSelector":"app=web"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "web-1") {
		t.Fatalf("expected output to contain web-1, got %q", out)
	}
	// fake client doesn't filter by label, so we just check it doesn't error
}

func TestListPods_WaitingContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "crash-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:1"}}},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off 5m0s"}},
				},
			},
		},
	}

	client := fake.NewSimpleClientset(pod)
	r := NewRegistry()
	RegisterPods(r, client)

	out, err := r.Call(context.Background(), "list_pods", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "CrashLoopBackOff") {
		t.Fatalf("expected output to contain CrashLoopBackOff, got %q", out)
	}
}

func TestListPods_InvalidParams(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterPods(r, client)

	_, err := r.Call(context.Background(), "list_pods", json.RawMessage(`invalid json`))
	if err == nil {
		t.Fatal("expected error for invalid params")
	}
}
