# 架构设计

> 状态：Phase 0 设计
> 原则：Go 单语言、CLI-first、纯 Go 优先、进程边界清晰、先只读后写入

---

## 1. 总体结构

```text
可执行 CLI：one-more-ffmpeg
        │
        ▼
internal/cli               命令解析、参数、help、version
        │
        ▼
internal/app               用例编排
   ├─ detect               系统发现
   ├─ validate             可用性校验
   ├─ capabilities         13 项能力命令
   ├─ resolve              provider release 解析
   ├─ download             下载与校验
   ├─ install              解压、安装、current 切换
   ├─ env                  PATH / 环境变量 / 回滚
   └─ doctor               健康检查
        │
        ▼
internal/core
   ├─ model                内部数据结构
   ├─ ffexec               FFmpeg 进程执行器
   ├─ parser               FFmpeg 输出解析
   ├─ checksum             SHA-256 / checksums.sha256
   ├─ archive              zip / tar.xz 安全解压
   ├─ journal              安装/环境变更事务
   ├─ cache                能力缓存
   ├─ config               TOML 1.1
   └─ errors               稳定错误码
        │
        ▼
internal/platform
   ├─ windows              PATH 注册表、文件系统细节
   └─ linux                shim、XDG 路径、文件系统细节
```

依赖方向：

```text
cmd → internal/cli → internal/app → internal/core
                                 ↘ internal/platform
```

禁止：

- `internal/core` 依赖 CLI；
- `internal/platform` 依赖 provider；
- provider 直接写 PATH；
- capability parser 直接写安装目录；
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

本阶段不使用 Go 标准库 `plugin` 包：Windows 不支持，跨版本 ABI 脆弱，且与“Go 1.27.3 精确锁定”冲突。

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

## 4. 核心数据模型（内部概念）

```go
type Installation struct {
    ID             string
    RootDir        string
    FFmpegPath     string
    FFprobePath    string
    FFplayPath     string
    Source         string
    ProviderHint   string
    Version        string
    BinarySHA256   map[string]string
    Status         string
    Warnings       []string
    Fingerprint    string
}
```

```go
type CapabilitySet struct {
    Schema         string
    InstallationID string
    Fingerprint    string
    Commands       map[string]CommandResult
    Decoders       []CapabilityEntry
    Encoders       []CapabilityEntry
    Muxers         []CapabilityEntry
    Demuxers       []CapabilityEntry
    Devices        []CapabilityEntry
    Filters        []CapabilityEntry
    PixelFormats   []string
    ChannelLayouts []string
    SampleFormats  []string
    License        LicenseInfo
    Version        VersionInfo
    CreatedAt      string
}
```

```go
type Release struct {
    Provider      string
    Channel       string
    ReleaseTag    string
    PublishedAt   string
    AssetName     string
    AssetURL      string
    AssetSHA256   string
    ChecksumsURL  string
    ChecksumsSHA  string
    Variant       string
    Linkage       string
    Target        string
}
```

这些是内部概念结构。对外 JSON 字段名以 CLI 输出 schema 为准，Go 结构体不对外承诺。

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

所有 FFmpeg 调用通过 `internal/core/ffexec`：

- `exec.CommandContext`；
- 不使用 shell；
- 参数数组传参；
- stdout/stderr 分离；
- 输出上限；
- 独立 timeout；
- 取消时杀进程；
- Windows 预留 Job Object；
- Linux 使用 process group；
- 记录 exit code、耗时、stdout hash、stderr 摘要。

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
- 能力命令可并行，但默认串行，避免 FFmpeg 进程风暴。

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
