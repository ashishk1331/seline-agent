# Agent Learning Checklist

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

> **src/ layout** — move package under `src/agent/`, install with `pip install -e ".[dev]"`;
> prevents accidental imports from project root.
> **pyproject.toml** — single config replacing `setup.py`, `requirements.txt`, `tox.ini`;
> declares dependencies, dev extras, and CLI entrypoint.
> **.env + .env.example** — all secrets and tunables in env vars; `.env.example` committed
> with empty values as documentation; `.env` always gitignored.
> **Provider-agnostic transport** — abstract OpenRouter behind one interface so swapping
> models means changing one constant, not touching agent logic.
> **Model pinning** — lock to explicit variant e.g. `zhipuai/glm-4-plus` in `constants.py`,
> never a floating alias; non-determinism is hard enough without surprise model changes.
> **Config from environment** — model name, API key, thresholds, prompts all in env vars or
> config file; nothing hardcoded in source.

---

## Tools & Actions
- [x] Web search tool
- [x] Run shell commands
- [x] Read/write files
- [x] Fetch and summarize a URL
- [x] Get current time/weather
- [x] Tool decorator + registry
- [ ] Code execution sandbox
- [ ] Tool result truncation
- [ ] Tool call hooks
- [ ] Toolset scoping

> **Tool decorator + registry** — `@tool` decorator introspects type hints and docstrings via
> `docstring-parser` to produce OpenAI-compatible schemas automatically; eliminates hand-written
> `TOOLS` list.
> **Code execution sandbox** — run agent-generated code in an isolated subprocess with resource
> limits; never exec in the main process.
> **Tool result truncation** — truncate or summarize large tool outputs before appending to
> context; unbounded results are the primary cause of context blowout.
> **Tool call hooks** — pre-hook for validation/logging, post-hook for sanitizing output before
> it hits context; applied on every tool call.
> **Toolset scoping** — send only tools relevant to the current task phase, not the full
> registry every turn; reduces prompt size and model confusion.

---

## Reliability
- [x] Recursion/depth guard on tool calls
- [x] Retry with exponential backoff on API failure
- [x] Graceful error reporting back to LLM
- [x] Timeout on tool execution
- [ ] Abort / cancel in-flight requests
- [ ] Tool input schema validation
- [ ] tool_calls response parsing
- [ ] Max token budget enforcement
- [ ] Convergence / loop detection
- [ ] Structured output enforcement

> **Abort / cancel** — `threading.Event` threaded through agent loop, every tool, and every
> subprocess; `proc.terminate()` on cancel; returns `{"status": "cancelled", "completed_steps":
> N, "partial_output": "..."}`; cleans up partial files, open handles, and dangling MCP
> sessions; wired to SIGINT and/or UI button.
> **Tool input schema validation** — validate model's `tool_calls` arguments against schema
> before dispatching; return schema errors as tool results, not crashes.
> **tool_calls response parsing** — GLM/OpenAI-compatible format returns `tool_calls` on the
> message object, not content blocks; handle both `content` and `tool_calls` fields.
> **Max token budget** — hard cap on total tokens per session using `prompt_tokens +
> completion_tokens` from `usage`; halt and surface to user on breach.
> **Convergence / loop detection** — if the same tool is called with identical args twice
> consecutively, or N turns pass with no user-visible progress, treat as stuck and surface
> rather than looping forever.
> **Structured output enforcement** — for tasks needing reliable JSON, use a dedicated LLM
> call with explicit JSON instructions rather than hoping the ReAct loop produces valid output.

---

## Context Engineering
- [x] Sliding window compaction
- [x] Token usage tracking
- [ ] Tool result masking
- [x] Auxiliary model routing
- [ ] Persist context across sessions
- [ ] Semantic memory
- [ ] Entity extraction
- [ ] Event sourcing for agent state

> **Sliding window compaction** — summarize old messages when approaching token limit; rebuild
> context as `[system] + [summary] + [recent N messages]`.
> **Token usage tracking** — OpenRouter returns `prompt_tokens`, `completion_tokens`,
> `total_tokens` under `response["usage"]`; track all three separately, not just total.
> **Tool result masking** — after a tool result is consumed, replace full content with a short
> reference e.g. `[web_fetch result used]`; frees context budget for future turns.
> **Auxiliary model routing** — route compaction and summarization to a cheaper GLM variant
> e.g. `glm-z1-flash`; reserve main model for reasoning; same OpenRouter endpoint.
> **Persist context** — save compacted context to JSONL or SQLite at end of session; load on
> next session start; save the already-compacted form, not raw history.
> **Semantic memory** — embed past messages and retrieve relevant ones by similarity; SQLite
> FTS5 is zero-dependency and sufficient before reaching for a vector DB.
> **Entity extraction** — track key nouns across turns: file paths, variable names, decisions,
> error messages; persist separately from raw message history.
> **Event sourcing** — append-only JSONL log of every input, tool call, tool result, and model
> output per session; enables replay and crash recovery; lives in `logs/` (gitignored).

---

## Planning & Reasoning
- [ ] Chain-of-thought via system prompt
- [ ] ReAct loop
- [ ] Multi-step task decomposition
- [ ] Self-critique
- [ ] Confidence / uncertainty signaling
- [ ] Phase gating

> **Chain-of-thought** — instruct the model to think step by step before choosing a tool or
> responding; one line in the system prompt, high leverage, no code changes needed.
> **ReAct loop** — Reason → Act → Observe → repeat; model emits a `Thought:` before every
> tool call and the result comes back as an `Observation:`; loop continues until no tool call
> is emitted.
> **Task decomposition** — explicit planning step before execution; model emits a numbered
> plan, then executes step by step and checks off as it goes.
> **Self-critique** — after generating a response, make a second LLM call asking if it fully
> answers the request; fix before returning to user.
> **Confidence signaling** — prompt the model to say "I'm not sure" rather than hallucinate;
> treat low-confidence tool selections as clarification requests.
> **Phase gating** — break long tasks into phases (plan → gather → execute → verify); do not
> allow execution before planning is confirmed; prevents premature irreversible actions.

---

## Guardrails & Safety
- [ ] Prompt injection detection
- [ ] Indirect injection in tool results
- [ ] Excessive agency prevention
- [ ] Tool permission tiers
- [ ] MCP tool description as untrusted input
- [ ] Cross-server escalation prevention
- [ ] PII / secrets detection
- [ ] Hardline command blocklist
- [ ] Input guardrail pipeline
- [ ] Circuit breaker

> **Prompt injection detection** — web pages, files, and command output can contain adversarial
> instructions; scan tool outputs before appending to context (OWASP LLM01:2025).
> **Indirect injection** — after a tool call, verify the model's next intended action still
> aligns with the original user goal; a fetched page could redirect the agent.
> **Excessive agency** — require explicit per-action confirmation for destructive or irreversible
> tool calls: delete, overwrite, network POST (OWASP LLM06:2025).
> **Tool permission tiers** — classify tools in the registry: read-only (auto), write/local
> (log + proceed), destructive (confirm); enforced in registry, not ad-hoc per call.
> **MCP tool descriptions** — tool metadata from third-party MCP servers is supply-chain input,
> not developer-authored config; scan for embedded directives before injecting into prompt
> (OWASP LLM05 / CVE-2025-54136).
> **Cross-server escalation** — validate that multi-tool sequences across MCP servers don't
> exceed the permissions of any single server involved.
> **PII / secrets detection** — scan outputs and tool results for credentials, tokens, and
> personal data before logging or returning to user.
> **Command blocklist** — shell tool maintains a hardcoded blocklist of commands that can never
> run regardless of model instruction e.g. `rm -rf /`, `git push --force`.
> **Input guardrail pipeline** — validate user input before it enters context: length limits,
> topic boundary, injection pattern detection; cheap regex first, LLM classifier only if needed.
> **Circuit breaker** — halt and surface to user if cost, turn count, or wall-clock time
> exceeds configured threshold.

---

## MCP Integration & Management

### Connection & Discovery
- [ ] MCP client implementation
- [ ] Runtime tool discovery
- [ ] MCP server config file
- [ ] Server health check on startup

> **MCP client** — connect to servers via STDIO (local process) or HTTP/SSE (remote); query
> `tools/list` at handshake and merge schemas into tool registry dynamically.
> **Runtime tool discovery** — MCP tools and native tools go through the same dispatch path;
> no special casing in the agent loop.
> **MCP server config** — declare servers in `mcp_servers.yaml` with transport type,
> command/URL, and env var references for secrets; no addresses hardcoded in source.
> **Health check on startup** — verify each configured MCP server responds to `initialize`
> before the session starts; surface dead servers as warnings, don't silently drop their tools.

### Access Control
- [ ] Tool allowlist per session
- [ ] Tool permission tiers for MCP tools
- [ ] OAuth 2.1 / API key management
- [ ] Least-privilege server scope

> **Tool allowlist** — maintain explicit approved tool names; reject any tool call not on the
> allowlist even if a server advertises it; prevents rogue tool advertisement mid-session.
> **MCP permission tiers** — apply the same read/write/destructive classification to MCP tools
> as native tools; don't auto-approve MCP write tools just because the server is trusted.
> **Auth management** — store MCP credentials in env vars or secrets file, never in the server
> config itself; rotate on a schedule.
> **Least-privilege scope** — connect each MCP server with minimum permissions it needs; a web
> search server should not have filesystem access even if the server supports it.

### Security
- [ ] Tool description sanitization
- [ ] Tool description hash pinning
- [ ] Indirect injection in MCP tool results
- [ ] MCP server version pinning
- [ ] mcp-scan audit
- [ ] No auto-approval for MCP destructive tools

> **Description sanitization** — scan MCP tool names, descriptions, and input schemas for
> embedded directives before injecting into prompt; this surface looks like config but the
> model reads it as instructions.
> **Hash pinning** — store a hash of each server's tool descriptions at first connection; abort
> if they change between sessions without a version bump; silent description changes are the
> primary tool poisoning vector.
> **MCP result injection** — apply the same injection scanning to MCP tool results as to
> `web_fetch` output before appending to context.
> **Server version pinning** — treat MCP servers like dependencies; pin to a specific version
> or commit; a server update that silently changes tool descriptions is a supply-chain risk.
> **mcp-scan** — run Invariant Labs' `mcp-scan` against server configs periodically; checks
> for known poisoning patterns, rug-pulls, and cross-server escalation risks.
> **No auto-approval** — tool poisoning achieves 84% success rate in testing when auto-approval
> is on; require explicit confirmation for any MCP tool that writes, deletes, sends, or posts.

### Observability
- [ ] Log MCP tool calls separately
- [ ] Track MCP tool latency

> **MCP call logging** — tag JSONL entries with `mcp_server: server_name` to distinguish
> MCP-sourced calls from native tool calls in post-session analysis.
> **MCP latency tracking** — remote MCP servers add network latency; measure per-server p50/p99
> and use to set realistic per-server timeouts rather than a global default.

---

## Multi-Agent
- [ ] Spawning subagents for subtasks
- [ ] Orchestrator / worker pattern
- [ ] Shared context or message passing
- [ ] Agent roles + permissions model
- [ ] Human-in-the-loop approval step
- [ ] Subagent isolation

> **Subagents** — spawn a fresh agent instance with its own context for a bounded subtask;
> collect and return only the result, not the full conversation.
> **Orchestrator / worker** — same OpenRouter/GLM endpoint, different system prompts and tool
> subsets per role; orchestrator plans and delegates, workers execute.
> **Message passing** — agents communicate via a shared JSONL file or in-memory queue; never
> share raw context windows between agents.
> **Roles + permissions** — each agent role gets only the tools it needs; a research worker
> shouldn't have `write_file`; a file worker shouldn't have `web_search`.
> **Human-in-the-loop** — checkpoint requiring human confirmation before high-stakes
> orchestrator decisions e.g. deploying, deleting, sending.
> **Subagent isolation** — pass only the task handoff artifact to subagents, not full parent
> context; prevents 4–15x token multiplication in multi-agent runs.

---

## Skills System

### Storage & Format
- [ ] Skills directory
- [ ] SKILL.md format
- [ ] Skills index

> **Skills directory** — store skills as `SKILL.md` files under `skills/` at project root
> (outside `src/`); gittrack as data, not library code.
> **SKILL.md format** — YAML frontmatter: `name`, `description`, `triggers`, `version`;
> sections: When to Use, Procedure, Pitfalls, Verification; compatible with agentskills.io.
> **Skills index** — `skills_index.json` lists all skill names, descriptions, and trigger
> keywords; agent scans index to decide what to load, not the full files; keeps token cost low.

### Loading & Activation
- [ ] Progressive disclosure
- [ ] Trigger matching
- [ ] Skill injection point

> **Progressive disclosure** — inject only the skills index at session start; load full skill
> content only when task matches its triggers; never inject all skills simultaneously.
> **Trigger matching** — keyword match first (cheap), LLM classifier second (only if ambiguous);
> check before each ReAct turn.
> **Injection point** — inject loaded skill as a system message immediately before the relevant
> ReAct step; not at session start, not as a user message.

### Creation & Improvement
- [ ] Autonomous skill creation
- [ ] Skill refinement
- [ ] Manual skill authoring

> **Autonomous creation** — after a task requiring 5+ tool calls, prompt the agent to write a
> `SKILL.md` capturing the procedure; store in `skills/` for future reuse.
> **Skill refinement** — on failed or significantly deviated execution, append a new entry to
> the skill's Pitfalls section; skills improve through use rather than being rewritten wholesale.
> **Manual authoring** — hand-write skills for known recurring workflows; highest quality since
> you control the procedure exactly.

### Lifecycle Management
- [ ] Skill curator job
- [ ] Skill versioning
- [ ] External skill directories
- [ ] agentskills.io compatibility

> **Curator job** — periodic task that grades skills by success rate, consolidates duplicates,
> and deprecates skills unused for N days; run on a schedule e.g. every 7 days.
> **Skill versioning** — `version` field in YAML frontmatter; bump on significant procedure
> changes; enables rollback if a refinement degrades performance.
> **External skill dirs** — support additional skill directories (project-local, team-shared,
> agentskills.io installs); resolved in order with local taking precedence.
> **agentskills.io compatibility** — following the SKILL.md spec gives access to 700+ community
> skills installable without reinventing every workflow.

---

## Observability & Cost
- [ ] Structured logging
- [ ] Per-session cost tracking
- [ ] Latency tracking per tool call
- [ ] Skills hit rate tracking
- [ ] Prompt diff tracking
- [ ] OpenRouter headers logging
- [ ] Action dry-run mode
- [ ] Sandboxed filesystem

> **Structured logging** — every tool call: name, input, output, latency, token delta as JSONL
> under `logs/`; `logs/` is gitignored.
> **Cost tracking** — compute cost from `prompt_tokens` and `completion_tokens` using
> OpenRouter's per-model pricing for your GLM variant; report at end of session.
> **Latency tracking** — measure per-tool call duration; use data to set realistic timeouts
> rather than a global default.
> **Skills hit rate** — log which skills activated per session; zero-activation skills over 30
> days are pruning candidates; high-activation skills are candidates for permanent system prompt
> inclusion.
> **Prompt diff tracking** — log before/after on every system prompt or compaction prompt
> change; correlate with eval regressions when quality drops.
> **OpenRouter headers** — log `HTTP-Referer` and model string sent per request; confirms which
> GLM variant OpenRouter actually routed to; useful when debugging unexpected behavior.
> **Dry-run mode** — agent describes intended actions before executing; user confirms; essential
> when adding new tools or running in unfamiliar environments.
> **Sandboxed filesystem** — constrain `write_file` to a working directory; agent cannot write
> outside it regardless of what path the model requests.

---

## Evals
- [ ] Task benchmark
- [ ] Pass/fail scoring
- [ ] Tool call accuracy + hallucination rate
- [ ] Regression testing
- [ ] Latency + cost per run
- [ ] Adversarial test cases
- [ ] CI integration

> **Task benchmark** — small set of tasks the agent should complete end-to-end; stored as
> `evals/tasks.jsonl`; covers happy path, edge cases, and multi-step workflows.
> **Pass/fail scoring** — each task has a verifiable expected outcome; scored automatically
> where possible, manually where not.
> **Tool call accuracy** — measure whether the agent calls the right tool with the right args;
> hallucination rate measures how often the model invents facts not in tool results.
> **Regression testing** — run the full eval suite before and after any prompt or tool schema
> change; a quality drop is a regression, treat it like a failing test.
> **Latency + cost** — track OpenRouter-reported token costs per GLM variant so you can compare
> model upgrades on both quality and cost axes simultaneously.
> **Adversarial cases** — include prompt injection attempts, conflicting tool results, and
> malformed inputs in the benchmark; these catch guardrail regressions.
> **CI integration** — eval suite triggers automatically on every prompt or tool schema change;
> not a manual step.