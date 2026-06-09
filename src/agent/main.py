from dotenv import load_dotenv
load_dotenv()

from .telegram_agent import telegram_loop


def main():
    telegram_loop()


if __name__ == "__main__":
    main()
