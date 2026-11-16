# 端到端测试

> 基础环境：PowerShell 7 + coreutils；Linux 服务器信息文件为仓库同级的 `Linux-Server-Info.txt`（由 `OMF_LINUX_SERVER_INFO` 覆盖）
> 工具：mise、uv（Python 3.14）、pytest + pytest-asyncio + pytest-xdist + pytest-timeout、Robot Framework、Bats 1.14.0（pin）
> 平台：Windows x64 本机 + Linux x86_64 远程服务器 + Docker 全新环境 + GitHub Actions matrix

---

## 1. Windows 本地

```powershell
pwsh -NoProfile -Command '
  $env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="amd64"
  go build -trimpath -o dist/one-more-ffmpeg-windows-amd64.exe ./cmd/one-more-ffmpeg
  $env:OMF_BINARY = (Resolve-Path "dist/one-more-ffmpeg-windows-amd64.exe").Path
  uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
    pytest -c tests/e2e/pytest.ini -n 2 -q tests/e2e/test_cli_e2e.py tests/e2e/test_cli_full_e2e.py
  uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
    robot --outputdir tests/e2e/results/robot tests/e2e/robot
'
```

或 `mise run e2e-pytest` / `mise run e2e-robot`。

## 2. Linux 服务器

`tests/e2e/remote_linux.py` 从 `OMF_LINUX_SERVER_INFO`（默认仓库同级的 `Linux-Server-Info.txt`）读取连接信息；文件与密码不会上传、不会打印。

```powershell
uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
  python tests/e2e/remote_linux.py check

$env:GOOS="linux"; go build -trimpath -o dist/one-more-ffmpeg-linux-amd64 ./cmd/one-more-ffmpeg
uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
  python tests/e2e/remote_linux.py all
```

远程流程：SFTP 部署源码与 Linux 二进制到 `/tmp/omf-e2e` → `mise trust` → `go test ./...` → pytest e2e → Robot → Bats。远程使用 mise 提供的 Python 3.14、uv、Go 1.27（工具链实际为 go1.27.4）。

## 3. CI

- `.github/workflows/e2e.yml`（`quality-e2e`）：`quality` job 跑 gofmt / go vet / go test / govulncheck / fuzz 短跑；
  `e2e` job 在 `windows-latest` + `ubuntu-latest` 矩阵构建 CLI 并跑 pytest（两个测试文件、`--junitxml`）与 Robot，
  Linux 额外跑 pin 到 `v1.14.0` 的 Bats；`workflow_dispatch` 可额外触发远程 Linux server job。
  远程连接信息由仓库 secret `OMF_LINUX_SERVER_INFO` 提供。
- `.github/workflows/release.yml`：tag（或 dispatch 指定已有 tag）触发，校验 release notes 存在、跑全量 go test、
  按 `VERSIONING.md` 命名构建双平台二进制、校验产物只含 CLI、生成 `SHA256SUMS.txt` 与 build provenance attestation，
  最后 `gh release create --verify-tag` 发布。

## 4. Docker 全新环境验证

在有 Docker + Docker Compose 的机器上（例如 Linux CI 服务器）：

```bash
docker compose up --build --abort-on-container-exit --exit-code-from e2e-fresh
```

`Dockerfile` 从 `mirror.gcr.io/library/golang:1.27-bookworm` 构建干净容器，安装 uv + Python 3.14，
从零构建 CLI 并跑 go test + 两个 pytest 文件 + Robot + Bats；不挂载任何宿主数据目录，
用于证明仓库在全新环境可独立构建与测试。注意不要在 compose 中把 `/tmp` 挂成 tmpfs（默认 noexec 会让 `go test` 失败）。

## 5. 覆盖

- version / 错误对象 / 未知命令 / `--in-detail-usage`；
- 单类能力探测 `--only`；
- 本地 HTTP 模拟的 stable/daily 版本目录；
- uninstall → journal list → journal rollback 恢复安装目录；
- provider list / provider add dry-run；
- env status / doctor；
- detect / validate / capabilities（临时编译 fake FFmpeg trio）；
- install dry-run → apply → LICENSE/锁文件/current → env set dry-run → uninstall；
- 非法 release 拒绝；
- asyncio 并行只读命令；
- Windows junction / Linux symlink 与 shim 路径差异由同一套 pytest 用例覆盖；
- `test_cli_full_e2e.py`：全部 18 个冻结命令的 JSON 信封形状、stdout JSON-only（含 help/`--json` 冲突）、
  capabilities 字段级解析（版本/许可证/13 条命令/9 类列表/缓存命中与刷新）、9 种 `--only`、
  参数互斥（`--release` + provider flags、`--only` + `--help-topic`）、各错误码退出码。

所有测试使用独立临时 `ONE_MORE_FFMPEG_HOME`；`env set` 只测 dry-run，不修改真实用户环境。
