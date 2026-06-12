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


async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if update.message:
        await update.message.reply_text(f"Hi {user}! I'm seline.")


async def consumption(update: Update, context: ContextTypes.DEFAULT_TYPE):
    consumption = LLMContext.get_consumption()
    message = (
        f"Current token usage: {consumption['current_tokens_in_words']} / "
        f"{consumption['max_tokens_in_words']} tokens "
        f"({consumption['percentage_used']}% used, "
        f"{consumption['remaining_tokens_in_words']} tokens remaining)"
    )
    if update.message:
        await update.message.reply_text(message, parse_mode=ParseMode.MARKDOWN)


async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if not update.message or not update.message.text:
        return

    GATEWAY_STATUS.set_update(update)
    await GATEWAY_STATUS.start()

    user_message = update.message.text
    log.info(f"Received message: {user_message}")

    response = await complete(user_message)

    if response:
        await GATEWAY_STATUS.stop()
        await update.message.reply_text(response, parse_mode=ParseMode.MARKDOWN)


def telegram_loop():
    app = ApplicationBuilder().token(CONFIG.TELEGRAM_BOT_TOKEN).build()

    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("consumption", consumption))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))

    log.info("Seline is up.")
    app.run_polling()
