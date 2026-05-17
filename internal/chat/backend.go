package chat

type Backend interface {
	Chat(messages []Message) (string, error)
	ChatStream(messages []Message, tools []map[string]any, onEvent func(StreamEvent) error) error
}
