package engine

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/internal/engine/llm"
	"github.com/drop-the-mic/operator/internal/engine/tool"
)

type mockAdapter struct {
	responses map[string]llm.VerifyResponse
	err       error
}

func (m *mockAdapter) Verify(ctx context.Context, req llm.VerifyRequest, callTool llm.ToolCaller) (llm.VerifyResponse, error) {
	if m.err != nil {
		return llm.VerifyResponse{}, m.err
	}
	if resp, ok := m.responses[req.CheckID]; ok {
		return resp, nil
	}
	return llm.VerifyResponse{Verdict: llm.VerdictPass, Reasoning: "default pass"}, nil
}

func TestEngine_RunChecks_AllPass(t *testing.T) {
	adapter := &mockAdapter{
		responses: map[string]llm.VerifyResponse{
			"check-1": {Verdict: llm.VerdictPass, Reasoning: "all good"},
			"check-2": {Verdict: llm.VerdictPass, Reasoning: "looking healthy"},
		},
	}

	registry := tool.NewRegistry()
	eng := New(adapter, registry, logr.Discard())

	policy := &dtmv1alpha1.ChecklistPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "test-policy", Namespace: "default"},
		Spec: dtmv1alpha1.ChecklistPolicySpec{
			TargetNamespaces: []string{"default"},
		},
	}

	checks := []dtmv1alpha1.CheckItem{
		{ID: "check-1", Description: "all pods running", Severity: "critical"},
		{ID: "check-2", Description: "nodes healthy", Severity: "warning"},
	}

	result, err := eng.RunChecks(context.Background(), policy, checks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Checks) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Checks))
	}

	if result.Summary == nil {
		t.Fatal("expected summary")
	}
	if result.Summary.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Summary.Total)
	}
	if result.Summary.Pass != 2 {
		t.Fatalf("expected pass=2, got %d", result.Summary.Pass)
	}
	if result.Summary.Fail != 0 {
		t.Fatalf("expected fail=0, got %d", result.Summary.Fail)
	}

	if result.CompletedAt == nil {
		t.Fatal("expected completedAt to be set")
	}
}

func TestEngine_RunChecks_MixedVerdicts(t *testing.T) {
	adapter := &mockAdapter{
		responses: map[string]llm.VerifyResponse{
			"c1": {Verdict: llm.VerdictPass, Reasoning: "ok"},
			"c2": {Verdict: llm.VerdictWarn, Reasoning: "high mem"},
			"c3": {Verdict: llm.VerdictFail, Reasoning: "pods crashing"},
		},
	}

	registry := tool.NewRegistry()
	eng := New(adapter, registry, logr.Discard())

	policy := &dtmv1alpha1.ChecklistPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "mixed-policy", Namespace: "default"},
	}

	checks := []dtmv1alpha1.CheckItem{
		{ID: "c1", Description: "check 1", Severity: "info"},
		{ID: "c2", Description: "check 2", Severity: "warning"},
		{ID: "c3", Description: "check 3", Severity: "critical"},
	}

	result, err := eng.RunChecks(context.Background(), policy, checks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.Pass != 1 {
		t.Fatalf("expected pass=1, got %d", result.Summary.Pass)
	}
	if result.Summary.Warn != 1 {
		t.Fatalf("expected warn=1, got %d", result.Summary.Warn)
	}
	if result.Summary.Fail != 1 {
		t.Fatalf("expected fail=1, got %d", result.Summary.Fail)
	}

	// Verify individual results
	for _, check := range result.Checks {
		switch check.ID {
		case "c1":
			if check.Verdict != dtmv1alpha1.VerdictPass {
				t.Fatalf("expected c1=PASS, got %s", check.Verdict)
			}
		case "c2":
			if check.Verdict != dtmv1alpha1.VerdictWarn {
				t.Fatalf("expected c2=WARN, got %s", check.Verdict)
			}
		case "c3":
			if check.Verdict != dtmv1alpha1.VerdictFail {
				t.Fatalf("expected c3=FAIL, got %s", check.Verdict)
			}
			if check.Severity != "critical" {
				t.Fatalf("expected severity=critical, got %s", check.Severity)
			}
		}
	}
}

func TestEngine_RunChecks_AdapterError(t *testing.T) {
	adapter := &mockAdapter{
		err: context.DeadlineExceeded,
	}

	registry := tool.NewRegistry()
	eng := New(adapter, registry, logr.Discard())

	policy := &dtmv1alpha1.ChecklistPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "error-policy", Namespace: "default"},
	}

	checks := []dtmv1alpha1.CheckItem{
		{ID: "c1", Description: "check that will error", Severity: "critical"},
	}

	result, err := eng.RunChecks(context.Background(), policy, checks)
	if err != nil {
		t.Fatalf("RunChecks itself should not error, got: %v", err)
	}

	// The individual check should be marked as FAIL with error reasoning
	if len(result.Checks) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Checks))
	}
	if result.Checks[0].Verdict != dtmv1alpha1.VerdictFail {
		t.Fatalf("expected FAIL on error, got %s", result.Checks[0].Verdict)
	}
	if result.Summary.Fail != 1 {
		t.Fatalf("expected fail=1, got %d", result.Summary.Fail)
	}
}

func TestEngine_RunChecks_WithToolCalls(t *testing.T) {
	adapter := &mockAdapter{
		responses: map[string]llm.VerifyResponse{
			"c1": {
				Verdict:   llm.VerdictPass,
				Reasoning: "verified via tools",
				ToolCalls: []llm.ToolCallRecord{
					{ToolName: "list_pods", Input: []byte(`{"namespace":"default"}`), Output: "3 pods running"},
					{ToolName: "list_nodes", Input: []byte(`{}`), Output: "2 nodes ready"},
				},
			},
		},
	}

	registry := tool.NewRegistry()
	eng := New(adapter, registry, logr.Discard())

	policy := &dtmv1alpha1.ChecklistPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "evidence-policy", Namespace: "default"},
	}

	checks := []dtmv1alpha1.CheckItem{
		{ID: "c1", Description: "check with evidence", Severity: "info"},
	}

	result, err := eng.RunChecks(context.Background(), policy, checks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	check := result.Checks[0]
	if check.Evidence == nil {
		t.Fatal("expected evidence")
	}
	if len(check.Evidence.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls in evidence, got %d", len(check.Evidence.ToolCalls))
	}
	if check.Evidence.ToolCalls[0].ToolName != "list_pods" {
		t.Fatalf("expected first tool call to be list_pods, got %s", check.Evidence.ToolCalls[0].ToolName)
	}
}

func TestEngine_RunChecks_Empty(t *testing.T) {
	adapter := &mockAdapter{}
	registry := tool.NewRegistry()
	eng := New(adapter, registry, logr.Discard())

	policy := &dtmv1alpha1.ChecklistPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-policy"},
	}

	result, err := eng.RunChecks(context.Background(), policy, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Checks) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result.Checks))
	}
	if result.Summary.Total != 0 {
		t.Fatalf("expected total=0, got %d", result.Summary.Total)
	}
}

func TestGenerateResultName(t *testing.T) {
	name := GenerateResultName("my-policy", dtmv1alpha1.ScanTypeFull)
	if name == "" {
		t.Fatal("expected non-empty name")
	}
	if len(name) < 20 {
		t.Fatalf("name seems too short: %s", name)
	}
}
