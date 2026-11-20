# 安装 OneMoreFFmpeg

当前稳定版：**v1.1.2-pre3**。所有产物由 GitHub Actions 从源码构建，带 SHA-256 校验和与 build provenance。
发布页：<https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/tag/v1.1.2-pre3>

> 仅 v1.1.2-pre3 受支持；更早的 Release/tag 已撤回，请勿使用旧版本下载链接。
>
> 本工具只发布 **Windows / Linux 二进制手动下载包**（附 `SHA256SUMS.txt`），**不提供 Scoop、dnf、RPM 等包管理器安装方式**。

## Windows（手动下载）

从 Release 下载 `one-more-ffmpeg-v1.1.2-pre3-windows-amd64.exe` 与 `SHA256SUMS.txt`：

```powershell
# 校验（详见下方「校验和验证」）
Get-FileHash .\one-more-ffmpeg-v1.1.2-pre3-windows-amd64.exe -Algorithm SHA256

New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\OneMoreFFmpeg\bin" | Out-Null
Move-Item .\one-more-ffmpeg-v1.1.2-pre3-windows-amd64.exe "$env:LOCALAPPDATA\OneMoreFFmpeg\bin\one-more-ffmpeg.exe"
# 把该目录加入用户 PATH 后开新终端
one-more-ffmpeg version --json
```

## Linux（手动下载）

```bash
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre3/one-more-ffmpeg-v1.1.2-pre3-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre3/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
chmod +x one-more-ffmpeg-v1.1.2-pre3-linux-amd64
./one-more-ffmpeg-v1.1.2-pre3-linux-amd64 version --json
```

把二进制放到 PATH 中并重命名为 `one-more-ffmpeg` 后即可直接调用。

## 校验和验证

每个 Release 提供 `SHA256SUMS.txt`，覆盖 Windows / Linux 双平台二进制：

```bash
sha256sum -c SHA256SUMS.txt --ignore-missing        # Linux/macOS
# Windows PowerShell:
Get-FileHash .\one-more-ffmpeg-v1.1.2-pre3-windows-amd64.exe -Algorithm SHA256
```

Release 同时附带 Sigstore build provenance attestation，可用 `gh attestation verify` 校验来源。

## 保持更新

v1.1 起工具可检查并更新自身（手动下载的二进制用 `update`）：

```bash
one-more-ffmpeg update            # 仅检查，输出 update_available
one-more-ffmpeg update --apply    # 下载校验后替换自身
```

`--apply` 会校验新版本的 SHA-256 digest 并原子替换，失败自动回滚；`--json` 下 stdout 只输出 JSON 信封，进度与日志走 stderr。

## 下一步

安装完成后按 [`新手上手指南`](HOWTO.md) 操作：`resolve`（默认官方 latest 快照）→ `download` → `install --apply` → `env set --apply` → 开新终端 → `ffmpeg` 可用。
