// Package tool provides the tool registry and built-in Kubernetes verification tools
// that are exposed to the LLM during cluster checks.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Parameter describes a single parameter accepted by a tool.
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Definition describes a tool that the LLM can invoke via tool_use / function_calling.
type Definition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
}

// Func is the execution signature every tool implementation must satisfy.
type Func func(ctx context.Context, params json.RawMessage) (string, error)

// Registry is a thread-safe store of tool definitions and their implementations.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]registeredTool
}

type registeredTool struct {
	def Definition
	fn  Func
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]registeredTool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(def Definition, fn Func) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[def.Name] = registeredTool{def: def, fn: fn}
}

// Definitions returns all registered tool definitions.
func (r *Registry) Definitions() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]Definition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.def)
	}
	return defs
}

// Call executes a tool by name with the given parameters.
func (r *Registry) Call(ctx context.Context, name string, params json.RawMessage) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return t.fn(ctx, params)
}
