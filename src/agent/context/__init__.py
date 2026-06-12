from ..prompts import get_system_prompt, get_compaction_prompt
from ..config import CONFIG
from ..constants import HEADERS, BASIC_PAYLOAD
from ..api import fetch
from ..logger import log
from .session import Session


class ContextManager(Session):
    def __init__(self):
        super().__init__()
        self.context = [
            {"role": "system", "content": get_system_prompt()},
            *self._load_messages_from_session(),
        ]
        self.compaction_context = [
            {"role": "system", "content": get_compaction_prompt()}
        ]
        self.max_tokens = CONFIG.MAX_CONTEXT_TOKENS
        self.current_tokens = self._retrive_consumption().get("current_tokens", 0)

    async def append(self, message, usage=None):
        self.context.append(message)
        if usage:
            self.current_tokens = usage["total_tokens"]
            await self.detect_and_compact()
        self._append_message_to_session(message)
        self._dump_consumption(self.get_consumption())

    def get_context(self):
        return self.context

    def _in_words(self, number):
        if number >= 1_000_000:
            return f"{round(number / 1_000_000, 1)}M"
        elif number >= 1_000:
            return f"{round(number / 1_000, 1)}K"
        else:
            return str(number)

    def get_consumption(self):
        return {
            "current_tokens": self.current_tokens,
            "current_tokens_in_words": self._in_words(self.current_tokens),
            "max_tokens": self.max_tokens,
            "max_tokens_in_words": self._in_words(self.max_tokens),
            "remaining_tokens": self.max_tokens - self.current_tokens,
            "remaining_tokens_in_words": self._in_words(
                self.max_tokens - self.current_tokens
            ),
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

    async def compaction(self, messages):
        data = await fetch(
            CONFIG.OPENROUTER_URL,
            headers=HEADERS,
            payload=BASIC_PAYLOAD | {"messages": messages},
        )

        if not data:
            log.error("No response from API during compaction.")
            return None, None

        summary = data["choices"][0]["message"]["content"]
        usage = data["usage"]
        return summary, usage

    async def detect_and_compact(self):
        if self.current_tokens < self.max_tokens * CONFIG.COMPACTION_THRESHOLD:
            return

        if len(self.context) <= CONFIG.COMPACTION_RECENT_N + 1:
            log.error(
                f"[CONTEXT] Context length is {len(self.context)}. Not enough messages to compact."
            )
            return

        log.info("[CONTEXT] Auto-compaction triggered.")

        recent_messages = self.context[-CONFIG.COMPACTION_RECENT_N :]
        previous_messages = self.context[1 : -CONFIG.COMPACTION_RECENT_N]
        prev_token_count = self.current_tokens

        summary, usage = await self.compaction(
            self.compaction_context
            + [{"role": "user", "content": self.messages_iron(previous_messages)}]
        )

        if summary is None:
            log.error("[CONTEXT] Compaction failed. Keeping existing context.")
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

        self._overwrite_messages_in_session(self.context[1:])
        self._dump_consumption(self.get_consumption())

        log.info(
            f"[CONTEXT] Compaction completed. {prev_token_count} -> {self.current_tokens}"
        )
