# 全量审查与修复记录（2026-11-11）

> 范围：`internal/*`、`cmd/`、`docs/`、`config.example.toml`、示例 JSON
> 方法：4 路只读并行审计 + 逐条代码核实 + 真实 FFmpeg/真实 CLI 复验
> 基线：Go 1.27.4 windows/amd64；FFmpeg `N-128713-g0f89d936b3-20261108`（win64 nonfree static）

---

## 1. 结论

本轮共核实 40+ 项代码与文档问题；可独立修复的代码缺陷已全部修复，保留 3 类需要平台级改造或产品决策的问题（见 §5）。审计过程中还发现并修正了 Phase 3 期间引入的跨平台回归（`proc_unix.go` 的进程组终止）。

```text
gofmt -l .                 无输出
go vet ./...               通过
go test -count=1 ./...     全部通过（新增 40+ 回归用例）
go build -tags deps ./...  通过
GOOS=windows/linux go build 通过
```

Phase 3 可以冻结；Phase 4 的路径、checksum、模型、配置、契约校验、GitHub 客户端、provider 注册表与 `provider list` 已就绪。

---

## 2. 严重缺陷（已修复）

### 2.1 发现/校验

| 缺陷 | 修复 | 回归测试 |
|---|---|---|
| 扫描预算耗尽后仍用过期 ctx 探测，健康安装被标 `unusable` | 探测改用父 ctx，单进程只受 `--probe-timeout` 约束；哈希可取消 | `TestDetectValidationIsNotCanceledByScanBudget` |
| `validate --dir C:\ffmpeg` 选中无二进制的父目录（`bin` 布局） | 排序优先含 `ffmpeg` 的安装；`ValidateDir` 取第一个含 `ffmpeg` 的报告 | `TestValidateDirFindsBinSubdirectory` |
| 无二进制的候选产出“幽灵安装”，`E1002` 不可达 | 过滤空候选；`ValidateDir` 无有效安装返回 `E1002` | `TestDetectDropsCandidatesWithoutBinaries`、`TestValidateDirWithoutBinariesFails` |
| `MaxFiles` 按 root 重置，多盘扫描实际远超上限 | 计数器提升到 `scanRoots`，全 root 共享 | `TestScanMaxFilesIsGlobalAcrossRoots` |
| 每个目录一次 `EvalSymlinks`，全盘扫描主要瓶颈 | `WalkDir` 不跟随链接，改用词法 key | 现有扫描测试 + 真实 `detect --dir` |
| 哈希失败导致安装 ID 漂移 | 失败项写 `unavailable` 占位；version 为空时 fingerprint 也写 `unavailable` | `TestDetectInstallationIDStableWithoutValidation` |
| 权限不足被报成“目录不存在” | `pathError` 区分 `E1001` / `E1005` / `E1009` | 代码路径覆盖 |
| 扫描超时产生重复 `E1008` | 只在 `Detect` 汇总一次；root 内部超限保留单条 | `TestScanMaxFilesIsGlobalAcrossRoots` |

### 2.2 ffexec

| 缺陷 | 修复 |
|---|---|
| 超时/取消/启动失败时 `ExitCode` 残留 0 | 统一 `ffexec.ExitCodeNotStarted = -1` |
| 启动失败映射成 `E1004`（安装不可用） | 新增 `OMF-E1012 EXEC_FAILED`；权限错误映射 `E1005` |
| `Signal` 字段塞 `exitErr.String()`，`ExitCode` 被改写为 1 | Unix 用 `WaitStatus.Signal()` 取真实信号名；`ExitCode` 保留 `-1` |
| `Truncated` 不区分流 | 新增 `stdout_truncated` / `stderr_truncated`；能力 warning 按实际流命名 |
| Windows `=C:=...` 环境项塌缩成空键 | 重建 env 时跳过空键条目，不再互相覆盖 |
| `taskkill` 无超时 | 3 秒超时 + 失败回退 `Process.Kill()` |

### 2.3 能力检查

| 缺陷 | 修复 |
|---|---|
| `--no-cache` 仍解析 data root，无 `LOCALAPPDATA` 时失败 | 仅在实际读写缓存时解析缓存路径 |
| 缓存键不含生效超时/输出上限，短 `--probe-timeout` 固化空结果 | 指纹加入 `command_timeout` / `max_output_bytes` / `wait_delay`；规格升 v3 |
| help 未命中只检查 stdout | 同时匹配 stdout/stderr；空 stdout 仍视为未命中 |
| 缓存提交失败前先删旧缓存 | 单次 `os.Rename`，失败保留旧文件 |
| 缓存写入失败复用 `W1003` | 新增 `OMF-W1007 CACHE_WRITE_FAILED` |
| 能力 goroutine panic 绕过 `E9002` | `runCommands` 内 recover 并返回 `E9002` |
| 列表解析不看 `scanner.Err()`，超长行静默截断 | 解析后检查，计入 unknown → `W1002` |
| help topic 名字允许内部 `=` | 字符集去掉 `=` |

### 2.4 契约/校验

| 缺陷 | 修复 |
|---|---|
| `ValidateExample` 把 help 命令键硬编码为 `-h full` | 接受任意 `-h <topic>`，并要求 `help[topic]` 对应 |
| `error.message` 按字节限制 200 | 改为 `utf8.RuneCountInString` |
| 错误码正则无边界，5 位编号被误判 | `\bOMF-[EW][0-9]{4}\b` |
| `ProviderSummary` 单序列化输出 `null` | 增加 `MarshalJSON`；`Normalize` 改为拷贝、不修改调用方 |

### 2.5 checksum

| 缺陷 | 修复 |
|---|---|
| deadline 被映射成“用户取消” `E4006` | deadline → `E4001`，取消仍 `E4006` |
| 重复且冲突的摘要静默取第一条 | 冲突返回 `E4002` |
| `./name`、`dist/name` 被整行丢弃 | 允许相对前缀，仍拒绝绝对路径与 `..` |
| checksums 读取 I/O 错误报 `E4002` | 改为 `E4001` |

### 2.6 CLI / 配置 / 日志

| 缺陷 | 修复 |
|---|---|
| 未知子命令退出 0 并打印帮助 | root `Action` 检测位置参数，返回 `E7001` + 退出 2 |
| 叶子命令多余参数被静默忽略 | `rejectExtraArgs` 统一拒绝 |
| `<data-root>/config.toml` 从不加载 | 新增 `config.LoadOptional` + 默认路径解析；`--config ""` 表示只用默认值 |
| `validate --installation-id` 静默忽略 `--scan/--roots/--recursive` | 冲突返回 `E7001` |
| `validate --no-validate` 被接受但无效果 | 直接 `E7001` |
| `capabilities` help 列出被拒绝的 `--no-validate` | 从该命令 flag 集合中移除 |
| `--probe-timeout` 大于 `--timeout` 时被整体预算截断 | 解析后校验，返回 `E7005`；秒数上限 86400 |
| `--include-removable/--include-network` 无 `--scan` 时静默无效 | 返回 `E7001`，Usage 标注依赖 |
| Windows `color=auto` 不加 VT 直接输出 ANSI | 检测并启用 `ENABLE_VIRTUAL_TERMINAL_PROCESSING`，失败则禁用颜色 |
| `version` 默认 `0.1.0`、非法 buildTime 原样输出 | 默认 `dev`；非 RFC 3339 归一为 `unknown`；`vcs.time` 兜底 |
| 只监听 `SIGINT` | 增加 `SIGTERM`；`os.Exit` 前显式 `stop()` |

---

## 3. 新增的 Phase 4 基础设施

| 组件 | 内容 |
|---|---|
| `internal/provider` | `Provider` 接口、`Request`、内置 `btbn` / `bnqdzj-nonfree` 元数据、`Registry`、`E3001` |
| `internal/provider/github` | `LatestRelease` / `ReleaseByTag`；https-only（loopback 例外）、3 次重试、`Retry-After`、403 限流、`len(via)` 重定向上限、4 MiB 响应上限、token 不进错误 |
| `provider list` | 离线输出两个内置 provider，schema `onemoreffmpeg.provider-list/v1` |
| 契约校验 | provider-list 非空、provider-add https/status/sha256、release 15 字段、artifact 必填与 `verified=true` |

`resolve` / `download` 仍未实现：先在 [`PHASE4_READINESS.md`](PHASE4_READINESS.md) 落文 D1（stable pin）与 D5（nonfree 显式同意）。

---

## 4. 修复后的真实行为

```text
validate --dir <ffmpeg 父目录>       exit 0  status=healthy（自动发现 bin/）
validate --dir <空目录>              exit 1  OMF-E1002
bogus --json                        exit 2  OMF-E7001
version extra --json                exit 2  OMF-E7001
capabilities --no-cache（HOME 无效） exit 0  13 项命令正常
<data-root>/config.toml 非法 level   exit 2  OMF-E7005
provider list --json                exit 0  2 个 provider
capabilities 连续两次               cache_hit=true
```

---

## 5. 保留问题（未修复）

| 项 | 原因 | 处理 |
|---|---|---|
| Windows 进程树无 Job Object | 需要 `CREATE_SUSPENDED` + `AssignProcessToJobObject` 的平台实现，改动面大 | **D7 已决定推迟到 Phase 5**；当前 `taskkill` 3s 超时 + `Process.Kill` 兜底，见 `ARCHITECTURE.md` §6 |
| `--trace` 与 `--debug` 无可观测差异 | 生产代码只有一条 debug 日志 | 保留 `trace` 作为 `debug` 的同义级别，随写入类命令补日志后再区分 |
| config 错误分类依赖 `go-toml` 文案 | — | **已修复**：先用非 strict 解析验证语法（`E7002`），typed decode 非 strict 失败统一 `E7005`，不再匹配错误文案 |
| `E1003/E1010/E1011/E1004` 部分不可达 | 当前设计把失败放进 `checks[]`/`status` | 在 `ERROR_CODES.md` 标注“保留/未来层使用” |
| `resolve` 的 `license_hint` 推导 | 依赖资产匹配 | Phase 4 `provider` 实现时按 provider + variant 推导（D1/D5 已定） |
| 缓存无 TTL、旧指纹文件不清理 | 低风险，1.9 MB/文件 | 后续加 `--cache-gc` |
| D1 stable pin / D5 nonfree 同意 | 产品决策 | 见 `PHASE4_READINESS.md` §4 |
