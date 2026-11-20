# 自定义 URL 产物规范（OneMoreFFmpeg 实现版）

> 适用 v1.1.2-pre4+。基于《自定义 URL 与软件产物规范 v2.0（简化版）》实现。
> 目标：普通静态文件服务器即可托管 FFmpeg 产物，**不锁死版本**——通过 `latest.txt` + 命名规则解析最新构建。

## 1. 两种 provider 模式

| 模式 | URL | SHA-256 | 适用 |
|---|---|---|---|
| `direct`（默认，兼容旧行为） | 指向**具体归档文件** | 必须 `--sha256` pin | 单文件直链 |
| `directory`（本规范新增） | 指向**目录**（可不带文件名） | 不传；从 `checksum.sha256` 获取 | 自建静态托管，可持续更新 |

```bash
# directory 模式
one-more-ffmpeg provider add --id my-ffmpeg --mode directory --name ffmpeg \
  --url https://example.com/ffmpeg --license-hint gpl --apply
one-more-ffmpeg resolve --provider my-ffmpeg --target linux64 --json > release.json
one-more-ffmpeg download --release release.json
```

## 2. 命名格式（7 字段）

```
{name}-{version}-{license}-{release_type}-{os}-{arch}-{linkage}.{archive_format}
ffmpeg-20261120-gpl-snapshot-linux-amd64-static.tar.xz
ffmpeg-8.1.0-gpl-stable-windows-amd64-static.zip
```

| 字段 | 允许值 |
|---|---|
| `version` | stable：`8.1.0`（数字点分≥3段，无 `v`）；snapshot：`20261120` 或 `20261120T153000Z`（UTC） |
| `license` | `gpl` / `lgpl` / `nonfree` / `custom`（**不使用 bsd**） |
| `release_type` | `stable` / `snapshot` |
| `os` | windows / linux / darwin / freebsd / openbsd / netbsd / android |
| `arch` | amd64 / arm64 / 386 / arm / ppc64le / riscv64 / s390x（**本工具当前只解析 amd64**） |
| `linkage` | static / shared |
| `archive_format` | `zip` / `tar.gz` / `tar.xz` / `tar.zst`（**不支持 7z**：Go 标准库无原生 7z，且无稳定依赖） |

文件名必须小写 ASCII（`T`/`Z` 时间戳可大写），字段用半角 `-` 分隔，无空格/中文/路径分隔符。

## 3. 目录结构与 latest.txt（可选）

```
ffmpeg/
  README.md
  stable/8.1.0/{checksum.sha256, ffmpeg-8.1.0-...}
  snapshots/latest.txt          # 一行：20261120
  snapshots/20261120/{checksum.sha256, ffmpeg-20261120-...}
```

- `latest.txt` 只写一行日期/时间戳（允许一个末尾换行），无 JSON/YAML；
- 客户端读取后校验格式，再拼接快照目录与文件名；
- 不提供 `latest.txt` 时可用 `--release-line` 显式指定版本；
- 服务器**不需要**目录索引、rewrite、脚本（见 [`deployment/README.md`](../deployment/README.md)）。

## 4. checksum.sha256

- 固定文件名，位于归档同目录（客户端自动推导，显式 `--checksums-url` 可覆盖）；
- GNU sha256sum 格式：`<64位小写hex>  <文件名>`；
- 文件名必须与归档文件名**完全一致**（不接受 `dist/name` 等路径）；
- 重复条目报错；CRLF 允许；**不对 checksum.sha256 自身做校验**。

## 5. 安全

- 只接受 http/https；拒绝 URL 内嵌凭据；
- HTTPS 最低 TLS 1.2，正常验证证书（无 `InsecureSkipVerify`）；可选私有 CA（`[network] ca_path`）；
- 重定向最多 10 次；**禁止 HTTPS→HTTP 降级（含 http→https→http）**；
- **不做 SSRF 主机过滤**（本机 CLI、用户自配 URL，按项目决定）；
- 下载先临时文件 → SHA-256 校验 → 原子落盘；失败删临时文件。

## 6. 与旧版的区别

- 旧的 btbn / bnqdzj-nonfree 内置 provider、ffmpeg.org 版本目录、`release list/install` 已移除；
- v1.1.2-pre4 的 `official` provider（FFmpeg-Builds latest）是新增的直接解析，不是恢复旧来源；
- 本规范仅用于**用户自建静态托管**，与 `official` 并存。

服务器配置与来源说明见 [`deployment/README.md`](../deployment/README.md)。
