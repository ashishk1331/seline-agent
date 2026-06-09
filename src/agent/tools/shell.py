from ..tools.registry import register_tool
import subprocess as SP


@register_tool()
def run_command(command):
    """Run a shell command and return its output.

    Args:
        command (str): The shell command to run.
    """
    result = SP.run(command, shell=True, capture_output=True, text=True, timeout=30)

    if result.returncode != 0:
        print(
            f"[TOOL] [ERROR] Command failed with return code {result.returncode}: {result.stderr}"
        )
    else:
        print(f'[TOOL] [RESULT_RAN] "{command}" = \n{result.stdout}')
    return result.stdout or result.stderr
