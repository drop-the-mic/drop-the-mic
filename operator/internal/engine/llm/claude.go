package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/drop-the-mic/operator/internal/engine/tool"
)

const (
	claudeAPIURL       = "https://api.anthropic.com/v1/messages"
	claudeDefaultModel = "claude-sonnet-4-20250514"
	anthropicVersion   = "2023-06-01"
)

// ClaudeAdapter implements the Adapter interface for Anthropic Claude.
type ClaudeAdapter struct {
	apiKey string
	model  string
	client *http.Client
}

// NewClaudeAdapter creates a new Claude adapter.
func NewClaudeAdapter(apiKey, model string) *ClaudeAdapter {
	if model == "" {
		model = claudeDefaultModel
	}
	return &ClaudeAdapter{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Claude API types
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
	Tools     []claudeTool    `json:"tools,omitempty"`
}

type claudeMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type claudeTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type claudeTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeToolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type claudeToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

type claudeResponse struct {
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
}

type contentBlock struct {
	Type string `json:"type"`
}

func (a *ClaudeAdapter) Verify(ctx context.Context, req VerifyRequest, callTool ToolCaller) (VerifyResponse, error) {
	tools := convertTools(req.Tools)
	systemPrompt := buildSystemPrompt(req.Namespace)

	messages := []claudeMessage{
		{
			Role: "user",
			Content: []interface{}{
				claudeTextBlock{
					Type: "text",
					Text: fmt.Sprintf("Please verify the following check and determine if it PASSES, has WARNINGS, or FAILS.\n\nCheck ID: %s\nCheck Description: %s\n\nUse the available tools to gather evidence from the Kubernetes cluster, then provide your verdict as exactly one of: PASS, WARN, or FAIL, along with your reasoning.",
						req.CheckID, req.Description),
				},
			},
		},
	}

	var allToolCalls []ToolCallRecord

	for iterations := 0; iterations < 10; iterations++ {
		resp, err := a.callAPI(ctx, claudeRequest{
			Model:     a.model,
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     tools,
		})
		if err != nil {
			return VerifyResponse{}, fmt.Errorf("claude API call: %w", err)
		}

		if resp.StopReason == "end_turn" || resp.StopReason == "stop" {
			return parseVerdict(resp, allToolCalls)
		}

		if resp.StopReason == "tool_use" {
			assistantContent, toolResults, toolCalls, err := a.processToolUse(ctx, resp, callTool)
			if err != nil {
				return VerifyResponse{}, fmt.Errorf("processing tool use: %w", err)
			}
			allToolCalls = append(allToolCalls, toolCalls...)

			messages = append(messages,
				claudeMessage{Role: "assistant", Content: assistantContent},
				claudeMessage{Role: "user", Content: toolResults},
			)
			continue
		}

		return parseVerdict(resp, allToolCalls)
	}

	return VerifyResponse{}, fmt.Errorf("exceeded maximum tool call iterations")
}

func (a *ClaudeAdapter) callAPI(ctx context.Context, req claudeRequest) (*claudeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
	}

	var resp claudeResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}
	return &resp, nil
}

func (a *ClaudeAdapter) processToolUse(ctx context.Context, resp *claudeResponse, callTool ToolCaller) ([]interface{}, []interface{}, []ToolCallRecord, error) {
	var assistantContent []interface{}
	var toolResults []interface{}
	var toolCalls []ToolCallRecord

	for _, raw := range resp.Content {
		var block contentBlock
		if err := json.Unmarshal(raw, &block); err != nil {
			continue
		}

		switch block.Type {
		case "text":
			var tb claudeTextBlock
			if err := json.Unmarshal(raw, &tb); err == nil {
				assistantContent = append(assistantContent, tb)
			}
		case "tool_use":
			var tu claudeToolUseBlock
			if err := json.Unmarshal(raw, &tu); err != nil {
				continue
			}
			assistantContent = append(assistantContent, tu)

			output, err := callTool(ctx, tu.Name, tu.Input)
			resultContent := output
			if err != nil {
				resultContent = fmt.Sprintf("Error: %v", err)
			}

			toolResults = append(toolResults, claudeToolResultBlock{
				Type:      "tool_result",
				ToolUseID: tu.ID,
				Content:   resultContent,
			})

			toolCalls = append(toolCalls, ToolCallRecord{
				ToolName: tu.Name,
				Input:    tu.Input,
				Output:   resultContent,
			})
		}
	}

	return assistantContent, toolResults, toolCalls, nil
}

func convertTools(defs []tool.Definition) []claudeTool {
	tools := make([]claudeTool, 0, len(defs))
	for _, d := range defs {
		schema := buildJSONSchema(d.Parameters)
		tools = append(tools, claudeTool{
			Name:        d.Name,
			Description: d.Description,
			InputSchema: schema,
		})
	}
	return tools
}

func buildJSONSchema(params []tool.Parameter) json.RawMessage {
	properties := make(map[string]map[string]string)
	var required []string

	for _, p := range params {
		properties[p.Name] = map[string]string{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	data, _ := json.Marshal(schema)
	return data
}

func buildSystemPrompt(namespace string) string {
	return fmt.Sprintf(`You are a Kubernetes cluster verification agent. Your job is to verify checks against a live Kubernetes cluster using the provided tools.

Rules:
1. Use the available tools to gather evidence before making a verdict.
2. Always base your verdict on actual data from the cluster, not assumptions.
3. Your final response MUST contain exactly one verdict line in this format: VERDICT: PASS, VERDICT: WARN, or VERDICT: FAIL
4. Provide clear reasoning explaining why you reached your verdict.
5. You are operating in a read-only capacity. You cannot modify the cluster.
6. Default namespace context: %s

Be thorough but efficient - use the minimum number of tool calls needed to verify the check.`, namespace)
}

func parseVerdict(resp *claudeResponse, toolCalls []ToolCallRecord) (VerifyResponse, error) {
	var fullText strings.Builder
	for _, raw := range resp.Content {
		var block contentBlock
		if err := json.Unmarshal(raw, &block); err != nil {
			continue
		}
		if block.Type == "text" {
			var tb claudeTextBlock
			if err := json.Unmarshal(raw, &tb); err == nil {
				fullText.WriteString(tb.Text)
			}
		}
	}

	text := fullText.String()
	verdict := VerdictFail

	upperText := strings.ToUpper(text)
	if strings.Contains(upperText, "VERDICT: PASS") {
		verdict = VerdictPass
	} else if strings.Contains(upperText, "VERDICT: WARN") {
		verdict = VerdictWarn
	} else if strings.Contains(upperText, "VERDICT: FAIL") {
		verdict = VerdictFail
	}

	return VerifyResponse{
		Verdict:   verdict,
		Reasoning: text,
		ToolCalls: toolCalls,
	}, nil
}
