"""Phase 7+ e2e: REAL environment-variable / PATH / shim modifications.

These tests use --apply (not dry-run) and verify that files/registries are
actually changed, then rolled back to their prior state. They are isolated:

* Linux: HOME is redirected to a temporary directory, so `~/.profile` and
  `~/.local/bin` writes never touch the developer's real home.
* Windows: env writes go to the real HKCU registry, so the mutating test is
  gated behind OMF_E2E_REAL_ENV=1 and always restores the previous values in a
  finally block. CI does not set the marker, so CI stays clean.
"""
from __future__ import annotations

import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.e2e


def _env_with_home(cli, home: Path) -> dict[str, str]:
    env = cli.env()
    env["HOME"] = str(home)
    env["USERPROFILE"] = str(home)  # Go's os.UserHomeDir on Windows
    return env


# --------------------------------------------------------------------------
# Linux: full real file-system env modification, fully isolated via HOME
# --------------------------------------------------------------------------

@pytest.mark.skipif(os.name == "nt", reason="Linux shim/PATH flow; Windows covered below")
def test_env_apply_writes_and_rolls_back_shims_and_profile(cli, tmp_path: Path):
    home = tmp_path / "fake-home"
    home.mkdir()
    env = _env_with_home(cli, home)
    shim_dir = home / ".local" / "bin"
    profile = home / ".profile"

    # Nothing exists before apply.
    assert not shim_dir.exists()
    assert not profile.exists()

    applied = cli.run("env", "set", "--apply", "--json", env=env)
    assert applied.returncode == 0, applied.stdout
    data = applied.data()
    assert data["applied"] is True
    assert data["dry_run"] is False
    assert data["journal_id"]
    jid = data["journal_id"]

    # Shims created with the managed template and executable mode.
    for name in ("ffmpeg", "ffprobe", "ffplay"):
        shim = shim_dir / name
        assert shim.exists(), f"missing shim {name}"
        body = shim.read_text(encoding="utf-8")
        assert body.startswith("#!/bin/sh\n")
        assert "# OneMoreFFmpeg managed shim" in body
        assert "exec " in body
        assert shim.stat().st_mode & 0o111, "shim must be executable"

    # .profile got an export pointing at the shim dir.
    assert profile.exists()
    profile_body = profile.read_text(encoding="utf-8")
    assert ".local/bin" in profile_body

    # env status reflects PATH/variables in THIS process view.
    status = cli.run("env", "status", "--json", env=env)
    assert status.returncode == 0, status.stdout

    # Rollback removes the shims and restores the (absent) profile.
    rolled = cli.run("env", "rollback", "--journal", jid, "--apply", "--json", env=env)
    assert rolled.returncode == 0, rolled.stdout
    assert rolled.data()["applied"] is True
    for name in ("ffmpeg", "ffprobe", "ffplay"):
        assert not (shim_dir / name).exists(), f"shim {name} not removed on rollback"
    # Profile backup restored: it did not exist, so it must be gone or empty.
    if profile.exists():
        assert profile.read_text(encoding="utf-8").strip() == ""

    # A second rollback is refused (already rolled back).
    again = cli.run("env", "rollback", "--journal", jid, "--apply", "--json", env=env)
    assert again.returncode != 0
    assert again.json()["schema"] == "onemoreffmpeg.error/v1"


@pytest.mark.skipif(os.name == "nt", reason="Linux shim flow")
def test_env_unset_removes_preexisting_managed_shim(cli, tmp_path: Path):
    home = tmp_path / "fake-home"
    home.mkdir()
    env = _env_with_home(cli, home)
    shim_dir = home / ".local" / "bin"
    shim_dir.mkdir(parents=True)
    # A managed shim that existed before this tool ran.
    managed = shim_dir / "ffmpeg"
    managed.write_text("#!/bin/sh\n# OneMoreFFmpeg managed shim\nexec '/old/ffmpeg' \"$@\"\n", encoding="utf-8")

    applied = cli.run("env", "unset", "--apply", "--json", env=env)
    assert applied.returncode == 0, applied.stdout
    assert not managed.exists(), "unset --apply must remove a managed shim"

    jid = applied.data()["journal_id"]
    rolled = cli.run("env", "rollback", "--journal", jid, "--apply", "--json", env=env)
    assert rolled.returncode == 0, rolled.stdout
    assert managed.exists(), "rollback must restore the shim"
    assert "'/old/ffmpeg'" in managed.read_text(encoding="utf-8")


@pytest.mark.skipif(os.name == "nt", reason="Linux shim flow")
def test_env_set_preserves_nonmanaged_shim(cli, tmp_path: Path):
    home = tmp_path / "fake-home"
    home.mkdir()
    env = _env_with_home(cli, home)
    shim_dir = home / ".local" / "bin"
    shim_dir.mkdir(parents=True)
    foreign = shim_dir / "ffmpeg"
    original = "#!/bin/sh\nexec /usr/bin/realffmpeg \"$@\"\n"
    foreign.write_text(original, encoding="utf-8")

    applied = cli.run("env", "set", "--apply", "--json", env=env)
    assert applied.returncode == 0, applied.stdout
    assert foreign.read_text(encoding="utf-8") == original, "non-managed shim must not be overwritten"
    assert any("skip shim" in op for op in applied.data()["operations"])


# --------------------------------------------------------------------------
# Windows: real HKCU registry writes, gated to avoid polluting normal runs
# --------------------------------------------------------------------------

@pytest.mark.skipif(os.name != "nt", reason="Windows registry flow")
@pytest.mark.skipif(os.environ.get("OMF_E2E_REAL_ENV") != "1",
                    reason="set OMF_E2E_REAL_ENV=1 to run real HKCU registry e2e")
def test_env_apply_writes_and_rolls_back_registry(cli, tmp_path: Path):
    import subprocess

    def reg_get(name: str) -> tuple[bool, str]:
        p = subprocess.run(["reg", "query", r"HKCU\Environment", "/v", name],
                           capture_output=True, text=True)
        if p.returncode != 0:
            return False, ""
        for line in p.stdout.splitlines():
            if "REG_" in line:
                return True, line.strip().split("    ")[-1]
        return True, ""

    name = "ONE_MORE_FFMPEG_DIR"
    existed_before, value_before = reg_get(name)
    jid = None
    try:
        applied = cli.run("env", "set", "--apply", "--json")
        assert applied.returncode == 0, applied.stdout
        data = applied.data()
        assert data["applied"] is True
        jid = data["journal_id"]
        exists_now, value_now = reg_get(name)
        assert exists_now, "env set --apply must write the variable to HKCU"
    finally:
        if jid:
            rolled = cli.run("env", "rollback", "--journal", jid, "--apply", "--json")
            assert rolled.returncode == 0, rolled.stdout
        # Belt-and-braces: restore exact prior state if rollback judgment differs.
        if existed_before:
            subprocess.run(["reg", "add", r"HKCU\Environment", "/v", name,
                            "/d", value_before, "/f"], capture_output=True)
        else:
            subprocess.run(["reg", "delete", r"HKCU\Environment", "/v", name,
                            "/f"], capture_output=True)
