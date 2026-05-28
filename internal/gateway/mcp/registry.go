package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
)

type Registry struct {
	mu        sync.RWMutex
	tools     map[string]*model.MCPTool
	piiEngine *privacy.Engine
}

func NewRegistry(pe *privacy.Engine) *Registry {
	r := &Registry{
		tools:     make(map[string]*model.MCPTool),
		piiEngine: pe,
	}
	r.registerDefaultTools()
	return r
}

func (r *Registry) registerDefaultTools() {
	r.tools["scrub_text"] = &model.MCPTool{
		Name:        "scrub_text",
		Description: "Sanitize prompt text by redacting PII values such as emails, phone numbers, and cards",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{"type": "string", "description": "The raw prompt text to sanitize"},
			},
			"required": []string{"text"},
		},
	}
}

func (r *Registry) ListTools() []*model.MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*model.MCPTool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

func (r *Registry) CallTool(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, errors.New("tool not found")
	}
	_ = tool

	switch name {
	case "scrub_text":
		var args struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, err
		}
		redacted := r.piiEngine.Redact(args.Text)
		return map[string]string{"sanitized_text": redacted}, nil
	}

	return nil, errors.New("tool execution not implemented")
}
