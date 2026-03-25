package llm

import (
	"encoding/json"
	"testing"

	"github.com/drop-the-mic/operator/internal/engine/tool"
)

func TestConvertTools(t *testing.T) {
	defs := []tool.Definition{
		{
			Name:        "list_pods",
			Description: "List pods in a namespace",
			Parameters: []tool.Parameter{
				{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: true},
				{Name: "labelSelector", Type: "string", Description: "Label selector", Required: false},
			},
		},
	}

	tools := convertTools(defs)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "list_pods" {
		t.Fatalf("expected list_pods, got %s", tools[0].Name)
	}

	// Verify schema structure
	var schema map[string]interface{}
	if err := json.Unmarshal(tools[0].InputSchema, &schema); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("expected type=object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be a map")
	}
	if _, ok := props["namespace"]; !ok {
		t.Fatal("expected namespace property")
	}

	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("expected required to be an array")
	}
	if len(required) != 1 || required[0] != "namespace" {
		t.Fatalf("expected required=[namespace], got %v", required)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := buildSystemPrompt([]string{"kube-system", "default"})
	if len(prompt) == 0 {
		t.Fatal("expected non-empty system prompt")
	}
	// Should contain the namespaces
	if !contains(prompt, "kube-system") {
		t.Fatal("expected kube-system in system prompt")
	}
	if !contains(prompt, "default") {
		t.Fatal("expected default in system prompt")
	}
}

func TestBuildSystemPrompt_NilNamespaces(t *testing.T) {
	prompt := buildSystemPrompt(nil)
	if !contains(prompt, "all namespaces") {
		t.Fatal("expected 'all namespaces' when namespaces is nil")
	}
}

func TestParseVerdict_Pass(t *testing.T) {
	resp := &claudeResponse{
		Content: []json.RawMessage{
			json.RawMessage(`{"type":"text","text":"Everything looks good.\n\nVERDICT: PASS"}`),
		},
	}

	result, err := parseVerdict(resp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict != VerdictPass {
		t.Fatalf("expected PASS, got %s", result.Verdict)
	}
}

func TestParseVerdict_Fail(t *testing.T) {
	resp := &claudeResponse{
		Content: []json.RawMessage{
			json.RawMessage(`{"type":"text","text":"Pods are crashing.\n\nVERDICT: FAIL"}`),
		},
	}

	result, err := parseVerdict(resp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict != VerdictFail {
		t.Fatalf("expected FAIL, got %s", result.Verdict)
	}
}

func TestParseVerdict_Warn(t *testing.T) {
	resp := &claudeResponse{
		Content: []json.RawMessage{
			json.RawMessage(`{"type":"text","text":"High memory usage detected.\n\nVERDICT: WARN"}`),
		},
	}

	result, err := parseVerdict(resp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict != VerdictWarn {
		t.Fatalf("expected WARN, got %s", result.Verdict)
	}
}

func TestParseVerdict_WithToolCalls(t *testing.T) {
	toolCalls := []ToolCallRecord{
		{ToolName: "list_pods", Input: []byte(`{}`), Output: "3 pods"},
		{ToolName: "list_nodes", Input: []byte(`{}`), Output: "2 nodes"},
	}

	resp := &claudeResponse{
		Content: []json.RawMessage{
			json.RawMessage(`{"type":"text","text":"VERDICT: PASS"}`),
		},
	}

	result, err := parseVerdict(resp, toolCalls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(result.ToolCalls))
	}
}

func TestParseVerdict_DefaultFail(t *testing.T) {
	// If no verdict keyword found, defaults to FAIL
	resp := &claudeResponse{
		Content: []json.RawMessage{
			json.RawMessage(`{"type":"text","text":"I couldn't determine the status"}`),
		},
	}

	result, err := parseVerdict(resp, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict != VerdictFail {
		t.Fatalf("expected default FAIL, got %s", result.Verdict)
	}
}

func TestBuildJSONSchema_NoRequired(t *testing.T) {
	params := []tool.Parameter{
		{Name: "optional", Type: "string", Description: "opt", Required: false},
	}

	schema := buildJSONSchema(params)
	var s map[string]interface{}
	if err := json.Unmarshal(schema, &s); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := s["required"]; ok {
		t.Fatal("expected no required field when all params are optional")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
