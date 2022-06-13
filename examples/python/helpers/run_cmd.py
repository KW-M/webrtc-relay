import shlex, subprocess, os


def run_cmd_string(cmd_string, show_stdout=True, show_stderr=True):
    args = shlex.split(cmd_string)
    print("PYTHON: Running command: " + str(args))
    return subprocess.Popen(args,
                            stdout=None if show_stdout else subprocess.DEVNULL,
                            stderr=None if show_stderr else subprocess.DEVNULL)
