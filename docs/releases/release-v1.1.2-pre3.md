# v1.1.2-pre3（2026-11-21，预发布）

> 新功能：默认 FFmpeg-Builds 官方 latest 快照 provider。CLI 契约保持 18 命令 / 17 个 `/v1` schema。
> 产物：Windows x64 / Linux x64 二进制、`SHA256SUMS.txt`、build provenance。
> **预发布版本，仅供测试。**

## 新功能

```bash
one-more-ffmpeg resolve --json > release.json     # 默认 official
one-more-ffmpeg download --release release.json
one-more-ffmpeg install --release release.json --apply
one-more-ffmpeg env set --apply
```

- **默认 official provider**：查询 <https://github.com/FFmpeg-Builds/FFmpeg-Builds/releases/tag/latest>，默认选 `master` 最新快照 / `gpl` / `static` / x86_64（`win64`/`linux64`）。
- **数据驱动命名规则**：上游资产命名约定集中在 `Rules` 结构（模板 + 映射 + 正则），上游改名只需改数据。
- **系统要求**：warning 直接给出上游 README 链接（<https://github.com/FFmpeg-Builds/FFmpeg-Builds#readme>），不解析内容——README 改版不影响本工具。
- `provider list` 显示 official + 自定义；自定义 URL provider 仍可用（`provider add --id --url --sha256 --apply`，http/https）。

## 限制（本版明确不支持）

- 稳定版 `n*` 资产（如 `n9.0`）：`OMF-E3004`，待命名规则确定；
- 非 x86_64 架构（win32/arm64 等）：`OMF-E3003`。

## 安装（手动二进制）

```bash
# Linux
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre3/one-more-ffmpeg-v1.1.2-pre3-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre3/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

Windows 下载 `one-more-ffmpeg-v1.1.2-pre3-windows-amd64.exe` 并校验。详见 [`安装指南`](../guides/INSTALL.md)。

## 验证

gofmt / vet / golangci-lint(0) / go test / fuzz / 三平台交叉编译全绿；真实 GitHub API 解析 official latest release 通过（拿到 asset + SHA-256 + README 链接 warning）；official 离线 mock 测试 6 项全过。
