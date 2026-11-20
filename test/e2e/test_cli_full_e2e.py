"""Phase 7 expanded e2e coverage.

These tests exercise every frozen result.command, assert the full JSON envelope
shape, verify capability parsing field-by-field against the fake FFmpeg, and
lock the "stdout is JSON-only" contract. They are intentionally black-box: all
interaction goes through the built CLI binary.
"""
from __future__ import annotations

import json
from pathlib import Path

import pytest

pytestmark = pytest.mark.e2e

RESULT_SCHEMA = "onemoreffmpeg.result/v1"
ERROR_SCHEMA = "onemoreffmpeg.error/v1"


def assert_result(result, command: str, schema: str) -> dict:
    """Every --json call must emit exactly one parseable result envelope."""
    assert result.returncode == 0, (result.stdout, result.stderr)
    body = result.json()
    assert body["schema"] == RESULT_SCHEMA
    assert body["ok"] is True
    assert body["command"] == command
    assert body["data"]["schema"] == schema
    assert set(body) == {"schema", "ok", "command", "data"}, body
    return body["data"]


def assert_error(result, code: str, exit_code: int) -> dict:
    assert result.returncode == exit_code, (result.stdout, result.stderr)
    body = result.json()
    assert body["schema"] == ERROR_SCHEMA
    assert body["ok"] is False
    assert body["code"] == code
    assert isinstance(body["message"], str) and body["message"]
    assert isinstance(body["details"], dict)
    return body


# --------------------------------------------------------------------------
# Envelope + JSON-only contract
# --------------------------------------------------------------------------

def test_all_help_under_json_is_refused_as_error_envelope(cli):
    for args in (
        ("--json",),                 # root, no subcommand
        ("provider", "--json"),      # parent group, no subcommand
        ("env", "--json"),
        ("journal", "--json"),
        ("--json", "--help"),
        ("version", "--json", "--help"),
    ):
        result = cli.run(*args)
        body = assert_error(result, "OMF-E1001", 1)
        assert "plain text" in body["message"]


def test_plain_help_never_emits_json(cli):
    result = cli.run("provider")
    assert result.returncode == 0
    body = result.stdout.strip()
    assert body and not body.startswith("{"), "plain help must be text"


def test_stderr_has_no_json_envelope(cli):
    result = cli.run("version", "--json")
    assert result.returncode == 0
    assert result.stderr.strip() == "", result.stderr
    assert result.stdout.count('"schema"') >= 2  # envelope + data schema


# --------------------------------------------------------------------------
# version / detect / validate — full field assertions
# --------------------------------------------------------------------------

def test_version_envelope_is_complete(cli):
    data = assert_result(cli.run("version", "--json"), "version", "onemoreffmpeg.version/v1")
    assert data["name"] == "OneMoreFFmpeg"
    assert data["version"]
    assert data["go_version"].startswith("go1.")
    assert data["os"] in ("windows", "linux")
    assert data["arch"] in ("amd64", "arm64")


def test_detect_field_shape(cli, fake_trio: Path):
    data = assert_result(
        cli.run("detect", "--dir", str(fake_trio), "--no-path", "--no-env", "--no-common", "--json"),
        "detect", "onemoreffmpeg.detect/v1",
    )
    assert data["installations"]
    inst = data["installations"][0]
    assert inst["id"] and inst["fingerprint"]
    assert inst["ffmpeg"] or inst["ffprobe"] or inst["ffplay"]
    assert isinstance(inst["sources"], list)
    assert inst["status"] in ("healthy", "degraded", "unknown")


def test_validate_missing_dir_is_error(cli, tmp_path: Path):
    result = cli.run("validate", "--dir", str(tmp_path / "does-not-exist"), "--json")
    assert result.returncode != 0
    body = result.json()
    assert body["schema"] == ERROR_SCHEMA
    assert body["code"] in {"OMF-E1001", "OMF-E3001", "OMF-E3002", "OMF-E3004", "OMF-E3008", "OMF-E7003"}


# --------------------------------------------------------------------------
# capabilities — field-by-field against the fake FFmpeg
# --------------------------------------------------------------------------

ALL_KINDS = [
    "muxers", "demuxers", "devices", "decoders", "encoders",
    "filters", "pixel_formats", "channel_layouts", "sample_formats",
]
EXPECTED_ENTRIES = {
    "decoders": "h264", "encoders": "h264",
    "muxers": "mp4", "demuxers": "mp4", "devices": "mp4",
    "filters": "scale", "pixel_formats": "yuv420p",
}


def test_capabilities_full_fields(cli, fake_trio: Path):
    data = assert_result(
        cli.run("capabilities", "--dir", str(fake_trio), "--no-cache", "--json"),
        "capabilities", "onemoreffmpeg.capabilities/v1",
    )
    assert len(data["commands"]) == 13
    for key in ("-L", "-license", "-version", "-muxers", "-decoders", "-filters", "-h full"):
        assert key in data["commands"], key
        assert data["commands"][key]["ok"] is True
        assert len(data["commands"][key]["stdout_sha256"]) == 64

    assert data["version"]["version"] == "9.9.9", data["version"]
    assert "GPL" in data["license"]["summary"]
    assert data["help"]["full"]["found"] is True

    for kind in ALL_KINDS:
        assert isinstance(data[kind], list), kind
    for kind, name in EXPECTED_ENTRIES.items():
        names = [entry["name"] for entry in data[kind]]
        assert name in names, (kind, names)
        assert all("raw" in entry for entry in data[kind])
    assert any(entry["name"] in ("mono", "stereo") for entry in data["channel_layouts"])
    assert {entry["name"] for entry in data["sample_formats"]} >= {"u8", "s16"}
    assert isinstance(data["warnings"], list)


@pytest.mark.parametrize("kind", ALL_KINDS)
def test_capabilities_only_each_kind(cli, fake_trio: Path, kind: str):
    data = assert_result(
        cli.run("capabilities", "--dir", str(fake_trio), "--only", kind, "--no-cache", "--json"),
        "capabilities", "onemoreffmpeg.capability-list/v1",
    )
    assert data["kind"] == kind
    assert data["entries"]
    assert "commands" not in data
    for entry in data["entries"]:
        assert entry["name"]
    if kind in EXPECTED_ENTRIES:
        assert EXPECTED_ENTRIES[kind] in [entry["name"] for entry in data["entries"]]


def test_capabilities_only_rejects_help_topic(cli, fake_trio: Path):
    result = cli.run("capabilities", "--dir", str(fake_trio), "--only", "encoders",
                     "--help-topic", "full", "--json")
    assert_error(result, "OMF-E1001", 1)


def test_capabilities_cache_then_refresh(cli, fake_trio: Path):
    first = cli.run("capabilities", "--dir", str(fake_trio), "--json")
    data = assert_result(first, "capabilities", "onemoreffmpeg.capabilities/v1")
    assert data.get("cache_hit", False) is False
    cache_files = list((cli.data_root / "cache").rglob("*.json"))
    assert cache_files, "capabilities run must write a cache entry"

    second = cli.run("capabilities", "--dir", str(fake_trio), "--json")
    cached = assert_result(second, "capabilities", "onemoreffmpeg.capabilities/v1")
    assert cached["cache_hit"] is True

    refreshed = cli.run("capabilities", "--dir", str(fake_trio), "--refresh", "--json")
    assert assert_result(refreshed, "capabilities", "onemoreffmpeg.capabilities/v1").get("cache_hit", False) is False


def test_capabilities_no_cache_writes_nothing(cli, fake_trio: Path):
    cli.run("capabilities", "--dir", str(fake_trio), "--no-cache", "--json")
    cache_dir = cli.data_root / "cache"
    assert not cache_dir.exists() or not list(cache_dir.rglob("*.json"))


# --------------------------------------------------------------------------
# provider / resolve / download / release — argument guards
# --------------------------------------------------------------------------

def test_provider_add_apply(cli):
    data = assert_result(
        cli.run("provider", "add", "--id", "e2e-applied", "--url", "https://example.invalid/x.zip",
                "--sha256", "a" * 64, "--apply", "--json"),
        "provider.add", "onemoreffmpeg.provider-add/v1",
    )
    assert data["status"] == "added"
    assert (cli.data_root / "providers.toml").exists()
    listed = assert_result(cli.run("provider", "list", "--json"),
                           "provider.list", "onemoreffmpeg.provider-list/v1")
    assert "e2e-applied" in {p["id"] for p in listed["providers"]}


def test_download_rejects_offline_release_with_provider_flag(cli, tmp_path: Path):
    release = tmp_path / "release.json"
    release.write_text(json.dumps({
        "schema": "onemoreffmpeg.release/v1", "provider": "custom",
        "repository": "custom", "channel": "daily",
        "release_tag": "latest", "release_id": "1", "published_at": "2026-01-01T00:00:00Z",
        "asset_name": "x.zip", "asset_url": "https://example.invalid/x.zip",
        "asset_sha256": "a" * 64, "variant": "gpl", "linkage": "static",
        "target": "win64", "license_hint": "gpl", "redistributable": True,
    }), encoding="utf-8")
    result = cli.run("download", "--release", str(release), "--provider", "custom", "--json")
    assert_error(result, "OMF-E1001", 1)


def test_install_rejects_short_sha(cli, tmp_path: Path):
    release = tmp_path / "release.json"
    release.write_text(json.dumps({
        "schema": "onemoreffmpeg.release/v1", "provider": "custom",
        "repository": "custom", "channel": "daily",
        "release_tag": "latest", "release_id": "1", "published_at": "2026-01-01T00:00:00Z",
        "asset_name": "x.zip", "asset_url": "https://example.invalid/x.zip",
        "asset_sha256": "abcd", "variant": "gpl", "linkage": "static",
        "target": "win64", "license_hint": "gpl", "redistributable": True,
    }), encoding="utf-8")
    result = cli.run("install", "--release", str(release), "--json")
    # Either schema validation or the new length guard must refuse it safely.
    assert result.returncode == 2
    assert result.json()["schema"] == ERROR_SCHEMA


# --------------------------------------------------------------------------
# env / uninstall / journal / doctor envelopes
# --------------------------------------------------------------------------

def test_env_unset_envelope(cli):
    unset = assert_result(cli.run("env", "unset", "--json"), "env.unset", "onemoreffmpeg.env-set/v1")
    assert unset["dry_run"] is True


def test_env_rollback_requires_journal(cli):
    assert_error(cli.run("env", "rollback", "--json"), "OMF-E6005", 1)


def test_env_rollback_missing_journal_is_runtime_error(cli):
    assert_error(cli.run("env", "rollback", "--journal", "nonexistent-id", "--json"),
                 "OMF-E6005", 1)


def test_uninstall_without_current_is_usage_error(cli):
    result = cli.run("uninstall", "--json")
    assert result.returncode in (1, 2)
    assert result.json()["schema"] == ERROR_SCHEMA


def test_journal_list_empty_is_valid(cli):
    data = assert_result(cli.run("journal", "list", "--json"),
                         "journal.list", "onemoreffmpeg.journal-list/v1")
    assert isinstance(data["records"], list)


def test_doctor_envelope_fields(cli):
    data = assert_result(cli.run("doctor", "--json"), "doctor", "onemoreffmpeg.doctor/v1")
    assert isinstance(data["healthy"], bool)
    assert data["checks"]
    for check in data["checks"]:
        assert check["name"]
        assert isinstance(check["ok"], bool)
