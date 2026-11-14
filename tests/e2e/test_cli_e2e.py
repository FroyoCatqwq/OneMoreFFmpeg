from __future__ import annotations

import asyncio
import hashlib
import http.server
import json
import threading
import zipfile
from pathlib import Path

import pytest

pytestmark = pytest.mark.e2e


def release_for(archive: Path) -> dict:
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    return {
        "schema": "onemoreffmpeg.release/v1",
        "provider": "btbn",
        "repository": "FFmpeg-Builds/FFmpeg-Builds",
        "channel": "stable",
        "release_tag": "latest",
        "release_id": "441698707",
        "published_at": "2026-11-11T13:00:32Z",
        "asset_name": archive.name,
        "asset_url": "https://example.invalid/" + archive.name,
        "asset_sha256": digest,
        "variant": "gpl",
        "linkage": "static",
        "target": "win64" if __import__("os").name == "nt" else "linux64",
        "license_hint": "gpl",
        "redistributable": True,
    }


def write_release(directory: Path, release: dict) -> Path:
    path = directory / "release.json"
    path.write_text(json.dumps(release), encoding="utf-8")
    return path


def test_version_and_envelope(cli):
    result = cli.run("version", "--json")
    assert result.returncode == 0
    data = result.data()
    assert data["schema"] == "onemoreffmpeg.version/v1"
    assert data["name"] == "OneMoreFFmpeg"
    assert data["os"] in ("windows", "linux")


def test_unknown_command_is_usage_error(cli):
    result = cli.run("definitely-not-a-command", "--json")
    assert result.returncode == 2
    assert result.json()["code"] == "OMF-E7001"


def test_provider_list_is_offline(cli):
    result = cli.run("provider", "list", "--json")
    assert result.returncode == 0
    providers = result.data()["providers"]
    assert {item["id"] for item in providers} >= {"btbn", "bnqdzj-nonfree"}


def test_provider_add_dry_run_and_list(cli):
    result = cli.run("provider", "add", "--id", "e2e-local",
                     "--url", "https://example.invalid/ffmpeg.zip",
                     "--sha256", "a" * 64, "--dry-run", "--json")
    assert result.returncode == 0
    assert result.data()["status"] == "dry-run"
    assert not (cli.data_root / "providers.toml").exists()


def test_env_status_and_doctor(cli):
    status = cli.run("env", "status", "--json")
    assert status.returncode == 0
    assert status.data()["schema"] == "onemoreffmpeg.env-status/v1"
    assert status.data()["current_type"] == "missing"

    report = cli.run("doctor", "--json")
    assert report.returncode == 0
    assert report.data()["schema"] == "onemoreffmpeg.doctor/v1"
    assert {item["name"] for item in report.data()["checks"]} >= {"current_exists", "no_pending_journal"}


def test_detect_validate_and_capabilities(cli, fake_trio: Path):
    detected = cli.run("detect", "--dir", str(fake_trio), "--no-path", "--no-env", "--no-common", "--json")
    assert detected.returncode == 0
    assert detected.data()["installations"]

    validated = cli.run("validate", "--dir", str(fake_trio), "--json")
    assert validated.returncode == 0
    assert validated.data()["status"] == "healthy"

    capabilities = cli.run("capabilities", "--dir", str(fake_trio), "--no-cache", "--json")
    assert capabilities.returncode == 0
    data = capabilities.data()
    assert data["schema"] == "onemoreffmpeg.capabilities/v1"
    assert len(data["commands"]) == 13
    assert data["decoders"]


def test_install_uninstall_and_env_dry_run(cli, fake_trio: Path, tmp_path: Path):
    archive = tmp_path / "ffmpeg-fake.zip"
    with zipfile.ZipFile(archive, "w") as writer:
        for executable in sorted(fake_trio.iterdir()):
            if executable.is_file() and not executable.name.endswith(".go"):
                writer.write(executable, "ffmpeg-build/bin/" + executable.name)
        writer.writestr("LICENSE.txt", "GPL-3.0-only\n")
    release = release_for(archive)
    release_path = write_release(tmp_path, release)

    preview = cli.run("install", "--release", str(release_path), "--json")
    assert preview.returncode == 0
    assert preview.data()["dry_run"] is True
    assert not Path(preview.data()["root_dir"]).exists()

    applied = cli.run("install", "--release", str(release_path),
                      "--artifact", str(archive), "--apply", "--emit-trio-lock", "--json")
    assert applied.returncode == 0, applied.stdout
    root = Path(applied.data()["root_dir"])
    assert (root / "install.json").exists()
    assert (root / "LICENSE.txt").exists()
    assert (root / "ffmpeg-trio.lock").exists()
    assert applied.data()["current_pointer"] in ("symlink", "directory_junction", "pointer_file")

    after = cli.run("env", "status", "--json")
    assert after.returncode == 0
    assert after.data()["current_type"] != "missing"

    dry = cli.run("env", "set", "--json")
    assert dry.returncode == 0
    assert dry.data()["dry_run"] is True
    assert dry.data()["applied"] is False

    removed = cli.run("uninstall", "--root", str(root), "--apply", "--json")
    assert removed.returncode == 0, removed.stdout
    assert removed.data()["applied"] is True
    assert not root.exists()

    journal_id = removed.data()["journal_id"]
    listed = cli.run("journal", "list", "--json")
    assert listed.returncode == 0
    assert journal_id in [item["id"] for item in listed.data()["records"]]

    rolled = cli.run("journal", "rollback", "--journal", journal_id, "--apply", "--json")
    assert rolled.returncode == 0, rolled.stdout
    assert rolled.data()["applied"] is True
    assert root.exists()


def test_invalid_release_is_rejected(cli, tmp_path: Path):
    dummy = tmp_path / "missing.zip"
    dummy.write_bytes(b"not a real archive")
    release = release_for(dummy)
    release["asset_sha256"] = "bad"
    path = write_release(tmp_path, release)
    result = cli.run("download", "--release", str(path), "--dir", str(tmp_path / "downloads"), "--json")
    assert result.returncode == 2
    assert result.json()["code"] == "OMF-E7003"


def test_capabilities_only_runs_one_kind(cli, fake_trio: Path):
    result = cli.run("capabilities", "--dir", str(fake_trio), "--only", "encoders", "--no-cache", "--json")
    assert result.returncode == 0, result.stdout
    data = result.data()
    assert data["schema"] == "onemoreffmpeg.capability-list/v1"
    assert data["kind"] == "encoders"
    assert data["entries"]
    assert "muxers" not in data


def test_in_detail_usage(cli):
    result = cli.run("--in-detail-usage")
    assert result.returncode == 0
    assert "OneMoreFFmpeg - detailed usage" in result.stdout
    assert "release list" in result.stdout
    assert "journal rollback" in result.stdout


def test_release_catalog_with_local_sources(cli):
    from datetime import datetime, timedelta, timezone

    now = datetime.now(timezone.utc)
    stable = {
        "id": 1, "tag_name": "latest", "published_at": now.isoformat(),
        "assets": [
            {"name": "ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
             "browser_download_url": "https://example.invalid/9.0.zip", "digest": "sha256:" + "a" * 64},
            {"name": "ffmpeg-n8.1-latest-win64-gpl-8.1.zip",
             "browser_download_url": "https://example.invalid/8.1.zip", "digest": "sha256:" + "b" * 64},
        ],
    }
    recent = (now - timedelta(days=1)).replace(microsecond=0).isoformat()
    old = (now - timedelta(days=30)).replace(microsecond=0).isoformat()
    daily = [
        {"id": 2, "tag_name": "autobuild-" + (now - timedelta(days=1)).strftime("%Y-%m-%d-%H-%M"),
         "published_at": recent,
         "assets": [{"name": "ffmpeg-N-128790-gabcdef-win64-gpl.zip",
                     "browser_download_url": "https://example.invalid/new.zip", "digest": "sha256:" + "c" * 64}]},
        {"id": 3, "tag_name": "autobuild-" + (now - timedelta(days=30)).strftime("%Y-%m-%d-%H-%M"),
         "published_at": old,
         "assets": [{"name": "ffmpeg-N-120000-gabcdef-win64-gpl.zip",
                     "browser_download_url": "https://example.invalid/old.zip", "digest": "sha256:" + "d" * 64}]},
    ]
    html = b'<a href="releases/ffmpeg-9.0.2.tar.xz">9.0.2</a>\n<a href="releases/ffmpeg-8.1.3.tar.xz">8.1.3</a>'

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            path = self.path.split("?", 1)[0]
            if path.endswith("/releases/latest"):
                payload = json.dumps(stable).encode()
            elif path.endswith("/releases"):
                payload = json.dumps(daily).encode()
            else:
                payload = html
            self.send_response(200)
            self.send_header("Content-Type", "application/json" if payload != html else "text/html")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def log_message(self, *args):
            pass

    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        port = server.server_address[1]
        (cli.data_root / "config.toml").write_text(
            "schema = 'onemoreffmpeg.config/v1'\n[network]\n"
            f"github_api = 'http://127.0.0.1:{port}'\n", encoding="utf-8")
        official = f"http://127.0.0.1:{port}/ffmpeg.html"

        stable_result = cli.run("release", "list", "--channel", "stable", "--target", "win64",
                                "--releases-url", official, "--json")
        assert stable_result.returncode == 0, stable_result.stdout
        versions = [entry["version"] for entry in stable_result.data()["entries"]]
        assert versions[:2] == ["9.0.2", "8.1.3"]

        daily_result = cli.run("release", "list", "--channel", "daily", "--target", "win64",
                               "--days", "7", "--count", "1", "--json")
        assert daily_result.returncode == 0, daily_result.stdout
        assert len(daily_result.data()["entries"]) == 1
        assert daily_result.data()["entries"][0]["version"].startswith("N-")
    finally:
        server.shutdown()
        thread.join(timeout=5)


@pytest.mark.asyncio
async def test_parallel_read_only_commands(cli):
    results = await asyncio.gather(
        cli.arun("version", "--json"),
        cli.arun("provider", "list", "--json"),
        cli.arun("env", "status", "--json"),
        cli.arun("doctor", "--json"),
    )
    assert all(result.returncode == 0 for result in results)
    assert all(result.json()["ok"] for result in results)


def test_doctor_default_unhealthy_exits_zero(cli):
    result = cli.run("doctor", "--json")
    assert result.returncode == 0
    assert result.data()["healthy"] is False


def test_doctor_strict_unhealthy_exits_nonzero(cli):
    result = cli.run("doctor", "--strict", "--json")
    assert result.returncode == 1
    body = result.json()
    assert body["code"] == "OMF-E6006"
    assert body["details"]["report"]["healthy"] is False


def test_env_system_consent_dry_run(cli):
    result = cli.run("env", "set", "--system", "--authorize-system", "--json")
    assert result.returncode == 0, result.stdout
    assert result.data()["dry_run"] is True
    assert result.data()["applied"] is False


def test_env_system_requires_authorization(cli):
    result = cli.run("env", "set", "--system", "--json")
    if result.returncode != 0:
        # Non-elevated processes must be refused with guidance.
        assert result.returncode == 2
        assert result.json()["code"] == "OMF-E6003"
    else:
        # Elevated (root/Administrator) processes are pre-authorized.
        assert result.data()["dry_run"] is True


def test_install_releases_transaction_lock(cli, fake_trio: Path, tmp_path: Path):
    archive = tmp_path / "ffmpeg-lock.zip"
    with zipfile.ZipFile(archive, "w") as writer:
        for executable in sorted(fake_trio.iterdir()):
            if executable.is_file() and not executable.name.endswith(".go"):
                writer.write(executable, "ffmpeg-build/bin/" + executable.name)
        writer.writestr("LICENSE.txt", "GPL-3.0-only\n")
    release_path = write_release(tmp_path, release_for(archive))

    applied = cli.run("install", "--release", str(release_path),
                      "--artifact", str(archive), "--apply", "--json")
    assert applied.returncode == 0, applied.stdout
    assert not (cli.data_root / ".transaction.lock").exists(), "lock must be released after install"
