# Go 依赖库评估

> 日期：2026-11-10
> 状态：Phase 0 依赖评估结论
> 适用：OneMoreFFmpeg 当前 CLI-first 范围
> 数据来源：Go module proxy `@latest`、GitHub API、项目官方 README

---

## 1. 评估标准

用户要求：

> 持续更新，经常维护，至少 6 个月内出新版本。

本次按 2026-11-10 回溯 6 个月，截止线为：

```text
2026-05-10
```

第三方库必须满足：

1. Go module proxy `@latest` 的 `Time` 不早于 `2026-05-10`；
2. 项目未 archived；
3. 有明确版本发布，或 Go `x/` 模块有近期 pseudo-version / tagged version；
4. 优先纯 Go、`CGO_ENABLED=0` 可用；
5. Windows x64 + Linux x86_64 可用；
6. 许可证清晰，Phase 1 再用 `go-licenses` / `cargo-deny` 等价工具复核；
7. 不引入与当前简单范围不匹配的大型框架；
8. 不因为“流行”就忽略版本新鲜度。

数据获取方式：

```text
https://proxy.golang.org/<escaped-module>/@latest
```

例如：

```text
https://proxy.golang.org/github.com/!burnt!sushi/toml/@latest
```

说明：

- `Time` 是 Go module proxy 记录的版本时间；
- 对 `golang.org/x/*` 这类高频模块，按 Go module proxy 的最新 tagged/pseudo-version 判断；
- 官方标准库不参与“6 个月内出新版本”筛选，但仍优先使用。

---

## 2. 当前项目范围变化

本评估基于最新范围：

| 项 | 当前状态 |
|---|---|
| Scoop / Homebrew / 包管理器识别 | **取消当前支持，纳入未来考虑** |
| 对外开放 Go API | **取消当前支持，纳入未来考虑** |
| 对外开放插件接口 / JSON-RPC / C ABI | **取消当前支持，纳入未来考虑** |
| 其他项目如何调用 | **只调用本项目产物可执行文件，即 CLI** |
| CLI 对外输出 | JSON / 文件 / 退出码 |
| 配置文件 | TOML 1.1 |
| 语言 | Go 1.27.3 |

因此当前依赖目标是：

```text
CLI 可执行文件 + 文件/IO + 进程调用 + 下载 + 解压 + 环境变量 + 日志 + 测试
```

不是：

```text
Go SDK + 进程插件框架 + gRPC 服务 + 包管理器集成 + GUI/TUI
```

---

## 3. 结论摘要

### 3.1 当前采用

| 类别 | 库 | 最新版本 | 发布时间 | 结论 |
|---|---|---:|---|---|
| CLI 框架 | `github.com/urfave/cli/v3` | v3.14.0 | 2026-10-02 | 采用 |
| TOML 1.1 | `github.com/pelletier/go-toml/v2` | v2.4.3 | 2026-07-05 | 采用 |
| 日志 | `log/slog` + `github.com/lmittmann/tint` | v1.2.1 | 2026-10-02 | 采用 |
| 并发 | `golang.org/x/sync` | v0.25.0 | 2026-10-15 | 采用 `errgroup` |
| 系统 API | `golang.org/x/sys` | v0.50.0 | 2026-11-06 | 采用 |
| 终端/系统 API | `golang.org/x/term` | v0.48.0 | 2026-11-09 | 采用 |
| 终端检测 | `github.com/mattn/go-isatty` | v0.0.24 | 2026-07-23 | 采用 |
| 压缩/解压 | `github.com/klauspost/compress` | v1.20.4 | 2026-11-09 | 采用 |
| xz 解压 | `github.com/ulikunitz/xz` | v0.5.17 | 2026-09-19 | 采用 |
| HTTP 重试 | `net/http` + `github.com/sethvargo/go-retry` | v0.5.0 | 2026-10-05 | 采用 |
| 测试 | `github.com/stretchr/testify` | v1.12.1 | 2026-08-17 | 采用 |
| 可读大小 | `github.com/dustin/go-humanize` | v1.1.0 | 2026-09-17 | 采用 |

### 3.2 未来候选，不进入当前 M0

| 类别 | 库 | 最新版本 | 发布时间 | 状态 |
|---|---|---:|---|---|
| Worker pool | `github.com/alitto/pond/v2` | v2.7.3 | 2026-10-17 | 候选 |
| Worker pool | `github.com/gammazero/workerpool` | v1.3.0 | 2026-10-19 | 候选 |
| Worker pool | `github.com/panjf2000/ants/v2` | v2.12.1 | 2026-05-25 | 候选 |
| 系统/进程信息 | `github.com/shirou/gopsutil/v4` | v4.26.10 | 2026-10-24 | 候选 |
| 环境变量配置 | `github.com/caarlos0/env/v11` | v11.4.2 | 2026-11-02 | 候选 |
| 环境变量配置 | `github.com/sethvargo/go-envconfig` | v1.4.3 | 2026-07-19 | 候选 |
| ID | `github.com/oklog/ulid/v2` | v2.1.2 | 2026-07-23 | 候选 |
| 进度条 | `github.com/vbauerster/mpb/v8` | v8.16.3 | 2026-10-20 | 候选 |
| 进度条 | `github.com/schollz/progressbar/v3` | v3.19.1 | 2026-06-30 | 候选 |
| Glob 扫描 | `github.com/bmatcuk/doublestar/v4` | v4.10.4 | 2026-10-24 | 候选 |
| 高速 JSON | `github.com/goccy/go-json` | v0.12.0 | 2026-10-29 | 候选 |
| 高速 JSON | `github.com/bytedance/sonic` | v1.15.5 | 2026-10-26 | 候选 |
| HTTP client | `github.com/imroc/req/v3` | v3.61.0 | 2026-08-13 | 候选 |
| HTTP router（未来本地服务） | `github.com/go-chi/chi/v5` | v5.5.0 | 2026-10-14 | 候选，仅未来 |
| Windows named pipe（未来 IPC） | `github.com/Microsoft/go-winio` | v0.6.4 | 2026-10-13 | 候选，仅未来 |
| 未来 JSON-RPC | `github.com/creachadair/jrpc2` | v1.3.6 | 2026-10-20 | 候选，仅未来 |
| 未来 JSON-RPC | `github.com/sourcegraph/jsonrpc2` | v0.2.3 | 2026-09-10 | 候选，仅未来 |
| 未来 RPC | `google.golang.org/grpc` | v1.85.0 | 2026-10-27 | 候选，仅未来 |
| 未来 RPC | `connectrpc.com/connect` | v1.21.1 | 2026-10-14 | 候选，仅未来 |
| 未来插件框架 | `github.com/hashicorp/go-plugin` | v1.8.0 | 2026-04-29 | **不采用**：发布时间早于 2026-05-10，差约 11 天 |

### 3.3 明确拒绝

| 库 | 最新版本 | 发布时间 | 拒绝原因 |
|---|---:|---|---|
| `github.com/spf13/cobra` | v1.10.2 | 2025-12-03 | 超过 6 个月无新版本；虽流行但不满足硬标准 |
| `github.com/spf13/pflag` | 随 cobra 生态 | — | 当前 CLI 用 `urfave/cli/v3`，不需要第二套参数体系 |
| `github.com/spf13/viper` | v1.21.0 | 2025-09-08 | 超过 6 个月；配置层过重 |
| `github.com/BurntSushi/toml` | v1.6.0 | 2025-12-18 | 超过 6 个月；且 TOML 1.1 支持不如 `go-toml/v2` 明确 |
| `github.com/rs/zerolog` | v1.35.1 | 2026-04-20 | 早于截止线；`log/slog` 已足够 |
| `go.uber.org/zap` | v1.28.0 | 2026-04-28 | 早于截止线；`log/slog` 已足够 |
| `github.com/charmbracelet/log` | v1.0.0 | 2026-03-09 | 早于截止线 |
| `github.com/google/go-cmp` | v0.7.0 | 2025-01-14 | 超过 6 个月；`testify` + 标准库足够 |
| `go.uber.org/goleak` | v1.3.0 | 2023-10-24 | 过旧 |
| `go.uber.org/mock` | v0.6.0 | 2025-08-18 | 超过 6 个月；当前不需要 mock 生成 |
| `github.com/hashicorp/go-retryablehttp` | v0.7.8 | 2025-06-18 | 超过 6 个月；用 `net/http` + `go-retry` |
| `github.com/cenkalti/backoff/v5` | v5.0.3 | 2025-07-23 | 超过 6 个月 |
| `github.com/avast/retry-go/v4` | v4.7.0 | 2025-10-14 | 超过 6 个月 |
| `github.com/valyala/fasthttp` | v1.78.0 | 2026-11-09 | 版本新，但与标准 `net/http` API 差异大，当前场景不需要 |
| `github.com/google/renameio/v2` | v2.0.2 | 2026-01-10 | 超过 6 个月；原子写用标准库实现 |
| `github.com/natefinch/atomic` | v1.0.1 | 2021-06-28 | 过旧 |
| `github.com/adrg/xdg` | v0.5.3 | 2024-10-31 | 超过 6 个月；`os.UserConfigDir` 等标准库足够 |
| `github.com/spf13/afero` | v1.15.0 | 2025-09-08 | 超过 6 个月；IO 抽象层当前过重 |
| `github.com/charlievieth/fastwalk` | v1.0.14 | 2025-09-09 | 超过 6 个月；先用 `filepath.WalkDir` |
| `github.com/sourcegraph/conc` | v0.3.0 | 2023-02-25 | 过旧 |
| `github.com/creack/pty` | v1.1.24 | 2024-10-31 | 超过 6 个月；当前不需要 PTY |
| `github.com/google/uuid` | v1.6.0 | 2024-01-23 | 超过 6 个月；若需要 ID 用 `oklog/ulid/v2` 候选 |
| `github.com/joho/godotenv` | v1.5.1 | 2023-02-05 | 过旧；且本项目默认 TOML |
| `github.com/Netflix/go-env` | v0.1.2 | 2024-10-28 | 过旧 |
| `github.com/mholt/archiver/v4` | v4.0.0-alpha.9 | 2024-11-08 | alpha 且过旧；用 `klauspost/compress` + `ulikunitz/xz` |
| `github.com/otiai10/copy` | v1.14.1 | 2025-01-05 | 超过 6 个月；标准库足够 |
| `github.com/eclipse/paho.mqtt.golang` | v1.5.1 | 2025-09-16 | 超过 6 个月；当前不需要 MQTT |
| `github.com/nats-io/nats.go` | v1.55.0 | 2026-10-23 | 版本新，但当前 CLI-first 范围不需要消息系统 |
| `github.com/redis/go-redis/v9` | v9.23.0 | 2026-10-05 | 版本新，但当前不需要 Redis |
| `github.com/mholt/archiver/v4` | alpha | 2024-11-08 | 重复列出：明确拒绝 |

---

## 4. 分类评估

### 4.1 文件读写 / IO

采用：

- 标准库：`os`、`io`、`fs`、`filepath`、`bufio`、`crypto/sha256`；
- `github.com/klauspost/compress`：zip / zstd 等高性能压缩；
- `github.com/ulikunitz/xz`：`.tar.xz` 解压；
- `github.com/dustin/go-humanize`：日志中的人类可读大小；
- `github.com/bmatcuk/doublestar/v4`：未来 glob 扫描候选。

不用：

- `afero`：抽象层过重，且版本过旧；
- `otiai10/copy`：标准库足够；
- `mholt/archiver/v4`：alpha、过旧；
- `google/renameio/v2` / `natefinch/atomic`：原子写通过 `os.CreateTemp` + `Rename` + `Sync` 自实现，减少依赖。

### 4.2 高性能并发

采用：

- `golang.org/x/sync/errgroup`：并发探测、能力命令、扫描；
- 标准库 semaphore pattern / channel：控制并发上限。

候选：

- `alitto/pond/v2`
- `gammazero/workerpool`
- `panjf2000/ants/v2`

当前不默认引入 worker pool，因为 M0 的并发规模不大；`errgroup` + 有界 channel 足够。若全盘扫描或下载并发出现瓶颈，再 A/B 引入。

拒绝：

- `sourcegraph/conc`：2023 后无新版本。

### 4.3 协议通信 / 程序通信

当前范围：

- 其他项目只调用 CLI 可执行文件；
- 进程通信通过：
  - 参数；
  - stdout JSON；
  - stderr 日志；
  - 退出码；
  - 输出文件 / `install.json` / 未来 `ffmpeg-trio.lock`。
- 不开放 Go API；
- 不开放 JSON-RPC / gRPC / C ABI 插件。

未来候选：

| 用途 | 候选 |
|---|---|
| JSON-RPC | `creachadair/jrpc2` / `sourcegraph/jsonrpc2` |
| gRPC | `google.golang.org/grpc` |
| Connect RPC | `connectrpc.com/connect` |
| Windows named pipe | `Microsoft/go-winio` |
| Go 插件框架 | `hashicorp/go-plugin`（但当前不满足 6 个月硬标准） |
| HTTP client | `imroc/req/v3` |
| HTTP router | `go-chi/chi/v5` |

明确拒绝当前引入：

- `hashicorp/go-plugin`：发布时间 2026-04-29，早于截止线；
- `nats.go` / `go-redis`：当前不需要消息/缓存中间件；
- `fasthttp`：API 与标准库差异大，当前无性能证据。

### 4.4 CLI 框架

采用：

- `github.com/urfave/cli/v3` v3.14.0，2026-10-02。

理由：

- 6 个月内有新版；
- Go module 主流；
- 适合子命令、flag、help、version；
- 比自己拼 `flag` 更稳定；
- 比 `cobra` 更符合版本新鲜度硬标准。

拒绝：

- `spf13/cobra`：最新 v1.10.2 是 2025-12-03，超过 6 个月；
- `alecthomas/kong` v1.17.0 2026-10-30 很新，但 `urfave/cli/v3` 更主流，作为备选观察；
- `spf13/viper`：配置层过重且版本过旧。

### 4.5 配置文件：TOML 1.1

采用：

- `github.com/pelletier/go-toml/v2` v2.4.3，2026-07-05。

已通过官方 README 核实：

```text
This library supports TOML v1.1.0.
Import: github.com/pelletier/go-toml/v2
License: MIT
```

采用点：

- TOML 1.1；
- 类 `encoding/json` 的 API；
- 支持 strict mode，可检测未知字段/拼写错误；
- 错误信息带行列上下文；
- 支持 commented config，适合生成带说明的配置文件；
- 纯 Go，无 cgo。

拒绝：

- `BurntSushi/toml`：v1.6.0 2025-12-18，超期，且 TOML 1.1 支持不如 `go-toml/v2` 明确；
- `viper`：过重且超期；
- `koanf/v2`：v2.4.1 2026-10-18 很新，但当前只需要单文件 TOML，不需要多源配置层，列为未来候选。

### 4.6 HTTP 相关通信

采用：

- 标准库 `net/http`；
- `github.com/sethvargo/go-retry` v0.5.0，2026-10-05，用于下载重试/backoff；
- `golang.org/x/time/rate`（来自 `golang.org/x/time` v0.16.0 2026-08-19）用于限流。

候选：

- `imroc/req/v3`：若标准库样板过多再评估；
- `go-chi/chi/v5`：仅未来本地 HTTP 服务；
- `valyala/fasthttp`：当前拒绝，API 差异大且无性能证据。

拒绝：

- `hashicorp/go-retryablehttp`：2025-06-18，超期；
- `cenkalti/backoff/v5`：2025-07-23，超期；
- `avast/retry-go/v4`：2025-10-14，超期。

### 4.7 系统 API

采用：

- `golang.org/x/sys`：Windows registry、文件系统、底层调用；
- `golang.org/x/term`：终端宽度/交互判断；
- `github.com/mattn/go-isatty`：是否 TTY；
- 标准库 `os/exec` / `os.UserConfigDir` / `os.UserHomeDir`。

候选：

- `shirou/gopsutil/v4`：进程/磁盘/系统信息，未来 `doctor` 可能需要；
- `Microsoft/go-winio`：未来 named pipe / Windows 容器场景；
- `adrg/xdg`：拒绝，版本过旧，标准库足够。

### 4.8 日志

采用：

- 标准库 `log/slog`：
  - 生产 JSON handler；
  - 结构化字段；
  - 与 `context` 集成；
- `github.com/lmittmann/tint` v1.2.1 2026-10-02：
  - 开发环境彩色文本 handler；
  - 不污染 stdout JSON。

拒绝：

- `rs/zerolog`：2026-04-20，早于截止线；
- `go.uber.org/zap`：2026-04-28，早于截止线；
- `charmbracelet/log`：2026-03-09，早于截止线；
- `sirupsen/logrus`：维护模式，不评估采用。

### 4.9 测试

采用：

- 标准库 `testing`；
- `github.com/stretchr/testify` v1.12.1 2026-08-17；
- 标准库 fuzz；
- fake ffmpeg 脚本；
- golden files。

拒绝：

- `google/go-cmp`：2025-01-14，超期；
- `go.uber.org/goleak`：2023-10-24，过旧；
- `go.uber.org/mock`：2025-08-18，超期；
- `testcontainers-go`：当前范围不需要容器测试。

### 4.10 插件

当前状态：

> **插件不实现，不引入插件依赖。**

未来重新开放时优先考虑：

- `creachadair/jrpc2`：JSON-RPC；
- `google.golang.org/grpc`：gRPC；
- `connectrpc.com/connect`：Connect RPC；
- `Microsoft/go-winio`：Windows named pipe；
- `hashicorp/go-plugin`：观察，但当前版本不满足 6 个月硬标准。

不使用 Go 标准库 `plugin` 包：Windows 不支持，跨版本 ABI 脆弱。

---

## 5. 建议的最小依赖集

Phase 1 可先写入 `go.mod` / `go.sum`：

```text
github.com/urfave/cli/v3
github.com/pelletier/go-toml/v2
github.com/lmittmann/tint
github.com/stretchr/testify          // test only
golang.org/x/sync
golang.org/x/sys
golang.org/x/term
golang.org/x/time
github.com/mattn/go-isatty
github.com/dustin/go-humanize
github.com/klauspost/compress
github.com/ulikunitz/xz
github.com/sethvargo/go-retry
```

当前仓库只提交 `go.mod` 的 Go 版本，不提前写入依赖；Phase 1 搭骨架时再加 `go.sum` 并锁定版本。

---

## 6. 待办

- Phase 1 用 `go get` 固定上述版本；
- `go mod tidy`；
- `go test ./...`；
- `go vet ./...`；
- `govulncheck` 评估；
- 许可证清单生成；
- Windows/Linux 双平台编译；
- 对 `urfave/cli/v3` / `go-toml/v2` / `tint` 做 API smoke test；
- 若未来重新开放 API/插件，再重新评估 RPC / 插件框架，不直接引入当前依赖。

---

## 7. 结论

> 当前采用“标准库为主 + 少量 6 个月内活跃第三方库”的策略。
>
> 核心第三方库：CLI 用 `urfave/cli/v3`，TOML 1.1 用 `go-toml/v2`，日志用 `slog+tint`，并发用 `x/sync`，压缩包用 `klauspost/compress` + `ulikunitz/xz`，测试用 `testify`。
>
> 包管理器识别、Go API、JSON-RPC、gRPC、C ABI 插件都明确推迟，不进入当前依赖范围。
