# 对外 API 与插件接口（未来设计）

> 状态：**DEFERRED / 当前不实现**
> 日期：2026-11-10
> 原因：程序功能尚不完善，过早开放 API/插件会冻结不成熟接口并增加复杂度

---

## 0. 当前对外方式

当前其他项目只能通过**调用本项目产物可执行文件**集成：

```text
LiveArchiveEncoder / lae-videodamage / 脚本
        │
        ▼
one-more-ffmpeg <command> --json
        │
        ├─ stdout：JSON / 文件路径
        ├─ stderr：结构化日志 / 进度 / 错误
        ├─ 退出码：0 / 1 / 2 / 130
        └─ 文件产物：artifact.json / install.json / 可选 ffmpeg-trio.lock
```

当前明确不提供：

- Go public API；
- `pkg/onemore` 公共 SDK；
- JSON-RPC 插件；
- gRPC / Connect RPC 服务；
- C ABI / c-shared；
- 进程内插件；
- 包管理器识别 API。

本文件以下内容仅为未来重新开放时的设计储备，**不是当前实施依据，也不进入 Phase 1**。

---

## 1. 未来可能的调用形态

> 未来清单以 [`docs/README.md`](README.md) 的“未来，明确不属于当前 Phase”为权威；本节只保留设计视图。

| 形态 | 未来优先级 | 说明 |
|---|---|---|
| Go library | 中 | 功能稳定后再评估 `pkg/onemore` |
| JSON-RPC 2.0 over stdio | 中 | 跨语言进程插件 |
| gRPC / Connect RPC | 低 | 需要长期服务或远程调用时再评估 |
| C ABI / c-shared | 低 | Rust/C++ 宿主 in-process 调用，风险最高 |
| Go `plugin` 包 | 不使用 | Windows 不支持，ABI 脆弱 |

不使用 Go `plugin` 包；Windows 不支持，且 Go 插件要求主程序与插件完全同版本同构建，部署脆弱。

---

## 2. 未来 Go API 草案

```go
type Manager interface {
    Detect(ctx context.Context, req DetectRequest) ([]Installation, error)
    Validate(ctx context.Context, inst Installation, opt ValidateOptions) (ValidationReport, error)
    Capabilities(ctx context.Context, inst Installation, opt CapabilityOptions) (CapabilitySet, error)
    Resolve(ctx context.Context, req ResolveRequest) (Release, error)
    Download(ctx context.Context, rel Release, opt DownloadOptions) (Artifact, error)
    Install(ctx context.Context, art Artifact, opt InstallOptions) (InstallRecord, error)
    ConfigureEnv(ctx context.Context, rec InstallRecord, opt EnvOptions) (EnvChange, error)
    Rollback(ctx context.Context, journalID string) error
    Doctor(ctx context.Context) (DoctorReport, error)
}
```

未来规则：

- 所有方法第一参数是 `context.Context`；
- 不暴露 `internal/*`；
- 错误实现 `Code() string`；
- 不返回未文档化的 Go-only 类型；
- 破坏性变更必须提升 `api/v2`；
- 重新开放前必须写 ADR 并冻结 schema。

---

## 3. 当前 CLI 契约才是外部契约

正式字段、示例与版本规则见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)、[`schemas/`](schemas/)；错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)。

当前 CLI：

```text
one-more-ffmpeg
```

可选别名：

```text
omf
```

主要命令：

```text
one-more-ffmpeg version
one-more-ffmpeg detect [--dir] [--scan] [--json]
one-more-ffmpeg validate --installation <id> [--json]
one-more-ffmpeg capabilities --installation <id> [--refresh] [--json]
one-more-ffmpeg provider list
one-more-ffmpeg provider add --id --url --sha256
one-more-ffmpeg resolve --provider btbn|bnqdzj-nonfree|custom --channel stable|daily --json
one-more-ffmpeg download --release <id> --json
one-more-ffmpeg install --artifact <id> [--emit-trio-lock] --dry-run|--yes
one-more-ffmpeg env status|set|unset|rollback --json
one-more-ffmpeg doctor --json
```

输出规则：

- stdout：机器可读 JSON 或结果路径；
- stderr：日志、进度、错误；
- `--json` 成功对象包含 `ok`, `schema`, `data`；
- 错误对象包含 `ok=false`, `code`, `message`, `details`；
- 退出码：`0` / `1` / `2` / `130`。

---

## 4. 未来 JSON-RPC 插件草案

传输：

```text
JSON-RPC 2.0 over stdio
UTF-8
一行一个 JSON object
```

候选方法：

```text
initialize
shutdown
detect
validate
capabilities
resolve
download
install
configure_env
rollback
doctor
```

候选通知：

```text
progress
log
warning
```

如果未来实现，候选库按 [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md) 选择：

- `creachadair/jrpc2`
- `sourcegraph/jsonrpc2`
- `google.golang.org/grpc`
- `connectrpc.com/connect`
- `Microsoft/go-winio`（Windows named pipe）

当前不引入这些依赖。

---

## 5. 未来插件 manifest 草案

```toml
schema = "onemoreffmpeg.plugin/v1"
id = "onemoreffmpeg"
name = "OneMoreFFmpeg"
version = "0.1.0"
protocol = "jsonrpc-stdio"
abi = 1
provides = [
  "omf.ffmpeg.detect",
  "omf.ffmpeg.capabilities",
  "omf.ffmpeg.install",
  "omf.ffmpeg.env"
]
requires = []
permissions = [
  "fs.read",
  "fs.write",
  "net",
  "env.write",
  "process.spawn"
]
```

当前不实现，也不要求宿主解析该 manifest。

---

## 6. 未来 C ABI 评估点

Go 可以用 `-buildmode=c-shared` 导出 C ABI，但注意：

- Go runtime 会进入宿主进程；
- Windows DLL 卸载 Go runtime 不安全；
- 线程/信号/GC 与宿主可能冲突；
- 不能跨边界传 Go string/slice/map/interface；
- 必须导出 opaque handle + `free` 函数；
- 必须 catch panic，不能让 Go panic 跨 C 边界；
- 更适合“一个宿主长期持有一个 OneMoreFFmpeg 实例”，不适合频繁 dlopen/dlclose。

结论：

> 当前不使用 C ABI；未来只有 LAE/VideoDamage 明确需要 in-process 性能时再开 ADR。

---

## 7. 当前与 LAE / VideoDamage 的集成

### 7.1 只读集成

```text
LAE / VideoDamage
  → 执行 one-more-ffmpeg detect --json
  → 执行 one-more-ffmpeg capabilities --json
  → 解析 stdout JSON
  → 得到 trio_dir + capabilities + fingerprint
```

### 7.2 生成锁文件

```text
one-more-ffmpeg install ... --emit-trio-lock
  → 写 install.json
  → 写符合 VideoDamage C1 的 ffmpeg-trio.lock
```

约束：

- 只有 BtbN stable GPL static 才允许 `--emit-trio-lock`；
- BnqDzj nonfree、daily、shared 不允许冒充 C1；
- 锁文件字段必须按 VideoDamage C1，不按 OneMoreFFmpeg 内部格式；
- 不把 `latest` 写入锁文件。

### 7.3 安装/环境变更

需要显式 CLI 参数：

```text
install / env set / rollback
```

VideoDamage core 默认不调用这些方法；它们属于部署/Extension Pack 场景。

---

## 8. 重新开放 API/插件的条件

未来必须同时满足：

1. CLI schema 稳定；
2. TOML config schema 稳定；
3. installation / artifact / install.json 稳定；
4. 错误码稳定；
5. 双平台 CI 稳定；
6. 有真实语料与回归测试；
7. ADR 明确为什么需要 SDK/插件；
8. 依赖库仍满足“6 个月内出新版本”标准；
9. 明确版本兼容策略；
10. 不破坏当前 CLI-first 调用方式。

在此之前，本文件只是未来设计储备。
