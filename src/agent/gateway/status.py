from telegram import Update, Message
from ..logger import log
from ..constants import THINKING_PHRASES
import random


class Status:
    def __init__(self):
        self._current_update: Update | None = None
        self._message: Message | None = None

    async def _update_message(self, message: str):
        if self._message:
            try:
                await self._message.edit_text(message)
            except Exception as e:
                log.error(f"Failed to update status message: {e}")

    def set_update(self, update: Update):
        self._current_update = update

    async def start(self):
        if self._current_update and self._current_update.message:
            await self.stop()
            self._message = await self._current_update.message.reply_text(
                random.choice(THINKING_PHRASES)
            )

    async def stop(self):
        if self._message:
            try:
                await self._message.delete()
            except Exception:
                log.error("Message not found to delete.")
            finally:
                self._message = None

    async def update(self, message: str):
        if self._message:
            await self._update_message(message)


GATEWAY_STATUS = Status()
