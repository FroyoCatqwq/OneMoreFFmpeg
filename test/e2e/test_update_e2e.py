"""e2e for the v1.1 self-update command.

The default `--check` path is exercised with a real GitHub API (marker-gated
so CI is not network-dependent); the atomic replacement itself is covered by
Go unit tests with a local HTTP server.
"""
from __future__ import annotations

import os

import pytest

pytestmark = pytest.mark.e2e


def test_update_help_is_plain_text(cli):
    r = cli.run("update", "--help")
    assert r.returncode == 0
    assert "self-update" in r.stdout.lower() or "update" in r.stdout.lower()


def test_update_json_help_is_refused(cli):
    r = cli.run("update", "--json", "--help")
    assert r.returncode == 1
    assert r.json()["code"] == "OMF-E1001"


@pytest.mark.parametrize("flags", [
    ("--apply", "--check"),
    ("--recover", "--apply"),
    ("--recover", "--check"),
])
def test_update_flags_mutually_exclusive(cli, flags):
    r = cli.run("update", *flags, "--json")
    assert r.returncode == 1
    assert r.json()["code"] == "OMF-E1001"


@pytest.mark.skipif(os.environ.get("OMF_E2E_NETWORK") != "1",
                    reason="set OMF_E2E_NETWORK=1 for real GitHub API check")
def test_update_check_real_github(cli):
    r = cli.run("update", "--json", timeout=90)
    assert r.returncode == 0, r.stdout
    data = r.data()
    assert data["schema"] == "onemoreffmpeg.update/v1"
    assert data["applied"] is False  # default dry-run never replaces the binary
    assert isinstance(data["update_available"], bool)
