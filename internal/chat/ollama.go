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

type ToolCall struct {
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type StreamEvent struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool
}

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type chatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []map[string]any `json:"tools,omitempty"`
}

type chatResponse struct {
	Message struct {
		Role      string     `json:"role"`
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done bool `json:"done"`
}

type Client struct {
	host  string
	Model string
	hc    *http.Client
}

func NewClient(host, model string) *Client {
	return &Client{
		host:  strings.TrimRight(host, "/"),
		Model: model,
		hc:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) Chat(messages []Message) (string, error) {
	body := chatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.hc.Post(c.host+"/api/chat", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama: %s — %s", resp.Status, string(bodyBytes))
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return cr.Message.Content, nil
}

func (c *Client) ChatStream(messages []Message, tools []map[string]any, onEvent func(StreamEvent) error) error {
	body := chatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   true,
		Tools:    tools,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.hc.Post(c.host+"/api/chat", "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama: %s — %s", resp.Status, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var cr chatResponse
		if err := json.Unmarshal(scanner.Bytes(), &cr); err != nil {
			continue
		}
		ev := StreamEvent{
			Content:   cr.Message.Content,
			ToolCalls: cr.Message.ToolCalls,
			Done:      cr.Done,
		}
		if err := onEvent(ev); err != nil {
			return err
		}
		if cr.Done {
			break
		}
	}
	return scanner.Err()
}
