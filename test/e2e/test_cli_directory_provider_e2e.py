"""e2e for custom-URL directory mode (plain static hosting, no version lock-in).

All tests use a local HTTP static server (stdlib http.server) and are fully
offline. They exercise the real CLI binary end-to-end:
    provider add --mode directory -> resolve -> download

The naming convention is {name}-{version}-{license}-{release_type}-{os}-{arch}-{linkage}.{format}
and checksum.sha256 is resolved in the artifact's own directory.
"""
from __future__ import annotations

import hashlib
import http.server
import socketserver
import threading
from pathlib import Path

import pytest

pytestmark = pytest.mark.e2e

ASSET = "ffmpeg-20261120-gpl-snapshot-linux-amd64-static.tar.xz"
ASSET_TS = "ffmpeg-20261120T153000Z-gpl-snapshot-linux-amd64-static.tar.xz"
PAYLOAD = b"fake-ffmpeg-archive-payload\n"


class _QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *args):
        pass


@pytest.fixture
def static_server(tmp_path):
    """Serve a directory over loopback HTTP. Returns (base_url, root)."""
    root = tmp_path / "static"
    root.mkdir()

    class ReuseTCPServer(socketserver.TCPServer):
        allow_reuse_address = True

    handler = lambda *a, **kw: _QuietHandler(*a, directory=str(root), **kw)
    server = ReuseTCPServer(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield f"http://127.0.0.1:{server.server_address[1]}", root
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def _write_snapshot(root: Path, version: str, asset: str = ASSET,
                    checksum_text: str | None = None) -> str:
    """Create snapshots/<version>/ with the asset and checksum.sha256.

    Returns the hex digest of the asset payload.
    """
    directory = root / "ffmpeg" / "snapshots" / version
    directory.mkdir(parents=True, exist_ok=True)
    (directory / asset).write_bytes(PAYLOAD)
    digest = hashlib.sha256(PAYLOAD).hexdigest()
    text = checksum_text if checksum_text is not None else f"{digest}  {asset}\n"
    (directory / "checksum.sha256").write_text(text, encoding="utf-8")
    return digest


def _add_directory_provider(cli, base_url: str, pid: str = "dirsrv",
                            license_hint: str = "gpl") -> None:
    r = cli.run("provider", "add", "--id", pid, "--mode", "directory",
                "--name", "ffmpeg", "--url", f"{base_url}/ffmpeg",
                "--license-hint", license_hint, "--apply", "--json")
    assert r.returncode == 0, r.stdout


def test_provider_add_directory_mode_apply_and_list(cli, static_server):
    base_url, root = static_server
    _add_directory_provider(cli, base_url, pid="dirlist")
    listed = cli.run("provider", "list", "--json")
    assert listed.returncode == 0, listed.stdout
    data = listed.data()
    assert any(p.get("id") == "dirlist" for p in data.get("providers", [])), data


def test_resolve_directory_latest_snapshot(cli, static_server):
    base_url, root = static_server
    (root / "ffmpeg" / "snapshots").mkdir(parents=True, exist_ok=True)
    (root / "ffmpeg" / "snapshots" / "latest.txt").write_text("20261120\n", encoding="utf-8")
    digest = _write_snapshot(root, "20261120")
    _add_directory_provider(cli, base_url, pid="dirlatest")

    r = cli.run("resolve", "--provider", "dirlatest", "--target", "linux64", "--json")
    assert r.returncode == 0, r.stdout
    data = r.data()
    assert data["schema"] == "onemoreffmpeg.release/v1"
    assert data["release_tag"] == "20261120"
    assert data["asset_name"] == ASSET
    assert data["asset_sha256"] == digest
    assert len(data["asset_sha256"]) == 64


def test_resolve_directory_release_line_timestamp_preserves_case(cli, static_server):
    base_url, root = static_server
    version = "20261120T153000Z"
    digest = _write_snapshot(root, version, asset=ASSET_TS)
    _add_directory_provider(cli, base_url, pid="dirts")

    r = cli.run("resolve", "--provider", "dirts", "--target", "linux64",
                "--release-line", version, "--json")
    assert r.returncode == 0, r.stdout
    data = r.data()
    assert data["release_tag"] == version, "T/Z must keep uppercase"
    assert data["asset_name"] == ASSET_TS, "asset name must keep uppercase T/Z"
    assert data["asset_sha256"] == digest


def test_resolve_directory_release_line_lowercase_tz_rejected(cli, static_server):
    base_url, root = static_server
    _write_snapshot(root, "20261120t153000z", asset=ASSET_TS)
    _add_directory_provider(cli, base_url, pid="dirlower")

    r = cli.run("resolve", "--provider", "dirlower", "--target", "linux64",
                "--release-line", "20261120t153000z", "--json")
    # Lowercase t/z is not a valid snapshot version per naming convention.
    assert r.returncode != 0
    assert r.json()["code"] in {"OMF-E4002", "OMF-E7003", "OMF-E3007"}


def test_resolve_directory_latest_missing_suggests_release_line(cli, static_server):
    base_url, _ = static_server
    _add_directory_provider(cli, base_url, pid="dirnolatest")
    r = cli.run("resolve", "--provider", "dirnolatest", "--target", "linux64", "--json")
    assert r.returncode != 0
    err = r.json()
    assert err["code"] in {"OMF-E3007", "OMF-E7003"}
    assert "release-line" in r.stdout


@pytest.mark.parametrize("checksum_text, expect_ok", [
    (None, True),  # valid default
    ("da39a3ee5e6b4b0d3255bfef95601890afd80709  other.tar.xz\n", False),  # missing entry
    ("{d}  {a}\n{d}  {a}\n", False),  # duplicate entry (formatted below)
    ("{d}  dist/{a}\n", False),  # path in checksum file
])
def test_resolve_directory_checksum_edges(cli, static_server, checksum_text, expect_ok):
    base_url, root = static_server
    (root / "ffmpeg" / "snapshots").mkdir(parents=True, exist_ok=True)
    (root / "ffmpeg" / "snapshots" / "latest.txt").write_text("20261120\n", encoding="utf-8")
    digest = hashlib.sha256(PAYLOAD).hexdigest()
    if checksum_text is not None:
        checksum_text = checksum_text.format(d=digest, a=ASSET)
    _write_snapshot(root, "20261120", checksum_text=checksum_text)
    _add_directory_provider(cli, base_url, pid="dirchecksum")

    r = cli.run("resolve", "--provider", "dirchecksum", "--target", "linux64", "--json")
    if expect_ok:
        assert r.returncode == 0, r.stdout
        assert r.data()["asset_sha256"] == digest
    else:
        assert r.returncode != 0
        assert r.json()["code"] in {"OMF-E4002", "OMF-E3007"}


def test_resolve_directory_nonfree_requires_consent(cli, static_server):
    base_url, root = static_server
    version = "20261120"
    asset = "ffmpeg-20261120-nonfree-snapshot-linux-amd64-static.tar.xz"
    _write_snapshot(root, version, asset=asset)
    # license_hint is authoritative in directory mode; set it to nonfree so
    # the consent guard is reached. Pass --release-line so the guard is hit
    # without depending on snapshots/latest.txt.
    _add_directory_provider(cli, base_url, pid="dirnonfree", license_hint="nonfree")

    no_consent = cli.run("resolve", "--provider", "dirnonfree", "--target", "linux64",
                         "--release-line", version, "--json")
    assert no_consent.returncode != 0
    assert no_consent.json()["code"] == "OMF-E7001"

    consented = cli.run("resolve", "--provider", "dirnonfree", "--target", "linux64",
                        "--release-line", version, "--allow-nonfree", "--json")
    assert consented.returncode == 0, consented.stdout
    data = consented.data()
    assert data["redistributable"] is False
    codes = [w.get("code") for w in data.get("warnings", [])]
    assert "OMF-W1006" in codes, data.get("warnings")


def test_directory_mode_resolve_download_roundtrip(cli, static_server):
    base_url, root = static_server
    (root / "ffmpeg" / "snapshots").mkdir(parents=True, exist_ok=True)
    (root / "ffmpeg" / "snapshots" / "latest.txt").write_text("20261120\n", encoding="utf-8")
    _write_snapshot(root, "20261120")
    _add_directory_provider(cli, base_url, pid="dirround")

    resolved = cli.run("resolve", "--provider", "dirround", "--target", "linux64", "--json")
    assert resolved.returncode == 0, resolved.stdout
    release = resolved.data()
    release_path = cli.data_root / "release.json"
    release_path.write_text(resolved.stdout, encoding="utf-8")

    downloaded = cli.run("download", "--release", str(release_path), "--json")
    assert downloaded.returncode == 0, downloaded.stdout
    data = downloaded.data()
    assert data["asset_sha256"] == release["asset_sha256"]
    assert not list(cli.data_root.rglob("*.part")), "no partial files after success"
