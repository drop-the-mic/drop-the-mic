package llm

import (
	"context"
	"fmt"
	"testing"
)

// MockAdapter is a test double for the LLM Adapter interface.
type MockAdapter struct {
	VerifyFunc func(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error)
}

func (m *MockAdapter) Verify(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(ctx, req, callTool)
	}
	return VerifyResponse{Verdict: VerdictPass, Reasoning: "mock pass"}, nil
}

func (m *MockAdapter) BatchVerify(ctx context.Context, req BatchVerifyRequest, callTool ToolCaller) ([]VerifyResponse, error) {
	results := make([]VerifyResponse, len(req.Checks))
	for i := range req.Checks {
		results[i] = VerifyResponse{Verdict: VerdictPass, Reasoning: "mock batch pass"}
	}
	return results, nil
}

// Verify MockAdapter satisfies the interface
var _ Adapter = (*MockAdapter)(nil)

func TestMockAdapter_DefaultPass(t *testing.T) {
	adapter := &MockAdapter{}
	resp, err := adapter.Verify(context.Background(), VerifyRequest{
		CheckID:     "test-1",
		Description: "test check",
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != VerdictPass {
		t.Fatalf("expected PASS, got %s", resp.Verdict)
	}
}

func TestMockAdapter_CustomVerdict(t *testing.T) {
	adapter := &MockAdapter{
		VerifyFunc: func(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error) {
			return VerifyResponse{
				Verdict:   VerdictFail,
				Reasoning: "pods are crashing",
				ToolCalls: []ToolCallRecord{
					{ToolName: "list_pods", Input: []byte(`{"namespace":"default"}`), Output: "crash detected"},
				},
			}, nil
		},
	}

	resp, err := adapter.Verify(context.Background(), VerifyRequest{
		CheckID:     "check-crash",
		Description: "verify no pods are crashing",
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != VerdictFail {
		t.Fatalf("expected FAIL, got %s", resp.Verdict)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ToolName != "list_pods" {
		t.Fatalf("expected list_pods tool call, got %s", resp.ToolCalls[0].ToolName)
	}
}

func TestMockAdapter_WithToolCaller(t *testing.T) {
	adapter := &MockAdapter{
		VerifyFunc: func(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error) {
			output, err := callTool(ctx, "list_pods", []byte(`{"namespace":"default"}`))
			if err != nil {
				return VerifyResponse{}, err
			}
			return VerifyResponse{
				Verdict:   VerdictPass,
				Reasoning: "verified: " + output,
				ToolCalls: []ToolCallRecord{
					{ToolName: "list_pods", Input: []byte(`{"namespace":"default"}`), Output: output},
				},
			}, nil
		},
	}

	toolCaller := func(ctx context.Context, name string, input []byte) (string, error) {
		if name == "list_pods" {
			return "Found 3 pods: all healthy", nil
		}
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	resp, err := adapter.Verify(context.Background(), VerifyRequest{
		CheckID:     "check-health",
		Description: "all pods healthy",
	}, toolCaller)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != VerdictPass {
		t.Fatalf("expected PASS, got %s", resp.Verdict)
	}
	if resp.Reasoning != "verified: Found 3 pods: all healthy" {
		t.Fatalf("unexpected reasoning: %s", resp.Reasoning)
	}
}

func TestMockAdapter_Error(t *testing.T) {
	adapter := &MockAdapter{
		VerifyFunc: func(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error) {
			return VerifyResponse{}, fmt.Errorf("API rate limited")
		},
	}

	_, err := adapter.Verify(context.Background(), VerifyRequest{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "API rate limited" {
		t.Fatalf("expected 'API rate limited', got %q", err.Error())
	}
}
