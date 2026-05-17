# Handoff — AI Tool Calling Fix

## The problem

The AI chat agent in the sica TUI cannot call tools (create/delete habits, create/complete tasks, etc.). When asked to "delete the teeth habit," the model just talks about doing it but never executes the tool.

## Two root causes identified and fixed

### 1. Incompatible model (FIXED)

`qwen2.5:7b` does **not** support Ollama's native tool-calling API. It generates text about tools but never emits structured `tool_calls` in the response.

**Fix applied**: Default model changed to `llama3.1:8b` in [config/config.go:53](config/config.go#L53).

### 2. Context window overflow (FIXED)

The Ollama server log showed: `msg="truncating input prompt" limit=4096 prompt=12225`

The 13 tool definitions with full JSON schemas + system prompt + conversation = ~12K tokens, but the model default context is only 4K. So the tools were being truncated before they reached the model.

**Fix applied**: Added `Options: map[string]any{"num_ctx": 16384}` to both `Chat()` and `ChatStream()` in [internal/chat/ollama.go](internal/chat/ollama.go).

## Changes made

| File | Change |
|------|--------|
| [internal/chat/ollama.go](internal/chat/ollama.go) | Added `Options` field to `chatRequest` struct; set `num_ctx: 16384` in `Chat()` and `ChatStream()` |
| [config/config.go](config/config.go) | Default model: `qwen2.5:7b` → `llama3.1:8b` |
| [internal/agent/agent.go](internal/agent/agent.go) | Updated empty-response diagnostic message |

All changes build and pass `go vet ./...` and `go test -count=1 ./...`.

## How the agent loop works

[internal/agent/agent.go](internal/agent/agent.go) `RunStream()`:
1. Build messages from DB conversation history + system prompt
2. Call `ChatStream()` with messages + tool definitions
3. If `tool_calls` come back in the stream → dispatch each tool → save result as "tool" role message → loop back to step 1
4. If no tool_calls → return final text response
5. Max 10 iterations

## Architecture

```
TUI (bubbletea) → Agent.RunStream → Backend.ChatStream → Ollama /api/chat
                        ↓
                 Registry.Dispatch → Tool functions (delete_habit, etc.) → SQLite stores
```

## Next step (NEEDS RUNTIME TEST)

The user must:
1. **Pull the model**: `ollama pull llama3.1:8b`
2. **Restart the app**: `go run ./cmd/sica/`
3. **Test**: Type "delete the teeth habit" in chat and see if the tool actually fires
4. **Watch logs**: `tail -f /home/mojitrk/.sica/sica.log` for errors

If it still doesn't work, check the Ollama server logs for truncation warnings:
```bash
journalctl -u ollama -f
```

## Key files

- [internal/chat/ollama.go](internal/chat/ollama.go) — Ollama HTTP client (tool calling)
- [internal/agent/agent.go](internal/agent/agent.go) — Agent loop, tool dispatch orchestration
- [internal/agent/tools.go](internal/agent/tools.go) — All 13 tool definitions + implementations
- [internal/agent/registry.go](internal/agent/registry.go) — Tool registry (register, definitions, dispatch)
- [internal/chat/backend.go](internal/chat/backend.go) — `Backend` interface
- [internal/chat/store.go](internal/chat/store.go) — Conversation/message persistence
- [internal/tui/app.go](internal/tui/app.go) — TUI (simplified, 4 keys only)
- [config/config.go](config/config.go) — Config loading, defaults

## Config location

`~/.sica/config.yaml` — generated on first run. The user's existing config may still have the old model name; delete this file to regenerate with defaults, or manually edit the `ai.ollama.model` field.

## Database

`~/.sica/data.db` — SQLite. Tables: `habits`, `habit_entries`, `projects`, `tasks`, `ai_conversations`, `ai_messages`.
