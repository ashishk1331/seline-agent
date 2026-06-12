import json as J
from .tools import TOOLS_MAP, TOOLS
from rich.console import Console
from .constants import HEADERS, BASIC_PAYLOAD
from .config import CONFIG
from .context import ContextManager
from .api import fetch
from .logger import log

console = Console()
context = ContextManager()


async def complete(message, max_tool_calls=CONFIG.MAX_TOOL_CALLS):

    if max_tool_calls <= 0:
        log.error("Maximum tool call limit reached.")
        return

    if message is not None:
        await context.append({"role": "user", "content": message})

    data = await fetch(
        CONFIG.OPENROUTER_URL,
        headers=HEADERS,
        payload=BASIC_PAYLOAD | {"messages": context.get_context(), "tools": TOOLS},
    )

    if not data:
        log.error("No response from API.")
        return

    message = data["choices"][0]["message"]
    usage = data["usage"]

    if message.get("tool_calls"):
        await context.append(message, usage)
        for tool_call in message["tool_calls"]:
            name = tool_call["function"]["name"]
            args = J.loads(tool_call["function"]["arguments"])

            result = await TOOLS_MAP[name](**args)

            await context.append(
                {
                    "role": "tool",
                    "tool_call_id": tool_call["id"],
                    "content": str(result),
                }
            )
        return await complete(None, max_tool_calls - 1)
    else:
        await context.append(
            {"role": "assistant", "content": message["content"]}, usage
        )
        log.info(f"[bold green]Assistant[/]: {message['content']}")

    return message["content"]


def debug_context():
    log.info(
        f"---\n[CONTEXT STARTS]\n\n{J.dumps(context.get_context(), indent=2)}\n\n[CONTEXT ENDS]\n---"
    )
