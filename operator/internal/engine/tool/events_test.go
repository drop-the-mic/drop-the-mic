package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListEvents_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterEvents(r, client)

	out, err := r.Call(context.Background(), "list_events", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No events found." {
		t.Fatalf("expected 'No events found.', got %q", out)
	}
}

func TestListEvents_WithEvents(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-event-1", Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Kind: "Pod",
			Name: "nginx-abc",
		},
		Type:    "Warning",
		Reason:  "FailedScheduling",
		Message: "0/3 nodes are available",
		Count:   5,
	}

	client := fake.NewSimpleClientset(event)
	r := NewRegistry()
	RegisterEvents(r, client)

	out, err := r.Call(context.Background(), "list_events", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "FailedScheduling") {
		t.Fatalf("expected output to contain reason, got %q", out)
	}
	if !strings.Contains(out, "Warning") {
		t.Fatalf("expected output to contain event type, got %q", out)
	}
	if !strings.Contains(out, "count=5") {
		t.Fatalf("expected output to contain count, got %q", out)
	}
}
