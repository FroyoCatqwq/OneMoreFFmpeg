# v1.1.2-pre4（2026-11-20，预发布）

> 新功能：自定义 URL 目录模式（自建静态文件托管，不锁死版本）。CLI 契约保持 18 命令 / 17 个 `/v1` schema。
> 产物：Windows x64 / Linux x64 二进制、`SHA256SUMS.txt`、build provenance。
> **预发布版本，仅供测试。**

## 新功能

```bash
one-more-ffmpeg provider add --id my-ffmpeg --mode directory --name ffmpeg \
  --url https://example.com/ffmpeg --license-hint gpl --apply
one-more-ffmpeg resolve --provider my-ffmpeg --target linux64 --json > release.json
one-more-ffmpeg download --release release.json
```

- 目录型 URL：服务器只需普通静态文件（`latest.txt` + 按 7 字段命名规则归档 + `checksum.sha256`），无需脚本/rewrite/JSON/目录列表。
- `--release-line` 可指定稳定版（如 8.1.0）或快照日期；不指定则读 `snapshots/latest.txt`。
- `internal/customurl` 四包：naming/checksum/latest/urlpolicy；TLS≥1.2、可选私有 CA、重定向链禁止 https 降级（含 http→https→http）。
- 契约放宽：`release` 允许单独 `checksums_url`（checksum.sha256 不自身校验）。

## 明确不做

- 7z（Go 标准库无原生支持，无稳定依赖）；
- SSRF 主机过滤（本机 CLI、用户自配 URL）；
- `bsd` license（保持 gpl/lgpl/nonfree/custom）。

## 服务器配置与来源

`docs/deployment/` 提供 httpd/nginx/Caddy 官方最小配置与来源说明。三款均以**官方最新稳定版**实测：
httpd 2.4.71（Apache 官方源码，SHA256 校验）、nginx 1.30.7（nginx.org 官方源）、Caddy 2.11.9（官方 COPR）。
实测确认：`latest.txt` 可读、`checksum.sha256` 200、目录列表关闭（403/404）、CLI resolve + download + SHA-256 校验全通。

详见 [`CUSTOM_URL_SPEC.md`](../reference/CUSTOM_URL_SPEC.md) 与 [`deployment/README.md`](../deployment/README.md)。

## 安装（手动二进制）

```bash
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre4/one-more-ffmpeg-v1.1.2-pre4-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre4/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

## 验证

gofmt / vet / golangci-lint(0) / go test / 三平台交叉编译全绿；Windows 47 passed；三款服务器实机端到端通过。
