# Provider、来源与 Release Pin

> 状态：v1.1.1 现行；CLI 外部契约冻结（18 命令 / 17 个 `/v1` schema）
> 重要：**项目没有内置 provider**，也不读取 ffmpeg.org 版本目录。所有 FFmpeg 来源都是用户自己登记的 URL + SHA-256。
> 解析细则：[`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)；CLI 输出见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)；错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)

---

## 1. 来源模型

唯一来源类型：**自定义 URL**（custom）。一个 provider 记录：

| 字段 | 说明 |
|---|---|
| `id` | 用户自定义 ID，`^[a-z0-9][a-z0-9-]{0,63}$`，不能是保留字 `custom` |
| `url` | 资产下载地址，必须 https；下载时强制 SHA-256 校验 |
| `sha256` | 资产 SHA-256（64 位 hex） |
| `license_hint` | `gpl` / `lgpl` / `nonfree` 等 |
| `checksums_url` | 可选：独立 checksums 文件 URL（https），resolve 时拉取校验 |

登记：`provider add --id my-ffmpeg --url https://.../ffmpeg.zip --sha256 <64hex> --apply`。查看：`provider list --json`（未登记时为空数组）。

## 2. 解析与 pin

```bash
omf resolve --provider my-ffmpeg --target win64 --json > release.json
omf download --release release.json --json
omf install --release release.json --apply --json
```

- `resolve` 用登记的 `asset_url` / `asset_sha256` 产出 `onemoreffmpeg.release/v1` pin；channel 固定 `daily`；
- 若登记 `checksums_url`，resolve 会拉取并校验 `checksums_sha256`（https-only、限流 4 MiB）；
- `--release-line` / `--release-tag` / `--channel stable` 不再支持（`OMF-E7001` / `OMF-E3004`）；
- `resolve` 只产出 pin，不下载资产；`asset_url` 必须为 https；
- 输出前做运行时字段级校验，非法 pin 返回 `OMF-E7003`。

## 3. 隔离与回滚

- 下载位于 `<data-root>/download/<sha16>/`，文件通过 SHA-256 校验后才落盘；
- 安装到 `<data-root>/apps/<provider>/<channel>/<sha16>/`，`current` 指针原子切换；
- 写入默认 dry-run，`--apply` 才落盘；所有写事务有 journal，可用 `journal rollback --journal <id> --apply` 恢复；
- 不覆盖非本项目管理的文件（`OMF-E6007`）。

## 4. 许可与 nonfree

- `variant`：`gpl` / `lgpl` / `nonfree`，`linkage`：`static` / `shared`；
- nonfree 变体需 `--allow-nonfree`，输出 `OMF-W1006`，且**仅本地使用、不可再分发**；
- 默认（`--variant gpl`）不因 provider 而改变——没有"nonfree-only 内置 provider"了。

## 5. 明确不做

- 内置 provider（btbn / bnqdzj-nonfree 已移除）、ffmpeg.org 版本目录、`release list/install` 命令；
- 包管理器识别 / Scoop / dnf / RPM 安装本工具；
- Go API / 插件 / JSON-RPC / C ABI；私有制品库；后台自动更新（用户主动 `update` 已支持）。

详见 [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md) 的解析规则与 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) 的字段契约。
