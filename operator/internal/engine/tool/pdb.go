package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type listPDBParams struct {
	Namespace string `json:"namespace"`
}

func RegisterPDB(r *Registry, client kubernetes.Interface) {
	r.Register(Definition{
		Name:        "list_pdbs",
		Description: "List PodDisruptionBudgets in a namespace with their status (allowed disruptions, current/desired/expected healthy).",
		Parameters: []Parameter{
			{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
		},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p listPDBParams
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("parsing params: %w", err)
		}

		pdbs, err := client.PolicyV1().PodDisruptionBudgets(p.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("listing PDBs: %w", err)
		}

		if len(pdbs.Items) == 0 {
			return "No PodDisruptionBudgets found.", nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d PDBs:\n", len(pdbs.Items)))
		for _, pdb := range pdbs.Items {
			minAvail := "N/A"
			maxUnavail := "N/A"
			if pdb.Spec.MinAvailable != nil {
				minAvail = pdb.Spec.MinAvailable.String()
			}
			if pdb.Spec.MaxUnavailable != nil {
				maxUnavail = pdb.Spec.MaxUnavailable.String()
			}
			sb.WriteString(fmt.Sprintf("- %s | MinAvailable: %s | MaxUnavailable: %s | AllowedDisruptions: %d | Current: %d | Desired: %d | Expected: %d\n",
				pdb.Name, minAvail, maxUnavail,
				pdb.Status.DisruptionsAllowed,
				pdb.Status.CurrentHealthy,
				pdb.Status.DesiredHealthy,
				pdb.Status.ExpectedPods))
		}
		return sb.String(), nil
	})
}
