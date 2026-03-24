package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPDBs_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	r := NewRegistry()
	RegisterPDB(r, client)

	out, err := r.Call(context.Background(), "list_pdbs", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "No PodDisruptionBudgets found." {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestListPDBs_WithPDB(t *testing.T) {
	minAvail := intstr.FromInt32(1)
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "web-pdb", Namespace: "default"},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvail,
		},
		Status: policyv1.PodDisruptionBudgetStatus{
			DisruptionsAllowed: 1,
			CurrentHealthy:     3,
			DesiredHealthy:     2,
			ExpectedPods:       3,
		},
	}

	client := fake.NewSimpleClientset(pdb)
	r := NewRegistry()
	RegisterPDB(r, client)

	out, err := r.Call(context.Background(), "list_pdbs", json.RawMessage(`{"namespace":"default"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "web-pdb") {
		t.Fatalf("expected PDB name, got %q", out)
	}
	if !strings.Contains(out, "MinAvailable: 1") {
		t.Fatalf("expected MinAvailable, got %q", out)
	}
	if !strings.Contains(out, "AllowedDisruptions: 1") {
		t.Fatalf("expected AllowedDisruptions, got %q", out)
	}
}
