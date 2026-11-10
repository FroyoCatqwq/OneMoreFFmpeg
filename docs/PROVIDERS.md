# Provider、来源与 Release Pin

> 状态：Phase 0 设计
> 目标：支持内置来源、自定义 URL、daily / stable，同时保证可复现
> 当前外部集成：其他项目只通过调用 CLI 可执行文件读取产物文件；不通过 Go API / 插件

---

## 1. 内置来源

| ID | 仓库 | 角色 | 备注 |
|---|---|---|---|
| `btbn` | `FFmpeg-Builds/FFmpeg-Builds` | GPL / LGPL / shared 预构建 | `BtbN/FFmpeg-Builds` 指向同一仓库 |
| `bnqdzj-nonfree` | `BnqDzj/FFmpeg-Builds-nonfree` | nonfree / nonfree-shared 预构建 | 二进制不可公开再分发 |

重要：

- 用户提到的 `bnqdzj/nonfree` 不是有效 GitHub 仓库路径；实际仓库是 `BnqDzj/FFmpeg-Builds-nonfree`。
- BtbN 当前公开 release 不提供 nonfree 资产；nonfree 来源由 BnqDzj 提供。
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
| `stable` | 稳定 release line，例如 BtbN `n9.0` | 必须解析为具体 release tag |
| `daily` | 每日/最新浮动 release，例如 `latest` / autobuild | 必须解析为具体 release tag，不允许记录 `latest` |

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
PublishedAt
AssetName
AssetURL
AssetSHA256
ChecksumsURL
ChecksumsSHA256
Variant
Linkage
Target
LicenseHint
```

安装记录必须包含以上字段；缺一不可。

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
- `stable` 优先匹配 `n9.0` release line 与稳定资产；
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
  --variant gpl \
  --linkage static
```

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

---

## 7. 下载流程

```text
resolve
  ↓
选择 asset
  ↓
下载 checksums.sha256（若存在）
  ↓
下载 asset 到 downloads/临时文件
  ↓
SHA-256 校验
  ↓
原子移动到 downloads/<asset-sha256>/
  ↓
生成 Artifact 记录
```

不自动解压，不自动安装，不自动改 PATH。

下载产物记录：

```json
{
  "schema": "onemoreffmpeg.artifact/v1",
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
release_line = "n9.0"
channel = "stable"
variant = "gpl"
linkage = "static"
target = 当前主机
```

BnqDzj nonfree / daily / shared 资产不能冒充 VideoDamage core 的 C1 锁文件；它们属于 OneMoreFFmpeg 的通用安装能力，或 VideoDamage Extension Pack 的未来能力。
