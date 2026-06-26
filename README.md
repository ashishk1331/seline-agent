# Seline

A personal AI agent that lives in your Telegram. Seline is context-aware, tool-capable, and built to feel like talking to a real person — not a chatbot.

---

## Features

- **Conversational AI** — multi-turn context with automatic compaction when the context window fills up
- **Tool calling** — web search, URL fetching, file read/write, and shell commands
- **Persistent sessions** — conversations are saved across restarts (JSONL per session)
- **Telegram-native UX** — live status bubble, markdown responses, debounced input
- **Provider-agnostic** — OpenRouter / Sarvam / NVIDIA / Cerebras behind one config switch
- **Concurrent & graceful** — Go goroutines end-to-end, graceful shutdown on SIGINT/SIGTERM

---

## Requirements

- Go 1.25+
- Docker (optional, for containerized deployment)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- An API key for your chosen provider (OpenRouter / Sarvam / NVIDIA / Cerebras)
- A TinyFish API key for web search/fetch

---

## Setup

### 1. Clone the repo

```bash
git clone https://github.com/ashishk1331/seline-agent.git
cd seline-agent
```

### 2. Configure environment

```bash
cp .env.sample .env
```

Fill in your `.env` (see [Environment Variables](#environment-variables)).

### 3. Run

```bash
go mod download
go run ./cmd/agent
```

Or build a binary:

```bash
go build -o bin/agent ./cmd/agent
./bin/agent
```

---

## Docker

### Build and run

```bash
docker build -t seline-agent .
# A named volume keeps sessions across restarts; the container runs as a
# non-root user, so a bare bind mount would hit permission errors.
docker run -d --name seline --env-file .env \
  -v seline-workspace:/app/workspace seline-agent
```

> Without the volume the agent still runs, but sessions live inside the
> container and are lost when it's removed.

### Docker Compose (recommended)

```bash
docker compose up -d       # start in background
docker compose logs -f     # watch logs
docker compose down        # stop
```

---

## Development

```bash
# live reload on file changes (https://github.com/air-verse/air)
# install once (kept out of go.mod — it drags in a large dependency tree):
go install github.com/air-verse/air@latest
air

# format and vet
gofmt -w ./cmd ./internal
go vet ./...
```

---

## Project Structure

```
seline-agent/
├── cmd/
│   └── agent/              # entrypoint: load env → wire deps → run
├── internal/
│   ├── config/             # environment config + validation
│   ├── logging/            # charmbracelet/log setup
│   ├── constants/          # TinyFish headers, thinking phrases
│   ├── prompts/            # system/compaction prompts (go:embed *.md)
│   ├── types/              # shared Message / Usage / CompletionResponse
│   ├── humanid/            # human-readable session IDs
│   ├── tools/              # tool registry + web/file/shell tools
│   ├── provider/           # LLM resolver (HTTP client + payload)
│   ├── contextmgr/         # ContextManager + Session + compaction
│   ├── llm/                # recursive tool-calling agent loop
│   └── gateway/            # Telegram bot, debounce, status, errors, rich messages
├── workspace/              # sessions (gitignored, runtime)
├── .env.sample
├── go.mod
└── Dockerfile
```

---

## Telegram Commands

| Command | Description |
|---|---|
| `/start` | Start the bot |
| `/consumption` | Show current context token usage |

---

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `AI_PROVIDER` | `OPENROUTER` \| `SARVAM` \| `NVIDIA` \| `CEREBRAS` | required |
| `<PROVIDER>_API_KEY` | API key for the chosen provider | required |
| `TINYFISH_API_KEY` | TinyFish API key (web search/fetch) | required |
| `TELEGRAM_BOT_TOKEN` | Bot token from BotFather | required |
| `TELEGRAM_ALLOWLIST` | Comma-separated `@usernames` allowed to use the bot | required |
| `<PROVIDER>_URL` | Override the provider chat-completions URL | provider default |
| `MODEL_NAME` | Primary LLM model | `z-ai/glm-4.5-air:free` |
| `MAX_TOKENS` | Max completion tokens | `1000` |
| `MAX_CONTEXT_TOKENS` | Context window size | `131000` |
| `TEMPERATURE` | Model temperature | `0.7` |
| `TINYFISH_SEARCH_URL` | TinyFish search endpoint | `https://api.search.tinyfish.ai` |
| `TINYFISH_FETCH_URL` | TinyFish fetch endpoint | `https://api.fetch.tinyfish.ai` |
| `COMPACTION_THRESHOLD` | Context fraction before compaction triggers | `0.9` |
| `COMPACTION_RECENT_N` | Recent messages kept after compaction | `5` |
| `MAX_TOOL_CALLS` | Max tool-call recursion depth per turn | `5` |
| `MESSAGE_DEBOUNCE_DELAY` | Base debounce delay (seconds) | `1.0` |
| `MESSAGE_DEBOUNCE_JITTER` | Per-message delay step (seconds) | `0.3` |
| `MESSAGE_DEBOUNCE_MAX_DELAY` | Max debounce delay (seconds) | `2.0` |
| `WORKSPACE_DIR` | Base path for sessions (`.workspace` is appended) | `$HOME` |
