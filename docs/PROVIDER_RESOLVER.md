# Provider Resolver 细则

> 状态：Phase 0 契约补充
> 日期：2026-11-10
> 配套：[`PROVIDERS.md`](PROVIDERS.md)、[`ERROR_CODES.md`](ERROR_CODES.md)、[`CLI_SCHEMA.md`](CLI_SCHEMA.md)

---

## 1. 目标

解决审计中的 P1-12：`PROVIDERS.md` 只定义来源、channel 与 pin 原则，但没有明确 GitHub API、限流、重试、资产匹配、checksums 与自定义 URL 细节。

Resolver 的输出必须是：

```text
具体 release tag
+ 具体 asset
+ asset SHA-256
+ checksums SHA-256（若有）
+ target / variant / linkage
+ redistributable
```

不允许把 `latest` 作为最终记录。

---

## 2. 输入

```text
provider: btbn | bnqdzj-nonfree | custom
channel: stable | daily
target: win64 | linux64
variant: gpl | lgpl | nonfree | nonfree-shared | ...
linkage: static | shared | unknown
```

由当前 `GOOS` / `GOARCH` 推导 target：

| Go 环境 | target |
|---|---|
| `windows/amd64` | `win64` |
| `linux/amd64` | `linux64` |

其他 target 当前不支持，返回 `OMF-E3003` 或 `OMF-E3004`。

---

## 3. Channel 支持矩阵

| provider | stable | daily | 说明 |
|---|---|---|---|
| `btbn` | 支持 | 支持 | stable 为当前 stable line（示例 `n9.0`）；daily 为 `latest` / autobuild |
| `bnqdzj-nonfree` | 视上游 | 支持 | 若上游无稳定 release line，`stable` 返回 `OMF-E3004` |
| `custom` | 由 manifest 声明 | 由 manifest 声明 | direct URL 无 channel 概念，可标 `pinned` |

---

## 4. GitHub API

### 4.1 端点

- daily / latest：

```text
GET /repos/{owner}/{repo}/releases/latest
```

- stable / pinned tag：

```text
GET /repos/{owner}/{repo}/releases/tags/{tag}
GET /repos/{owner}/{repo}/releases
```

- 资产下载：

```text
browser_download_url
```

### 4.2 认证

- 优先匿名；
- 若设置 `GITHUB_TOKEN`，放入 `Authorization: Bearer`；
- token 只在内存/环境变量，不写日志、不写 journal、不写 artifact；
- 日志只显示“authenticated/anonymous”，不显示 token。

### 4.3 限流与重试

处理：

- HTTP 403 / 429；
- `X-RateLimit-Remaining: 0`；
- `Retry-After`；
- 网络超时；
- 5xx。

策略：

- 最大尝试次数：3；
- 指数退避；
- 尊重 `Retry-After`；
- 超过上限返回 `OMF-E3006`；
- 其他 API 失败返回 `OMF-E3007`。

实现层使用 `net/http` + `github.com/sethvargo/go-retry`；不引入过旧的 retry 库。

### 4.4 下载限制

- 只允许 HTTPS；
- 禁止 `http://` / `file://`；
- 最大重定向次数：5；
- 连接/读取超时可配置；
- 最大 asset 大小可配置；
- 超过限制返回 `OMF-E4003` / `OMF-E4004`；
- 下载到项目管理的 `downloads/` 临时文件，不直接写目标目录。

---

## 5. 资产匹配

匹配顺序：

1. target；
2. variant；
3. linkage；
4. 压缩格式：Windows zip、Linux tar.xz 优先；
5. 排除源码包、调试符号、checksums 文件本身；
6. 若多个资产匹配，按 provider 规则或用户显式参数选择，不随机选。

资产名识别不能写死单一文件名；以 release API 返回的 `assets[].name` 为准。

常见形态：

```text
ffmpeg-n9.0-latest-win64-gpl-9.0.zip
ffmpeg-n9.0-latest-linux64-gpl-9.0.tar.xz
ffmpeg-master-latest-win64-gpl.zip
ffmpeg-master-latest-linux64-gpl.tar.xz
```

BnqDzj nonfree：

```text
nonfree
nonfree-shared
win64
linux64
winarm64
linuxarm64
```

ARM 当前不作为本项目 target。

---

## 6. Checksums

### 6.1 查找

优先查找名为：

```text
checksums.sha256
*checksums*.sha256
*.sha256
```

的文件，具体以 provider 资产清单为准。

若找不到可信 checksums 文件：

- `resolve` 可返回候选；
- `download` 前必须要求用户提供 `--sha256`；
- 否则返回 `OMF-E3005` / `OMF-E3009`。

### 6.2 解析格式

支持：

```text
<64-lowercase-hex>  <filename>
<64-lowercase-hex> *<filename>
```

规则：

- 只接受 64 位小写 hex；
- 文件名必须做干净路径检查，拒绝绝对路径与 `..`；
- 只匹配目标 asset name；
- 不匹配返回 `OMF-E4002`。

### 6.3 校验

- 流式计算本地 asset SHA-256；
- 与 pin 的 `asset_sha256` 比较；
- 再与 checksums 文件中的目标行比较；
- 两者都通过才写 `artifact.json`；
- 失败删除临时文件，不保留半下载文件。

---

## 7. Release Pin

解析结果写入 `release` 对象：

```json
{
  "schema": "onemoreffmpeg.release/v1",
  "provider": "btbn",
  "repository": "FFmpeg-Builds/FFmpeg-Builds",
  "channel": "stable",
  "release_tag": "autobuild-...",
  "published_at": "...",
  "asset_name": "...",
  "asset_url": "...",
  "asset_sha256": "...",
  "checksums_url": "...",
  "checksums_sha256": "...",
  "variant": "gpl",
  "linkage": "static",
  "target": "win64",
  "license_hint": "gpl",
  "redistributable": true
}
```

禁止：

- `release_tag = "latest"`；
- `asset_url` 为最终版本记录；
- 用发布时间代替 tag；
- 用 provider 名代替 release；
- 不记录 asset SHA-256。

`RELEASE_NOT_PINNED` 对应 `OMF-E3010`。

---

## 8. 自定义 URL

### 8.1 直接资产 URL

必须提供：

```text
--sha256 <64-hex>
```

可选：

```text
--checksums-url <https-url>
```

规则：

- 只允许 HTTPS；
- 禁止 `file://`；
- 禁止本地路径作为 provider；
- 本地目录用 `detect --dir`；
- 非法返回 `OMF-E3008`；
- 缺 hash 返回 `OMF-E3009`。

### 8.2 GitHub Release 仓库 URL

解析为：

```text
owner/repo
```

然后走 GitHub API；仍要求具体 release tag + asset SHA-256。

### 8.3 自定义 provider manifest

建议字段：

```toml
schema = "onemoreffmpeg.provider/v1"
id = "my-ffmpeg"
repository = "owner/repo"       # 或 url
url = "https://example.com/ffmpeg.zip"
sha256 = "..."
variant = "gpl"
linkage = "static"
license_hint = "gpl"
redistributable = true
```

自定义 provider 不得复用内置 id：

```text
btbn
bnqdzj-nonfree
```

---

## 9. nonfree 合规

BnqDzj nonfree 资产：

- `redistributable = false`；
- `license_hint = "nonfree"`；
- 不进入 public GitHub Release；
- 不上传为公开 CI artifact；
- `install.json` 必须记录 `redistributable = false`；
- CLI 输出/日志显示 `OMF-W1006`；
- 不把二进制提交到 Git。

---

## 10. 与 VideoDamage C1 的关系

只有满足：

```text
provider = "btbn"
channel = "stable"
release_line = "<resolver 解析到的 stable line>"  # 当前示例为 n9.0
variant = "gpl"
linkage = "static"
target = 当前主机
```

才允许通过 CLI 产物生成 VideoDamage C1 的 `ffmpeg-trio.lock`。

BnqDzj nonfree / daily / shared 资产不能冒充 C1。

---

## 11. 错误码

主要使用：

```text
OMF-E3001  PROVIDER_UNKNOWN
OMF-E3002  RELEASE_NOT_FOUND
OMF-E3003  ASSET_NOT_FOUND
OMF-E3004  CHANNEL_UNSUPPORTED
OMF-E3005  CHECKSUMS_FILE_MISSING
OMF-E3006  GITHUB_RATE_LIMITED
OMF-E3007  GITHUB_API_FAILED
OMF-E3008  CUSTOM_URL_INVALID
OMF-E3009  CUSTOM_SHA256_REQUIRED
OMF-E3010  RELEASE_NOT_PINNED
```

完整注册表见 [`ERROR_CODES.md`](ERROR_CODES.md)。
