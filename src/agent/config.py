import os as OS
from pathlib import Path
from rich.markdown import Markdown
from .logger import console
import re

ALLOWLIST_PATTERN = re.compile(
    r"^(@[a-zA-Z][a-zA-Z0-9_]{4,31})(,@[a-zA-Z][a-zA-Z0-9_]{4,31})*$"
)

PROVIDER_URLS = {
    "OPENROUTER": "https://openrouter.ai/api/v1/chat/completions",
    "SARVAM": "https://api.sarvam.ai/v1/chat/completions",
    "NVIDIA": "https://integrate.api.nvidia.com/v1/chat/completions",
    "CEREBRAS": "https://api.cerebras.ai/v1/chat/completions",
}


class ConfigManager:
    def __init__(self):
        # Non-optional settings
        self.AI_PROVIDER = self._required("AI_PROVIDER")
        self.TINYFISH_API_KEY = self._required("TINYFISH_API_KEY")
        self.TELEGRAM_BOT_TOKEN = self._required("TELEGRAM_BOT_TOKEN")
        self.TELEGRAM_ALLOWLIST = self._required(
            "TELEGRAM_ALLOWLIST",
            "Add comma-separated Telegram usernames to allow access to Seline. Example: @john_doe,@jane_doe",
        )
        self.TELEGRAM_ALLOWLIST_CLEANED = self._sanitize_telegram_allowlist(
            self.TELEGRAM_ALLOWLIST
        )

        if self.AI_PROVIDER not in PROVIDER_URLS:
            raise ValueError(
                f"Pick 'OPENROUTER' or 'SARVAM' for AI_PROVIDER. Found {self.AI_PROVIDER} instead."
            )

        self.AI_PROVIDER_API_KEY = self._required(f"{self.AI_PROVIDER}_API_KEY")
        self.AI_PROVIDER_LLM_URL = OS.getenv(
            f"{self.AI_PROVIDER}_URL", PROVIDER_URLS[self.AI_PROVIDER]
        )

        # optional settings with defaults
        self.MODEL_NAME = OS.getenv("MODEL_NAME", "z-ai/glm-4.5-air:free")
        self.MAX_TOKENS = int(OS.getenv("MAX_TOKENS", 1000))
        self.MAX_CONTEXT_TOKENS = int(OS.getenv("MAX_CONTEXT_TOKENS", 131000))
        self.TEMPERATURE = float(OS.getenv("TEMPERATURE", 0.7))

        self.TINYFISH_SEARCH_URL = OS.getenv(
            "TINYFISH_SEARCH_URL", "https://api.search.tinyfish.ai"
        )
        self.TINYFISH_FETCH_URL = OS.getenv(
            "TINYFISH_FETCH_URL", "https://api.fetch.tinyfish.ai"
        )
        self.COMPACTION_THRESHOLD = float(OS.getenv("COMPACTION_THRESHOLD", 0.9))
        self.COMPACTION_RECENT_N = int(OS.getenv("COMPACTION_RECENT_N", 5))
        self.MAX_TOOL_CALLS = int(OS.getenv("MAX_TOOL_CALLS", 5))
        self.MESSAGE_DEBOUNCE_DELAY = float(OS.getenv("MESSAGE_DEBOUNCE_DELAY", 1.0))
        self.MESSAGE_DEBOUNCE_JITTER = float(OS.getenv("MESSAGE_DEBOUNCE_JITTER", 0.3))
        self.MESSAGE_DEBOUNCE_MAX_DELAY = float(
            OS.getenv("MESSAGE_DEBOUNCE_MAX_DELAY", 2.0)
        )

        # Paths
        self.WORKSPACE_DIR = (
            Path(OS.getenv("WORKSPACE_DIR", Path.home())) / ".workspace"
        )

        self.print_config()

    def _required(self, key: str, error_message: str | None = None) -> str:
        value = OS.getenv(key)
        if value is None:
            raise ValueError(
                error_message
                if error_message
                else f"Environment variable '{key}' is required but not set.",
            )
        return value

    def print_config(self):
        console.print(
            Markdown(f"""
| Key | Value |
|------|-----|
| Provider | {self.AI_PROVIDER} |
| Provider LLM Url | {self.AI_PROVIDER_LLM_URL} |
| Model Name | {self.MODEL_NAME} |
| Max Tokens | {self.MAX_TOKENS} |
| Context Window Size | {self.MAX_CONTEXT_TOKENS} |
| Workspace Dir | {self.WORKSPACE_DIR} |
| Compaction Threshold | {round(self.COMPACTION_THRESHOLD * 100)}% |
| Message Delay | [{self.MESSAGE_DEBOUNCE_DELAY}s, {self.MESSAGE_DEBOUNCE_MAX_DELAY}s] (step={self.MESSAGE_DEBOUNCE_JITTER}s)  |
| Telegram Allowlist | {", ".join(self.TELEGRAM_ALLOWLIST_CLEANED)} |
""")
        )

    def _sanitize_telegram_allowlist(self, allowlist: str):
        cleaned_list = allowlist.strip().replace(" ", "")
        if not ALLOWLIST_PATTERN.match(cleaned_list):
            raise ValueError(
                f"Invalid TELEGRAM_ALLOWLIST format. Expected: @john_doe,@jane_doe. Got: {cleaned_list}"
            )
        return cleaned_list.split(",")


CONFIG = ConfigManager()
