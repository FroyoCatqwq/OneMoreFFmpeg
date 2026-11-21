"""e2e for network error paths using a local HTTP server (fully offline).

Verifies that the CLI maps upstream failures to stable typed error codes and
that stdout carries exactly one JSON error envelope while stderr stays clean.
"""
from __future__ import annotations

import hashlib
import http.server
import socketserver
import threading

import pytest

pytestmark = pytest.mark.e2e

VA = "ffmpeg-20261120-gpl-snapshot-linux-amd64-static.tar.xz"
PAYLOAD = b"payload\n"


class _Handler(http.server.BaseHTTPRequestHandler):
    mode = "ok"

    def log_message(self, *args):
        pass

    def do_GET(self):
        if self.path.endswith("latest.txt"):
            if self.mode == "latest_404":
                self.send_response(404)
                self.end_headers()
                return
            if self.mode == "latest_500":
                self.send_response(500)
                self.end_headers()
                return
            body = b"20261120\n"
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path.endswith("checksum.sha256"):
            digest = hashlib.sha256(PAYLOAD).hexdigest()
            if self.mode == "checksum_dup":
                body = f"{digest}  {VA}\n{digest}  {VA}\n".encode()
            elif self.mode == "checksum_missing":
                body = f"{digest}  other.tar.xz\n".encode()
            elif self.mode == "checksum_oversize":
                body = (b"#" * (4 << 20)) + b"\n"
            else:
                body = f"{digest}  {VA}\n".encode()
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path.endswith("redirect-loop"):
            self.send_response(302)
            self.send_header("Location", self.path)
            self.end_headers()
            return
        self.send_response(404)
        self.end_headers()


@pytest.fixture
def error_server():
    class Reuse(socketserver.TCPServer):
        allow_reuse_address = True

    handler = _Handler
    server = Reuse(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield f"http://127.0.0.1:{server.server_address[1]}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def _add(cli, base, pid="errsrv"):
    r = cli.run("provider", "add", "--id", pid, "--mode", "directory",
                "--name", "ffmpeg", "--url", f"{base}/ffmpeg",
                "--license-hint", "gpl", "--apply", "--json")
    assert r.returncode == 0, r.stdout


@pytest.mark.parametrize("mode, expected", [
    ("latest_404", "OMF-E3007"),
    ("latest_500", "OMF-E3007"),
    ("checksum_dup", "OMF-E4002"),
    ("checksum_missing", "OMF-E4002"),
    ("checksum_oversize", "OMF-E4004"),
])
def test_directory_error_codes(cli, error_server, mode, expected):
    _Handler.mode = mode
    _add(cli, error_server)
    r = cli.run("resolve", "--provider", "errsrv", "--target", "linux64", "--json")
    assert r.returncode != 0
    err = r.json()
    assert err["schema"] == "onemoreffmpeg.error/v1"
    assert err["code"] == expected, (mode, err)
    assert r.stderr.strip() == "", "errors must go to stdout JSON, not stderr"
