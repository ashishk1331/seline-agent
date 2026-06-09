from .prompts import get_system_prompt, get_compaction_prompt
from .config import CONFIG
from .constants import HEADERS, COMPACTION_PAYLOAD
from .api import fetch


class ContextManager:
    def __init__(self):
        self.context = [{"role": "system", "content": get_system_prompt()}]
        self.compaction_context = [
            {"role": "system", "content": get_compaction_prompt()}
        ]
        self.max_tokens = CONFIG.MAX_CONTEXT_TOKENS
        self.current_tokens = 0

    def append(self, message, usage=None):
        self.context.append(message)
        if usage:
            self.current_tokens = usage["total_tokens"]
            self.detect_and_compact()

    def get_context(self):
        return self.context

    def get_consumption(self):
        return {
            "current_tokens": self.current_tokens,
            "current_tokens_in_words": f"{round(self.current_tokens / 1_000, 1)}K"
            if self.current_tokens >= 1_000
            else str(self.current_tokens),
            "max_tokens": self.max_tokens,
            "max_tokens_in_words": f"{self.max_tokens // 1_000}K",
            "remaining_tokens": self.max_tokens - self.current_tokens,
            "percentage_used": round((self.current_tokens / self.max_tokens) * 100, 2),
        }

    def messages_iron(self, messages):
        ironed = []
        for m in messages:
            content = m["content"]
            if isinstance(content, list):
                content = " ".join(
                    c.get("text", "") for c in content if isinstance(c, dict)
                )
            ironed.append(f"[{m['role']}] {content}")
        return "\n".join(ironed)

    def compaction(self, messages):
        data = fetch(
            CONFIG.OPENROUTER_URL,
            headers=HEADERS,
            payload=COMPACTION_PAYLOAD | {"messages": messages},
        )

        if not data:
            print("[ERROR] No response from API during compaction.")
            return None, None

        summary = data["choices"][0]["message"]["content"]
        usage = data["usage"]
        return summary, usage

    def detect_and_compact(self):
        if self.current_tokens < self.max_tokens * CONFIG.COMPACTION_THRESHOLD:
            return

        if len(self.context) <= CONFIG.COMPACTION_RECENT_N + 1:
            print(
                f"[CONTEXT] Context length is {len(self.context)}. Not enough messages to compact."
            )
            return

        print(f"[CONTEXT] Auto-compaction triggered.")

        recent_messages = self.context[-CONFIG.COMPACTION_RECENT_N :]
        previous_messages = self.context[1 : -CONFIG.COMPACTION_RECENT_N]
        prev_token_count = self.current_tokens

        summary, usage = self.compaction(
            self.compaction_context
            + [{"role": "user", "content": self.messages_iron(previous_messages)}]
        )

        if summary is None:
            print(f"[CONTEXT] Compaction failed. Keeping existing context.")
            return

        self.context = (
            [{"role": "system", "content": get_system_prompt()}]
            + [
                {
                    "role": "system",
                    "content": f"[Compacted summary of earlier conversation: {summary}]",
                }
            ]
            + recent_messages
        )
        if usage:
            self.current_tokens = usage["total_tokens"]

        print(
            f"[CONTEXT] Compaction completed. {prev_token_count} -> {self.current_tokens}"
        )
