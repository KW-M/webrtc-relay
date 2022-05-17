import shlex, subprocess, os


def run_cmd_string(cmd_string):
    args = shlex.split(cmd_string)
    print("PYTHON: Running command: " + str(args))
    return subprocess.Popen(args, stdout=None, stderr=None)
