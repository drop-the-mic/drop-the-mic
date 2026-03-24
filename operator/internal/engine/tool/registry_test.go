package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRegistry_RegisterAndCall(t *testing.T) {
	r := NewRegistry()

	r.Register(Definition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  []Parameter{{Name: "msg", Type: "string", Description: "message", Required: true}},
	}, func(ctx context.Context, params json.RawMessage) (string, error) {
		var p struct{ Msg string `json:"msg"` }
		if err := json.Unmarshal(params, &p); err != nil {
			return "", err
		}
		return "echo: " + p.Msg, nil
	})

	// Definitions should return 1 tool
	defs := r.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "test_tool" {
		t.Fatalf("expected tool name test_tool, got %s", defs[0].Name)
	}

	// Call should succeed
	output, err := r.Call(context.Background(), "test_tool", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "echo: hello" {
		t.Fatalf("expected 'echo: hello', got %q", output)
	}
}

func TestRegistry_CallNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Call(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
}

func TestRegistry_MultipleTools(t *testing.T) {
	r := NewRegistry()

	for _, name := range []string{"tool_a", "tool_b", "tool_c"} {
		n := name
		r.Register(Definition{Name: n, Description: n}, func(ctx context.Context, params json.RawMessage) (string, error) {
			return n, nil
		})
	}

	defs := r.Definitions()
	if len(defs) != 3 {
		t.Fatalf("expected 3 definitions, got %d", len(defs))
	}

	for _, name := range []string{"tool_a", "tool_b", "tool_c"} {
		out, err := r.Call(context.Background(), name, nil)
		if err != nil {
			t.Fatalf("unexpected error calling %s: %v", name, err)
		}
		if out != name {
			t.Fatalf("expected %q, got %q", name, out)
		}
	}
}
