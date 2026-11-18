# Fresh-environment e2e harness. Builds and tests OneMoreFFmpeg in a clean
# container with no developer state, proving the repo is self-contained.
FROM mirror.gcr.io/library/golang:1.27-bookworm

ENV CGO_ENABLED=0 \
    DEBIAN_FRONTEND=noninteractive \
    UV_PYTHON=3.14

# git + curl needed for Bats and uv; ca-certificates for TLS.
RUN apt-get update && apt-get install -y --no-install-recommends \
        git curl ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && curl -LsSf https://astral.sh/uv/install.sh | sh \
    && ln -sf /root/.local/bin/uv /usr/local/bin/uv

WORKDIR /src
COPY . /src

# Build the CLI for this (linux) platform. The image has no mise and the tests
# do not use it, so no `mise trust` step is needed here.
RUN mkdir -p dist && go build -trimpath -o dist/one-more-ffmpeg-linux-amd64 ./cmd/one-more-ffmpeg

ENV OMF_BINARY=/src/dist/one-more-ffmpeg-linux-amd64

CMD ["sh", "-c", "set -e; \
  go test -count=1 ./...; \
  uv run --python 3.14 --with-requirements tests/e2e/requirements.txt \
    pytest -c tests/e2e/pytest.ini -n 2 -q \
    tests/e2e/test_cli_e2e.py tests/e2e/test_cli_full_e2e.py tests/e2e/test_env_real_e2e.py \
    tests/e2e/test_update_e2e.py; \
  uv run --python 3.14 --with-requirements tests/e2e/requirements.txt \
    robot --outputdir tests/e2e/results/robot tests/e2e/robot; \
  git clone --depth 1 --branch v1.14.0 https://github.com/bats-core/bats-core.git /tmp/bats-core; \
  /tmp/bats-core/bin/bats tests/e2e/bats; \
  echo FRESH_ENV_E2E_OK"]
