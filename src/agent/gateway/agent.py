from telegram import Update
from telegram.constants import ParseMode
from telegram.ext import (
    ApplicationBuilder,
    CommandHandler,
    ContextTypes,
    MessageHandler,
    filters,
)
from ..config import CONFIG
from ..llm import complete
from ..logger import log
from .status import GATEWAY_STATUS
from ..llm import context as LLMContext
from .debounce import DEBOUNCER
from asyncio import CancelledError


async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if update.message:
        await update.message.reply_text(f"Hi {user}! I'm seline.")


async def consumption(update: Update, context: ContextTypes.DEFAULT_TYPE):
    stats = LLMContext.get_consumption()
    message = (
        f"Current token usage: {stats['current_tokens_in_words']} / "
        f"{stats['max_tokens_in_words']} tokens "
        f"({stats['percentage_used']}% used, "
        f"{stats['remaining_tokens_in_words']} tokens remaining)"
    )
    if update.message:
        await update.message.reply_text(message, parse_mode=ParseMode.MARKDOWN)


async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if not update.message or not update.message.text:
        return

    if update.effective_user:
        sender = update.effective_user
        log.info(f"{sender.name} send: {update.message.text}")
    else:
        log.info(f"Recieved: {update.message.text}")

    DEBOUNCER.add(update, _process)


async def _process(update: Update, text: str):
    try:
        GATEWAY_STATUS.set_update(update)
        await GATEWAY_STATUS.start()

        response = await complete(text)

        if response:
            await GATEWAY_STATUS.stop()
            if update.message:
                await update.message.reply_text(response, parse_mode=ParseMode.MARKDOWN)
    except CancelledError:
        await GATEWAY_STATUS.stop()
        raise


COMMANDS = [
    ("start", "Start the bot"),
    ("consumption", "Check token consumption"),
]


async def post_init(app):
    await app.bot.set_my_commands(COMMANDS)

    log.info("Bot commands updated.")
    for cmd, desc in COMMANDS:
        log.info(f"/{cmd} = {desc}")


def telegram_loop():
    app = ApplicationBuilder().token(CONFIG.TELEGRAM_BOT_TOKEN).build()
    app.post_init = post_init

    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("consumption", consumption))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))

    log.info("Seline is up.")
    app.run_polling()
