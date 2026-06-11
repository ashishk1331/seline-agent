from telegram import Update
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


async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if update.message:
        await update.message.reply_text(f"Hi {user}! I'm seline.")


async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if not update.message or not update.message.text:
        return

    placeholder = await update.message.reply_text("thinking...")

    user_message = update.message.text
    log.info(f"Received message: {user_message}")

    response = await complete(user_message)

    if response:
        await placeholder.delete()
        await update.message.reply_text(response)


def telegram_loop():
    app = ApplicationBuilder().token(CONFIG.TELEGRAM_BOT_TOKEN).build()

    app.add_handler(CommandHandler("start", start))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))

    log.info("Seline is up.")
    app.run_polling()
