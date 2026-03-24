// Package llm defines the common LLM adapter interface and types used by
// the verification engine to interact with different LLM providers.
package llm

import (
	"context"

	"github.com/drop-the-mic/operator/internal/engine/tool"
)

// Verdict represents the result of a verification check.
type Verdict string

const (
	VerdictPass Verdict = "PASS"
	VerdictWarn Verdict = "WARN"
	VerdictFail Verdict = "FAIL"
)

// VerifyRequest contains everything needed for an LLM to verify a check.
type VerifyRequest struct {
	CheckID     string
	Description string
	Tools       []tool.Definition
	Namespace   string
}

// ToolCallRecord stores a single tool invocation and its result.
type ToolCallRecord struct {
	ToolName string
	Input    []byte
	Output   string
}

// VerifyResponse contains the LLM's verdict after verification.
type VerifyResponse struct {
	Verdict   Verdict
	Reasoning string
	ToolCalls []ToolCallRecord
}

// Adapter is the interface all LLM implementations must satisfy.
type Adapter interface {
	Verify(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error)
}

// ToolCaller is a function that executes a tool and returns its output.
type ToolCaller func(ctx context.Context, name string, input []byte) (string, error)
