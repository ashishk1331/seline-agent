from ..config import CONFIG
from telegram import Update
from asyncio import Task, create_task as CREATE_TASK, sleep, CancelledError


class Debouncer:
    def __init__(self):
        self._delay: float = CONFIG.MESSAGE_DEBOUNCE_DELAY
        self._jitter: float = CONFIG.MESSAGE_DEBOUNCE_JITTER
        self._max_delay: float = CONFIG.MESSAGE_DEBOUNCE_MAX_DELAY
        self._counter: int = 0

        self._texts: list[str] = []
        self._task: Task | None = None

    def add(self, update: Update, processor):
        if update.message and update.message.text:
            self._texts.append(update.message.text)

        if self._task and not self._task.done():
            self._task.cancel()
            self._counter += 1

        self._task = CREATE_TASK(self._dispatch(update, processor))

    def _calculate_delay(self):
        return min(self._delay + self._jitter * self._counter, self._max_delay)

    async def _dispatch(self, update: Update, processor):
        try:
            await sleep(self._calculate_delay())
            combined_text = "\n".join(self._texts)
            await processor(update, combined_text)
            self._texts.clear()
            self._counter = 0
        except CancelledError:
            raise


DEBOUNCER = Debouncer()
