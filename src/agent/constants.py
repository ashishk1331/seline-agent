from .config import CONFIG

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
