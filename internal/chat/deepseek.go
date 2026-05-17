package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type DeepSeekClient struct {
	apiKey string
	model  string
	hc     *http.Client
}

func NewDeepSeekClient(apiKey, model string) *DeepSeekClient {
	if apiKey == "" {
		return nil
	}
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekClient{
		apiKey: apiKey,
		model:  model,
		hc:     &http.Client{Timeout: 120 * time.Second},
	}
}

type dsRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []map[string]any `json:"tools,omitempty"`
}

type dsChoice struct {
	Message struct {
		Role      string     `json:"role"`
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Delta struct {
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type dsResponse struct {
	Choices []dsChoice `json:"choices"`
}

func (c *DeepSeekClient) Chat(messages []Message) (string, error) {
	body := dsRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("deepseek: %s — %s", resp.Status, string(bodyBytes))
	}

	var ds dsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ds); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(ds.Choices) == 0 {
		return "", fmt.Errorf("deepseek: no choices in response")
	}
	return ds.Choices[0].Message.Content, nil
}

func (c *DeepSeekClient) ChatStream(messages []Message, tools []map[string]any, onEvent func(StreamEvent) error) error {
	body := dsRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
		Tools:    tools,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deepseek: %s — %s", resp.Status, string(bodyBytes))
	}

	var accumulated []ToolCall
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")

		var ds dsResponse
		if err := json.Unmarshal([]byte(payload), &ds); err != nil {
			continue
		}
		if len(ds.Choices) == 0 {
			continue
		}
		ch := ds.Choices[0]

		content := ch.Delta.Content
		for _, tc := range ch.Delta.ToolCalls {
			accumulated = append(accumulated, tc)
		}

		done := ch.FinishReason == "stop" || ch.FinishReason == "tool_calls"
		ev := StreamEvent{Content: content, Done: done}
		if done && len(accumulated) > 0 {
			ev.ToolCalls = accumulated
		}
		if err := onEvent(ev); err != nil {
			return err
		}
		if done {
			break
		}
	}

	return scanner.Err()
}
