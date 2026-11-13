# 端到端测试

> 基础环境：PowerShell 7 + `C:\Program Files\coreutils`；Linux 使用 `d:\cb-workpath\Linux-Server-Info.txt` 指向的服务器
> 工具：mise、uv（最新 Python）、pytest + pytest-asyncio + pytest-xdist + pytest-timeout、Robot Framework、Bats
> 平台：Windows x64 本机 + Linux x86_64 远程服务器 + GitHub Actions matrix

---

## 1. Windows 本地

```powershell
pwsh -NoProfile -Command '
  $env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="amd64"
  go build -trimpath -o dist/one-more-ffmpeg-windows-amd64.exe ./cmd/one-more-ffmpeg
  $env:OMF_BINARY = (Resolve-Path "dist/one-more-ffmpeg-windows-amd64.exe").Path
  uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
    pytest -c tests/e2e/pytest.ini -n 2 -q tests/e2e/test_cli_e2e.py
  uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
    robot --outputdir tests/e2e/results/robot tests/e2e/robot
'
```

或 `mise run e2e-pytest` / `mise run e2e-robot`。

## 2. Linux 服务器

`tests/e2e/remote_linux.py` 从 `OMF_LINUX_SERVER_INFO`（默认 `d:\cb-workpath\Linux-Server-Info.txt`）读取连接信息；文件与密码不会上传、不会打印。

```powershell
uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
  python tests/e2e/remote_linux.py check

$env:GOOS="linux"; go build -trimpath -o dist/one-more-ffmpeg-linux-amd64 ./cmd/one-more-ffmpeg
uv run --python 3.14 --with-requirements tests/e2e/requirements.txt `
  python tests/e2e/remote_linux.py all
```

远程流程：SFTP 部署源码与 Linux 二进制到 `/tmp/omf-e2e` → `mise trust` → `go test ./...` → pytest e2e → Robot → Bats。远程使用 mise 提供的 Python 3.15、uv、Go 1.27。

## 3. CI

`.github/workflows/e2e.yml`：

- `windows-latest` + `ubuntu-latest` matrix；
- 每平台：Go 单测、构建、pytest e2e（`-n auto`、`--timeout=180`）、Robot；
- Linux 额外跑 Bats；
- Linux 服务器部署/开发/CI 统一以 `Linux-Server-Info.txt` 为准，CI 中对应内容由仓库 secret 提供（`secrets.OMF_LINUX_SERVER_INFO`）。

## 4. 覆盖

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
- Windows junction / Linux symlink 与 shim 路径差异由同一套 pytest 用例覆盖。

所有测试使用独立临时 `ONE_MORE_FFMPEG_HOME`；`env set` 只测 dry-run，不修改真实用户环境。
