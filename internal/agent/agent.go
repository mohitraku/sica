package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mojitrk/sica/internal/chat"
)

const maxToolIterations = 10

type Callbacks struct {
	OnChunk func(string)
	OnTool  func(string, string) // (toolName, status: "start"|"end")
}

type Agent struct {
	backend  chat.Backend
	store    *chat.Store
	registry *Registry
	convID   int64
	model    string
}

func New(backend chat.Backend, store *chat.Store, registry *Registry, convID int64, model string) *Agent {
	return &Agent{
		backend:  backend,
		store:    store,
		registry: registry,
		convID:   convID,
		model:    model,
	}
}

func SystemPrompt() chat.Message {
	return chat.Message{
		Role: "system",
		Content: fmt.Sprintf(`You are an AI assistant for sica, a personal productivity application. You help users manage their habits and tasks.

You have access to tools that let you query and modify the user's data. When you need information or need to make changes, use the appropriate tool. Always explain what you're doing in a friendly, concise way.

Today's date is %s.

When presenting data:
- For habits, show current streaks and today's progress
- For tasks, indicate priority (high/med/low) and status (todo/done)
- Include IDs in tool results so you can reference them in follow-up calls
- Format dates as YYYY-MM-DD

If a tool returns an error, explain it to the user. If you don't have enough information, ask the user for clarification.`, time.Now().Format("2006-01-02")),
	}
}

func (a *Agent) buildMessages() ([]chat.Message, error) {
	msgs, err := a.store.Messages(a.convID)
	if err != nil {
		return nil, err
	}

	result := make([]chat.Message, 0, len(msgs)+1)
	result = append(result, SystemPrompt())

	for _, m := range msgs {
		cm := chat.Message{
			Role:    m.Role,
			Content: m.Content,
		}
		if m.ToolCalls != "" && m.Role == "assistant" {
			var tcs []chat.ToolCall
			if err := json.Unmarshal([]byte(m.ToolCalls), &tcs); err == nil {
				cm.ToolCalls = tcs
			}
		}
		result = append(result, cm)
	}
	return result, nil
}

func (a *Agent) RunStream(cb Callbacks) (string, error) {
	for iter := 0; iter < maxToolIterations; iter++ {
		msgs, err := a.buildMessages()
		if err != nil {
			return "", fmt.Errorf("build messages: %w", err)
		}

		tools := a.registry.Definitions()

		var fullContent strings.Builder
		var toolCalls []chat.ToolCall

		err = a.backend.ChatStream(msgs, tools, func(event chat.StreamEvent) error {
			fullContent.WriteString(event.Content)
			if cb.OnChunk != nil && event.Content != "" {
				cb.OnChunk(event.Content)
			}
			for _, tc := range event.ToolCalls {
				toolCalls = append(toolCalls, tc)
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("chat stream: %w", err)
		}

		if fullContent.Len() == 0 && len(toolCalls) == 0 {
			fullContent.WriteString("(no response)")
		}

		var toolCallsJSON string
		if len(toolCalls) > 0 {
			b, _ := json.Marshal(toolCalls)
			toolCallsJSON = string(b)
		}
		if _, err := a.store.AddMessage(a.convID, "assistant", fullContent.String(), toolCallsJSON); err != nil {
			return "", fmt.Errorf("save assistant msg: %w", err)
		}

		if len(toolCalls) == 0 {
			return fullContent.String(), nil
		}

		for _, tc := range toolCalls {
			if cb.OnTool != nil {
				cb.OnTool(tc.Function.Name, "start")
			}

			var args json.RawMessage
			if tc.Function.Arguments != "" {
				args = json.RawMessage(tc.Function.Arguments)
			}

			result, err := a.registry.Dispatch(tc.Function.Name, args)
			if err != nil {
				result = fmt.Sprintf("Error executing %s: %v", tc.Function.Name, err)
			}

			if _, err := a.store.AddMessage(a.convID, "tool", result); err != nil {
				return "", fmt.Errorf("save tool result: %w", err)
			}

			if cb.OnTool != nil {
				cb.OnTool(tc.Function.Name, "end")
			}
		}
	}

	return "I've taken too many steps. Please try again or rephrase your request.", nil
}
