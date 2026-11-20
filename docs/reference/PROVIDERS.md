# Provider、来源与 Release Pin

> 状态：v1.1.2-pre4 现行；CLI 外部契约冻结（18 命令 / 17 个 `/v1` schema）
> 默认 provider：**official**（FFmpeg-Builds 官方 latest 快照）；可选：用户自定义 URL。
> 解析细则：[`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)；CLI 输出见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)

---

## 1. official provider（默认，推荐）

`resolve`（不带 `--provider`）与 `resolve --provider official` 都走内置 official provider：

- **数据源**：<https://github.com/FFmpeg-Builds/FFmpeg-Builds/releases/tag/latest>（由 FFmpeg 官方推荐）；
- **默认选择**：`master` 最新快照、`gpl` variant、`static` linkage、x86_64（`win64` / `linux64`）；
- **稳定版**：`n*` 版本（如 `n9.0`）的资产命名尚在评估，**暂不支持**，返回 `OMF-E3004`；
- **系统要求**：工具**不解析**上游 README（它会变化），而是在 resolve 结果的 warning 中给出 README 链接
  <https://github.com/FFmpeg-Builds/FFmpeg-Builds#readme>，请自行查看 Windows 版本 / UCRT、glibc / kernel 要求。

### 命名规则（数据驱动，不写死在逻辑中）

所有上游命名约定集中在 `internal/provider/official` 的 `Rules` 结构中：

| 字段 | 默认值 | 说明 |
|---|---|---|
| `Owner` / `Repo` / `Tag` | `FFmpeg-Builds` / `FFmpeg-Builds` / `latest` | GitHub 仓库与 tag |
| `AssetTemplate` | `ffmpeg-{{.Line}}-latest-{{.Target}}-{{.Variant}}{{.LinkageSuffix}}{{.Ext}}` | Go 模板，变量 `.Line/.Target/.Variant/.LinkageSuffix/.Ext` |
| `TargetDir` | `win64→win64`、`linux64→linux64` | 逻辑 target 到资产 token 的映射 |
| `Extensions` | `win64→.zip`、`linux64→.tar.xz` | 归档扩展名 |
| `SnapshotLine` | `master` | 快照 line token |
| `StableLinePattern` | `^n[0-9]+\.[0-9]+$` | 稳定版 line（拒绝） |

上游将来若改资产命名，只需更新 `Rules`（数据），无需改解析逻辑。

## 2. 自定义 provider（可选）

用你自己的 URL + SHA-256 登记来源（支持 http/https，便于自建服务器）：

```bash
omf provider add --id my-ffmpeg --url https://example.invalid/ffmpeg.zip --sha256 <64位SHA256> --apply
omf provider list --json
omf resolve --provider my-ffmpeg --json > release.json
```

- URL 必须指向 `zip` / `tar.xz` / `tar.gz` / `tar.zst` 归档；
- SHA-256 必填（64 位 hex），下载时强制校验；可选 `checksums_url` 提供第二信源；
- `provider list` 显示内置 official + 用户自定义。

## 3. 共享流程

```bash
omf resolve --json > release.json     # 默认 official
omf download --release release.json   # 强制 SHA-256 校验
omf install --release release.json --apply
omf env set --apply
```

`resolve` 只产出 `onemoreffmpeg.release/v1` pin，不下载；download/install 消费同一 pin，与来源无关。

## 4. 明确不做

- 稳定版（n*）下载安装（待命名规则确定）；
- 非 x86_64 架构（win32/arm64 等）自动解析；
- Go API / 插件 / JSON-RPC / C ABI；私有制品库；后台自动更新（用户主动 `update` 已支持）。

详见 [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md) 与 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)。
