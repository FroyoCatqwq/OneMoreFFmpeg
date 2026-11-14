# 架构设计

> 状态：Phase 0 设计
> 原则：Go 单语言、CLI-first、纯 Go 优先、进程边界清晰、先只读后写入

---

## 1. 总体结构（当前实现）

> 早期设计中的 `internal/app`、`internal/core`、`internal/platform` 三层没有落地；实现采用扁平包结构，命令只依赖具体能力包。若未来需要分层，必须同时更新本节，不允许文档先于代码。

```text
cmd/one-more-ffmpeg       main、信号、buildinfo 注入
        │
        ▼
internal/cli              命令解析、全局 flags、JSON/人类输出、退出码
   ├─ provider            provider list（离线）
   ├─ detect/validate     调用 internal/detect
   └─ capabilities        调用 internal/capabilities
        │
        ├──────────────┬───────────────┬──────────────┬─────────────┐
        ▼              ▼               ▼              ▼             ▼
internal/detect   internal/capabilities  internal/provider  internal/config  internal/logging
        │              │               │                    │
        ▼              ▼               ▼                    ▼
internal/ffexec   internal/paths   internal/provider/github  internal/contracts
        │              │               │
        ▼              ▼               ▼
internal/model    internal/errors  internal/errors
        │
        ├─ internal/buildinfo
        └─ internal/checksum
```

包职责：

| 包 | 职责 |
|---|---|
| `internal/ffexec` | 进程执行、超时/取消、输出上限、进程树终止 |
| `internal/detect` | 只读发现、候选去重、三件套校验、fingerprint/ID |
| `internal/capabilities` | 13 项能力命令、解析、缓存 |
| `internal/paths` | data root / cache / downloads / config 路径 |
| `internal/checksum` | 流式 SHA-256、`checksums.sha256` 解析 |
| `internal/provider` | provider 注册表、资产匹配、pin 解析、nonfree 同意、自定义 provider 存储 |
| `internal/provider/github` | GitHub Releases API 客户端（重试/限流/脱敏） |
| `internal/download` | 流式下载、双重校验、原子落盘、artifact.json |
| `internal/archive` | zip / tar.xz / tar.gz / tar.zst 安全解压 |
| `internal/install` | 版本化安装、install.json、current 指针、安装后校验 |
| `internal/journal` | 安装/切换事务 journal |
| `internal/env` | current/PATH/变量只读检查 |
| `internal/doctor` | 安装健康检查 |
| `internal/contracts` | JSON 示例与错误码注册表校验 |
| `internal/config` | 严格 TOML 1.1 配置 |
| `internal/model` | JSON 契约结构体 |
| `internal/errors` | 稳定错误码注册表 |
| `internal/logging` | slog / tint、颜色能力 |
| `internal/buildinfo` | 版本/提交/构建时间 |
| `internal/testutil` | 测试用 fake FFmpeg（仅测试） |

依赖方向（实际）：

```text
cmd → cli → {detect, capabilities, provider, download, config, logging, contracts}
detect → {ffexec, model, errors}
capabilities → {ffexec, model, paths, errors}
provider → {github, checksum, model, errors}
provider/github → errors
download → {checksum, model, errors}
checksum → errors
paths → errors
```

禁止：

- 能力包反向依赖 CLI；
- provider 直接写 PATH 或安装目录；
- capability parser 直接写安装目录；
- provider/github 依赖 CLI 或 logging（token 不得进日志）；
- 暴露 `internal/*` 作为公共 API；
- 当前阶段引入 `pkg/onemore` 公共 SDK；
- 当前阶段引入插件协议层。

---

## 2. 外部形态

| 形态 | 当前状态 | 说明 |
|---|---|---|
| CLI `one-more-ffmpeg` | **当前唯一对外形态** | 其他项目只调用它 |
| Go library `pkg/onemore` | 未来 | 功能完善后再评估 |
| JSON-RPC / gRPC 插件 | 未来 | 当前不实现 |
| C ABI / c-shared | 未来 | 需要单独 ADR |
| 包管理器识别 | 未来 | 当前不实现 |

当前外部契约：

```text
CLI 参数
stdout：JSON / 文件路径
stderr：日志 / 进度 / 错误
退出码：0 / 1 / 2 / 130
文件产物：artifact.json / install.json / 可选 ffmpeg-trio.lock
```

其他项目不得直接 import 本项目 Go 包，也不得依赖内部结构体。

本项目不使用 Go 标准库 `plugin` 包：Windows 不支持，跨版本 ABI 脆弱，且与“Go 1.27 锁定（工具链 1.27.4）”冲突。

---

## 3. 依赖库落点

| 层 | 采用库 |
|---|---|
| CLI | `urfave/cli/v3` |
| config | `pelletier/go-toml/v2` |
| log | `log/slog` + `lmittmann/tint` |
| concurrency | `golang.org/x/sync` |
| system | `golang.org/x/sys` / `golang.org/x/term` |
| tty | `mattn/go-isatty` |
| archive | `klauspost/compress` / `ulikunitz/xz` |
| HTTP retry | `net/http` + `sethvargo/go-retry` |
| rate limit | `golang.org/x/time/rate` |
| human size | `dustin/go-humanize` |
| test | `stretchr/testify` |

详见 [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md)。

---

## 4. 核心数据模型（与实现一致）

字段名即 JSON 契约名；未标注 `omitempty` 的字段总是输出。

```go
type Installation struct {
    ID                  string            `json:"id"`
    RootDir             string            `json:"root_dir"`
    FFmpeg              string            `json:"ffmpeg,omitempty"`
    FFprobe             string            `json:"ffprobe,omitempty"`
    FFplay              string            `json:"ffplay,omitempty"`
    Sources             []string          `json:"sources"`
    Status              string            `json:"status"`
    Version             string            `json:"version,omitempty"`
    Fingerprint         string            `json:"fingerprint"`
    VersionStdoutSHA256 string            `json:"version_stdout_sha256,omitempty"`
    LicenseStdoutSHA256 string            `json:"license_stdout_sha256,omitempty"`
    BinarySHA           map[string]string `json:"binary_sha256,omitempty"`
    Warnings            []Warning         `json:"warnings"`
}
```

```go
type CapabilitySet struct {
    Schema                  string
    InstallationID          string
    InstallationFingerprint string
    Fingerprint             string
    CreatedAt               string
    CacheHit                bool
    Commands                map[string]CapabilityCommand
    Version                 VersionInfo
    License                 LicenseInfo
    Muxers                  []CapabilityEntry
    Demuxers                []CapabilityEntry
    Devices                 []CapabilityEntry
    Decoders                []CapabilityEntry
    Encoders                []CapabilityEntry
    Filters                 []CapabilityEntry
    PixelFormats            []CapabilityEntry
    ChannelLayouts          []CapabilityEntry
    SampleFormats           []CapabilityEntry
    Help                    map[string]HelpInfo // 含 found
    Warnings                []Warning
}
```

```go
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
```

Go 结构体仍不是公共 API；对外只承诺 CLI JSON，字段改动必须先改 `CLI_SCHEMA.md` 与示例。

---

## 5. 配置与输出

| 类型 | 格式 | 说明 |
|---|---|---|
| 用户配置 | TOML 1.1 | `pelletier/go-toml/v2`，strict mode |
| CLI 输出 | JSON | `--json` 机器可读；正式契约见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) 与 [`schemas/`](schemas/) |
| 文件产物 | JSON + SHA-256 | artifact / install / journal |
| 能力缓存 | JSON | 按 installation fingerprint |
| 日志 | `log/slog` JSON/text | stderr，不污染 stdout |

TOML 解析要求：

- 纯 Go；
- 明确支持 TOML 1.1.0；
- 严格模式拒绝未知字段；
- 无 cgo。

---

## 6. 进程执行模型

所有 FFmpeg 调用通过 `internal/ffexec`：

- `exec.CommandContext`；
- 不使用 shell；
- 参数数组传参；
- stdout/stderr 分离；
- 输出上限；
- 独立 timeout；
- 取消时杀进程；
- Windows：`taskkill /T /F`（3s 超时）+ `Process.Kill` 兜底；Job Object 推迟到 Phase 5（D7，见下）；
- Linux：`Setpgid` + 进程组 `SIGKILL`；
- 记录 exit code、耗时、stdout hash、stderr 摘要。

Windows Job Object（D7）：

- FFmpeg 不派生子进程，现有 `taskkill` + `Kill` 已能覆盖超时/取消场景；
- 引入 Job Object 需要在启动前后分配进程句柄，且 `CREATE_SUSPENDED` 版本要自定义线程恢复，成本高于收益；
- 结论：Phase 5 写入类命令（install / env）前再实现，届时必须补 Windows 专测；
- 在此之前，`ffexec` 的取消语义视为“尽力终止进程树”，文档与 help 不得宣称强保证。

能力命令和校验命令都不得绕过此执行器。

---

## 7. 安装与事务模型

目录示例：

```text
<data-root>/
├─ apps/
│  └─ <provider>/
│     └─ <release-tag>/
│        └─ <sha256-16hex>/
│           ├─ ffmpeg
│           ├─ ffprobe
│           ├─ ffplay
│           └─ install.json
├─ current（symlink / Windows junction / directory pointer） -> apps/<...>
├─ downloads/
├─ cache/
├─ journal/
└─ config.toml
```

写入流程：

```text
download → verify → extract(temp) → validate(temp) → journal begin
  → install to versioned dir → switch current → env change
  → journal commit
```

失败：

```text
journal rollback → remove temp → restore current/env → report
```

不覆盖未由本项目管理的文件。

---

## 8. 错误模型

- 稳定字符串错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)；Phase 1 用 golden test 锁定；
- 错误对象：`code` / `message` / `details` / `cause`；
- stdout JSON 永不混入日志；
- CLI 退出码：`0` 成功、`1` 运行错误、`2` 用法/配置错误、`130` 取消；
- 多来源探测错误用 `errors.Join` 聚合，但最终对外为稳定 JSON 列表。

---

## 9. 并发模型

- 所有用例内部第一关键对象是 `context.Context`；
- 探测源可并发，但有上限；
- 全盘扫描使用 `errgroup` + 有界 channel；
- 下载单连接 + 重试；
- 安装与 env 写入加进程内互斥锁；
- 同一 data root 跨进程用文件锁；
- 能力命令有界并行（默认最多 4 个 FFmpeg 进程，`Collector.MaxParallel`），避免进程风暴；每个命令仍使用独立 timeout/context；
- 缓存写入使用唯一临时文件 + 原子 rename；取消或全部失败时不写缓存。

---

## 10. 与 LAE / VideoDamage 的边界

```text
LiveArchiveEncoder / lae-videodamage
        │ 调用可执行文件
        ▼
one-more-ffmpeg <command> --json
        │
        ├─ stdout JSON
        ├─ stderr 日志
        ├─ 退出码
        └─ 文件产物：artifact.json / install.json / 可选 ffmpeg-trio.lock
```

规则：

- 不 import OneMoreFFmpeg Go 包；
- 不链接 Go runtime；
- 不调用 JSON-RPC / gRPC / C ABI；
- 不依赖 OneMoreFFmpeg 内部结构体；
- 只依赖 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)、[`ERROR_CODES.md`](ERROR_CODES.md) 与退出码；
- OneMoreFFmpeg 不读取 LAE 业务配置；
- VideoDamage core 默认不调用 install/env；这些属于部署/Extension Pack 场景；
- 未来若重新开放 API/插件，必须另写 ADR 并升版本。
