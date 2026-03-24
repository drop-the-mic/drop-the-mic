package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

func TestListHPAs_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterHPA(r, client)

	out, err := r.Call(context.Background(), "list_hpas", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No HorizontalPodAutoscalers found." {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestListHPAs_WithHPA(t *testing.T) {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "web-hpa", Namespace: "default"},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "web",
			},
			MinReplicas: ptr.To[int32](2),
			MaxReplicas: 10,
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 5,
		},
	}

	client := fake.NewSimpleClientset(hpa)
	r := NewRegistry()
	RegisterHPA(r, client)

	out, err := r.Call(context.Background(), "list_hpas", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "web-hpa") {
		t.Fatalf("expected HPA name in output, got %q", out)
	}
	if !strings.Contains(out, "Deployment/web") {
		t.Fatalf("expected target ref in output, got %q", out)
	}
	if !strings.Contains(out, "Current: 3") {
		t.Fatalf("expected current replicas in output, got %q", out)
	}
}
