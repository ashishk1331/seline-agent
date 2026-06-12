# Telegram AI Agent — Learning Checklist

---

## Foundation
- [x] LLM completions via HTTP
- [x] Multi-turn context management
- [x] Tool calling loop
- [x] Modular project structure
- [x] src/ layout
- [x] pyproject.toml
- [x] .env + .env.example
- [ ] Provider-agnostic transport layer
- [x] Model pinning
- [x] Config from environment

> **Provider-agnostic transport** — abstract your LLM provider behind one interface so swapping
> models means changing one constant, not touching agent logic.
> **Model pinning** — lock to an explicit variant in `constants.py`, never a floating alias;
> non-determinism is hard enough without surprise model changes.
> **Config from environment** — model name, API key, bot token, thresholds, prompts all in env
> vars; nothing hardcoded in source.

---

## Telegram Bot Layer

### Bot Setup & Lifecycle
- [x] Bot token from BotFather
- [x] `python-telegram-bot` (async) integration
- [x] `/start` and `/help` command handlers
- [ ] Webhook mode (production) vs polling (dev) toggle
- [ ] Graceful shutdown on SIGINT/SIGTERM
- [ ] Bot restart recovery — resume pending sessions on startup
- [ ] Health-check endpoint alongside webhook server

> **Webhook vs polling** — polling is fine for dev; for production, use a webhook behind HTTPS
> (e.g. via `ngrok` locally, or a VPS/Railway deployment); Telegram requires HTTPS for webhooks.
> **Graceful shutdown** — flush in-flight conversations and close aiohttp/httpx sessions cleanly
> on SIGTERM; avoids dropping mid-task agent runs.
> **Restart recovery** — persist the last conversation state to SQLite or JSONL so the bot can
> resume or summarise what was in-progress rather than silently forgetting after a redeploy.

### Message Handling
- [x] Text message handler
- [ ] Photo / document upload handler (pass to vision or file tools)
- [ ] Voice message handler → transcribe → agent
- [ ] `/cancel` command to abort in-flight agent task
- [ ] Edited message handling (ignore or re-run)
- [ ] Forwarded message handling
- [ ] Sticker / reaction handling (optional UX touch)

> **Photo/document handler** — encode uploaded images as base64 and pass to a vision-capable
> model; for documents, download via Telegram's `getFile`, then pass to file-reading tools.
> **Voice handler** — download `.ogg` voice note, run Whisper (or OpenAI transcription) to get
> text, then feed into the normal agent loop; enables fully hands-free use.
> **`/cancel`** — wire to a `threading.Event` or `asyncio.Event` threaded through the agent
> loop; returns partial output and cleanup summary on cancel.

### UX & Interaction Patterns
- [ ] Typing indicator while agent is running (`send_chat_action`)
- [ ] Message chunking for long responses (Telegram's 4096-char limit)
- [ ] Markdown / HTML formatting in replies
- [ ] Inline keyboard buttons for confirmations and choices
- [ ] Reply keyboard for common commands
- [ ] Streaming responses via message edits (progressive output)
- [ ] Progress messages for multi-step tool chains

> **Typing indicator** — call `send_chat_action(ChatAction.TYPING)` at the start of every agent
> turn; re-call every 4s for long-running tasks; users see the bot "thinking" rather than
> assuming it crashed.
> **Message chunking** — split responses at paragraph boundaries, not mid-sentence; send each
> chunk as a separate message or use `edit_message_text` to stream content progressively.
> **Streaming via edits** — send a placeholder message, then edit it as the LLM streams tokens;
> gives a real-time feel without spamming the chat.
> **Confirmation keyboards** — use `InlineKeyboardMarkup` for destructive tool confirmations
> (delete, send, overwrite) instead of asking the user to type "yes".

### Multi-User & Session Management
- [ ] Per-user conversation context (keyed by `chat_id`)
- [ ] Per-user session isolation (no context leakage between users)
- [ ] Allowlist / blocklist by `user_id` or `chat_id`
- [ ] Group chat support (respond only when mentioned / replied to)
- [ ] Admin commands gated by `user_id`
- [ ] Concurrent user handling (async per-chat locks)

> **Per-user context** — store conversation history in a dict or DB keyed by `chat_id`; never
> share context windows between users.
> **Async locks** — use `asyncio.Lock` per `chat_id` to prevent concurrent messages from the
> same user interleaving tool calls and corrupting context.
> **Group chat** — in group mode, only trigger the agent when the bot is @-mentioned or the
> message is a direct reply; otherwise ignore; prevents noise.
> **Admin commands** — `/flush`, `/stats`, `/broadcast` etc. should check `user_id` against an
> `ADMIN_IDS` env var before executing.

---

## Tools & Actions
- [x] Web search tool
- [x] Run shell commands
- [x] Read/write files
- [x] Fetch and summarize a URL
- [x] Get current time/weather
- [x] Tool decorator + registry
- [ ] Send Telegram message as a tool (agent can message proactively)
- [ ] Download Telegram file as a tool input
- [ ] Code execution sandbox
- [ ] Tool result truncation
- [ ] Tool call hooks
- [ ] Toolset scoping

> **Send message tool** — expose `send_message(chat_id, text)` as a tool so the agent can
> proactively notify the user mid-task rather than only replying at the end.
> **Download Telegram file** — given a `file_id`, download via `bot.get_file()` and return the
> local path; lets the agent process user-uploaded images, PDFs, and audio natively.
> **Code execution sandbox** — run agent-generated code in an isolated subprocess with resource
> limits; never exec in the main process.
> **Tool result truncation** — truncate or summarize large tool outputs before appending to
> context; unbounded results are the primary cause of context blowout.
> **Tool call hooks** — pre-hook for validation/logging, post-hook for sanitizing output before
> it hits context; applied on every tool call.
> **Toolset scoping** — send only tools relevant to the current task phase; reduces prompt size
> and model confusion.

---

## Reliability
- [x] Recursion/depth guard on tool calls
- [x] Retry with exponential backoff on API failure
- [x] Graceful error reporting back to LLM
- [x] Timeout on tool execution
- [ ] Abort / cancel in-flight requests (wired to `/cancel` command)
- [ ] Tool input schema validation
- [ ] Max token budget enforcement
- [ ] Convergence / loop detection
- [ ] Structured output enforcement
- [ ] Telegram API rate limit handling (30 msg/s global, 1 msg/s per chat)
- [ ] Dead-letter queue for failed Telegram sends

> **`/cancel` wiring** — `threading.Event` or `asyncio.Event` threaded through every tool and
> subprocess; on `/cancel`, terminate in-flight work, return partial output summary to the user.
> **Telegram rate limits** — Telegram enforces 30 messages/second globally and 1/second per
> chat; back off with jitter on `RetryAfter` errors; queue outgoing messages rather than
> firing in a tight loop.
> **Dead-letter queue** — if a Telegram send fails after retries (e.g. user blocked the bot),
> log the failed message to a JSONL dead-letter file for later inspection; don't crash the agent.

---

## Context Engineering
- [x] Sliding window compaction
- [x] Token usage tracking
- [x] Auxiliary model routing
- [ ] Per-chat persistent context (SQLite or JSONL)
- [ ] Tool result masking
- [ ] Semantic memory
- [ ] Entity extraction
- [ ] Event sourcing for agent state
- [ ] Session summary on `/reset` or idle timeout

> **Per-chat persistence** — save compacted context per `chat_id` to SQLite at end of each
> turn; load on the next message; users expect the bot to remember across app restarts.
> **Idle timeout** — after N minutes of inactivity, compact and archive the session; on next
> message, start fresh but offer a one-line summary of the last session.
> **Session summary on `/reset`** — before wiping context, send a brief bullet-point summary of
> what was accomplished; gives the user a natural checkpoint.
> **Tool result masking** — after a tool result is consumed, replace full content with a short
> reference e.g. `[web_fetch result used]`; frees context budget for future turns.
> **Entity extraction** — track key nouns across turns: file paths, variable names, decisions;
> persist separately from raw message history for fast retrieval.

---

## Planning & Reasoning
- [ ] Chain-of-thought via system prompt
- [ ] ReAct loop
- [ ] Multi-step task decomposition
- [ ] Self-critique
- [ ] Confidence / uncertainty signaling
- [ ] Phase gating
- [ ] Clarification requests before long tasks

> **Clarification before long tasks** — if the user's intent is ambiguous, ask one clarifying
> question before starting a multi-step tool chain; cheaper than executing the wrong plan and
> having to undo it.
> **ReAct loop** — Reason → Act → Observe → repeat; model emits a `Thought:` before every
> tool call and the result comes back as an `Observation:`; loop continues until no tool call
> is emitted.
> **Phase gating** — break long tasks into phases (plan → gather → execute → verify); confirm
> the plan with the user via an inline keyboard before execution; prevents premature irreversible
> actions.

---

## Guardrails & Safety
- [ ] Prompt injection detection
- [ ] Indirect injection in tool results
- [ ] Excessive agency prevention (confirm destructive actions via inline keyboard)
- [ ] Tool permission tiers
- [ ] PII / secrets detection
- [ ] Hardline command blocklist
- [ ] Input guardrail pipeline
- [ ] Circuit breaker (cost/turn/time threshold)
- [ ] Per-user rate limiting (prevent abuse)
- [ ] Message content filtering before sending to Telegram

> **Confirmation via inline keyboard** — for destructive or irreversible tool calls, send an
> `InlineKeyboardMarkup` with Yes/No buttons; do not proceed until the user taps Yes.
> **Per-user rate limiting** — track messages per `user_id` per minute; return a friendly
> throttle message rather than processing a flood; prevents abuse and runaway API costs.
> **Content filtering before send** — scan agent output for secrets, credentials, or PII before
> sending to Telegram; Telegram messages are logged server-side and may be forwarded by users.
> **Input guardrail pipeline** — validate user input before it enters context: length limits,
> topic boundary, injection pattern detection; cheap regex first, LLM classifier only if needed.

---

## MCP Integration & Management

### Connection & Discovery
- [ ] MCP client implementation
- [ ] Runtime tool discovery
- [ ] MCP server config file (`mcp_servers.yaml`)
- [ ] Server health check on startup

### Access Control
- [ ] Tool allowlist per session
- [ ] Tool permission tiers for MCP tools
- [ ] OAuth 2.1 / API key management
- [ ] Least-privilege server scope

### Security
- [ ] Tool description sanitization
- [ ] Tool description hash pinning
- [ ] Indirect injection in MCP tool results
- [ ] MCP server version pinning
- [ ] No auto-approval for MCP destructive tools

### Observability
- [ ] Log MCP tool calls separately
- [ ] Track MCP tool latency

> **MCP + Telegram confirmations** — for any MCP tool in the write/destructive tier, send an
> `InlineKeyboardMarkup` confirm prompt to the user before dispatching; never auto-approve.

---

## Multi-User & Deployment
- [ ] Dockerized deployment
- [ ] Environment-based config (no secrets in image)
- [ ] Process supervisor (systemd / supervisord / Railway)
- [ ] Auto-restart on crash
- [ ] Rolling deploys without dropping active sessions
- [ ] Separate staging bot token for testing

> **Dockerized** — package the bot as a Docker image; use `CMD ["uv", "run", "agent"]`; inject
> all secrets via environment variables at runtime, never baked into the image.
> **Staging bot token** — maintain a separate `@YourBot_staging` bot for testing; route to it
> via a `BOT_TOKEN_STAGING` env var; never test against the production bot.
> **Rolling deploys** — drain active sessions before restarting (or use `/notify` to warn
> active users of a brief restart); SQLite-persisted context ensures they can continue after.

---

## Multi-Agent
- [ ] Spawning subagents for subtasks
- [ ] Orchestrator / worker pattern
- [ ] Shared context or message passing
- [ ] Agent roles + permissions model
- [ ] Human-in-the-loop approval via Telegram inline keyboard
- [ ] Subagent isolation

> **Human-in-the-loop via Telegram** — orchestrator sends a summary and an
> `InlineKeyboardMarkup` approval prompt to the user before high-stakes actions; no
> out-of-band approval mechanism needed since Telegram is already the interface.

---

## Skills System

### Storage & Format
- [ ] Skills directory (`skills/` at project root)
- [ ] `SKILL.md` format with YAML frontmatter
- [ ] Skills index (`skills_index.json`)
- [ ] Telegram-specific skills (e.g. `send-report`, `poll-creation`, `media-download`)

### Loading & Activation
- [ ] Progressive disclosure (index only at session start)
- [ ] Trigger matching (keyword → LLM classifier)
- [ ] Skill injection point (before relevant ReAct step)

### Creation & Improvement
- [ ] Autonomous skill creation after complex tasks
- [ ] Skill refinement on failure
- [ ] Manual skill authoring for known workflows

---

## Observability & Cost
- [ ] Structured JSONL logging per `chat_id`
- [ ] Per-session cost tracking (reported to user on `/stats`)
- [ ] Latency tracking per tool call
- [ ] Skills hit rate tracking
- [ ] OpenRouter headers logging
- [ ] Action dry-run mode
- [ ] Sandboxed filesystem for file writes
- [ ] Admin `/stats` command showing total cost, active sessions, error rate

> **Per-chat JSONL logs** — tag every log entry with `chat_id` and `user_id`; enables per-user
> cost attribution and debugging without mixing up conversations.
> **`/stats` command** — admin-only command that reports total API spend, number of active
> sessions, tool call counts, and error rate for the current day; pulled from the JSONL logs.
> **Dry-run mode** — agent describes its intended actions and sends them via Telegram before
> executing; user taps Confirm; essential when onboarding the bot to a new environment.

---

## Evals
- [ ] Task benchmark (`evals/tasks.jsonl`)
- [ ] Pass/fail scoring
- [ ] Tool call accuracy + hallucination rate
- [ ] Regression testing on prompt changes
- [ ] Latency + cost per run
- [ ] Adversarial test cases (prompt injection, malformed input)
- [ ] CI integration
- [ ] Telegram-specific eval: message chunking, inline keyboard flows, voice round-trip

> **Telegram-specific evals** — test that long responses chunk correctly, that inline keyboard
> confirmation flows complete end-to-end, and that voice → transcribe → agent → reply round
> trips within an acceptable latency budget.
> **Regression testing** — run the full eval suite before and after any system prompt, tool
> schema, or handler change; a quality drop is a regression, treat it like a failing test.