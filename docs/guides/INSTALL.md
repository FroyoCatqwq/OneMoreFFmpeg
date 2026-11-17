# 安装 OneMoreFFmpeg

当前稳定版：**v1.0.1**。所有产物由 GitHub Actions 从源码构建，带 SHA-256 校验和与 build provenance。
发布页：<https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/tag/v1.0.1>

> 仅 v1.0.1 受支持；更早的 Release/tag 已撤回，请勿使用旧版本下载链接。

## Windows（推荐 Scoop）

```powershell
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg
one-more-ffmpeg version --json
```

后续升级：

```powershell
scoop update
scoop update one-more-ffmpeg
```

Scoop bucket 的 manifest（`one-more-ffmpeg.json`）由 release 流水线在每次发布时自动更新，
包含 Windows x64 二进制的 SHA-256，并配置 `autoupdate` 以跟随新版本。

## Windows（手动）

下载 `one-more-ffmpeg-v1.0.1-windows-amd64.exe`，放到 PATH 中的目录，重命名为 `one-more-ffmpeg.exe`：

```powershell
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\OneMoreFFmpeg\bin" | Out-Null
Move-Item .\one-more-ffmpeg-v1.0.1-windows-amd64.exe "$env:LOCALAPPDATA\OneMoreFFmpeg\bin\one-more-ffmpeg.exe"
# 把该目录加入用户 PATH 后开新终端
one-more-ffmpeg version --json
```

## RHEL 9 及以上（x86_64）

支持 RHEL 9 / AlmaLinux 9 / Rocky Linux 9 等兼容发行版。纯静态 Go 二进制，无运行时依赖：

```bash
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.1/one-more-ffmpeg-1.0.1-1.el9.x86_64.rpm
one-more-ffmpeg version --json
```

升级：`sudo dnf upgrade one-more-ffmpeg`。卸载：`sudo dnf remove one-more-ffmpeg`。

RPM 安装内容：`/usr/bin/one-more-ffmpeg`、man 页、LICENSE。

## Linux 通用（手动二进制）

```bash
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.1/one-more-ffmpeg-v1.0.1-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.1/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
chmod +x one-more-ffmpeg-v1.0.1-linux-amd64
./one-more-ffmpeg-v1.0.1-linux-amd64 version --json
```

## 校验和验证

每个 Release 提供 `SHA256SUMS.txt`，覆盖双平台二进制与 RPM：

```bash
sha256sum -c SHA256SUMS.txt --ignore-missing        # Linux/macOS
# Windows PowerShell:
Get-FileHash .\one-more-ffmpeg-v1.0.1-windows-amd64.exe -Algorithm SHA256
```

Release 同时附带 Sigstore build provenance attestation，可用 `gh attestation verify` 校验来源。

## 下一步

安装完成后按 [`HOWTO.md`](HOWTO.md) 的黄金顺序操作：`install --apply` → `env set --apply` → 开新终端 → `ffmpeg` 可用。
