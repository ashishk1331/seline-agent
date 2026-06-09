import json as J
import requests as R
from .tools import TOOLS_MAP, TOOLS
from rich.console import Console
from rich.markdown import Markdown
from .constants import HEADERS, BASIC_PAYLOAD
from .config import CONFIG
from .context import ContextManager
from .api import fetch

console = Console()
context = ContextManager()


def complete(message, max_tool_calls=CONFIG.MAX_TOOL_CALLS):

    if max_tool_calls <= 0:
        print("[ERROR] Maximum tool call limit reached.")
        return

    if message is not None:
        context.append({"role": "user", "content": message})

    data = fetch(
        CONFIG.OPENROUTER_URL,
        headers=HEADERS,
        payload=BASIC_PAYLOAD | {"messages": context.get_context(), "tools": TOOLS},
    )

    if not data:
        print("[ERROR] No response from API.")
        return

    message = data["choices"][0]["message"]
    usage = data["usage"]

    if message.get("tool_calls"):
        context.append(message, usage)
        for tool_call in message["tool_calls"]:
            name = tool_call["function"]["name"]
            args = J.loads(tool_call["function"]["arguments"])

            result = TOOLS_MAP[name](**args)

            context.append(
                {
                    "role": "tool",
                    "tool_call_id": tool_call["id"],
                    "content": str(result),
                }
            )
        complete(None, max_tool_calls - 1)
    else:
        context.append({"role": "assistant", "content": message["content"]}, usage)
        console.print(Markdown(message["content"]))
    
    debug_context()


def debug_context():
    print(
        f"---\n[CONTEXT STARTS]\n\n{J.dumps(context.get_context(), indent=2)}\n\n[CONTEXT ENDS]\n---"
    )
