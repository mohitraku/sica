package agent

import "encoding/json"

type ToolFunc func(args json.RawMessage) (string, error)

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Tool struct {
	Definition ToolDefinition
	Fn         ToolFunc
}
