from docstring_parser import parse
from string import Template
from ..logger import log
from ..gateway.status import GATEWAY_STATUS

TOOLS = []
TOOLS_MAP = {}
TYPE_MAP = {
    "str": "string",
    "int": "integer",
    "float": "number",
    "bool": "boolean",
}
INPUT_TYPES = {}


def register_tool(max_chars=1000, mask_after_use=False, status_message=None):
    def decorator(func):
        """A decorator to mark a function as a tool."""
        func.is_tool = True
        func.doc = parse(func.__doc__)
        INPUT_TYPES[func.__name__] = {
            param.arg_name: param.type_name for param in func.doc.params
        }
        TOOLS.append(
            {
                "type": "function",
                "function": {
                    "name": func.__name__,
                    "description": func.doc.short_description,
                    "parameters": {
                        "type": "object",
                        "properties": {
                            param.arg_name: {
                                "type": TYPE_MAP.get(param.type_name, "string")
                                if param.type_name
                                else "string",
                                "description": param.description,
                            }
                            for param in func.doc.params
                        },
                        "required": [param.arg_name for param in func.doc.params],
                    },
                },
            }
        )

        async def wrapper(*args, **kwargs):
            log.info(f"[bold cyan]{func.__name__}[/] args={args} kwargs={kwargs}")
            if status_message:
                await GATEWAY_STATUS.update(
                    Template(status_message).safe_substitute(**kwargs)
                )
            result = str(func(*args, **kwargs))
            if max_chars > -1 and len(result) > max_chars:
                result = result[:max_chars] + "... [truncated]"
            return result

        wrapper._max_chars = max_chars
        wrapper._mask_after_use = mask_after_use
        TOOLS_MAP[func.__name__] = wrapper
        return wrapper

    return decorator
