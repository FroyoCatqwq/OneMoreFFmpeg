# 架构设计

> 状态：Phase 0 设计
> 原则：Go 单语言、纯 Go 优先、进程边界清晰、先只读后写入

---

## 1. 总体结构

```text
CLI / JSON-RPC / Go API
        │
        ▼
pkg/onemore                 公共 API v1
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
internal/platform
   ├─ windows              Scoop、PATH 注册表、Job Object 预留
   └─ linux                Homebrew、shim、XDG 路径

internal/core
   ├─ model                公共数据结构
   ├─ ffexec               FFmpeg 进程执行器
   ├─ parser               FFmpeg 输出解析
   ├─ checksum             SHA-256 / checksums.sha256
   ├─ archive              zip / tar.xz 安全解压
   ├─ journal              安装/环境变更事务
   ├─ cache                能力缓存
   ├─ config               TOML 1.1
   └─ errors               稳定错误码
```

依赖方向：

```text
cmd → pkg/onemore → internal/app → internal/core
                              ↘ internal/platform
```

禁止：

- `internal/core` 依赖 CLI；
- `internal/platform` 依赖 provider；
- provider 直接写 PATH；
- capability parser 直接写安装目录；
- Go API 暴露 `internal/*` 类型。

---

## 2. 外部形态

| 形态 | 入口 | 用途 |
|---|---|---|
| CLI | `one-more-ffmpeg` | 人工使用、脚本调用 |
| Go library | `pkg/onemore` | Go 项目直接 import |
| 进程插件 | JSON-RPC 2.0 over stdio | LAE / VideoDamage / 其他语言 |
| C ABI | 后续评估 | Rust/C++ 宿主 in-process 调用 |

优先做前三；C ABI 需要单独 ADR，因为 Go `-buildmode=c-shared` 会把 Go runtime 带进宿主进程，卸载和线程模型都有约束。

本阶段不使用 Go 标准库 `plugin` 包：Windows 不支持，跨版本 ABI 脆弱，且与“Go 1.27.3 精确锁定”冲突。

---

## 3. 核心数据模型（概念）

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

这些是概念结构，不代表最终 Go 结构体字段已冻结；字段名以 `API_AND_PLUGIN.md` 的 JSON 契约为准。

---

## 4. 配置与输出

| 类型 | 格式 | 说明 |
|---|---|---|
| 用户配置 | TOML 1.1 | provider、路径、扫描策略、权限 |
| CLI 输出 | JSON | `--json` 机器可读 |
| Go API | Go 类型 | `pkg/onemore` |
| 插件协议 | JSON-RPC 2.0 | stdio，一行一个 JSON object |
| 能力缓存 | JSON | 按 installation fingerprint |
| 安装记录 | JSON + SHA-256 | 写入版本化安装目录 |
| 日志 | `log/slog` JSON/text | stderr，不污染 stdout |

TOML 1.1 解析库在 Phase 1 选定，要求：

- 纯 Go；
- 持续维护；
- 明确支持 1.1；
- 支持 `deny_unknown_fields` 或等价严格解码；
- 无 cgo。

若生态不满足，则配置改 JSON，并通过 ADR 记录变更。

---

## 5. 进程执行模型

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

## 6. 安装与事务模型

目录示例：

```text
<data-root>/
├─ apps/
│  └─ <provider>/
│     └─ <release-tag>/
│        └─ <asset-sha256-prefix>/
│           ├─ ffmpeg
│           ├─ ffprobe
│           ├─ ffplay
│           └─ install.json
├─ current -> apps/<...>
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

## 7. 错误模型

- 稳定字符串错误码，例如 `OMF-E1001`；
- 错误对象：`code` / `message` / `details` / `cause`；
- stdout JSON 永不混入日志；
- CLI 退出码：`0` 成功、`1` 运行错误、`2` 用法/配置错误、`130` 取消；
- 多来源探测错误用 `errors.Join` 聚合，但最终对外为稳定 JSON 列表。

---

## 8. 并发模型

- 所有公开 API 第一参数是 `context.Context`；
- 探测源可并发，但有上限；
- 全盘扫描使用 `errgroup` 风格有限 worker；
- 下载单连接 + 重试，不做无上限并发；
- 安装与 env 写入加进程内互斥锁；
- 同一 data root 跨进程用文件锁；
- 能力命令可并行，但默认串行，避免 FFmpeg 进程风暴。

---

## 9. 与 LAE / VideoDamage 的边界

```text
LiveArchiveEncoder / lae-videodamage
        │ JSON-RPC / Go API / CLI
        ▼
OneMoreFFmpeg
        │
        ├─ 返回已安装 trio 路径
        ├─ 返回 capabilities
        ├─ 返回 install.json / release pin
        └─ 可选：安装、切换、env 配置
```

OneMoreFFmpeg 不读取 LAE 业务配置，不生成 `ffmpeg-trio.lock` 以外的业务文件；但可以为 VideoDamage 生成符合其 C1 契约的 `ffmpeg-trio.lock`。

VideoDamage 不直接下载/安装 FFmpeg；它调用 OneMoreFFmpeg 或读取 OneMoreFFmpeg 产出的锁文件。
