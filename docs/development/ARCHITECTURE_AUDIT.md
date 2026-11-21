# OneMoreFFmpeg 全项目架构审计报告

> 审计日期：2026-11-21
> 审计基准：v1.1.2（commit `24a3f9f`）
> 审计范围：整个仓库（代码、测试、文档、CI、依赖）
> 审计性质：**第一阶段——只审计，不修改任何源码**
> 后续：等待用户批准目标架构与迁移计划，再按 Phase 1–8 分阶段实施

---

## 0. 审计原则与约束

本次审计遵循以下原则：

1. 优先保留现有正确实现，不为了采用某种架构而强制改造代码。
2. 优先消除重复代码和职责混乱，避免过度工程化。
3. 避免为了拆分而产生大量只有一个函数的小 package。
4. 不允许循环依赖，不允许所有模块依赖一个超级 common/utils 包。
5. 不引入没有实际必要的第三方依赖，不为了使用接口而使用接口。
6. 不随意 Export 原本只需要包内使用的函数。
7. 不允许在缺乏测试的情况下大规模移动和重命名代码。
8. **本次先完成架构审计、架构设计和模块整理方案，不实施**：
   - SSRF 策略加强
   - Journal 事务机制大改
   - Ed25519 签名验证
   - 自更新安全策略大改
   - 其他与结构重构无关的新功能
9. 安全发现登记到本文件末尾的安全问题清单，并指明未来负责模块。

---

## 1. 当前完整目录树（真实，非文档推测）

```
OneMoreFFmpeg/
├── cmd/one-more-ffmpeg/
│   ├── main.go
│   ├── script_test.go
│   └── testdata/{json_contract,provider_list,version}.txtar
├── internal/                       # 34 个包（见第 2 节）
├── test/
│   ├── e2e/
│   │   ├── test_cli_e2e.py
│   │   ├── test_cli_full_e2e.py
│   │   ├── test_env_real_e2e.py
│   │   ├── test_update_e2e.py
│   │   ├── conftest.py / pytest.ini / requirements.txt
│   │   ├── robot/cli.robot
│   │   └── bats/cli.bats
│   ├── infra/remote_linux.py
│   └── README.md
├── docs/
│   ├── README.md
│   ├── guides/{HOWTO,INSTALL,INTEGRATION,MIGRATION,USAGE}.md
│   ├── reference/{CAPABILITIES,CLI_SCHEMA,CUSTOM_URL_SPEC,
│   │   DETECTION_AND_VALIDATION,ENVIRONMENT,ERROR_CODES,
│   │   PLATFORM_ENVIRONMENT,PROVIDERS,PROVIDER_RESOLVER,
│   │   SYSTEM_REQUIREMENTS,VERSIONING}.md
│   │   ├── man/one-more-ffmpeg.1
│   │   └── schemas/{README.md, contract.schema.json, examples/*.json}
│   ├── development/{ARCHITECTURE,E2E_TESTING,TECHNICAL_NOTES}.md
│   │   └── adr/{0001,0002,0003}.md
│   ├── releases/{CHANGELOG, release-v*.md}
│   └── deployment/{README.md, httpd.conf, nginx.conf, Caddyfile, nginx.repo}
├── examples/external-caller/
├── .github/workflows/{e2e.yml, release.yml}
├── scripts/{elevate.ps1, enable-developer-mode.ps1}
├── third_party/licenses/           # 26 份许可证
├── packaging/ / publish/ / bin/ / dist/   # 构建与历史产物
└── config.example.toml, go.mod, go.sum, mise.toml, .golangci.yml,
    Dockerfile, docker-compose.yml, .dockerignore, .gitignore,
    README.md, SECURITY.md, THIRD_PARTY_NOTICES.md, LICENSE
```

---

## 2. 当前 package 清单（`go list ./...`，34 个）

```
cmd/one-more-ffmpeg
internal/archive
internal/buildinfo
internal/cache
internal/capabilities
internal/checksum
internal/cli
internal/config
internal/contracts
internal/customurl/checksum
internal/customurl/latest
internal/customurl/naming
internal/customurl/urlpolicy
internal/deps
internal/detect
internal/doctor
internal/download
internal/env
internal/errors
internal/ffexec
internal/install
internal/journal
internal/lock
internal/logging
internal/model
internal/output
internal/paths
internal/platform
internal/provider
internal/provider/github
internal/provider/official
internal/selfupdate
internal/testutil
```

---

## 3. 当前实际依赖关系图（生产代码）

```
cmd/one-more-ffmpeg → cli, buildinfo

cli → buildinfo, config, errors, logging, paths, contracts, output, model,
      detect, capabilities, provider, provider/official, download, checksum,
      ffexec, install, journal, lock, env, platform, doctor, selfupdate   (22 个)

provider → errors, model, checksum, archive, lock,
           customurl/checksum, customurl/latest, customurl/naming,
           customurl/urlpolicy                                           (9 个)
provider/github   → errors
provider/official → errors, model

contracts   → capabilities, errors, model
capabilities→ errors, ffexec, logging, model, cache, paths
detect      → errors, ffexec, model, platform
download    → checksum, errors, logging, model, output, platform
install     → archive, checksum, errors, ffexec, journal, logging, model, platform
env         → errors, install, journal, model, platform
doctor      → env, errors, ffexec, journal, model
selfupdate  → errors, logging, output, platform, provider/github
archive     → errors, platform
logging     → config, errors
platform    → errors, model                    ← 基础设施反向依赖 DTO（问题）
journal / lock / config / paths / checksum / ffexec / customurl/* → errors
errors, model, output, cache, buildinfo, deps, testutil → 零内部依赖
```

**用户点名的错误方向均未发现**：
- `errors → cli`：否（errors 零依赖底座）
- `checksum → download`：否（方向相反，正确）
- `archive → install`：否（方向相反，正确）
- `feature`/`shared` 包：不存在

**异常点**：
- `platform → model`（基础设施依赖业务 DTO）
- `contracts → capabilities`（契约层依赖能力层）
- `cli` 扇出 22 个包（扇出极端点）
- `provider` 扇出 9 个包（中枢耦合）

---

## 4. 每个 package 的职责审查表

> 行 = 生产/测试；导出 = 顶层导出函数/导出类型；依 = 直接内部依赖数；被依 = 被生产代码引用数（估算）

| 包 | 职责 | 行(产/测) | 导出(函/类) | 依 | 被依 | 问题标签 |
|---|---|---|---|---|---|---|
| `internal/cli` | 命令解析+编排+输出 | 2362/1044 | ~15/3 | **22** | 1 | **越层编排、胖 CLI、扇出过高** |
| `internal/provider` | 注册表+自定义 provider 存储 | 920/482 | ~15/~6 | **9** | 3 | **职责过载、中枢耦合** |
| `internal/contracts` | JSON 契约/错误码校验 | 787/687 | 9/~3 | 3 | 2 | **巨型文件 746 行、依赖能力包** |
| `internal/detect` | 只读发现/校验/fingerprint | 1045/454 | 2/~10 | 4 | 3 | 体量偏大 |
| `internal/capabilities` | 13 项能力命令+解析+缓存 | 930/584 | 5/~3 | 6 | 2 | 含表现依赖(logging) |
| `internal/env` | PATH/变量写入+回滚 | 918/448 | 5/~3 | 5 | 3 | 复杂函数 152 行 |
| `internal/install` | 版本化安装+current+回滚 | 976/394 | 6/~2 | 8 | 3 | 依赖面广 |
| `internal/selfupdate` | 自身更新事务 | 775/330 | ~1/2 | 5 | 1 | **Apply() 229 行巨型函数** |
| `internal/download` | 流式下载+双重校验 | 530/491 | 1/2 | 6 | 2 | 依赖表现层(output/logging) |
| `internal/platform` | 平台适配(链接/权限/环境) | 659/154 | ~14/1 | 2 | 8 | **反向依赖 model** |
| `internal/model` | JSON DTO+schema 注册表 | 495/157 | 0/~31 | 0 | 13 | **上帝 DTO、可变全局 map** |
| `internal/archive` | zip/tar 安全解压 | 407/347 | ~4/0 | 2 | 2 | 依赖 platform |
| `internal/provider/github` | GitHub Releases 客户端 | 402/288 | 1/3 | 1 | 2 | 职责单一 |
| `internal/errors` | 稳定错误码注册表 | 306/44 | 8/3 | 0 | ~24 | 健康（零依赖底座） |
| `internal/provider/official` | 官方 provider 元数据 | 305/139 | 1/1 | 2 | 1 | 职责单一 |
| `internal/cache` | 通用缓存读写 | 268/160 | ~6/3 | 0 | 1 | **Store 无生产调用** |
| `internal/ffexec` | 进程执行/超时/进程树终止 | 333/209 | 3/3 | 1 | 6 | Runner 不可注入 |
| `internal/checksum` | 流式 SHA256+清单解析 | 132/154 | 5/0 | 1 | 3 | 职责纯净 |
| `internal/output` | JSON/进度输出 | 322/155 | 3/5 | 0 | 3 | 表现层，健康 |
| `internal/config` | 严格 TOML 配置 | 201/319 | 6/4 | 1 | 2 | 职责单一 |
| `internal/journal` | 事务 journal | 153/71 | 6/1 | 1 | 5 | 职责单一 |
| `internal/logging` | slog/tint 配置 | 122/45 | 5/1 | 2 | 5 | 依赖 config（可接受） |
| `internal/paths` | data root/路径 | 114/120 | 8/1 | 1 | 3 | 职责单一 |
| `internal/customurl/naming` | 资产命名解析 | 173/51 | 5/1 | 1 | 1 | 职责单一 |
| `internal/customurl/checksum` | 校验和 URL 解析 | 128/41 | 3/0 | 1 | 1 | 职责单一 |
| `internal/customurl/urlpolicy` | URL/TLS 策略 | 95/50 | 2/1 | 1 | 1 | 职责单一 |
| `internal/customurl/latest` | latest 版本读取 | 66/27 | 2/0 | 1 | 1 | 职责单一 |
| `internal/lock` | 跨进程文件锁 | 82/115 | 4/0 | 1 | 2 | 职责单一 |
| `internal/buildinfo` | 版本/提交/构建时间 | 57/33 | 1/1 | 0 | 2 | 职责单一 |
| `internal/doctor` | 安装健康检查 | 163/176 | ~6/1 | 5 | 1 | 依赖面较广 |
| `internal/testutil` | 测试用 fake FFmpeg | 231/0 | 4/1 | 0 | 0 | 非 `_test.go` 会进构建 |
| `internal/deps` | 未来依赖占位 | 5/0 | 0/0 | 0 | 0 | **死代码** |
| `cmd/one-more-ffmpeg` | main+信号+buildinfo 注入 | 30/42 | 0/0 | 2 | 0 | 职责单一 |

**覆盖率**：本次未运行 `go test -cover`，全部 N/A（审计阶段不改代码，未编造数字）。建议 Phase 7 建立基线。

---

## 5. 过大文件和复杂函数清单

**过大文件（生产代码 > 400 行）**

| 文件 | 行数 |
|---|---|
| `internal/contracts/contracts.go` | 746 |
| `internal/download/download.go` | 530 |
| `internal/env/apply.go` | 498 |
| `internal/selfupdate/selfupdate.go` | 486 |
| `internal/capabilities/capabilities.go` | 427 |
| `internal/archive/archive.go` | 407 |
| `internal/provider/github/client.go` | 402 |

**复杂/超长函数**

| 函数 | 文件:行 | 估算行数 |
|---|---|---|
| `selfupdate.(Options).Apply` | `internal/selfupdate/selfupdate.go:161` | **~229** |
| `env.buildEnvPlan` | `internal/env/apply.go:141` | ~152 |
| `cli.root.New` | `internal/cli/root.go:113` | ~108 |
| `download.(*Downloader).download` | `internal/download/download.go:211` | ~107 |
| `archive.extractTar` | `internal/archive/archive.go:224` | ~101 |
| `download.(*Downloader).Fetch` | `internal/download/download.go:90` | ~98 |
| `contracts.validateRelease` | `internal/contracts/contracts.go:563` | ~87 |
| `archive.extractZip` | `internal/archive/archive.go:143` | ~81 |
| `env.runEnvTransaction` | `internal/env/apply.go:60` | ~81 |
| `contracts.validateCapabilities` | `internal/contracts/contracts.go:501` | ~62 |

不能简单以行数判断是否拆包；应根据职责划分文件和 package。

---

## 6. 重复函数和方法清单（含精确行号）

### 6.1 真重复（应提取）

| 类别 | 位置 | 结论 |
|---|---|---|
| `sha256Hex(data []byte)` ×5 | `detect/paths.go:163`、`ffexec/runner.go:105`、`capabilities/cache.go:108`、`install/files.go:57`(checksumOfString)、`cache/cache.go:215`(内联) | 输入/输出/错误/副作用/并发/平台/安全全同。提取 `checksum.SumHex` |
| `isLowerHex` ×3 + 变体 2 | `checksum/checksum.go:125`、`customurl/checksum/checksum.go:119`、`provider/official/official.go:189`；变体 `selfupdate.go:443 isHex64`、`download.go:39 sha256Pattern` | A/B/C 逐字符相同。提取 `checksum.IsLowerHex64`/`IsHex64` |
| `isLoopback` ×2 + 弱化 1 | `provider/github/client.go:385`、`config/config.go:195`（逐字符相同）；弱化 `selfupdate.go:466`（仅字符串比较，漏 IP 变体） | 提取 `netutil.IsLoopbackHost` |
| URL 校验严格族 ×3 | `config.go:177`、`selfupdate.go:473`、`github/client.go:393` | 逻辑一致（https 放行、http 仅 loopback、拒 userinfo）。提取 `netutil.ValidateStrictURL` |
| URL 校验宽松族 ×3 | `download.go:524`、`customurl/urlpolicy/urlpolicy.go:26`、`provider/fetch.go:18-28` | 自托管 http 允许。**与严格族绝对不可合并** |
| redirect 闭包 ×5 | `download.go:396-411`、`urlpolicy.go:75-93`（唯一检查整条链）、`fetch.go:37-50`、`github/client.go:114-125`、`selfupdate.go:194-206` | 安全语义不同；共享策略构造器但保留 `urlpolicy` 的整链检查；`selfupdate` 用裸 `fmt.Errorf` 是唯一丢失错误码者 |
| 有界文本抓取 ×2 | `provider/fetch.go:18 fetchURL`、`customurl/checksum/checksum.go:44 Fetch` | 近乎行级重复（UA、状态码、LimitReader(max+1)、超限 E4004）。提取 `netutil.FetchLimited` |
| 原子写 ×7 | `cache/fileio.go:35`、`cache/cache.go:103`、`journal/journal.go:68`、`download.go:429`、`install/record.go:52`、`env/shell.go:182`、`selfupdate/ops.go:47` | 1/2/3 跨进程锁（lockedfile），4/5/6/7 无锁靠 rename。**5 与 6 真重复**。抽底层原语，不可合并为单一函数 |
| 临时路径预留 ×2 | `platform/link.go:67 reservePath`、`download.go:470-476` | 同一原语两次实现。提取 `fsatomic.ReservePath` |
| `linkKind` ≡ `createdKind` | `install/current.go:27`、`platform/link.go:60` | 逻辑完全相同（Lstat + ModeSymlink）。提取 `platform.LinkKindAt` |
| `containedIn` ≡ `LexicallyInside` | `install/rollback.go:17`、`platform/pathguard.go:12` | 完全等价（Rel + 拒绝 `.`/`..`/`..sep`）。`containedIn` 直接调用后删除 |
| `findBinary` ×2 | `install/files.go:103`、`doctor/doctor.go:150` | 逐字符相同。提取 `ffexec.FindBinary` |
| context 错误映射 ×4 | `download.go:514`、`github/client.go:331`、`checksum.go:31-44`、`ffexec/runner.go:204-211` | 结构同构，领域错误码不同。提取 `errors.ContextError(err, timeoutCode, cancelCode)` |
| RFC3339 UTC 格式化 ×7 | journal/capabilities/directory/install/official 等 | `time.Now().UTC().Format(time.RFC3339)`。提取 `clock.FormatUTC(now)` |
| `toReleaseInfo` 映射 ×2 | `selfupdate/gh_latest.go:68-84`、`selfupdate.go:109-115` | 字段逐一对应。包内提取 |
| 重试退避 ×2 | `github/client.go:185 getJSON` + `backoff:319`、`selfupdate/retry.go:18` | 层级不同不可合并，但 Backoff/Sleep 原语可共享 |
| 文件 SHA-256 ×1 真重复 | `detect/paths.go:122 hashFile`（手写循环）vs `install/files.go:62 checksumFile`（委托 `checksum.StreamSHA256`） | B 应改为委托，错误码映射为 E1009 |

### 6.2 伪重复（不应抽取）

1. `output/output.go:34 WriteJSON`（stdout）vs `cache/fileio.go:35 WriteJSON`（带锁文件持久化）——仅同名。
2. `install/files.go:145 copyFile`（本地复制）vs `download.go:211 download`（HTTP 流+校验+进度）。
3. `download.go:460 commitFile`（普通文件+唯一 .bak）vs `platform/link.go:24 ReplaceLink`（符号链接/junction）。
4. `detect/paths.go:90 findBinary`（含可执行位与平台扩展名语义）vs `install/files.go:103 findBinary`。
5. **严格 URL 策略 vs 宽松 URL 策略**——安全语义必须保持不同。
6. `checksum.ParseChecksums`（宽松，允许 `dist/name`，相同重复静默）vs `customurl/checksum.Parse`（严格裸文件名，任何重复报错）——只共享行扫描内核。
7. `install/current.go:34 currentType`（带 target 校验与外来文件识别的语义超集）vs `linkKind`。
8. `model/provider.go:98,103` 两个 `MarshalJSON`——Go 方法无法泛型抽取，Normalize 内容不同。
9. `ffexec/runner.go:193`、`lock/lock.go:58,73` 的 `time.Now()`——单调计时/超时，不应改成 Clock 注入。
10. ID 拼接（`inst-`/`art-`/`journal-` + digest[:16]）——过于 trivial。
11. `doctor.dirExists` 单行判断、各处 `WithTempFile` cleanup 闭包——生命周期与 journal/进度耦合，只抽 `ReservePath`。

---

## 7. 推荐共享 package 清单（只列 ≥2 处真实重复者）

| 包 | 新增职责 | 覆盖的重复点 |
|---|---|---|
| `internal/checksum`（扩展） | `SumHex` / `IsLowerHex64` / `IsHex64` / `scanChecksums` / `CopyAndHash` | 5×sha256Hex、3×isLowerHex、2×解析内核、detect.hashFile |
| `internal/netutil`（新建） | `IsLoopbackHost` / `ValidateStrictURL` / `ValidateCustomURL` / `RedirectPolicy` / `FetchLimited` | 3×loopback、6×URL 校验、5×redirect、2×有界抓取 |
| `internal/fsatomic`（新建，可由 `cache/fileio.go` 升格） | `WriteFile` / `WriteJSONLocked` / `ReservePath` | 7×原子写、2×临时路径预留 |
| `internal/ffexec`（扩展） | `FindBinary` / `HasBanner` | install/doctor 的 findBinary、detect/doctor 的 banner |
| `internal/platform`（扩展） | `LinkKindAt` / `Canonicalize`；让 `containedIn` 改用 `LexicallyInside` | linkKind/createdKind、containedIn、detect 的 Abs/EvalSymlinks |
| `internal/clock`（新建） | `FormatUTC(now)` / `UTCNow(now)` | 7×RFC3339 |
| `internal/errors`（扩展） | `ContextError(err, timeoutCode, cancelCode)` | download/github/checksum/ffexec 四处 |
| `internal/retry`（可选新建） | `Backoff` / `Sleep` | github.getJSON、selfupdate.retryTransport |
| selfupdate 包内提取（无需新包） | `toReleaseInfo` | gh_latest.go 与 selfupdate.go 的映射重复 |

**接口原则**：只有 `selfupdate.FileOps` 需要保留接口（它有真实测试注入价值，属正当抽象）；其余提取不需要接口。

---

## 8. 当前不应抽取的函数清单

见第 6.2 节伪重复清单，以及：
- `internal/lock` 的 `DataRoot`/`FilePath` 已委托共用 `acquire`，**已经去重**
- `internal/errors` 的 `New`/`WithDetails`/`WithCause`/`As` 统一，**无重复**
- `internal/logging` 的 `New`/`FromContext` 单一实现
- `ffexec.Runner.Run` 被 detect/capabilities/install/doctor 统一使用
- `journal.NewID` 是唯一随机 ID 生成器

---

## 9. 过度抽象和疑似死代码清单

| # | 位置 | 结论 |
|---|---|---|
| 1 | `internal/deps/` | **死代码**：仅 5 行 `doc.go`，声称的 `future.go`（build tag deps）不存在。建议删除 |
| 2 | `selfupdate.Options.Now`（`selfupdate.go:49`） | **死注入**：在 `normalized():77-79` 被赋值，但 Check/Apply/Recover 全文未使用。删除或真正用于 state 时间戳 |
| 3 | `cache.Store[K,V]`（`cache/cache.go:22-55`）及 TTL/LRU/Prune | **未接入框架**：仅测试调用，无生产调用（capabilities 走 `cache.ReadJSON/WriteJSON`）。删除或接入 |
| 4 | `platform.LinkKind()`（`link.go:14`） | 无生产调用点；与 `createdKind`/`linkKind` 的"实际类型"概念混淆 |
| 5 | `github.LatestRelease` / `ReleaseByTag` | 生产路径未调用（go-github 取代） |
| 6 | `provider.Provider` 接口 + `Registry.Add()` | 注释称 "used for custom entries if needed in future"，当前无真实外部实现者 |
| 7 | `errors.E5006` / `E6004` | 注释明确无代码路径发出——**保留常量（数据兼容），但不补实现** |
| 8 | `KnownCodes`（`registry.go:207`）、`defaultWait`（`lock.go:82`） | 仅测试使用，可保留但标注为测试辅助 |
| 9 | `install/files.go:20 TargetDir` | 为跨包提供的一行导出薄壳，伪 API |
| 10 | `testutil/fake_ffmpeg.go`（231 行，非 `_test.go`） | 会进入正常构建；建议加 build tag |
| 11 | `selfupdate/gh_latest.go:30-32` 注释 | 自称 "foundation for future features"——去掉未来叙事或确实提取 `toReleaseInfo` |
| 12 | 双 HTTP 客户端体系（4 套注入） | 非死代码，但应通过 `netutil.RedirectPolicy` 统一策略来源 |

---

## 10. Model / DTO 耦合审查

`model` 495 行、~31 导出类型、零依赖、被 13 包依赖。主体是纯 DTO（健康），但混入三类非 DTO 内容：

1. `KnownDataSchemas`（`model.go:28`）可变全局 map——schema 注册表
2. `CurrentJunction` / `CurrentSymlink` 领域常量——导致 `platform → model` 反向依赖
3. `ProviderSummary.MarshalJSON`（`provider.go:98,103`）的 Normalize 逻辑——表现层关注点

**建议**：
- 不引入完整 DDD domain 包（无明确理由）
- 把 `Current*` 常量下沉到 `platform`，消除反向依赖
- 把 `KnownDataSchemas` 改为函数返回副本（或 `sync.Once` 只读结构）
- 不为采用 DDD 而强行分层

相关可变全局状态：

| 位置 | 形态 | 风险 |
|---|---|---|
| `model.KnownDataSchemas` | 导出 `map[string]struct{}` | 高（外部可写） |
| `contracts.CommandDataSchemas` | 导出 `map[string][]string` | 高（外部可写） |
| `contracts.knownCommands` | 包级 map | 中 |
| `detect.sourcePriority` | 包级 map | 中 |
| `provider.licenseHints` | 包级 map | 中 |
| `download.chunkSize` | 包级可变 int | 中 |

**无**全局 `sync` 锁、**无** `func init()`——锁通过 flock 实例在函数内创建，logger 通过 context 传递，设计良好。

---

## 11. 测试结构和覆盖情况

- **单元测试**：119 个 `*_test.go`，覆盖主要包；`testscript`×3、`_internal_test`×6、`_fuzz_test`×3
- **集成/E2E**：pytest×4 + Robot + Bats + 真实 env 修改 + 真实网络（FFmpeg-Builds / yt-dlp）+ 三款官方服务器实测
- **故障注入能力**：

| 维度 | 状态 | 证据 |
|---|---|---|
| HTTP Client | 良好 ✓ | `download.Options.HTTPClient`、`github.Client.HTTPClient`、customurl 的 `Read/Fetch(ctx,*http.Client,...)`、`urlpolicy.NewClient(base)` |
| Clock/Now | 部分 ✓ | `download.Options.Now`、`github.Client.Sleep`；其余直接 `time.Now()` |
| Filesystem | 部分 ✓ | 仅 `selfupdate.FileOps` 接口；其余直接 `os.*` |
| Runner（进程） | **弱 ✗** | `ffexec.Runner` 是配置 struct 非接口，无法注入 fake runner |
| EnvStore | 良好 ✓ | `platform.EnvStore` 接口 + `env.RollbackWithStore` |
| Progress | 良好 ✓ | `output.Progress` 接口 |

- **覆盖率**：本次未运行 `go test -cover`，无基线数字。建议 Phase 7 建立基线，不追求单一百分比。

---

## 12. 外部依赖审计

- 17 个直接依赖**全部真实使用**（含测试用 `jsonschema`/`testify`）
- 9 个间接依赖符合定位
- **无 CGO**、无 `import "C"`——`CGO_ENABLED=0` 静态编译成立
- 许可证：go.mod 26 模块 ↔ `THIRD_PARTY_NOTICES.md` 26 行 ↔ `third_party/licenses/` 26 份**一一对应**
- 低优先：`go.sum` 有 `go-cmp v0.7.0` 残留（tidy 可清）；`output` 两种字节格式化（humanize vs go-pretty）；`tidwall/pretty` 可由标准库替代（非阻塞）
- `encoding/json/v2` 是 Go 1.27 标准库，不是第三方依赖

文档错误：
- `docs/development/ARCHITECTURE.md:122-123` 写 `sethvargo/go-retry` 保留在 go.mod、`golang.org/x/time/rate` 使用——**两者均不在 go.mod**（当前重试在 `internal/selfupdate/retry.go`）
- `adr/0002-dependencies.md` 把标准库 `encoding/json/v2` 算进"17 个直接依赖"，且历史段落称"16 直接 + 2 间接"与 go.mod 实际 17+9 不符

---

## 13. 文档一致性问题（高严重度）

| 文件:行 | 当前内容 | 事实 |
|---|---|---|
| `README.md:38` | "无内置 provider，只支持自定义 URL" | 与同文 :14 矛盾；official 是默认 provider |
| `SECURITY.md:21` | "自定义 URL 必须提供 pin" | directory 模式不要求 `--sha256`，改读 checksum.sha256 |
| `CLI_SCHEMA.md:302,313` | "HTTPS 直链""仅 https" | 当前允许 http/https |
| `CLI_SCHEMA.md:338` | "`--release-line` / `--release-tag` / `--channel stable` 不再支持" | `--release-line` 已恢复并用于 official/directory |
| `CLI_SCHEMA.md:282-288`、`provider-list.example.json:9-22` | provider 示例仍是 `btbn`/`bnqdzj-nonfree`、channels `stable/daily` | 当前为 official + 自定义，变体 `gpl/lgpl/nonfree` |
| `PROVIDER_RESOLVER.md:14,19,24-27,42` | "provider 必填（无默认值）""release-line 不接受""查 providers.toml""asset_url 必须 https" | 与默认 official、`--release-line`、http/https 全部相反 |
| `PROVIDER_RESOLVER.md:34` | "checksums 拉取失败时降级为仅 digest" | 代码直接返回错误，无降级 |
| `MIGRATION.md:39` | `omf install --channel stable --apply --force` | flag 已不存在，**可直接复制失败** |
| `man/one-more-ffmpeg.1:44-54` | "There are no built-in providers""release-line rejected" | 与默认 official、directory、`--release-line` 冲突 |
| `man/one-more-ffmpeg.1:26-86` | COMMANDS 无 `update` | 18 命令包含 `update --check/--apply/--recover` |
| `ARCHITECTURE.md:10` | "platform 三层没有落地" | platform 已有 22 个文件 |
| `ARCHITECTURE.md:37-58` | 包职责表缺 output/lock/platform/selfupdate/deps | 这些包均存在 |
| `ARCHITECTURE.md:63-70` | 依赖方向表不完整 | cli 还直接依赖 checksum/ffexec/install/lock/paths 等 |
| `THIRD_PARTY_NOTICES.md:7` | "v1.1.2 is the current prerelease" | 与正式版矛盾 |
| `CHANGELOG.md`、`VERSIONING.md` | pre3 日期 11-21 > pre4 日期 11-20 | 日期倒置（疑似笔误） |
| `HOWTO.md:29,105` | "install --apply 下载+解压" | install 消费已下载的 release/归档，不负责下载 |

未发现明确断链（现行文档的 Markdown 相对链接目标均存在）。

---

## 14. 目标架构目录树（轻量化 Modular Monolith）

```
cmd/one-more-ffmpeg/              # 组合根：main、信号、buildinfo 注入
internal/
  cli/                            # Presentation：命令注册、flags、JSON 输出、退出码
  app/                            # 【新】Application：跨模块用例编排
      install.go / download.go / env.go / update.go / doctor.go
  detect/ capabilities/ provider/ download/ archive/
  install/ env/ journal/ doctor/ selfupdate/                  # Feature Modules
  checksum/ netutil/ fsatomic/ clock/ retry/                  # 【新/扩展】Shared Infrastructure
  ffexec/ platform/ paths/                                    # Platform Adapters
  model/ errors/ contracts/ config/ logging/ output/
  buildinfo/ testutil/                                        # Cross-cutting
```

**依赖方向**：`cmd → cli → app → feature → shared → platform`

---

## 15. 新旧模块对应关系

| 现状 | 目标 | 动作 |
|---|---|---|
| `cli/install.go:61 installRelease` | `app/install.go` | 上移编排，cli 只解析+输出 |
| `cli/env.go`/`uninstall.go`/`update.go` 中的编排 | `app/*.go` | 同上（按实际调用关系确定，不强行包装） |
| `provider/custom.go` TOML+lock | `provider/store/` | 拆出配置持久化 |
| `provider/*.go` URL 解析 | `customurl/*`（已存在） | 归位 |
| `cache/fileio.go` | `fsatomic/` | 升格为共享原语 |
| 6 处 URL/redirect | `netutil/` | 统一策略入口（保留两族） |
| 5×sha256Hex、3×isLowerHex | `checksum` | 提取 |
| `install.linkKind`/`containedIn` | `platform.LinkKindAt`/`LexicallyInside` | 去重 |
| `model.Current*` 常量 | `platform` | 消除反向依赖 |
| `deps/` | 删除 | 死代码 |
| `cache.Store`（未接入） | 删除或接入 | 明确去留 |

---

## 16. 迁移风险分析

| 风险 | 等级 | 说明 |
|---|---|---|
| CLI 契约破坏 | **高** | 18 命令/17 schema/error code 被 freeze_test + golden 守护，任何移动须保持 JSON 字段与命令名不变 |
| 原子写语义改变 | **高** | journal 用 lockedfile（跨进程锁），download 用唯一 .bak+并发守卫，selfupdate 用注入 FileOps——统一接口**不能统一实现** |
| URL 安全策略合并 | **高** | 严格族 vs 宽松族合并会引入 https→http 降级或误伤自托管，必须保持两套 |
| 安装/回滚流程 | **高** | 未经测试不得调整（明令禁止） |
| 文档与代码不同步 | 中 | 每 Phase 后校验 |
| 导出面扩大 | 中 | 提取共享函数时克制 Export |
| Go 方法泛型限制 | 低 | `MarshalJSON` 无法泛型抽取，保留 |

---

## 17. 分阶段重构计划（获批后执行）

- **Phase 1 基础架构**：建 `netutil/fsatomic/clock/retry`；删 `deps`；消 `platform→model`；`checksum` 扩展 3 个纯函数。**不动业务流程**。
- **Phase 2 统一重复**：5×sha256Hex、3×isLowerHex、3×loopback、2×findBinary、2×linkKind、containedIn、4×contextError、7×RFC3339、toReleaseInfo 替换为共享调用；detect.hashFile 委托 checksum。
- **Phase 3 拆分大型模块**：`provider` 拆 store；`contracts.go` 按对象拆文件并解除 `→capabilities`；`selfupdate.Apply`/`env.buildEnvPlan` 拆小函数（保持签名）。
- **Phase 4 Application Layer**：新建 `internal/app`，上移 cli 中 4–5 处跨模块编排。
- **Phase 5 Platform/Feature 边界**：统一 redirect 策略入口（保留两族）；FS 注入仅 selfupdate 保留；Runner 接口评估（可选）。
- **Phase 6 Model/DTO/Contracts**：`KnownDataSchemas` 改函数；`Current*` 常量下沉；仅在有明确理由时分 domain。
- **Phase 7 测试与文档**：覆盖率基线、故障注入测试、`ARCHITECTURE`/`PACKAGE_GUIDELINES`/`DEPENDENCY_GRAPH`/ADR-0004。
- **Phase 8 完整回归**：gofmt/vet/test/fuzz/race/交叉编译/E2E/契约 freeze 全绿，输出前后依赖图对比。

**每 Phase 完成**：输出改动清单 + 迁移前后依赖图 + 相关测试 + CLI 契约检查 + 重复实现检查。

---

## 18. 重构完成验收标准

1. `cmd → cli → app → feature → shared → platform` 单向，无环。
2. 无 `cli`/`app` 被 feature/shared 反向 import；`platform` 不 import 业务包。
3. 第 6.1 节真重复全部收敛到共享包，第 6.2 节伪重复保持独立。
4. `deps` 删除；`cache.Store` 有明确去留；`selfupdate.Now` 不再是死注入。
5. CLI 18 命令 / 17 schema / error code / 目录布局 / 文件格式**零变化**。
6. 安全检查强度不降低（两族 URL 策略并存、journal 跨进程锁不变）。
7. gofmt / vet / test / fuzz / race / build / 双平台交叉 / E2E 全绿。
8. 文档（`ARCHITECTURE`/`PACKAGE_GUIDELINES`/`DEPENDENCY_GRAPH`/ADR-0004）与最终代码一致，含禁止依赖清单。

---

## 19. 安全问题清单（本次不修，指定未来负责模块）

| 问题 | 严重度 | 未来负责模块 |
|---|---|---|
| `urlpolicy` 之外的 4 处 redirect 只看原始 scheme，存在 http→https→http 降级面 | 中 | `netutil.RedirectPolicy`（Phase 5）；注意严格/宽松两族需分别处理 |
| `selfupdate.Apply` 用裸 `fmt.Errorf`，脱离 typed error 体系 | 低 | `errors` + `selfupdate`（Phase 2） |
| `download` 任意 host http（自托管有意设计）缺少显式风险接受记录 | 低 | `netutil` + `SECURITY.md`（Phase 5） |
| `selfupdate/ops.go:53` 固定 `.tmp` 名，并发自更新可互踩 | 中 | `fsatomic.ReservePath`（Phase 1/2） |
| `install/record.go:52 writeAtomic` 无显式 Chmod，文件可能 0600 而非预期 | 低 | `fsatomic.WriteFile`（Phase 1） |
| 7 处 RFC3339 直接 `time.Now()` 使时间戳不可注入测试 | 低 | `clock`（Phase 1） |
| 导出可变全局 map（`KnownDataSchemas`/`CommandDataSchemas`）可被外部包写入 | 低 | `model`/`contracts`（Phase 6） |

---

## 20. 审计方法与限制说明

- 通过 `go list ./...` 获取真实包清单（34 个），通过 `go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./...` 获取真实生产依赖。
- 通过全仓文件遍历与行数统计得到目录树与规模数据（119 个 `*_test.go`）。
- 重复函数审计逐文件阅读 101 个非测试 Go 源文件，并比较输入/输出/错误处理/副作用/并发语义/平台差异/安全语义，而非仅凭函数名。
- 依赖审计读取 go.mod/go.sum 与全仓 import，核对许可证清单；未执行 `go mod tidy`（只读约束）。
- 覆盖率未运行（审计阶段不改代码），表中标注 N/A，未编造数字。
- 个别"无生产调用点"判断基于内容检索，建议实施阶段用 IDE "Find References" 复核。

**本次审计未修改任何源码、文档或测试。**
