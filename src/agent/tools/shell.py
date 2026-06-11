from ..tools.registry import register_tool
from ..logger import log
import subprocess as SP


@register_tool(status_message="Running $command")
def run_command(command):
    """Run a shell command and return its output.

    Args:
        command (str): The shell command to run.
    """
    result = SP.run(command, shell=True, capture_output=True, text=True, timeout=30)

    if result.returncode != 0:
        log.error(
            f"Command failed with return code {result.returncode}: {result.stderr}"
        )
    else:
        log.info(f"Command succeeded: {command}")
    return result.stdout or result.stderr
