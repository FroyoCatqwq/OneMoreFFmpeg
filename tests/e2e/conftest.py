from __future__ import annotations

import asyncio
import json
import os
import shutil
import subprocess
from dataclasses import dataclass
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]
FAKE_GO = r'''package main
import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)
func main() {
    name := strings.TrimSuffix(strings.ToLower(filepath.Base(os.Args[0])), filepath.Ext(os.Args[0]))
    arg := ""
    if len(os.Args) > 1 { arg = os.Args[1] }
    switch arg {
    case "-version":
        fmt.Printf("%s version 9.9.9\nconfiguration: --enable-gpl\n", name)
    case "-L", "-license":
        fmt.Println("GPL version 3 or later")
    case "-h":
        fmt.Println("Help: full")
    case "-decoders", "-encoders":
        fmt.Printf("%s:\n V..... = Video\n ------\n V....D h264 H.264 / AVC\n", strings.TrimPrefix(arg, "-"))
    case "-muxers", "-demuxers", "-devices":
        fmt.Printf("%s:\n D. = Demuxing supported\n .E = Muxing supported\n --\n E mp4 MP4 (MPEG-4 Part 14)\n", strings.TrimPrefix(arg, "-"))
    case "-filters":
        fmt.Println("Filters:\n T.. = Timeline support\n ... scale V->V Scale the input video")
    case "-pix_fmts":
        fmt.Println("Pixel formats:\nFLAGS NAME NB_COMPONENTS BITS_PER_PIXEL BIT_DEPTHS\n-----\nIO... yuv420p 3 12 8-8-8")
    case "-layouts":
        fmt.Println("Individual channels:\nNAME DESCRIPTION\nFL front left\n\nStandard channel layouts:\nNAME DECOMPOSITION\nmono FC\nstereo FL+FR")
    case "-sample_fmts":
        fmt.Println("name depth\n u8 8\n s16 16")
    default:
        fmt.Printf("%s %s\n", name, arg)
    }
}
'''


def pytest_configure(config):
    config.addinivalue_line("markers", "e2e: end-to-end CLI tests")


@dataclass
class CLIResult:
    returncode: int
    stdout: str
    stderr: str

    def json(self):
        return json.loads(self.stdout)

    def data(self):
        return self.json()["data"]


async def _run(binary: Path, args: tuple[str, ...], env: dict[str, str], timeout: int) -> CLIResult:
    process = await asyncio.create_subprocess_exec(
        str(binary), *args,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
        env=env,
    )
    try:
        stdout, stderr = await asyncio.wait_for(process.communicate(), timeout=timeout)
    except asyncio.TimeoutError:
        process.kill()
        await process.communicate()
        raise AssertionError(f"CLI timed out: {args}")
    return CLIResult(process.returncode, stdout.decode("utf-8", "replace"), stderr.decode("utf-8", "replace"))


@pytest.fixture(scope="session")
def binary() -> Path:
    value = os.environ.get("OMF_BINARY")
    if value:
        path = Path(value).resolve()
    elif os.name == "nt":
        path = ROOT / "dist" / "one-more-ffmpeg-windows-amd64.exe"
    else:
        path = ROOT / "dist" / "one-more-ffmpeg-linux-amd64"
    assert path.exists(), f"build the CLI first: {path}"
    return path


@pytest.fixture
def cli(binary: Path, tmp_path: Path):
    home = tmp_path / "data-root"
    home.mkdir()

    def base_env() -> dict[str, str]:
        env = dict(os.environ)
        env["ONE_MORE_FFMPEG_HOME"] = str(home)
        env.pop("OMF_FAKE_EXIT", None)
        return env

    class Runner:
        data_root = home

        def env(self, **extra: str) -> dict[str, str]:
            env = base_env()
            env.update(extra)
            return env

        def run(self, *args: str, env: dict[str, str] | None = None, timeout: int = 40) -> CLIResult:
            return asyncio.run(_run(binary, args, env or base_env(), timeout))

        async def arun(self, *args: str, env: dict[str, str] | None = None, timeout: int = 40) -> CLIResult:
            return await _run(binary, args, env or base_env(), timeout)

    return Runner()


@pytest.fixture(scope="session")
def fake_trio(tmp_path_factory, worker_id: str) -> Path:
    directory = tmp_path_factory.mktemp(f"fake-trio-{worker_id}")
    source = directory / "main.go"
    source.write_text(FAKE_GO, encoding="utf-8")
    extension = ".exe" if os.name == "nt" else ""
    executable = directory / ("ffmpeg" + extension)
    env = dict(os.environ, GO111MODULE="off", CGO_ENABLED="0",
               GOOS="windows" if os.name == "nt" else "linux", GOARCH="amd64")
    subprocess.run(["go", "build", "-o", str(executable), str(source)], check=True, env=env, timeout=180)
    for name in ("ffprobe", "ffplay"):
        shutil.copy2(executable, directory / (name + extension))
    return directory
