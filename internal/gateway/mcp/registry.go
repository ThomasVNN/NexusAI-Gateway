package mcp

import (
	"sync"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]*model.MCPTool
}

func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]*model.MCPTool),
	}
	r.registerDefaultTools()
	return r
}

func (r *Registry) registerDefaultTools() {
	r.tools["search_web"] = &model.MCPTool{
		Name:        "search_web",
		Description: "Perform privacy-filtered web search via DuckDuckGo",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
			},
			"required": []string{"query"},
		},
	}
}
