# Phase 3 代码审查与修复记录

> 日期：2026-11-11
> 范围：`internal/capabilities/`、`internal/model/`、`internal/cli/`、`internal/testutil/` 及 Phase 3 相关文档
> 状态：P1 已修复，Phase 3 可冻结
> 基线工具链：Go 1.27.4 windows/amd64（`go.mod` 要求 `go 1.27.3`，`GOTOOLCHAIN=auto`）
> 真实样本：FFmpeg `N-128713-g0f89d936b3-20261108`（win64 nonfree static，本机 PATH）

---

## 1. 结论

Phase 3 的结构、错误码、进程边界与缓存设计可以保留，但初版测试全部使用手写 golden，未覆盖真实 FFmpeg 输出形态，导致 3 个会污染公开 JSON 的解析缺陷和 3 个状态/缓存缺陷。

当前判定：

> **Phase 3 可以冻结，进入 Phase 4：Provider / Resolve / Download。**

进入 Phase 4 的前置项（数据模型、路径、checksum、HTTP 可测性）见 [`PHASE4_READINESS.md`](PHASE4_READINESS.md)。

---

## 2. 发现的问题

### 2.1 已修复（P1）

#### P1-1：`-pix_fmts` 表头被解析成能力条目

真实输出：

```text
FLAGS NAME            NB_COMPONENTS BITS_PER_PIXEL BIT_DEPTHS
-----
IO... yuv420p                3             12      8-8-8
```

`flagPattern` 把 `FLAGS` 当作 flag、`NAME` 当作能力名，输出中出现：

```json
{ "name": "NAME", "flags": "FLAGS", "description": "NB_COMPONENTS BITS_PER_PIXEL BIT_DEPTHS" }
```

修复：

- `isColumnHeader` 跳过 `FLAGS NAME ...` 表头；
- `description` 改为 `strings.Join(fields[2:], " ")`，避免把列间填充空格写进 JSON；
- `raw` 仍保留原始行。

#### P1-2：`-layouts` 把独立声道混入 `channel_layouts[]`

真实 `-layouts` 同时输出：

```text
Individual channels:
NAME           DESCRIPTION
FL             front left
...

Standard channel layouts:
NAME           DECOMPOSITION
mono           FC
stereo         FL+FR
```

初版把两节都解析进 `channel_layouts[]`，真实样本得到 78 条，其中 `FL` / `FR` / `LFE` 等并非布局。

修复：

- 新增 `parseChannelLayouts`：只解析 `Standard channel layouts:` 之后的内容；
- 找不到该节时回退到旧行为，兼容老版本 FFmpeg；
- 真实样本条目数从 78 变为 42，与 FFmpeg 标准布局数量一致。

#### P1-3：13 个命令全部失败仍返回成功

`ERROR_CODES.md` §5 规定“只有所有能力命令都失败……才返回整体错误”，但初版只写 `OMF-W1002` 并返回空能力集，消费方无法区分“FFmpeg 损坏”和“没有该能力”。

修复：

- 全部命令失败时返回 `OMF-E2001`，`details` 带 `installation_id` / `ffmpeg` / `commands`；
- 该路径不写缓存；
- 部分失败仍是成功 + `OMF-W1002`。

#### P1-4：取消/超时会污染缓存

初版在父 context 取消后仍把 13 个 `ok=false` 的结果写入缓存；下次运行直接 `cache_hit=true` 返回空能力集。

修复：

- `Collect` 在写缓存前检查 `ctx.Err()`：取消 → `OMF-E9004`，超时 → `OMF-E1006`；
- 两条路径都不写缓存；
- 新增 `TestCanceledContextDoesNotWriteCache`。

#### P1-5：`-h <topic>` 未命中被当成成功

真实行为（FFmpeg 退出码 0，stdout 为错误文本）：

```text
Codec 'nosuchcodec' is not recognized by FFmpeg.
Unknown filter 'nosuch'.
Unknown format 'nosuch'.
```

初版记录 `ok=true`，仅把错误文本放进 `help.raw`。

修复：

- 新增 `helpTopicMissing` 识别三类消息；
- 未命中时该命令 `ok=false`、`error="help topic not found"`、`warnings` 增加 `OMF-W1002`；
- `help[topic]` 新增 `found` 字段；`found=false` 表示 topic 未命中；
- 整体命令仍成功（单个 topic 未命中不影响其他 12 项）。

#### P1-6：缓存临时文件名冲突

初版固定使用 `<fingerprint>.json.tmp`，两个 CLI 进程并发写同一指纹时会互相覆盖；`os.Remove` + `os.Rename` 也不具备原子性。

修复：

- `os.CreateTemp(dir, "<fingerprint>.json.*.tmp")` 生成唯一临时文件；
- 写完 `Close` 后 `Remove` 旧文件再 `Rename`；
- 失败路径清理临时文件；
- 进程间仍可能“后写覆盖先写”，但不再互相破坏文件内容。

#### P1-7：`--dir` 与 `--scan` / `--recursive` / `--roots` 冲突被静默忽略

`capabilities` 与 `validate` 在 `--dir` 分支强制关闭发现选项，用户显式传入的 `--scan` / `--roots` / `--recursive` 不生效也不报错。

修复：

- 新增 `rejectDirDiscoveryConflict`；
- 冲突时返回 `OMF-E7001`，退出码 2；
- 文档同步到 `CLI_SCHEMA.md` 与 `CAPABILITIES.md`。

### 2.2 已确认但未改代码（P2 / 记录）

| 项 | 说明 | 处理 |
|---|---|---|
| `-h full` 体积 | 真实样本 stdout 1,376,963 字节，缓存文件约 1.9 MB | 保留原文；8 MiB 上限保护；消费方需容忍大 JSON |
| 缓存无 TTL | 失效只依赖 fingerprint + `--refresh` | 保持现状；Phase 4 起 fingerprint 含二进制 hash，足够 |
| 命令规格版本 | 解析规则变化必须让旧缓存失效 | 本次 `capability-commands-v1` → `v2` |
| `E2002` / `E2003` / `E2004` / `E2006` | 注册但未使用 | 在 `ERROR_CODES.md` 标注为保留码 |
| `--no-path` / `--no-env` / `--no-common` 配 `--dir` | 仍被接受但无效果 | 文档标注仅对 `--installation-id` 有效 |
| `license.summary` | `-L` 的 banner 写 stderr、许可文本写 stdout；已加 banner 行防御 | 保持首行语义，测试覆盖真实样本 |
| 并发上限 | 代码固定 4，`ARCHITECTURE.md` 曾写“默认串行” | 增加 `Collector.MaxParallel`，文档改为有界并行 4 |
| nil 归一化重复 | `model.CapabilitySet.MarshalJSON` 与缓存读取各写一份 | 抽出 `CapabilitySet.Normalize()` |

---

## 3. 代码变更清单

```text
internal/capabilities/parser.go       表头/分节解析、help 未命中、license banner 防御、description 归一
internal/capabilities/capabilities.go  E2001/E9004/E1006、取消不写缓存、唯一临时文件、MaxParallel、spec v2
internal/capabilities/parser_test.go  真实输出 testdata 用例
internal/capabilities/capabilities_test.go  全失败/单失败/help 未命中/取消不写缓存
internal/capabilities/testdata/       pix_fmts / layouts / license_gpl / license_nonfree / help_not_found
internal/model/model.go               HelpInfo.Found、CapabilitySet.Normalize
internal/cli/options.go               rejectDirDiscoveryConflict
internal/cli/capabilities.go          --dir 冲突校验
internal/cli/validate.go              --dir 冲突校验
internal/cli/root_test.go             --dir 冲突用例
internal/testutil/fake_ffmpeg.go      OMF_FAKE_FAIL_ARGS 单命令失败
docs/CAPABILITIES.md                  解析规则、found、缓存写入条件、E2001
docs/CLI_SCHEMA.md                    help.found、--dir 互斥、E2001
docs/ERROR_CODES.md                   E2001 实际语义、保留码说明
docs/ARCHITECTURE.md                  CapabilitySet 模型、并发与缓存写入
docs/schemas/examples/capabilities.example.json  help.full.found
```

---

## 4. 测试覆盖

新增：

- `-pix_fmts` 表头不产生 `NAME` 条目；
- `-layouts` 只取标准布局，不含 `FL` / `FR` / `BR`；
- 真实 GPL / nonfree license 文本与 banner 泄漏场景；
- `helpTopicMissing` 三类真实消息 + 正常 help + 空输出；
- 单命令失败保留其他结果；
- 13 个命令全部失败 → `OMF-E2001` 且不写缓存；
- help topic 未命中 → `ok=false` / `found=false` / `OMF-W1002`；
- context 取消 → `OMF-E9004` 且不写缓存；
- CLI `--dir` + `--scan` / `--recursive` / `--roots` → `OMF-E7001`。

保留：

- 13 命令全部成功（真实 fake 输出）；
- 缓存命中、`--refresh`、缓存损坏、`--no-cache`；
- ffplay 缺失 `OMF-W1001`；
- 非法 help topic `OMF-E2005`；
- 契约示例与错误码注册表双向校验。

---

## 5. 验证结果

```text
gofmt -l .                             无输出
go build ./...                         通过
go test -count=1 ./...                 全部通过
go vet ./...                           通过
go build -tags deps ./...              通过
GOOS=windows GOARCH=amd64 go build     通过
GOOS=linux   GOARCH=amd64 go build     通过
```

真实 FFmpeg 冒烟：

```text
capabilities --dir <real ffmpeg dir> --no-cache --json
  license.summary = "This version of ffmpeg has nonfree parts compiled in."
  version.version = "N-128713-g0f89d936b3-20261108"
  warnings = 0
  pixel_formats = 270（无 NAME 条目；修复前 271 含表头伪条目）
  channel_layouts = 42（无 FL/FR/LFE）
  help.full.found = true

capabilities --dir <real ffmpeg dir> --help-topic decoder=nosuch --no-cache --json
  commands["-h decoder=nosuch"].ok = false
  error = "help topic not found"
  help["decoder=nosuch"].found = false
  warnings = [OMF-W1002 help topic not found]

capabilities --dir <dir> --scan --json
  OMF-E7001，退出码 2

capabilities --dir <dir> --cache-dir <tmp> --json（连续两次）
  第一次 cache_hit=false；第二次 cache_hit=true；缓存目录仅 1 个文件
```

---

## 6. 第二轮修复（Phase 3 收尾）

| 问题 | 修复 | 测试 |
|---|---|---|
| `--probe-timeout` 被 capabilities 接受但完全忽略 | `Options.CommandTimeout` 覆盖所有命令超时；仅在用户显式传参时生效 | `TestCommandTimeoutOverrideAppliesToEveryCommand` |
| 缓存命中只校验 13 个命令键，`help` 条目缺失也命中 | `readCache` 额外校验 `help[topic]` 与 `installation_id` | `TestCacheWithoutHelpEntryIsRejected` |
| 默认缓存目录自行拼接，与 `<data-root>` 布局脱节 | 改用 `internal/paths.CacheDir()` | `internal/paths` 测试 + 现有缓存测试 |
| `--no-cache` 仍解析 data root，无 `LOCALAPPDATA` 时直接失败 | 仅在实际读写缓存时解析路径 | `TestNoCacheDoesNotRequireResolvableDataRoot` |
| 缓存键不含生效的超时/输出上限，短 `--probe-timeout` 会固化空结果 | fingerprint 加入 `command_timeout` / `max_output_bytes` / `wait_delay`；规格版本升 v3 | `TestCacheKeyIncludesCommandTimeout` |
| help 未命中只读 stdout，stderr 诊断被忽略 | 同时匹配两路输出；空 stdout 仍视为未命中 | `TestHelpTopicMissing` |
| `Rename` 失败时先删旧缓存，提交失败会丢失有效缓存 | 只做一次 `os.Rename`，失败保留旧文件 | 现有缓存测试覆盖 |
| 缓存写入失败复用 `W1003`（语义为 stale） | 新增 `OMF-W1007 CACHE_WRITE_FAILED` | 注册表/文档双向校验 |

## 7. 保留的 Tech Debt

1. `-h full` 原文可达 1.4 MB，缓存与 stdout JSON 都在 MB 级；如消费方压力大，后续可加 `--help-raw-limit`。
2. 能力缓存无 TTL，长期不刷新依赖二进制 hash 变化或 `--refresh`。
3. 并发进程写同一指纹缓存时“后写覆盖先写”，没有跨进程锁（已改为优先原子 rename）。
4. help 未命中依赖 FFmpeg 英文消息文本；上游改词需要同步 `helpMissPattern`。
5. `--dir` 下的 `--no-path` / `--no-env` / `--no-common` 仍被接受但无效果。
6. 能力缓存键包含 `commandSpecVer`，解析器规则变更必须手动升版本，否则旧缓存不会失效。
7. 旧指纹缓存文件不会自动清理；频繁切换二进制时缓存目录会累积（每个约 1.9 MB）。

这些不阻塞 Phase 4。

---

## 8. 最终判定

> **Phase 3 的 13 项能力命令、真实输出解析、错误语义、缓存写入与 CLI 冲突校验已达到可冻结水平。**
>
> Phase 4 必须复用 `ffexec` 的进程/超时/输出上限，复用 `apperrors` 注册表，并遵守 [`PHASE4_READINESS.md`](PHASE4_READINESS.md) 中的前置项与决策项；不要绕过本文修复的 `found` / `E2001` / 缓存写入语义。
