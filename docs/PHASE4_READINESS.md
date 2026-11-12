# Phase 4 准备与准入清单

> 日期：2026-11-11
> 前置：[`PHASE3_CODE_REVIEW.md`](PHASE3_CODE_REVIEW.md)（Phase 3 已冻结）
> 目标：进入 Phase 4（Provider / Resolve / Download）前，把契约冲突、代码缺口、真实上游事实与决策项一次列清
> 原则：先 pin 后下载；只读上游 API；不自动安装；不写 PATH；nonfree 二进制不进公开产物

---

## 1. 判定

Phase 3 已提供可复用的进程边界（`ffexec`）、错误码（`apperrors`）、只读安装模型（`model.Installation`）与能力缓存。Phase 4 的**契约文本基本齐备**（`PROVIDERS.md`、`PROVIDER_RESOLVER.md`、`CLI_SCHEMA.md`、`ERROR_CODES.md`、`schemas/examples/`）。

截至 2026-11-11 的代码状态：

```text
internal/paths            data root / cache / downloads / config 路径        完成
internal/checksum         流式 SHA-256 + checksums.sha256 解析               完成
internal/model/provider   ProviderSummary / Release / Artifact              完成
internal/config           [network] / [download] + 校验                     完成
internal/contracts        Phase 4 示例字段级校验                            完成
internal/provider         注册表 + 内置 provider 元数据 + provider list     完成
internal/provider/github  Releases API 客户端（重试/限流/重定向/脱敏）      完成
internal/download         下载与 artifact.json                              未开始
resolve / provider add    provider 资产匹配与 pin                           未开始（依赖 D1/D5）
```

结论：

> **Phase 4 的基础设施与全部决策（D1–D7）已就绪：路径、checksum、模型、配置、契约校验、HTTP 客户端、provider 注册表、`provider list`、D1 pin 字段与 D5 同意策略。剩余工作只有 `resolve` 资产匹配、`download` 落盘与 `provider add`，不再有阻塞性决策。**

### 1.1 已落地的基础件（2026-11-11）

| 项 | 位置 | 状态 |
|---|---|---|
| data root / cache / downloads | `internal/paths` | 完成，`capabilities` 已复用 |
| 流式 SHA-256 + checksums 解析 | `internal/checksum` | 完成，含真实格式测试 |
| Release / Artifact / ProviderSummary | `internal/model/provider.go` | 完成，nil-safe JSON |
| `[network]` / `[download]` 配置 | `internal/config` | 完成，默认值与校验 |
| Phase 4 示例字段级校验 | `internal/contracts` | 完成，release / artifact / provider list / add |
| provider 注册表 + `provider list` | `internal/provider`、`internal/cli/provider.go` | 完成，离线运行 |
| GitHub HTTP 客户端 | `internal/provider/github` | 完成，含 403/429/5xx/重定向/大小上限/脱敏测试 |

---

## 2. 已有可直接依赖的契约

| 契约 | 内容 | 状态 |
|---|---|---|
| `PROVIDERS.md` §2 | `stable` / `daily` channel 语义 | 已按 D1 修正：stable 允许 latest + pin 字段 |
| `PROVIDERS.md` §3 | Release 记录 15 个字段（含 `schema` 共 16 个 JSON 字段） | 与 `CLI_SCHEMA.md` §4.7 一致 |
| `PROVIDERS.md` §6 | 自定义 URL 必须 HTTPS + `--sha256` | 可直接实现 |
| `PROVIDER_RESOLVER.md` §4.3 | 3 次尝试、指数退避、尊重 `Retry-After` | 依赖 `sethvargo/go-retry`（已在 `go.mod`） |
| `PROVIDER_RESOLVER.md` §4.4 | 仅 HTTPS、最多 5 次重定向、大小上限 | 可直接实现 |
| `PROVIDER_RESOLVER.md` §6.2 | checksums 行格式 | 已用真实文件核对，见 §3.3 |
| `PROVIDER_RESOLVER.md` §7 | Release pin 结构与禁止项 | 已按 D1 改写：latest + pin 字段合法，缺字段 E3010 |
| `CLI_SCHEMA.md` §4.5–4.8 | `provider list` / `provider add` / `resolve` / `download` 的 data 字段 | 示例齐全 |
| `ERROR_CODES.md` §6–7 | `E3001–E3010`、`E4001–E4006` | 全部已注册，无需新增 |
| `model.KnownDataSchemas` | 四个 Phase 4 schema 均已登记 | `contracts.ValidateData` 可直接用 |

---

## 3. 真实上游事实核查（2026-11-11，匿名 GitHub API）

> 以下为实际 API 返回，用于校正 `PROVIDERS.md` / `PROVIDER_RESOLVER.md` 中的示例假设。

### 3.1 BtbN（`FFmpeg-Builds/FFmpeg-Builds`）

Release 列表：

```text
tag=latest                          name="Latest Auto-Build (2026-11-11 12:39)"  49 assets
tag=autobuild-2026-11-11-12-39      name="Auto-Build 2026-11-11 12:39"            49 assets
tag=autobuild-2026-11-10-12-39      ...
```

`latest` 的资产命名（节选）：

```text
checksums.sha256
ffmpeg-master-latest-win64-gpl.zip
ffmpeg-master-latest-win64-gpl-shared.zip
ffmpeg-master-latest-win64-lgpl.zip
ffmpeg-master-latest-win64-lgpl-shared.zip
ffmpeg-n9.0-latest-win64-gpl-9.0.zip
ffmpeg-n9.0-latest-win64-gpl-shared-9.0.zip
ffmpeg-n8.1-latest-win64-lgpl-8.1.zip
（同结构覆盖 linux64 / winarm64 / linuxarm64）
```

`autobuild-*` 的资产命名（节选）：

```text
ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip
ffmpeg-N-128790-g9bee1e603c-linux64-gpl-shared.tar.xz
```

关键结论：

1. **`latest` 同时承载 master 与 `n9.0` / `n8.1` 稳定线资产**；
2. **`autobuild-*` 只承载 master / `N-*` 构建，不含 `n9.0` 资产**；
3. 因此 `stable` 无法既“使用 n9.0 资产”又“pin 到具体 autobuild tag”——这是 D1 的根因；
4. `checksums.sha256` 在 `latest` 与每个 `autobuild-*` 中都存在，BtbN 的 `OMF-E3005` 只在异常时出现；
5. 文档中的 `ffmpeg-n9.0-latest-win64-gpl-9.0.zip` 示例与真实资产一致，未被证伪；
6. `docs/schemas/examples/provider-resolve.example.json` 已按 D1 替换为真实可达组合：`channel=stable`、`release_tag=latest`、`release_id=441698707`、`published_at=2026-11-11T13:00:32Z`、`ffmpeg-n9.0-latest-win64-gpl-9.0.zip`（2026-11-11 实测）。

### 3.2 BnqDzj nonfree（`BnqDzj/FFmpeg-Builds-nonfree`）

```text
tag=latest                      "Latest Auto-Build (2026-11-10 19:41)"   9 assets
tag=autobuild-2026-11-10-19-41  "Auto-Build 2026-11-10 19:41"             9 assets

checksums.sha256
ffmpeg-master-latest-win64-nonfree-shared.zip
ffmpeg-master-latest-win64-nonfree.zip
ffmpeg-N-128756-gd0b9c63c9f-linux64-nonfree.tar.xz
（同结构覆盖 linuxarm64 / winarm64）
```

关键结论：**只有 master/N-* 资产，没有稳定线**；`stable` 必须返回 `OMF-E3004`，与 `PROVIDERS.md` §2 一致。

### 3.3 `checksums.sha256` 实际格式

```text
f1b10f0edd0f2e52b084a721ab644c064d3e618bdfce9198299073cd23b678e9  ffmpeg-master-latest-win64-gpl-shared.zip
```

- 64 位小写 hex + 两个空格 + 文件名 + LF；
- 文件约 5.3 KB / 49 行；
- 与 `PROVIDER_RESOLVER.md` §6.2 完全一致，可直接实现解析器。

---

## 4. 决策记录

| 编号 | 问题 | 决定 | 状态 |
|---|---|---|---|
| D1 | `stable` 如何 pin | 允许 `release_tag=latest` **仅当**同时记录 `release_id` + `published_at` + `asset_sha256`；Release schema 增加可选 `release_id` / `pinned_at`；`E3010` 仅在缺 pin 字段时触发；具体 tag 不需要 `release_id` | **已定（A）并落文**：`PROVIDERS.md` §2/§3、`PROVIDER_RESOLVER.md` §7、`CLI_SCHEMA.md` §4.7、`ERROR_CODES.md` §6、`contracts` 校验与示例 |
| D2 | target / variant / linkage 默认值 | target 由 `GOOS/GOARCH` 推导（win64/linux64，其他 `E3003`）；默认 `variant=gpl`、`linkage=static`；显式 flag 覆盖 | 已写入 `CLI_SCHEMA.md` §4.7，待实现 |
| D3 | stable 线选择 | `--release-line master|n8.1|n9.0`；默认在 `latest` 资产中取数值最大的 `nX.Y`，master 只在 daily 默认 | 已写入 `CLI_SCHEMA.md` §4.7，待实现 |
| D4 | data root / downloads 目录 | `ONE_MORE_FFMPEG_HOME`（必须绝对路径）> Windows `%LOCALAPPDATA%\OneMoreFFmpeg` > Linux `$XDG_DATA_HOME/onemoreffmpeg` > `~/.local/share/onemoreffmpeg`；下载到 `<data-root>/downloads/<asset-sha256:16>/<asset-name>` | **已定并落地**：`internal/paths`，见 `ENVIRONMENT.md` §1.1 |
| D5 | nonfree 显式同意 | `bnqdzj-nonfree` 必须显式 `--allow-nonfree`；缺失返回 `OMF-E7001`（退出 2，不新增错误码）；成功时始终 `redistributable=false` + `OMF-W1006` | **已定（A）并落地**：`provider.CheckNonfreeConsent` + 测试；文档见 `PROVIDERS.md` §5、`CLI_SCHEMA.md` §4.7 |
| D6 | `download` 是否需要先 `resolve` | `download` 支持 `--release <artifact-input.json>`（离线）或自带 `resolve` 参数（在线）；两条路径产出同一 `artifact.json` | 已写入方案，待实现 |
| D7 | Windows 进程树 Job Object | 推迟到 Phase 5（写入类命令前）；Phase 4 继续使用 `taskkill` 3s 超时 + `Process.Kill` 兜底；FFmpeg 本身不派生子进程 | **已定（A）并记录**：`ARCHITECTURE.md` §6、`FULL_AUDIT_2026-11-11.md` §5 |

> 契约冻结：D1/D4/D5 已落文并有代码/校验支撑；D2/D3/D6 的 CLI 形态已冻结在 `CLI_SCHEMA.md` §4.7，实现时不得改语义。

---

## 5. 代码缺口与基础件

### G1：model 只有 schema 常量，没有结构体（已完成）

已新增 `internal/model/provider.go`：

```go
type ProviderSummary struct {
    ID              string   `json:"id"`
    Repository      string   `json:"repository"`
    Channels        []string `json:"channels"`
    Variants        []string `json:"variants"`
    Linkages        []string `json:"linkages"`
    Redistributable bool     `json:"redistributable"`
}

type Release struct {
    Schema          string `json:"schema"`
    Provider        string `json:"provider"`
    Repository      string `json:"repository"`
    Channel         string `json:"channel"`
    ReleaseTag      string `json:"release_tag"`
    PublishedAt     string `json:"published_at"`
    AssetName       string `json:"asset_name"`
    AssetURL        string `json:"asset_url"`
    AssetSHA256     string `json:"asset_sha256"`
    ChecksumsURL    string `json:"checksums_url,omitempty"`
    ChecksumsSHA256 string `json:"checksums_sha256,omitempty"`
    Variant         string `json:"variant"`
    Linkage         string `json:"linkage"`
    Target          string `json:"target"`
    LicenseHint     string `json:"license_hint"`
    Redistributable bool   `json:"redistributable"`
}

type Artifact struct { /* CLI_SCHEMA.md §4.8 字段 */ }
```

要求：

- 所有 slice/map 字段实现 nil → `[]`，与 `CapabilitySet.Normalize` 同样的做法；`ProviderListData` / `ProviderSummary` 已实现 `Normalize` + `MarshalJSON`；
- `contracts.ValidateExample` 已增加 release / artifact / provider-list / provider-add 的字段级检查（schema、枚举、HTTPS、SHA-256、RFC 3339、nonfree 不可再分发）；
- 不把 Go 结构体当公共 API，字段名以 JSON 契约为准。

### G2：CLI 命令缺失

需要新增 `internal/cli/provider.go`、`resolve.go`、`download.go`，并在 `root.go` 注册：

```text
provider list
provider add
resolve
download
```

`contracts.knownCommands` 与 `model.KnownDataSchemas` 已就绪，不需要改注册表。

### G3：config 没有网络与下载段（已完成）

`config.Config` 已在 v1 内补齐（strict mode 下新增字段对旧配置向后兼容）：

```toml
schema = "onemoreffmpeg.config/v1"

[network]
timeout = "30s"
max_attempts = 3
github_api = "https://api.github.com"

[download]
max_bytes = 536870912
max_redirects = 5
dir = ""            # 空 = <data-root>/downloads
```

校验规则（已实现）：`timeout` 可解析且 > 0；`max_attempts` 1..10；`github_api` 必须是 https URL；`max_bytes` > 0；`max_redirects` 0..20；`dir` 为空或绝对路径。违反时返回 `OMF-E7005`。

注意：新增字段对旧配置是向后兼容的（缺省走默认值）；但**字段一旦发布不能改语义**，需在 Phase 4 一次定好。

### G4：没有统一路径解析（已完成）

已新增 `internal/paths`：

```go
type Layout struct { DataRoot, CacheDir, DownloadsDir, ConfigFile string }

func Resolve() (Layout, error)
func DataRoot() (string, error)
func CacheDir() (string, error)
func DownloadsDir() (string, error)
func ConfigFile() (string, error)
```

`internal/capabilities` 的 `resolveCacheDir` 已改为复用 `paths.CacheDir()`，避免 Phase 5 再改一遍。

### G5：没有 checksum 工具（已完成）

已新增 `internal/checksum`：

```go
func StreamSHA256(ctx context.Context, r io.Reader) (string, int64, error)
func StreamSHA256Limit(ctx context.Context, r io.Reader, limit int64) (string, int64, error)
func ParseChecksums(r io.Reader, assetName string) (string, error)
```

要求：

- 流式计算，不把资产读进内存；
- 超限返回 `OMF-E4004`；deadline 返回 `OMF-E4001`，用户取消返回 `OMF-E4006`；
- 拒绝绝对路径、`..` 与带目录的 asset 名，但接受 checksums 行里的相对前缀（`./x`、`dist/x`）；
- 输入 hex 大小写均接受并统一为小写；重复且冲突的条目返回 `OMF-E4002`；
- 找不到目标行返回 `OMF-E4002`。

### G6：HTTP 不可注入，无法离线测试（已完成）

`internal/provider/github` 已提供可注入客户端：

```go
type Client struct {
    BaseURL          string        // 默认 https://api.github.com
    HTTPClient       *http.Client  // 可替换；CheckRedirect 使用 len(via) 计数
    Token            string        // 仅内存，不落盘、不进错误
    MaxAttempts      int           // 默认 3
    MaxResponseBytes int64         // 默认 4 MiB
    RetryBase        time.Duration // 默认 500ms，指数退避
    MaxRedirects     int           // 默认 5
    Sleep            func(ctx context.Context, d time.Duration) error
}
```

- 单元测试使用 `httptest.Server`，默认不联网；
- 覆盖 200 / 403 rate limit / 429 + `Retry-After` / 5xx / 重定向上限 / 响应超限 / 取消 / 404；
- 仅允许 https；http 只允许 loopback（测试）；
- 真实网络测试仍留到 `//go:build network`。

### G7：token 脱敏没有落地（已完成）

- `Authorization` 只存在于请求头；
- 错误对象只含状态码与 path，不含 header；
- `TestTokenIsNotLeaked` 断言错误文本、cause、details 均不含 token。

### G8：`license_hint` 来源未定义

`resolve` 的 `license_hint` 不能依赖本机已安装二进制（Phase 4 阶段还没有安装）。应从 provider + variant 推导：

```text
btbn + gpl*        -> gpl
btbn + lgpl*       -> lgpl
bnqdzj-nonfree     -> nonfree
custom             -> 用户显式 --license-hint（默认 custom）
```

Phase 3 的 `-L` 解析不参与 Phase 4 的 `license_hint`，只用于安装后校验。

### G9：限流与重试策略未落地

- `E3006`：HTTP 403/429 且 `X-RateLimit-Remaining: 0` 或响应体含 rate limit 信息；
- 尊重 `Retry-After`（秒数或 HTTP date）；
- 5xx / 网络错误重试 3 次，指数退避；
- 其他 4xx 直接 `E3007`，不重试；
- `golang.org/x/time/rate` 可选，v1 先用退避即可，不引入全局 limiter。

### G10：artifact 缺少 provenance 字段

`CLI_SCHEMA.md` §4.8 的 artifact 只有 `asset_sha256`，但 `PROVIDERS.md` §3 要求安装记录包含 `ChecksumsURL` / `ChecksumsSHA256`。建议 artifact 增加可选字段：

```json
"checksums_url": "...",
"checksums_sha256": "...",
"variant": "gpl",
"linkage": "static",
"license_hint": "gpl",
"redistributable": true
```

否则 Phase 5 生成 `install.json` 时需要重新解析 release。

---

## 6. 建议包结构

```text
internal/paths        data root / downloads / cache
internal/checksum     流式 SHA-256、checksums.sha256 解析
internal/provider     Provider 接口、registry、btbn、bnqdzj-nonfree、custom
internal/provider/github   Releases API client、重试、限流、脱敏
internal/download     asset 下载、大小/重定向限制、原子落盘、artifact 记录
internal/cli          provider.go / resolve.go / download.go
internal/model        provider.go（Release / Artifact / ProviderSummary）
```

依赖方向：

```text
cli → provider → github
            ↘ checksum
cli → download → checksum
cli/provider/download → paths
```

禁止：

- `provider` 写安装目录或 PATH；
- `download` 解压或执行下载内容；
- `checksum` 依赖 CLI；
- 把 nonfree 二进制写入仓库或公开 artifact。

### 6.1 Provider 接口草案

```go
type Request struct {
    Provider   string
    Channel    string // stable | daily
    Target     string // win64 | linux64
    Variant    string // gpl | lgpl | nonfree | nonfree-shared
    Linkage    string // static | shared
    ReleaseTag string // 可选：显式 pin
    ReleaseLine string // 可选：n9.0 / n8.1 / master
}

type Provider interface {
    ID() string
    Summary() model.ProviderSummary
    Resolve(ctx context.Context, req Request) (model.Release, error)
}
```

registry 只注册三个内置 id；custom 通过 `provider add` 写入用户配置，**不得复用内置 id**。

### 6.2 资产匹配算法

```text
1. 排除 checksums.sha256 / *.sha256 / 调试符号 / 源码包
2. 必须命中 target token：win64 或 linux64
3. 必须命中 variant token：gpl | lgpl | nonfree
4. 含 shared token -> linkage=shared，否则 static
5. 包含 nX.Y token -> release_line=nX.Y；含 master -> master；含 N-* -> master 构建
6. 扩展名：win64 -> .zip；linux64 -> .tar.xz
7. 多个命中 -> 按 (release_line 数值, 无 shared 优先) 排序，仍不唯一则报 E3003，不随机选
```

### 6.3 下载流程草案

```text
resolve（或读取 --release）
  → 下载 checksums.sha256（若存在）
  → 解析目标行得到 expected_sha
  → 下载 asset 到 <downloads>/<sha256:16>/.<name>.part
  → 流式 SHA-256
  → 与 release.asset_sha256 和 checksums 行双重比较
  → rename 为最终文件
  → 写 artifact.json
失败：删除 .part，返回 E4001/E4002/E4003/E4004/E4005/E4006
```

---

## 7. 错误码映射

| 场景 | code | 退出 |
|---|---|---:|
| 未注册 provider id | `OMF-E3001` | 2 |
| release/tag 不存在 | `OMF-E3002` | 1 |
| target/variant/linkage 无匹配资产 | `OMF-E3003` | 1 |
| 不支持的 channel | `OMF-E3004` | 2 |
| nonfree provider 未加 `--allow-nonfree` | `OMF-E7001` | 2 |
| 无可信 checksums 且未给 `--sha256` | `OMF-E3005` | 1 |
| GitHub 限流（403/429 + rate limit） | `OMF-E3006` | 1 |
| GitHub API 其他失败 | `OMF-E3007` | 1 |
| 自定义 URL 非 HTTPS / 非法 | `OMF-E3008` | 2 |
| 自定义 direct asset 缺 SHA-256 | `OMF-E3009` | 2 |
| pin 字段缺失（D1 后） | `OMF-E3010` | 1 |
| 下载失败 / 连接中断 | `OMF-E4001` | 1 |
| checksums 不匹配或缺少目标行 | `OMF-E4002` | 1 |
| 重定向超过 5 次 | `OMF-E4003` | 1 |
| 超过 `max_bytes` | `OMF-E4004` | 1 |
| 临时文件写入失败 | `OMF-E4005` | 1 |
| 用户取消 | `OMF-E4006` | 130 |

---

## 8. 测试策略

单元（默认，不联网）：

- `checksum`：真实 `checksums.sha256` 片段、`*name` 形式、绝对路径/`..`、大小写 hex、缺失行；
- 资产匹配：master / n8.1 / n9.0 / nonfree / shared 的真实资产名表驱动；
- GitHub client：`httptest` 覆盖 200 / 403+`Retry-After` / 429 / 500 / 重定向链 / 超过重定向 / 截断 body；
- 下载：`Content-Length` 超限、chunked 超限、SHA 不匹配、ctx 取消（检查 `.part` 被删除）；
- token 脱敏：JSON 与日志均不含 token；
- 自定义 URL：`http://`、`file://`、UNC、缺 `--sha256`；
- 契约：release / artifact / provider-list / provider-add 示例通过 `contracts.ValidateExample` 的字段级校验；该校验当前只作用于示例，Phase 4 实现 `writeResult` 时可复用同一套规则。

集成（`//go:build network`）：

- `resolve --provider btbn --channel daily`；
- `resolve --provider btbn --channel stable --release-line n9.0`；
- `resolve --provider bnqdzj-nonfree --channel daily --allow-nonfree`；
- 只读元数据，不下载大文件；下载集成测试用 `--max-bytes` 限制或跳过。

---

## 9. 安全清单

- [ ] 只允许 `https://`，拒绝 `http://` / `file://` / UNC；
- [ ] `CheckRedirect` 上限 5；
- [ ] 连接/读取超时可配置，默认 30s；
- [ ] `max_bytes` 默认 512 MiB，同时校验 `Content-Length` 与流式计数；
- [ ] `GITHUB_TOKEN` 只从环境读取，不落盘、不进 JSON、不进日志；
- [ ] 不把 asset URL 当作最终版本记录；
- [ ] 不执行下载包内任何脚本或二进制；
- [ ] 下载失败删除 `.part`，不保留半文件；
- [ ] nonfree 不进入公开 Release / CI artifact；
- [ ] `redistributable=false` 必须出现在 release 与 artifact；
- [ ] 文件名做干净路径检查，落盘路径由 SHA-256 决定，不使用上游文件名拼路径。

---

## 10. 实施顺序

| 步骤 | 内容 | 产出 |
|---:|---|---|
| 1 | 落文 D1/D4/D5（`CLI_SCHEMA.md`、`PROVIDERS.md`、`PROVIDER_RESOLVER.md`） | 契约无冲突（D4 已完成） |
| 2 | ~~`internal/paths` + `config` 新增 `[network]` / `[download]`~~ | **已完成** |
| 3 | ~~`internal/model/provider.go` + 示例字段级校验~~ | **已完成** |
| 4 | ~~`internal/checksum` + 真实 checksums 测试~~ | **已完成** |
| 5 | ~~`internal/provider/github` + 重试/限流/脱敏测试~~ | **已完成** |
| 6 | ~~`provider list` + registry~~；`provider add` 待做 | **provider list 已完成** |
| 7 | `btbn`（daily 先行）、`bnqdzj-nonfree`、`custom` | `resolve` 可用 |
| 8 | `internal/download` + `artifact.json` | `download` 可用 |
| 9 | 文档、示例、`PHASE4_STATUS.md`、DoD 对照 | Phase 4 收尾 |

### DoD 对照

| `PROJECT_PLAN.md` Phase 4 DoD | 对应步骤 |
|---|---|
| BtbN stable / daily 都能解析 | 1、7（依赖 D1） |
| BnqDzj nonfree 能解析 | 7 |
| 自定义 URL 必须提供 checksum | 6、7 |
| HTTP 重定向、超时、重试策略明确 | 5、9 |
| 下载不写目标安装目录 | 8 |
| 不修改 PATH | 全流程 |
| 不把 nonfree 二进制加入 Git 或 Release | 8、9 |

---

## 11. 明确不做

- 解压、安装、`current` 切换（Phase 5）；
- PATH / 环境变量写入（Phase 5）；
- `install.json` 生成（Phase 5）；
- 包管理器识别；
- Go API / JSON-RPC / 插件；
- nonfree 二进制再分发；
- 自动更新服务；
- 下载内容的签名验证（仅 SHA-256 pin）；
- Windows Job Object（D7：推迟到 Phase 5 写入类命令前）。

---

## 12. 进入 Phase 4 的检查清单

- [x] D1（stable pin）已定 A 并落文：`release_id` / `pinned_at` / `E3010` 规则；
- [x] D4（data root）写入 `ENVIRONMENT.md` §1.1；
- [x] D5（nonfree 显式同意）已定 A：`--allow-nonfree` + `E7001` + `W1006`，并有 `provider.CheckNonfreeConsent` 测试；
- [x] D7（Job Object）已决定推迟到 Phase 5，并记录理由；
- [x] `internal/paths` 合入，`capabilities` 缓存目录复用；
- [x] `internal/checksum` 合入并有真实 checksums 测试；
- [x] `model` 的 Release / Artifact / ProviderSummary 有 nil-safe JSON；
- [x] `contracts` 对 release / artifact 示例做字段级校验；
- [x] `provider list` 可离线运行（不触发网络）；
- [x] GitHub 客户端具备注入点、重试、限流、重定向上限、大小上限与 token 脱敏测试；
- [x] Phase 3 的 `OMF-E2001`、`help.found`、缓存写入语义未被破坏（全量测试通过）。
