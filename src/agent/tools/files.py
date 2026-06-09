from ..tools.registry import register_tool


@register_tool()
def read_file(path):
    """Read the contents of a file.

    Args:
        path (str): The path to the file.
    """
    with open(path, "r") as file:
        return file.read()


@register_tool()
def write_file(path, content):
    """Write content to a file, replacing any existing content.

    Args:
        path (str): The path to the file.
        content (str): The content to write to the file.
    """
    with open(path, "w") as file:
        file.write(content)
        return f"{path} written."
