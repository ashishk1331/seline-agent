import os as OS
from pathlib import Path

class ConfigManager:
    def __init__(self):
        # Non-optional settings
        self.AI_PROVIDER = self._required("AI_PROVIDER")
        self.TINYFISH_API_KEY = self._required("TINYFISH_API_KEY")
        self.TELEGRAM_BOT_TOKEN = self._required("TELEGRAM_BOT_TOKEN")

        if self.AI_PROVIDER == "OPENROUTER":
            self.OPENROUTER_API_KEY = self._required("OPENROUTER_API_KEY")
            self.OPENROUTER_URL = OS.getenv(
                "OPENROUTER_URL", "https://openrouter.ai/api/v1/chat/completions"
            )

            self.AI_PROVIDER_API_KEY = self.OPENROUTER_API_KEY
            self.AI_PROVIDER_LLM_URL = self.OPENROUTER_URL

        elif self.AI_PROVIDER == "SARVAM":
            self.SARVAM_API_KEY = self._required("SARVAM_API_KEY")
            self.SARVAM_URL = OS.getenv(
                "SARVAM_URL", "https://api.sarvam.ai/v1/chat/completions"
            )

            self.AI_PROVIDER_API_KEY = self.SARVAM_API_KEY
            self.AI_PROVIDER_LLM_URL = self.SARVAM_URL
        else:
            raise ValueError(
                f"Pick 'OPENROUTER' or 'SARVAM' for AI_PROVIDER. Found {self.AI_PROVIDER} instead."
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

    def _required(self, key: str) -> str:
        value = OS.getenv(key)
        if value is None:
            raise ValueError(f"Environment variable '{key}' is required but not set.")
        return value


CONFIG = ConfigManager()
