from pathlib import Path
from ..config import CONFIG
from string import Template

_PROMPTS_DIR = Path(__file__).parent


class PromptBase:
    def __init__(self):
        pass

    def get_system_prompt(self) -> str:
        with open(_PROMPTS_DIR / "system.md", "r") as file:
            return file.read() + "\n\n" + self.get_environment_information()
        return "You are a helpful assistant who always speak briefly."

    def get_compaction_prompt(self) -> str:
        with open(_PROMPTS_DIR / "compaction.md", "r") as file:
            return file.read()
        return "Please summarize the following conversation:"

    def get_soul_prompt(self) -> str:
        with open(_PROMPTS_DIR / "soul.md", "r") as file:
            return file.read()
        return "Speak in short sentences."

    def get_environment_information(self) -> str:
        with open(_PROMPTS_DIR / "soul.md", "r") as file:
            mapping = {
                "AI_PROVIDER": CONFIG.AI_PROVIDER,
                "AI_PROVIDER_LLM_URL": CONFIG.AI_PROVIDER_LLM_URL,
                "MODEL_NAME": CONFIG.MODEL_NAME,
                "MAX_TOKENS": str(CONFIG.MAX_TOKENS),
                "MAX_CONTEXT_TOKENS": str(CONFIG.MAX_CONTEXT_TOKENS),
                "COMPACTION_THRESHOLD": f"{round(CONFIG.COMPACTION_THRESHOLD * 100)}%",
                "WORKSPACE_DIR": CONFIG.WORKSPACE_DIR,
                "DATE": CONFIG.DATE,
                "TIME": CONFIG.TIME,
                "PLATFORM": CONFIG.PLATFORM,
                "PYTHON_VERSION": CONFIG.PYTHON_VERSION,
                "TIMEZONE": CONFIG.TIMEZONE,
                "LOCALE": CONFIG.LOCALE,
                "MAX_TOOL_CALLS": str(CONFIG.MAX_TOOL_CALLS),
                # "TRANSPORT":            self.TRANSPORT,
                # "LOG_LEVEL":            self.LOG_LEVEL,
                # "VERSION":              self.VERSION,
            }
            return Template(file.read()).safe_substitute(mapping)
        return ""


PROMPT_BASE = PromptBase()
