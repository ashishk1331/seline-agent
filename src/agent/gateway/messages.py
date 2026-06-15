import httpx
from ..config import CONFIG


async def _send_rich_text_message(
    chat_id: int, markdown: str, reply_to: int | None = None
):
    payload = {
        "chat_id": chat_id,
        "rich_message": {
            "markdown": markdown,
        },
    }
    if reply_to:
        payload["reply_parameters"] = {"message_id": reply_to}

    async with httpx.AsyncClient() as client:
        resp = await client.post(
            f"https://api.telegram.org/bot{CONFIG.TELEGRAM_BOT_TOKEN}/sendRichMessage",
            json=payload,
        )

    if resp.status_code != 200:
        return None

    return resp.json()
