from pathlib import Path

_PROMPTS_DIR = Path(__file__).parent / "prompts"


def get_system_prompt():
    with open(_PROMPTS_DIR / "system.md", "r") as file:
        return file.read()
    return "You are a helpful assistant who always speak briefly."


def get_compaction_prompt():
    with open(_PROMPTS_DIR / "compaction.md", "r") as file:
        return file.read()
    return "Please summarize the following conversation:"
