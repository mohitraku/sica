package agent

import (
	"encoding/json"
	"fmt"
	"sort"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Definition.Name] = t
}

func (r *Registry) Definitions() []map[string]any {
	var result []map[string]any
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t := r.tools[name]
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Definition.Name,
				"description": t.Definition.Description,
				"parameters":  t.Definition.Parameters,
			},
		})
	}
	return result
}

func (r *Registry) Dispatch(name string, args json.RawMessage) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s (available: %d tools)", name, len(r.tools))
	}
	return tool.Fn(args)
}
