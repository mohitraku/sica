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

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type chatResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
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

func (c *Client) ChatStream(messages []Message, onChunk func(chunk string) error) error {
	body := chatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   true,
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
		if cr.Message.Content != "" {
			if err := onChunk(cr.Message.Content); err != nil {
				return err
			}
		}
		if cr.Done {
			break
		}
	}
	return scanner.Err()
}
