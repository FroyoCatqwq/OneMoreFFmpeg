# 对外 API 与插件接口

> 状态：Phase 0 设计
> 目标：让 LiveArchiveEncoder、lae-videodamage 与其他语言调用 OneMoreFFmpeg

---

## 1. 三种调用形态

| 形态 | 版本 | 优先实现 |
|---|---|---|
| Go library `pkg/onemore` | `api/v1` | 是 |
| CLI `one-more-ffmpeg --json` | `cli/v1` | 是 |
| 进程插件 JSON-RPC 2.0 over stdio | `plugin/v1` | 是 |
| C ABI / `-buildmode=c-shared` | 未来 | 否 |

不使用 Go `plugin` 包；Windows 不支持，且 Go 插件要求主程序与插件完全同版本同构建，部署脆弱。

---

## 2. Go API v1

模块：

```text
github.com/FroyoCatqwq/OneMoreFFmpeg/pkg/onemore
```

概念接口：

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

规则：

- 所有方法第一参数是 `context.Context`；
- 不暴露 `internal/*`；
- 返回稳定结构体；
- 错误实现 `Code() string`；
- 不返回未文档化的 Go-only 类型；
- 破坏性变更必须提升 `api/v2`。

---

## 3. CLI v1

二进制：

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
one-more-ffmpeg install --artifact <id> --dry-run|--yes
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

## 4. JSON-RPC 插件 v1

传输：

```text
JSON-RPC 2.0 over stdio
UTF-8
一行一个 JSON object
```

方法：

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

通知：

```text
progress
log
warning
```

请求示例：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "capabilities",
  "params": {
    "installation_id": "inst-...",
    "refresh": false
  }
}
```

错误示例：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": "OMF-E2001",
    "message": "installation not found",
    "data": {}
  }
}
```

规则：

- 不用 Go 类型名作为协议字段；
- 不把 `error` 文本当稳定机器接口；
- `code` 才是稳定接口；
- 二进制不直接走 JSON base64，走文件路径；
- 下载/安装权限必须显式授予；
- 支持 `cancel` 通知；
- 超时由宿主与插件双方共同限制。

---

## 5. 插件 manifest

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

权限说明：

| 权限 | 默认 |
|---|---|
| `fs.read` | 允许 |
| `fs.write` | 仅 install/env 方法 |
| `net` | 仅 resolve/download |
| `env.write` | 仅 configure_env |
| `process.spawn` | 调用 ffmpeg / scoop / brew 时 |

宿主应拒绝未授权方法，不依赖插件自律。

---

## 6. C ABI 后续评估

Go 可以用 `-buildmode=c-shared` 导出 C ABI，但注意：

- Go runtime 会进入宿主进程；
- Windows DLL 卸载 Go runtime 不安全；
- 线程/信号/GC 与宿主可能冲突；
- 不能跨边界传 Go string/slice/map/interface；
- 必须导出 opaque handle + `free` 函数；
- 必须 catch panic，不能让 Go panic 跨 C 边界；
- 更适合“一个宿主长期持有一个 OneMoreFFmpeg 实例”，不适合频繁 dlopen/dlclose。

结论：

> v1 用 JSON-RPC 进程插件；C ABI 只在 LAE/VideoDamage 明确需要 in-process 性能时再开 ADR 评估。

---

## 7. LAE / VideoDamage 集成

### 7.1 只读集成

```text
LAE / VideoDamage
  → OneMoreFFmpeg detect
  → OneMoreFFmpeg capabilities
  → 得到 trio_dir + capabilities + fingerprint
  → 自己决定是否使用
```

### 7.2 生成锁文件

```text
OneMoreFFmpeg install --provider btbn --channel stable --variant gpl --linkage static
  → 写 install.json
  → 可选：--emit-trio-lock
  → 生成 VideoDamage C1 ffmpeg-trio.lock
```

约束：

- 只有 BtbN stable GPL static 才允许 `--emit-trio-lock`；
- BnqDzj nonfree、daily、shared 不允许冒充 C1；
- 锁文件字段必须按 VideoDamage C1，不按 OneMoreFFmpeg 内部格式；
- 不把 `latest` 写入锁文件。

### 7.3 安装/环境变更

需要显式权限：

```text
install / configure_env / rollback
```

VideoDamage core 默认不调用这些方法；它们属于 Extension Pack / 部署工具。

---

## 8. 版本策略

| 对象 | 版本字段 |
|---|---|
| Go API | `onemore/api/v1` |
| CLI JSON | `schema` |
| capabilities | `onemoreffmpeg.capabilities/v1` |
| artifact | `onemoreffmpeg.artifact/v1` |
| install record | `onemoreffmpeg.install/v1` |
| journal | `onemoreffmpeg.journal/v1` |
| plugin | `onemoreffmpeg.plugin/v1` |

不向下兼容旧实验版本；主版本变化即拒绝，不静默迁移。
