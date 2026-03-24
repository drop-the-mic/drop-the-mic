package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListNodes_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterNodes(r, client)

	out, err := r.Call(context.Background(), "list_nodes", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No nodes found." {
		t.Fatalf("expected 'No nodes found.', got %q", out)
	}
}

func TestListNodes_WithNodes(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			},
		},
	}

	client := fake.NewSimpleClientset(node)
	r := NewRegistry()
	RegisterNodes(r, client)

	out, err := r.Call(context.Background(), "list_nodes", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "worker-1") {
		t.Fatalf("expected output to contain node name, got %q", out)
	}
	if !strings.Contains(out, "Ready") {
		t.Fatalf("expected output to contain Ready, got %q", out)
	}
	if !strings.Contains(out, "Taint") {
		t.Fatalf("expected output to contain taint info, got %q", out)
	}
}

func TestListNodes_NotReady(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "sick-node"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse, Message: "kubelet stopped"},
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}

	client := fake.NewSimpleClientset(node)
	r := NewRegistry()
	RegisterNodes(r, client)

	out, err := r.Call(context.Background(), "list_nodes", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "NotReady") {
		t.Fatalf("expected output to contain NotReady, got %q", out)
	}
}
