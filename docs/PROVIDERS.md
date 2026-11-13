# Provider、来源与 Release Pin

> 状态：Phase 0 设计
> 目标：支持内置来源、自定义 URL、daily / stable，同时保证可复现
> 当前外部集成：其他项目只通过调用 CLI 可执行文件读取产物文件；不通过 Go API / 插件
> 解析细则：[`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)；CLI 输出见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)；错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)

---

## 1. 内置来源

| ID | 仓库 | 角色 | 备注 |
|---|---|---|---|
| `btbn` | `FFmpeg-Builds/FFmpeg-Builds` | GPL / LGPL / shared 预构建 | `BtbN/FFmpeg-Builds` 指向同一仓库 |
| `bnqdzj-nonfree` | `BnqDzj/FFmpeg-Builds-nonfree` | nonfree / nonfree-shared 预构建 | 二进制不可公开再分发 |

重要：

- 用户提到的 `bnqdzj/nonfree` 不是有效 GitHub 仓库路径；实际仓库是 `BnqDzj/FFmpeg-Builds-nonfree`。
- 截至 2026-11-10，BtbN 当前公开 release 不提供 nonfree 资产；nonfree 来源由 BnqDzj 提供。实际以 resolver 查询为准。
- OneMoreFFmpeg 只保存仓库 URL、release 元数据、资产名与 SHA-256；不把 FFmpeg 二进制提交到 Git。
- public GitHub Release 不得包含 BnqDzj nonfree 二进制。

---

## 2. Channel 模型

对外暴露两个抽象 channel：

```text
stable
daily
```

含义：

| Channel | 含义 | 运行时要求 |
|---|---|---|
| `stable` | 稳定 release line，例如 BtbN `n9.0` | 优先 pin 到具体 tag；上游把稳定线资产放在 `latest` release 时，允许 `release_tag=latest` 但必须带 `release_id` + `published_at` + `asset_sha256`（D1） |
| `daily` | 每日/最新浮动 release，例如 autobuild | 必须解析为具体 autobuild tag，不允许只记录 `latest` |

支持矩阵：

| provider | stable | daily | 说明 |
|---|---|---|---|
| `btbn` | 支持 | 支持 | stable 为当前 stable line（示例 `n9.0`）；daily 为 `latest` / autobuild |
| `bnqdzj-nonfree` | 视上游 | 支持 | 若上游无稳定 release line，`stable` 返回 `OMF-E3004` |
| `custom` | 由 manifest 声明 | 由 manifest 声明 | direct URL 没有 channel 概念；输出 `channel=daily` 以匹配枚举，pin 由 `release_tag` + `asset_sha256` 表达，不写 `pinned` |

GitHub API、限流、重试、资产匹配、checksums 与 pin 细则见 [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)。

禁止：

- 把 `latest` 写进安装记录；
- 把 URL 当作版本；
- 每次启动重新解析同一安装；
- 静默从 stable 切到 daily；
- 混用不同 provider 的 asset。

如果某 provider 不支持某 channel，返回明确错误 `OMF-E3004`，不做猜测。

---

## 3. Release 记录

解析完成后生成：

```text
Provider
Channel
Repository
ReleaseTag
ReleaseID        # 可选；release_tag=latest 时必填
PublishedAt
PinnedAt         # 可选；本次 pin 时间
AssetName
AssetURL
AssetSHA256
ChecksumsURL
ChecksumsSHA256
Variant
Linkage
Target
LicenseHint
Redistributable
```

安装记录必须包含以上字段；缺一不可，但 `ChecksumsURL` / `ChecksumsSHA256` 有例外：GitHub Release 来源必须同时记录两者；自定义直链只有 `asset_sha256` pin 时这两个字段可以为空（`CLI_SCHEMA.md` §4.7 与 `contracts` 的校验一致）。

pin 规则（D1）：

```text
release_tag != latest  -> tag 本身不可变，release_id 可选
release_tag == latest  -> 必须同时有 release_id、published_at、asset_sha256
缺少任一 pin 字段      -> OMF-E3010
```

`pinned_at` 只用于审计“什么时候解析到该 release”，不参与完整性判定。

`Linkage`：

```text
static
shared
unknown
```

v1 不强制所有 provider 都 static；但 VideoDamage 消费侧要求 static GPL 时，由 VideoDamage 自己校验或由 OneMoreFFmpeg 生成符合其 C1 的锁文件。

---

## 4. BtbN 适配器

仓库：

```text
https://github.com/FFmpeg-Builds/FFmpeg-Builds
```

解析规则：

- 通过 GitHub Releases API 获取 release；
- `stable` 匹配当前解析到的 stable line（示例 `n9.0`）与稳定资产；
- `daily` 匹配 `latest` / autobuild 资产；
- 资产按 `GOOS`/`GOARCH` 过滤：
  - `windows/amd64` → `win64`
  - `linux/amd64` → `linux64`
- variant 从资产名识别：
  - `gpl`
  - `lgpl`
  - `gpl-shared`
  - `lgpl-shared`
- 必须找到 `checksums.sha256` 或等价校验文件；
- 如果只有 direct asset 而无可信 checksum，则 `resolve` 成功但 `download` 失败。

BtbN 常见资产名形态：

```text
ffmpeg-n9.0-latest-win64-gpl-9.0.zip
ffmpeg-n9.0-latest-linux64-gpl-9.0.tar.xz
ffmpeg-master-latest-win64-gpl.zip
ffmpeg-master-latest-linux64-gpl.tar.xz
```

实际以 release API 返回资产为准，不写死单一文件名。

---

## 5. BnqDzj nonfree 适配器

仓库：

```text
https://github.com/BnqDzj/FFmpeg-Builds-nonfree
```

解析规则：

- 通过 GitHub Releases API 获取 release；
- `daily` 支持 `latest` / autobuild；
- `stable` 若上游没有稳定 release line，则返回 unsupported；
- 资产识别：
  - `nonfree`
  - `nonfree-shared`
  - `win64` / `linux64` / `winarm64` / `linuxarm64`
- 记录 `license_hint = "nonfree"`；
- `install.json` 明确标记 `redistributable = false`；
- 公开 CI 可以跑元数据解析，但不把 nonfree 资产上传为 artifact。

显式同意（D5）：

```text
resolve --provider bnqdzj-nonfree ...             -> OMF-E7001（缺少同意）
resolve --provider bnqdzj-nonfree --allow-nonfree -> 成功 + OMF-W1006
```

`--allow-nonfree` 只对 nonfree variant 生效；缺失时不得发起下载，也不得静默降级到其他 provider。

法律边界：

- nonfree 二进制不能随 GPL 项目公开分发；
- 不能放进 public GitHub Release；
- 不能作为 Docker 镜像公开层；
- 可以在用户本机下载并使用，但必须显示 license 风险提示。

---

## 6. 自定义 URL Provider

用户可以添加：

```text
one-more-ffmpeg provider add \
  --id my-ffmpeg \
  --url https://example.com/ffmpeg.zip \
  --sha256 <64-hex> \
  --apply
```

`provider add` 只登记来源与身份（`id` / `url` / `sha256` / `status`）；`variant` / `linkage` / `target` 属于 `resolve` 的参数，不在 add 阶段固定，避免同一来源无法解析不同变体。

支持两类 URL：

1. **直接资产 URL**
   必须提供 `--sha256`；可附带 `--checksums-url`。

2. **GitHub Release 仓库 URL**
   走 GitHub API；仍要求 release tag + asset SHA-256；不允许只写 `latest`。

约束：

- 只允许 `https://`；
- 禁止 `file://` / `http://` / 本地 UNC 作为远程来源；本地目录用 `detect` 而不是 provider；
- 重定向次数限制；
- 下载超时；
- 文件大小上限；
- 必须显示来源、license 与 checksum；
- 不允许自动执行下载包里的脚本；
- 不允许把自定义 provider 标记为 `btbn` / `bnqdzj-nonfree`。

存储与命令（已实现）：

```text
provider add --id my-ffmpeg --url https://example.com/ffmpeg.zip \
             --sha256 <64-hex> [--checksums-url https://...] \
             [--license-hint gpl|lgpl|nonfree|custom] --apply
```

- 默认 dry-run；`--apply` 才写入，`--dry-run` 可强制预览；
- 写入 `<data-root>/providers.toml`（schema `onemoreffmpeg.providers/v1`），临时文件 + rename；保存前重新校验，拒绝重复/保留 id 与非法 URL；
- 全链路 `Load → AddCustom → Save` 使用 `providers.toml.lock` 跨进程锁；
- 重复 id 直接失败，不覆盖既有 pin；
- URL 必须 https 且指向解压器支持的 `.zip` / `.tar.xz` / `.tar.gz` / `.tgz` / `.tar.zst`；`.7z` 不受支持并在 add 阶段拒绝；不接受 userinfo、目录 URL 或带查询参数的文件名推导；
- `resolve --provider my-ffmpeg` 输出 `repository=custom`、`release_tag=direct-<sha256[:12]>`、`published_at=pinned_at=解析时间`；
- 自定义 provider 的 `redistributable` 固定 false，`license_hint` 取 `--license-hint`（默认 `custom`）；
- 自定义 provider 在 `provider list` 中 `channels=["daily"]`；`resolve --channel stable` 返回 `OMF-E3004`；`--release-tag` / `--release-line` 返回 `OMF-E7001`；
- `checksums_url` 在 resolve 时拉取：计算 `checksums_sha256` 并校验目标行，与 `sha256` 不一致返回 `OMF-E4002`。

---

## 7. 下载流程

```text
resolve（digest / checksums 双重来源，产出 pin）
  ↓
download 读取 release（--release 或在线 resolve）
  ↓
已有 <sha16>/<asset> 且摘要匹配 -> 直接复用
  ↓
下载 asset 到同目录 *.part，边写边算 SHA-256
  ↓
与 pin 比较（不匹配删除临时文件）
  ↓
存在 checksums_url 时再校验其自身 SHA-256 与目标行
  ↓
rename 到 downloads/<sha256[:16]>/<asset>
  ↓
写同目录 artifact.json
```

失败时删除 `.part` 与空目录；不自动解压，不自动安装，不自动改 PATH。

校验源降级规则：

```text
asset.digest 存在，checksums.sha256 拉取失败 -> 仅用 digest 完成 resolve（不写 checksums_url/sha256）
asset.digest 缺失，checksums.sha256 可用     -> 用 checksums 行
两者都存在但不一致                           -> OMF-E4002
两者都不可用                                 -> OMF-E3005
```

下载阶段仍会用 release 里记录的 `checksums_url` 做第二次校验（存在时）。

下载产物记录：

```json
{
  "schema": "onemoreffmpeg.artifact/v1",
  "artifact_id": "art-0123456789abcdef",
  "provider": "btbn",
  "release_tag": "autobuild-...",
  "asset_name": "...",
  "asset_sha256": "...",
  "local_path": "...",
  "downloaded_at": "...",
  "verified": true
}
```

---

## 8. 安全与复现

- GitHub API 可匿名，也可使用 `GITHUB_TOKEN`；
- token 只进内存/环境变量，不写日志；
- API 限流要退避并报错；
- 不缓存 floating `latest` 的解析结果超过配置 TTL；
- 安装记录永远 pin 具体 release；
- `install.json` 写入 provider、release tag、asset SHA-256、三件套二进制 SHA-256；
- 每次升级重新生成记录，不复用旧 hash；
- 不下载到系统临时目录后直接执行，先校验再解压。

---

## 9. 与 VideoDamage C1 的关系

当前只通过 CLI 产物文件集成，不通过 Go API / 插件：

OneMoreFFmpeg 可生成：

```text
ffmpeg-trio.lock
```

但要满足 VideoDamage C1，必须额外保证：

```text
provider = "btbn"
release_line = "<resolver 解析到的 stable line>"  # 当前示例为 n9.0
channel = "stable"
variant = "gpl"
linkage = "static"
target = 当前主机
```

BnqDzj nonfree / daily / shared 资产不能冒充 VideoDamage core 的 C1 锁文件；它们属于 OneMoreFFmpeg 的通用安装能力，或 VideoDamage Extension Pack 的未来能力。
