#!/usr/bin/env python3
"""Deploy and run the OneMoreFFmpeg Linux e2e suite on the configured server.

Connection details come from the file pointed to by OMF_LINUX_SERVER_INFO
(default: d:/cb-workpath/Linux-Server-Info.txt). The file is never copied to
the remote host and credentials are never printed.
"""
from __future__ import annotations

import argparse
import os
import posixpath
import stat
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_INFO = Path("d:/cb-workpath/Linux-Server-Info.txt")
REMOTE_DIR = "~/omf-e2e"
IGNORED_DIRS = {".git", ".venv", "dist", "__pycache__", "results", ".pytest_cache", ".mypy_cache"}
IGNORED_SUFFIXES = {".pyc", ".log", ".html", ".xml"}


def connection_info(path: Path) -> tuple[str, str, str]:
    text = path.read_text(encoding="utf-8", errors="replace")
    user_host = None
    password = None
    for line in text.splitlines():
        line = line.strip()
        if line.startswith("ssh "):
            user_host = line[4:].strip()
        elif line.lower().startswith("password:"):
            password = line.split(":", 1)[1].strip()
    if not user_host or not password:
        raise SystemExit(f"cannot parse ssh/password from {path}")
    user, host = user_host.split("@", 1)
    return host, user, password


def connect(info: Path):
    host, user, password = connection_info(info)
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, username=user, password=password, timeout=30)
    return client


def run(client, command: str, check: bool = True) -> tuple[int, str, str]:
    stdin, stdout, stderr = client.exec_command(command, timeout=900)
    # Combine stderr into stdout; reading stdout then stderr can deadlock when
    # either channel fills its SSH window.
    stdout.channel.set_combine_stderr(True)
    out = stdout.read().decode("utf-8", "replace")
    code = stdout.channel.recv_exit_status()
    print(f"$ {command}\n{out}", end="")
    if check and code != 0:
        raise SystemExit(f"remote command failed ({code}): {command}")
    return code, out, ""


def should_upload(relative: Path) -> bool:
    if any(part in IGNORED_DIRS for part in relative.parts):
        return False
    if relative.suffix in IGNORED_SUFFIXES:
        return False
    return True


def deploy(client, local_binary: Path | None):
    sftp = client.open_sftp()
    remote_root = posixpath.join("/tmp", "omf-e2e")
    run(client, f"rm -rf {remote_root} && mkdir -p {remote_root}")

    for source in ROOT.rglob("*"):
        if not source.is_file():
            continue
        relative = source.relative_to(ROOT)
        if not should_upload(relative):
            continue
        target = posixpath.join(remote_root, relative.as_posix())
        parent = posixpath.dirname(target)
        run_sftp_mkdir(sftp, parent)
        sftp.put(str(source), target)

    if local_binary:
        target = posixpath.join(remote_root, "dist", local_binary.name)
        run_sftp_mkdir(sftp, posixpath.dirname(target))
        sftp.put(str(local_binary), target)
        sftp.chmod(target, 0o755)
    sftp.close()
    return remote_root


def run_sftp_mkdir(sftp, path: str):
    parts = path.strip("/").split("/")
    current = ""
    for part in parts:
        current += "/" + part
        try:
            sftp.stat(current)
        except IOError:
            sftp.mkdir(current)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["check", "deploy", "e2e", "all"])
    parser.add_argument("--info", default=os.environ.get("OMF_LINUX_SERVER_INFO", str(DEFAULT_INFO)))
    parser.add_argument("--binary", default=str(ROOT / "dist" / "one-more-ffmpeg-linux-amd64"))
    args = parser.parse_args()
    info = Path(args.info)
    client = connect(info)
    try:
        if args.action == "check":
            run(client, "uname -a; ls ~/.local/bin 2>/dev/null | head -20; "
                "ls ~/.local/share/mise/shims 2>/dev/null | head -20; "
                "export PATH=$HOME/.local/bin:$HOME/.local/share/mise/shims:$PATH; "
                "which python uv go bats mise 2>/dev/null || true; "
                "uv --version 2>/dev/null || true; python --version 2>/dev/null || true; "
                "go version 2>/dev/null || true")
            return
        remote_root = deploy(client, Path(args.binary) if args.binary else None)
        if args.action in ("e2e", "all"):
            binary = posixpath.join(remote_root, "dist", Path(args.binary).name)
            env = ("export PATH=$HOME/.local/bin:$HOME/.local/share/mise/shims:$PATH; "
                   f"export OMF_BINARY={binary}; cd {remote_root}; "
                   "mise trust mise.toml >/dev/null 2>&1 || true; ")
            run(client, env + "go test -count=1 ./...")
            uv = "uv run --python 3.15 --with-requirements tests/e2e/requirements.txt"
            run(client, env + uv + " pytest -c tests/e2e/pytest.ini -n 2 -q tests/e2e/test_cli_e2e.py")
            run(client, env + uv + " robot --outputdir tests/e2e/results/robot tests/e2e/robot")
            run(client, env + "rm -rf /tmp/bats-core && git clone --depth 1 "
                "https://github.com/bats-core/bats-core.git /tmp/bats-core >/dev/null 2>&1 && "
                "/tmp/bats-core/bin/bats tests/e2e/bats")
        print("linux e2e completed")
    finally:
        client.close()


if __name__ == "__main__":
    main()
