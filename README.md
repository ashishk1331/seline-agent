# Seline

A personal AI agent that lives in your Telegram. Seline is context-aware, tool-capable, and built to feel like talking to a real person — not a chatbot.

---

## Features

- **Conversational AI** — multi-turn context with automatic compaction when the context window fills up
- **Tool calling** — web search, URL fetching, file read/write, and more
- **Persistent sessions** — conversations are saved across restarts
- **Telegram-native UX** — live status bubble, typing indicator, markdown responses
- **Async end-to-end** — fully async Python stack with `httpx` and `python-telegram-bot`

---

## Requirements

- Python 3.12+
- [uv](https://github.com/astral-sh/uv)
- Docker (optional, for containerized deployment)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- An OpenRouter API key from [openrouter.ai](https://openrouter.ai)

---

## Setup

### 1. Clone the repo

```bash
git clone https://github.com/yourname/seline-agent.git
cd seline-agent
```

### 2. Install dependencies

```bash
uv sync
```

### 3. Configure environment

```bash
cp .env.example .env
```

Fill in your `.env`:

```dotenv
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
OPENROUTER_API_KEY=your_openrouter_api_key
MODEL_NAME=z-ai/glm-4.5-air:free
WORKSPACE_DIR=/home/youruser/Documents/seline-agent
```

### 4. Run

```bash
uv run agent
```

---

## Docker

### Build and run

```bash
docker build -t seline-agent .
docker run -d --name seline --env-file .env seline-agent
```

### Docker Compose (recommended)

```bash
docker compose up -d       # start in background
docker compose logs -f     # watch logs
docker compose down        # stop
```

---

## Development

```bash
# run with auto-reload on file changes
uv run watchfiles "uv run agent" src/

# lint and format
uv run ruff check --fix .
uv run ruff format .
```

---

## Project Structure

```
seline-agent/
├── src/
│   └── agent/
│       ├── __init__.py       # loads .env
│       ├── main.py           # entrypoint
│       ├── config.py         # environment config
│       ├── constants.py      # model headers and payloads
│       ├── llm.py            # complete() — main agent loop
│       ├── api.py            # async fetch()
│       ├── logger.py         # rich logging setup
│       ├── prompts.py        # system and compaction prompts
│       ├── context/          # ContextManager + Session + compaction
│       ├── gateway/          # Telegram bot handlers and status
│       └── tools/            # tool registry, web, file tools
├── workspace/                # sessions, logs (gitignored)
├── .env.example
├── pyproject.toml
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
| `TELEGRAM_BOT_TOKEN` | Bot token from BotFather | required |
| `OPENROUTER_API_KEY` | OpenRouter API key | required |
| `MODEL_NAME` | Primary LLM model | required |
| `MAX_TOKENS` | Max completion tokens | `1000` |
| `MAX_CONTEXT_TOKENS` | Context window size | `131000` |
| `TEMPERATURE` | Model temperature | `0.7` |
| `WORKSPACE_DIR` | Path for sessions and logs | `~/.workspace` |
| `COMPACTION_THRESHOLD` | Context % before compaction triggers | `0.7` |
| `COMPACTION_RECENT_N` | Recent messages kept after compaction | `20` |