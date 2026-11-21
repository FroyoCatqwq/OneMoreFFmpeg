"""e2e for concurrent read-only CLI commands (offline).

Read commands (version/detect/capabilities/doctor/provider.list) must be safe
to run concurrently: no shared cache corruption, no JSON crosstalk, no lock
contention. The conftest arun helper is used here (previously unused).
"""
from __future__ import annotations

import asyncio

import pytest

pytestmark = pytest.mark.e2e


@pytest.mark.asyncio
async def test_parallel_read_commands_are_isolated(cli, fake_trio):
    commands = [
        ("version", "--json"),
        ("doctor", "--json"),
        ("provider", "list", "--json"),
        ("detect", "--json"),
        ("capabilities", "--dir", str(fake_trio), "--json"),
    ]
    results = await asyncio.gather(*(cli.arun(*args) for args in commands))
    expected_commands = {"version", "doctor", "provider.list", "detect", "capabilities"}
    seen = set()
    for args, r in zip(commands, results):
        assert r.returncode == 0, (args, r.stdout, r.stderr)
        payload = r.json()
        assert payload["schema"] == "onemoreffmpeg.result/v1"
        assert payload["ok"] is True
        assert payload["command"] in expected_commands
        seen.add(payload["command"])
    assert seen == expected_commands


@pytest.mark.asyncio
async def test_repeated_capabilities_cache_is_concurrency_safe(cli, fake_trio):
    args = ("capabilities", "--dir", str(fake_trio), "--json")
    results = await asyncio.gather(*(cli.arun(*args) for _ in range(4)))
    for r in results:
        assert r.returncode == 0, (r.stdout, r.stderr)
        assert r.data()["schema"] == "onemoreffmpeg.capabilities/v1"
