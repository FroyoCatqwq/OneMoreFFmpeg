"""Real-network e2e for provider/release flows.

Gated behind OMF_E2E_NETWORK=1 so CI is not flaky when GitHub/ffmpeg.org are
unreachable. When enabled, these hit the real ffmpeg.org releases page and the
real BtbN GitHub release API (no local HTTP stubs).

Note: https://www.gyan.dev/ffmpeg/builds/ serves .7z archives, which this tool
does not support (zip/tar.xz/tar.gz/tar.zst only), so it is not used as a
custom-provider download target. Tests use the built-in BtbN provider instead.
"""
from __future__ import annotations

import os

import pytest

pytestmark = pytest.mark.e2e
NETWORK = pytest.mark.skipif(
    os.environ.get("OMF_E2E_NETWORK") != "1",
    reason="set OMF_E2E_NETWORK=1 to run real-network tests",
)


@NETWORK
def test_real_release_list_stable(cli):
    r = cli.run("release", "list", "--channel", "stable", "--count", "3", "--json", timeout=90)
    assert r.returncode == 0, r.stdout
    data = r.data()
    assert data["schema"] == "onemoreffmpeg.release-catalog/v1"
    assert data["source"] == "https://ffmpeg.org/download.html#releases"
    assert len(data["entries"]) >= 1
    for e in data["entries"]:
        assert e["channel"] == "stable"
        assert e["asset_url"].startswith("https://github.com/")


@NETWORK
def test_real_resolve_btbn_daily(cli):
    r = cli.run("resolve", "--provider", "btbn", "--channel", "daily",
                "--target", "win64", "--json", timeout=90)
    assert r.returncode == 0, r.stdout
    data = r.data()
    assert data["schema"] == "onemoreffmpeg.release/v1"
    assert data["provider"] == "btbn"
    assert len(data["asset_sha256"]) == 64, "real asset digest must be a full sha256"
    assert data["asset_name"].endswith((".zip", ".tar.xz", ".tar.gz", ".tar.zst"))
    assert data["asset_url"].startswith("https://github.com/FFmpeg-Builds/")
    # A real release carries an independent checksums source.
    if data.get("checksums_url"):
        assert data["checksums_url"].startswith("https://")
        assert len(data["checksums_sha256"]) == 64


@NETWORK
def test_real_resolve_rejects_unknown_target(cli):
    r = cli.run("resolve", "--provider", "btbn", "--target", "nonsense",
                "--json", timeout=60)
    assert r.returncode != 0
    body = r.json()
    assert body["schema"] == "onemoreffmpeg.error/v1"
