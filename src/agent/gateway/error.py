from telegram import Update
from telegram.error import TelegramError
import httpx
from ..logger import log


async def _notify(update: Update, error_message: str) -> None:
    try:
        if update.message:
            await update.message.reply_text(f"[⚠️ Error] {error_message}")
    except TelegramError:
        log.error(f"Unable to notify user about the error: {error_message}")


async def handle_error(update: Update, err: Exception) -> None:
    match err:
        case httpx.ReadTimeout():
            log.error("LLM read timeout.")
            await _notify(update, "Timed out waiting on model. Try again")

        case httpx.ConnectTimeout():
            log.warning("LLM connect timeout")
            await _notify(
                update, "Couldn't reach the model server. Check your connection."
            )

        case httpx.HTTPStatusError() if err.response.status_code == 429:
            log.warning("Rate limited by provider")
            await _notify(update, "Rate limited. Wait a moment and try again.")

        case httpx.HTTPStatusError():
            log.error(f"HTTP {err.response.status_code} from provider")
            await _notify(
                update, f"Provider returned an error ({err.response.status_code})."
            )

        case httpx.HTTPError():
            log.error(f"HTTP error: {err}")
            await _notify(update, "Network error. Please try again.")

        case _:
            log.exception("Unhandled error in agent pipeline", exc_info=err)
            await _notify(update, "Something went wrong. Please try again.")
