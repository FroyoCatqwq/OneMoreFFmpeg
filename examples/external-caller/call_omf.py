#!/usr/bin/env python3
"""External-caller example: consume OneMoreFFmpeg's CLI JSON contract.

This is the pattern a downstream project (e.g. LiveArchiveEncoder /
VideoDamage) uses to locate FFmpeg and query its capabilities. It only needs
Python + the published `one-more-ffmpeg` binary:

  * no import of OneMoreFFmpeg Go packages,
  * no linking against the Go runtime,
  * all machine-readable results arrive on stdout as a v1 JSON envelope.

Usage:
  OMF_BINARY=/path/to/one-more-ffmpeg python call_omf.py
  python call_omf.py /path/to/one-more-ffmpeg
"""
from __future__ import annotations

import json
import os
import subprocess
import sys

RESULT_SCHEMA = "onemoreffmpeg.result/v1"
ERROR_SCHEMA = "onemoreffmpeg.error/v1"


class OmfError(RuntimeError):
    def __init__(self, code: str, message: str, exit_code: int):
        super().__init__(f"{code}: {message}")
        self.code = code
        self.exit_code = exit_code


def call(binary: str, *args: str) -> dict:
    """Run a command and return its data object, raising on the error envelope.

    stdout is reserved for JSON; diagnostics go to stderr. The process exit
    code is decided solely by the error code (0 ok, 2 usage, 1 runtime, 130
    cancelled), so callers can branch on `code` rather than parsing text.
    """
    proc = subprocess.run(
        [binary, *args, "--json"],
        capture_output=True, text=True, timeout=120,
    )
    try:
        envelope = json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"non-JSON stdout (exit {proc.returncode}): {proc.stderr}") from exc

    if envelope.get("schema") == ERROR_SCHEMA or not envelope.get("ok", False):
        raise OmfError(envelope.get("code", "OMF-E9003"),
                       envelope.get("message", "unknown error"), proc.returncode)
    if envelope.get("schema") != RESULT_SCHEMA:
        raise RuntimeError(f"unexpected envelope schema: {envelope.get('schema')}")
    return envelope["data"]


def main() -> int:
    binary = (sys.argv[1] if len(sys.argv) > 1
              else os.environ.get("OMF_BINARY", "one-more-ffmpeg"))

    version = call(binary, "version")
    print(f"OneMoreFFmpeg {version['version']} ({version['os']}/{version['arch']})")

    # Health gate: doctor always exits 0 by default, so read `healthy`.
    report = call(binary, "doctor")
    print(f"healthy: {report['healthy']}")
    for check in report["checks"]:
        print(f"  - {check['name']}: {check['ok']} ({check['detail']})")

    # Discover an installation, then query a single capability list.
    detected = call(binary, "detect")
    installations = detected.get("installations", [])
    if not installations:
        print("no FFmpeg discovered; install one first (see docs/HOWTO.md)")
        return 0
    root = installations[0]["root_dir"]
    encoders = call(binary, "capabilities", "--dir", root, "--only", "encoders", "--no-cache")
    names = [entry["name"] for entry in encoders.get("entries", [])]
    print(f"encoders at {root}: {len(names)} (e.g. {names[:5]})")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except OmfError as err:
        print(f"CLI error {err.code} (exit {err.exit_code}): {err}", file=sys.stderr)
        sys.exit(err.exit_code)