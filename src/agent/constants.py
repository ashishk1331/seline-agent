from .config import CONFIG

HEADERS = {
    "Authorization": f"Bearer {CONFIG.OPENROUTER_API_KEY}",
    "Content-Type": "application/json",
}

BASIC_PAYLOAD = {
    "model": CONFIG.MODEL_NAME,
    "max_tokens": CONFIG.MAX_TOKENS,
    "temperature": CONFIG.TEMPERATURE,
}

COMPACTION_PAYLOAD = {
    "model": CONFIG.AUXILIARY_MODEL_NAME,
    "max_tokens": CONFIG.AUXILIARY_MAX_TOKENS,
    "temperature": CONFIG.TEMPERATURE,
}

TINYFISH_HEADERS = {
    "X-API-Key": str(CONFIG.TINYFISH_API_KEY),
    "Content-Type": "application/json",
}

THINKING_PHRASES = [
    "soch rahi hoon...",
    "ek second...",
    "haan haan, dekh rahi hoon...",
    "abhi batati hoon...",
    "hmm...",
    "thoda socha jaaye...",
    "ek minute...",
    "haan, samjhi...",
    "dekhte hain...",
]