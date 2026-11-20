# v1.1.2（2026-11-21，正式版）

首个包含"自定义 URL 目录模式"的稳定版本。CLI 外部契约保持 18 命令 / 17 个 `/v1` schema。
产物：Windows x64 / Linux x64 二进制、`SHA256SUMS.txt`、build provenance（手动下载，无包管理器）。

## 主要能力

- **official provider（默认）**：`resolve` 默认解析 FFmpeg-Builds/FFmpeg-Builds 的 `latest` release，master 快照 / gpl / static / x86_64；命名规则数据驱动（`Rules`），README 链接提示系统要求。
- **自定义 URL 两种模式**：
  - `direct`（默认）：URL 指向具体归档 + `--sha256` pin；
  - `directory`：`--mode directory --name <软件名>`，URL 指向静态目录，按 `latest.txt` + 7 字段命名规则 + 同目录 `checksum.sha256` 解析，不锁死版本。
- **自更新**：`update --check` / `--apply` / `--recover`（崩溃恢复），失败保留唯一备份并报错。
- **安全**：TLS≥1.2、可选私有 CA（`[network] ca_path`）、重定向禁止 https 降级（含 http→https→http）、下载/缓存目录 symlink 防护、压缩炸弹防护、journal 路径校验。
- **质量门禁**：gofmt/vet/golangci-lint（含 gosec）/test/fuzz/govulncheck；Windows+Linux e2e + Docker 全新环境。

## 本版相对 pre4 的修复

- `--release-line` 保留大小写，`YYYYMMDDTHHMMSSZ` 时间戳快照可从 CLI 选中。
- official `Rules` 支持**逐字段覆盖**（此前仅靠 Owner 判断，部分覆盖失效）；`SnapshotLine` 从规则读取；`StableLinePattern` 安全编译。
- GitHub Enterprise API base 不再二次追加 `/api/v3`。
- digest 严格校验 64 位 hex；directory 模式 nonfree 需 `--allow-nonfree` 并给出 `W1006`。
- 文档全量更新到 v1.1.2。

## 限制（明确）

- 不支持 7z（Go 标准库无原生支持）；不做 SSRF 过滤；不使用 `bsd` license；
- 稳定版 `n*` 资产暂不支持（`OMF-E3004`）；仅 x86_64；
- 历史产物不可覆盖属发布纪律，客户端不校验。

## 安装

```bash
# Linux
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2/one-more-ffmpeg-v1.1.2-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

Windows 下载 `one-more-ffmpeg-v1.1.2-windows-amd64.exe` 并按 `SHA256SUMS.txt` 校验。
详见 [`安装指南`](../guides/INSTALL.md)。

## 验证

真实 provider URL 端到端验证（FFmpeg-Builds、yt-dlp/FFmpeg-Builds）；三款官方最新静态服务器（httpd 2.4.71 / nginx 1.30.7 / Caddy 2.11.9）实测；gofmt/vet/lint(0)/test/fuzz/交叉编译/race/双机 e2e 全绿。
