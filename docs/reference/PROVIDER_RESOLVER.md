# Provider 解析细则

> 状态：v1.1.2-pre2 现行；只支持**自定义 URL provider**（无内置 provider、无 ffmpeg.org 版本目录）
> CLI 输出见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)；错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)

---

## 1. 输入与规范化

`resolve` 的输入是一个已登记的 provider ID。`NormalizeRequest` 做以下处理：

| 字段 | 规则 |
|---|---|
| `provider` | 必填（无默认值）；小写、去首尾空格 |
| `channel` | 固定 `daily`；传 `stable` 返回 `OMF-E3004` |
| `target` | 默认当前平台；仅 `win64` / `linux64`，其它 `OMF-E3003` |
| `variant` | 默认 `gpl`；`gpl` / `lgpl` / `nonfree` |
| `linkage` | 默认 `static`；`static` / `shared` |
| `release-line` / `release-tag` | 不接受，返回 `OMF-E7001` |
| `allow-nonfree` | nonfree 变体必填，缺失 `OMF-E7001` |

## 2. 解析流程

1. 查 `providers.toml`：`provider id is not registered` → `OMF-E3001`；
2. 取登记的 `asset_url`（https）与 `asset_sha256`（64 hex）；
3. 若登记了 `checksums_url`：用 https-only、最多 5 次重定向、4 MiB 上限拉取文件，按 `checksums.sha256 → *checksums*.sha256 → *.sha256` 查找目标行，计算文件自身的 SHA-256 填入 `checksums_sha256`；内容不一致返回 `OMF-E4002`；
4. 组装 `onemoreffmpeg.release/v1` pin（`provider=custom`、`repository=custom`、`channel=daily`、`release_tag=direct-<sha16>`、`pinned_at=<now>`）；
5. 运行时字段级校验（非法 → `OMF-E7003`），然后输出；**不下载资产**。

## 3. 校验来源

- **权威来源**：登记的 `asset_sha256`。下载层强制按此校验，不匹配 → `OMF-E4002`；
- **第二信源**：可选 `checksums_url` 的独立校验文件；两者不一致 → `OMF-E4002`；
- checksums 拉取失败但 `asset_sha256` 存在时，降级为仅 digest（不写 `checksums_url` / `checksums_sha256`）。

## 4. 下载与 pin 的关系

`download --release <pin.json>` 与 `install --release <pin.json>` 只消费 `onemoreffmpeg.release/v1` 中的 `asset_url` + `asset_sha256`，与 pin 的生成方式无关。因此即便将来有其它来源，只要产出同 schema 的 pin 就能被消费。

## 5. 安全约束

- `asset_url` 与 `checksums_url` 必须 https，禁止 userinfo（`user:pass@host`）；重定向禁止 https→http 降级；
- checksums 文件读取限 4 MiB，防止内存耗尽；
- 资产名经 `safeAssetName` 校验（无路径分隔符、无 `..`）；
- 所有路径写入受 `internal/lock` 的 OS 级跨进程锁保护。

## 6. 已移除的能力

以下曾属于 v1.0.x 的内置来源能力，现已全部移除：

- btbn / bnqdzj-nonfree 内置 provider 与 GitHub release 目录；
- ffmpeg.org 官网版本列表（`release list --channel stable`）；
- daily/stable 双通道与 `--release-line` / `--release-tag` / `--snapshot`；
- `release list` / `release install` 命令。

详见 [`PROVIDERS.md`](PROVIDERS.md) 的当前来源模型。
