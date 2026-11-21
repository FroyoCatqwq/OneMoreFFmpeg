# OneMoreFFmpeg 架构设计

> **文档性质**：当前实现说明 + 重构边界约束（As-Is first）  
> **项目形态**：Go CLI / Modular Monolith  
> **审计日期**：2026-11-21  
> **审计依据**：仓库 `main` 分支可访问源码、`go.mod`、当前架构文档，以及本轮提供的全仓架构审计材料。  
> **重构设计**：见 [`ADR-0004：轻量化 Modular Monolith 架构重构方案`](adr/0004-modular-architecture.md)。  
> **重要说明**：架构图描述真实包依赖，不代表所有包必须迁入某个机械分层。没有落地的 `internal/app`、`netpolicy`、`httpfetch`、`fsatomic` 不得画成当前实现。

---

## 1. 架构定位与边界

OneMoreFFmpeg 是一个以命令行为唯一稳定集成边界的 FFmpeg 管理工具，负责：

- 发现、识别和验证本机 FFmpeg / ffprobe / ffplay；
- 查询官方 FFmpeg-Builds 发布信息，或解析用户注册的自定义来源；
- 下载、校验、解压和版本化安装；
- 管理 `current` 指针、环境变量和 PATH；
- 读取安装状态、运行健康检查、查询 FFmpeg 能力；
- 通过 journal 支持写操作回滚；
- 检查并应用 CLI 自更新。

**当前唯一对外集成形态是 CLI 可执行文件。** 其他项目通过命令行参数、JSON 输出、退出码和文件产物集成；`internal/*` 是内部实现，不是公共 Go API。当前不提供承诺稳定的 Go SDK、插件协议、JSON-RPC/gRPC 或 C ABI。

核心约束：

1. CLI 外部契约优先于内部包结构。重构不得无意改变命令、参数、JSON schema、错误码、退出码和文件布局。
2. 默认只读；会修改安装、环境或配置状态的命令使用 dry-run / `--apply` 明确区分。
3. stdout 是机器接口；日志、进度和诊断信息写 stderr。
4. 下载内容在校验前不执行、不解压；归档处理与安装目录写入分离。
5. 跨平台行为必须基于 Windows / Linux 实际实现与测试，不以单个平台行为推定另一个平台。
6. 先保留已正确的实现，再收敛重复与依赖方向；不为“分层”或“接口化”而改造。

## 2. 当前总体结构（As-Is）

当前代码采用**以 `internal/cli` 为入口的扁平模块化结构**，没有独立 Application Layer。`internal/platform` 已经是实际存在的跨平台适配包，不能再描述为“尚未落地”。

```text
cmd/one-more-ffmpeg
  └─ internal/cli
       ├─ 命令解析 / flags / 配置加载
       ├─ stdout JSON 与人类可读输出
       ├─ stderr 日志与进度接线
       ├─ 单模块命令调用 feature 包
       └─ 多模块操作编排、DataRoot 锁

internal/ feature packages
  ├─ detect          发现与验证
  ├─ capabilities    FFmpeg 能力采集与缓存协调
  ├─ provider        自定义 provider 数据与解析
  │   ├─ official    官方 FFmpeg-Builds resolver
  │   └─ github      GitHub Releases API 客户端
  ├─ download        归档下载、校验与提交
  ├─ archive         ZIP / TAR 安全解压
  ├─ install         版本化安装、current、卸载与恢复基础操作
  ├─ env             PATH / 环境变量 / shell 配置写入与回滚
  ├─ journal         写操作记录
  ├─ doctor          安装健康检查
  └─ selfupdate      CLI 自身更新

internal/ supporting packages
  ├─ checksum        SHA-256 与 checksums 文件解析
  ├─ ffexec          FFmpeg 进程执行
  ├─ platform        文件系统、链接、权限和环境适配
  ├─ paths           data-root 与数据目录定位
  ├─ lock            跨进程文件锁
  └─ cache           JSON 缓存持久化及缓存容器

internal/ cross-cutting packages
  ├─ model           JSON DTO / 数据结构
  ├─ errors          稳定错误码与错误对象
  ├─ contracts       JSON 示例、命令数据和错误码契约校验
  ├─ config          TOML 配置
  ├─ logging         slog handler 与日志上下文
  ├─ output          JSON / human / progress 输出实现
  ├─ buildinfo       版本、commit、build time
  └─ testutil        测试辅助程序
```

上图是职责分组，不是新的 Go 依赖白名单。当前代码没有 `internal/app`、`internal/netpolicy`、`internal/httpfetch`、`internal/fsatomic`；这些只能在 ADR-0004 通过评审并经迁移验证后创建。

### 2.1 入口与 Composition Root

`cmd/one-more-ffmpeg/main.go` 是薄进程入口：

1. 使用 `signal.NotifyContext` 处理 interrupt / SIGTERM；
2. 从构建变量创建 `buildinfo.Info`；
3. 调用 `cli.Run` 并传入参数、stdout、stderr 和版本信息；
4. 根据 `cli.Run` 返回值退出。

它不负责构造所有业务客户端或事务依赖。

当前真正的组合与编排分散在 `internal/cli`：

- `cli.Run` / `cli.New` 处理全局参数、配置、logger 和命令树；
- 各命令 Action 构造 feature options、HTTP client / Runner 等依赖；
- install、env、uninstall、journal rollback 等多模块写操作在 CLI 侧持有 DataRoot 锁并协调 feature 包。

因此，准确表述是：**当前 Composition Root 由 CLI 入口及各命令 Action 共同承担**。不要把 `main.go` 描述成完整组合根，也不要把尚不存在的 `app` 写成当前层。

### 2.2 当前包数量与审计快照

提供的全仓审计记录在基准 `24a3f9f` 上通过 `go list ./...` 统计到 34 个 Go package。该清单如下：

```text
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

`internal/deps` 是无生产用途的占位包，列入待清理项；不能把它误写成运行时依赖。实施前必须在实际 checkout 的固定 commit 上重新执行 `go list ./...`，以该结果更新本节和 ADR，而不是把 34 作为永久常数。

## 3. 当前生产代码依赖关系（As-Is）

下图以 Go package 为单位，列出本次审计快照中的直接内部依赖。它不是“理想架构图”，而是重构前用于比较的真实基线。

```text
cmd/one-more-ffmpeg
  └─ cli, buildinfo

cli
  ├─ buildinfo, config, errors, logging, paths, contracts, output, model
  ├─ detect, capabilities, provider, provider/official
  ├─ download, checksum, ffexec, install, journal, lock
  └─ env, platform, doctor, selfupdate

provider
  ├─ errors, model, checksum, archive, lock
  └─ customurl/checksum, customurl/latest, customurl/naming, customurl/urlpolicy
provider/github   └─ errors
provider/official └─ errors, model

contracts    └─ capabilities, errors, model
capabilities └─ errors, ffexec, logging, model, cache, paths
detect       └─ errors, ffexec, model, platform
download     └─ checksum, errors, logging, model, output, platform
install      └─ archive, checksum, errors, ffexec, journal, logging, model, platform
env          └─ errors, install, journal, model, platform
doctor       └─ env, errors, ffexec, journal, model
selfupdate   └─ errors, logging, output, platform, provider/github
archive      └─ errors, platform
logging      └─ config, errors
platform     └─ errors, model

journal / lock / config / paths / checksum / ffexec / customurl/*
  └─ errors

errors / model / output / cache / buildinfo / deps / testutil
  └─ 无内部 package 依赖
```

> 注意：`cli` 的 22 个依赖是**整个 `internal/cli` package 的聚合依赖**，不是 `root.go` 单文件的 import 数。每个命令拆分在独立 Go 文件中，Go 编译时按 package 汇总。

### 3.1 已有的正确依赖方向

- `errors` 不依赖 CLI 或 feature；它是稳定错误类型底座。
- `checksum` 不依赖 download；下载使用校验能力，方向正确。
- `archive` 不依赖 install；安装调用解压能力，方向正确。
- `model` 不依赖其他内部包，仍是低耦合 DTO 包。
- `output` 不依赖 feature；但当前 `download` / `selfupdate` 依赖 `output`，需要通过依赖倒置消除。
- 没有 `common` / `utils` 之类超级共享包，也没有观察到 package import cycle。

### 3.2 当前主要结构问题

| 优先级 | 位置 | 当前事实 | 影响 / 方向 |
|---|---|---|---|
| P1 | `cli` | 直接依赖约 22 个内部包，同时承担命令入口、输出和跨 feature 编排 | 扇出过高；只把真正跨模块用例移出，不为单模块命令加包装层 |
| P1 | `provider` | 聚合 provider 解析、custom provider 文件持久化、锁、checksum/archive/customurl 等依赖 | 职责偏多；先文件级拆分，只有有独立数据所有权和多个调用方才建子包 |
| P1 | `contracts` | 约 746 行大文件，并依赖 `capabilities` 获取能力类型/命令元数据 | 契约层依赖 feature 层；纯数据定义应下沉到中立 DTO/metadata 位置 |
| P1 | `platform` | 依赖 `model` 中的 current/link 表示常量 | 基础设施反向依赖 DTO；在平台边界定义自己的类型并显式映射 |
| P1 | `download` / `selfupdate` | 直接依赖 `output` 进度实现 | 业务层依赖表现层；接口放在使用方 package，CLI 注入 `output` 实现 |
| P2 | `model` / `contracts` | 导出可变 map：`KnownDataSchemas`、`CommandDataSchemas` | 任意导入者可篡改契约元数据；提供副本 getter 或私有只读数据 |
| P2 | `selfupdate` / `env` / `download` / `archive` | 存在大函数和较大源文件 | 先同 package 拆职责函数，不先拆 package；保持调用顺序和事务边界 |
| P2 | `cache` | 通用 `Store[K,V]` 容器没有生产调用，生产路径使用 JSON cache helpers | 删除或接入须基于真实调用方，不为保留泛型抽象而保留 |
| P2 | `deps` | 仅有占位文档代码，无实际依赖声明 | 复核引用后删除 |
| P2 | `ffexec.Runner` | 配置结构体，不是可替换接口 | 目前有其他测试注入点；仅在测试确实无法覆盖时再加窄接口 |

行数、行号和调用点是审计时快照，不应在后续 PR 中作为永久事实。修改前应重新检索并以函数名、行为和测试作为定位依据。

## 4. Package 职责与边界

| Package | 当前职责 | 边界要求 |
|---|---|---|
| `cli` | 命令树、参数/配置、日志和输出接线、调用 feature、部分多模块事务编排 | 不把业务规则散落进 flag callback；不把单模块调用包成 App 转发层 |
| `detect` | 查找、去重、校验 FFmpeg 三件套、生成 fingerprint / installation id | 只发现与验证，不写安装布局或环境变量 |
| `capabilities` | 运行 FFmpeg 能力命令、解析结果、协调缓存 | 进程调用统一走 `ffexec`；缓存路径走 `paths` / `cache` |
| `provider` | 自定义 provider 数据、解析和 URL 命名协调 | 不直接写 PATH / install current；官方 provider 保持独立 package |
| `provider/official` | 官方 FFmpeg-Builds release / asset 选择规则 | 上游命名规则集中在 `Rules`；API/网络失败映射稳定错误 |
| `provider/github` | GitHub Releases API 客户端 | 不依赖 CLI；token 不写日志；网络策略语义需与其他客户端明确对齐 |
| `customurl/*` | URL 命名、checksums、latest 元数据和 TLS/URL policy | 子包独立治理；不要只因目录相邻就把不同语义合并 |
| `download` | 归档下载、大小限制、校验、进度和目标提交 | 只处理下载，不承担解压与安装事务；`.bak` 提交并发语义不能被普通原子写替代 |
| `archive` | ZIP / TAR 安全解压 | 不依赖 install；路径逃逸、链接和权限边界必须由测试覆盖 |
| `install` | 版本化目录、install metadata、current 指针、卸载/恢复基础动作 | 不自己取得 DataRoot 事务锁；由最外层协调者持锁 |
| `env` | 环境变量、PATH、shell rc / shim 写入与回滚计划 | 不自行取得 DataRoot 事务锁；平台 IO 走 `platform` |
| `journal` | 写操作状态记录与恢复数据 | journal 自己的文件级跨进程保护不能被 DataRoot 锁替代 |
| `selfupdate` | 当前 CLI 检查、下载、暂存、替换、恢复 | 与安装 data-root 事务分开；保留 `FileOps` 测试注入 |
| `doctor` | 安装健康状态检查 | 尽量只读；汇总 feature 状态而不是实施修复 |
| `ffexec` | 执行 FFmpeg 进程、超时/取消、输出限制、平台进程树终止 | 不拼 shell 命令；参数数组；统一进程终止行为 |
| `platform` | Windows/Linux 文件系统、链接、环境和权限差异适配 | 不依赖 feature；目标状态不依赖 `model` |
| `paths` | data-root、cache、downloads、配置等路径计算 | 所有命令使用统一入口，不自行拼系统路径 |
| `lock` | 文件级锁与 DataRoot 事务锁原语 | 调用方负责锁所有权和持锁范围；不在 feature 内悄悄重复加锁 |
| `cache` | JSON cache 读取/写入及缓存容器 | 保持缓存持久化与通用内存 LRU 概念分离 |
| `checksum` | SHA-256、checksum 行解析 | 纯校验能力，不依赖网络或下载流程 |
| `model` | CLI JSON DTO、结果载荷、schema 名称 | 保持无内部依赖；不成为所有业务规则的归宿 |
| `contracts` | 命令和数据 schema、错误码及样例校验 | 不依赖 CLI；目标不依赖 `capabilities` 具体实现 |
| `errors` | 稳定错误码、错误构造和错误映射 | 零内部依赖；已有错误码视为兼容数据，不随意删除 |
| `config` | TOML 配置解析、默认值与约束 | 不执行实际业务；网络上界等跨包常量需避免反向依赖 |
| `logging` | slog / tint 设置、context logger | 日志写 stderr；禁止敏感 token / header 泄露 |
| `output` | JSON、human-readable、进度输出实现 | 不依赖任何 feature；具体进度接口由调用方定义 |
| `buildinfo` | 编译注入的版本、commit、构建时间 | 尽量保持零内部依赖 |
| `testutil` | fake FFmpeg 等测试辅助 | 不被生产路径引用；不因缺少 build tag 就声称代码会打包进主 CLI |
| `deps` | 占位 | 确认无引用后删除 |

## 5. 外部契约

对外兼容边界是 CLI 行为，不是 Go struct 的字段或 package API。

```text
调用方
  └─ one-more-ffmpeg <command> [flags]
       ├─ stdout：JSON（--json）或命令约定的人类输出 / 文件路径
       ├─ stderr：日志、进度、警告和错误诊断
       ├─ exit code：成功 / 运行失败 / 用法或配置失败 / 中断
       └─ 文件产物：release / artifact / install / journal / 可选 ffmpeg-trio.lock
```

- JSON 字段、schema 和命令映射以 [`CLI_SCHEMA.md`](../reference/CLI_SCHEMA.md)、[`schemas/`](../reference/schemas/) 为准。
- 错误码以 [`ERROR_CODES.md`](../reference/ERROR_CODES.md) 为准。
- 参数、默认值和命令行为以 CLI 实现与对应 reference 为准；若文档冲突，以源码和冻结测试复核后修正文档，不能靠口头描述确定契约。
- `--json` 下 stdout 只能输出 JSON envelope；日志和进度仍走 stderr。
- 内部 `model` struct 不是 SDK；修改 JSON tag 或 `omitempty` 必须经过 schema、golden 和契约测试。

### 5.1 Provider 与 URL 模式

当前支持的来源不是“仅自定义 URL”或“只有官方 provider”二选一：

- 默认来源是 `official`，由 `internal/provider/official` 解析 FFmpeg-Builds release；
- 用户还可以注册自定义 provider；自定义来源支持 direct 与 directory 等模式；
- `internal/provider/provider.go` 中关于“没有内置 provider”的旧注释不能作为产品行为依据；CLI 的默认解析路径实际使用 `provider/official`，应在源码维护时同步修正文案；
- 用户自定义来源与官方 GitHub API 的 URL 风险模型不同，不能强制套用同一个 URL 校验策略；
- 自定义源的 HTTP/HTTPS 支持、checksums 获取失败行为和 redirect 规则，必须以 resolver 与调用点实际逻辑为准。

### 5.2 配置与持久化

- TOML 严格解析，拒绝未知键；
- data-root 由 `paths` 统一解析；
- 下载、cache、安装、journal 和 provider 配置位于确定的数据根目录；
- 文件写入有不同语义，详见第 8 节；
- 配置文件、provider 存储和 journal 的文件级锁不能被粗略合并为一个“原子写”函数。

## 6. 进程执行模型

所有 FFmpeg/ffprobe 等外部工具进程通过 `internal/ffexec`：

- 使用 `exec.CommandContext` 和参数数组，不使用 shell 字符串拼接；
- stdout / stderr 分开捕获并设置输出上限；
- 每个操作有独立 timeout / context；
- 超时或取消时按平台策略尽力终止进程树；
- 记录必要的退出码、耗时和输出摘要/hash；
- 不把完整、可能含敏感路径或异常内容的 stderr 无差别写入日志。

Windows 与 Linux 的进程树终止实现不同。当前文档描述的是“尽力终止”，不得承诺操作系统在所有环境下都能强制清理全部子进程。新增进程抽象必须有测试注入的实际需求，不能仅为了接口数量增加 `Runner` 接口。

## 7. 安装与事务边界

当前安装布局的主要元素：

```text
<data-root>/
├─ apps/<provider>/<release-tag>/<sha256-prefix>/
├─ current                     # symlink / junction / fallback pointer
├─ downloads/
├─ cache/capabilities/
├─ journal/
├─ tmp/
├─ config.toml
└─ providers.toml
```

典型业务路径由命令组合多个 feature：

```text
resolve release
    ↓
fetch archive (when needed) → verify SHA-256
    ↓
extract into temporary directory → validate FFmpeg trio
    ↓
write installation metadata → switch current pointer
    ↓
write/update journal → optional env update
```

具体命令参数可能让用户提供已经存在的 release JSON 或 artifact 路径，因此上图表达能力流程，不代表所有命令调用都必须执行全部步骤。`download` 与 `install` 是独立命令，也存在 install 路径中按参数获取归档的组合行为；实现和 HOWTO 必须统一说明，不能在文档中绝对声称 install 永远不触发下载。

### 7.1 DataRoot 事务锁

- 当前 DataRoot 事务锁的最外层所有者在 CLI 命令 Action。
- `install`、`env`、`journal` 等 feature 不应为同一事务再次取得 DataRoot 锁。
- 若未来引入 `app`，锁所有权必须在同一个变更中从 CLI 移到 App，不能两层同时持锁，也不能出现无锁窗口。
- `download` 的下载文件提交、`journal` 文件级锁和 `selfupdate` 的更新互斥是其他锁语义，不应与 DataRoot 锁混为一谈。

## 8. 文件写入与锁语义

文件操作不能因为函数名相似都收敛为一个 `AtomicWrite`。至少需要保留以下不同语义：

| 语义 | 当前典型用途 | 必须保留的属性 |
|---|---|---|
| 临时文件 + rename 替换 | install metadata、shell 等普通文件写入 | 临时文件与目标位置关系、mode、rename 行为和失败清理 |
| 带文件锁写入 | journal、部分 cache / provider 配置 | 跨进程串行化；不等于 temp+rename 原子替换 |
| 下载目标提交 | download 的目标文件与 `.bak` 恢复 | 并发提交检查、唯一备份、失败恢复 |
| Self-update FileOps | CLI 可执行文件替换与恢复 | 注入能力、阶段状态与 rollback 语义 |
| 链接替换 / 预留 | current symlink / junction | 链接类型、目标解析、Windows junction 与 symlink 差异 |

任何共享文件系统抽象必须先写明支持的语义、平台保证、失败行为和调用方，再迁移真实调用点。尤其不能：

- 用 `lockedfile.Write` 替换下载提交逻辑；
- 用简单 rename 替换 journal 的跨进程互斥；
- 把普通文件 rename 和 symlink / junction replace 合为同一函数；
- 删除 `selfupdate.FileOps` 测试注入；
- 声称所有 Windows 文件系统都具有已验证的原子替换或掉电持久性。

## 9. 网络与 URL 安全边界

网络调用至少分成不同风险域：

1. **官方控制平面 API**（例如 GitHub Releases）：按受限 HTTPS 策略处理，谨慎保留 loopback 测试例外、host/userinfo 检查、redirect 约束和 token 脱敏。
2. **用户自定义资源来源**（direct / directory）：当前支持用户显式提供的 HTTP/HTTPS 地址；对这类来源强行套用“仅 HTTPS 或 loopback HTTP”的严格策略会改变现有契约。
3. **归档下载**：流式写文件、校验摘要、控制体积；不能用小文本抓取 helper 替代。
4. **小型文本 / 元数据抓取**：checksums、latest 元数据等可以考虑统一有界读取，但必须保留调用方 HTTP client、TLS、代理、超时、认证、重试和错误码语义。

重构网络策略时，至少区分：

- URL 单点校验（scheme / host / userinfo）；
- redirect 目标校验；
- 整条 redirect chain 的 downgrade 约束；
- 调用方自定义 `CheckRedirect` 回调；
- `http.ErrUseLastResponse` 的标准库语义。

不得把“所有重定向使用同一套规则”作为未经验证的默认。尤其要测试回调改写 `req.URL` 的情况，避免项目校验被调用方回调绕过。

**完整 SSRF 防护目前不属于本架构文档的既有保证**。DNS 解析后的私网 / metadata IP 约束、DNS rebinding、代理环境变量和自托管服务放行策略，需要单独安全设计。

## 10. 依赖策略与当前 go.mod

当前仓库 `go.mod` 声明 `go 1.27`。审计读取到的依赖分组为 17 个直接依赖、9 个间接依赖；依赖版本和 license 以仓库 `go.mod` / `go.sum` / `THIRD_PARTY_NOTICES.md` 为准。

主要落点：

| 用途 | 当前采用 |
|---|---|
| CLI | `github.com/urfave/cli/v3` |
| TOML | `github.com/pelletier/go-toml/v2` |
| GitHub API | `github.com/google/go-github/v93` |
| 日志 | `log/slog` + `github.com/lmittmann/tint` |
| 文件锁 | `github.com/gofrs/flock` |
| 并发 | `golang.org/x/sync` |
| OS / terminal API | `golang.org/x/sys`、`golang.org/x/term` |
| Archive | `github.com/klauspost/compress`、`github.com/ulikunitz/xz` |
| TTY | `github.com/mattn/go-isatty` |
| 人类可读大小 | `github.com/dustin/go-humanize` |
| JSON schema / tests | `github.com/santhosh-tekuri/jsonschema/v6`、`github.com/stretchr/testify` 等 |

重要纠错：当前 `go.mod` 不包含 `sethvargo/go-retry` 或 `golang.org/x/time/rate`。不要在架构图或依赖表中继续把它们写成现有依赖。重试/退避实际由项目当前代码控制；每次调整后必须以 `go list -m all` 和源码 import 重新核验直接 / 间接依赖。

依赖治理原则：

- 不为了抽象引入新第三方依赖；
- 新 package 至少要有清晰职责和两个以上真实使用点，或有明确边界收益；
- 避免一函数一 package；
- 禁止超级 `common` / `util` / `utils`；
- 依赖方向必须可通过 `go list` 自动核验；
- license 清单与实际模块清单保持一致。

## 11. 测试与质量门禁

现有测试体系由 Go package tests、命令行为测试以及独立 E2E 组成。既有审计快照记录了约 119 个 `*_test.go`，并包含 testscript、pytest、Robot Framework、Bats 和真实环境测试。具体数量应在实施前重新统计。

既有测试能力：

| 测试边界 | 当前注入能力 | 观察 / 约束 |
|---|---|---|
| HTTP | 多个 package 支持注入 `*http.Client` 或 fetch 函数 | 较好，但策略与客户端构造重复；不应丢失 Transport/TLS/Proxy 配置 |
| Clock | 个别 options / resolver 可注入 Now 或 Sleep | 不完整；不要为格式化时间建立全局时钟框架 |
| Filesystem | `selfupdate.FileOps` 可注入；其他包多直接使用 `os.*` | 迁移文件 API 前先补行为测试 |
| Environment | `platform.EnvStore` 与 rollback 测试路径 | 保留平台差异和回滚语义 |
| Progress | 输出侧有 progress 实现 | 将接口移至使用方，不让 feature 依赖 output |
| Process runner | `ffexec.Runner` 是配置 struct | 只有出现具体测试缺口时才考虑窄接口 |

**覆盖率基线不能使用失败命令生成的部分数字。** 既有 ADR 记载：`go test -coverprofile=... ./...` 曾因 CLI testscript 的 stderr 断言失败，部分 `67.3%` 并非完整、可信基线。该问题应先在固定 commit 上复现并修复，再采集完整覆盖率。

### 11.1 每次重构至少执行

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...
```

涉及并发、重定向、归档和事务状态时还必须执行相应 race / fuzz / 故障注入 / 跨平台 E2E。不能把“之前某 commit 测试通过”当作本 PR 的测试结果。

## 12. 文档的权威来源

| 内容 | 权威来源 |
|---|---|
| 命令、参数、全局 flags | `internal/cli` 实现 + `docs/reference/CLI_SCHEMA.md` |
| JSON 结构和示例 | `internal/model`、`internal/contracts`、`docs/reference/schemas/` |
| 错误码 | `internal/errors` + `docs/reference/ERROR_CODES.md` |
| Provider / release 行为 | `internal/provider/*`、`internal/cli/resolve.go` + `PROVIDERS.md` / `PROVIDER_RESOLVER.md` |
| data-root / 文件布局 | `internal/paths`、`internal/config` + `ENVIRONMENT.md` |
| 环境变量 / 平台行为 | `internal/env`、`internal/platform` + `PLATFORM_ENVIRONMENT.md` |
| 架构及依赖关系 | 本文件 + ADR-0004 |
| 构建依赖 | `go.mod` / `go.sum` + ADR-0002 |

文档、注释、CLI help 和源码发生冲突时，先确认代码与冻结测试的实际行为，再在同一改动内修正文档；不要在架构文档中复制未经验证的命令默认值。

## 13. 重构规则（To-Be 约束，而非当前状态）

以下规则指导 ADR-0004 的目标状态，不表示当前代码已经满足：

1. `cmd` 保持薄入口；`cli` 保持 Presentation / Composition Root 职责，但不继续承载大量跨模块事务细节。
2. 只有真实协调两个以上 feature、锁或事务边界的用例，才考虑 `internal/app`；单模块命令直接调用 feature。
3. `errors`、`model`、`output`、`buildinfo` 保持零内部依赖。
4. `platform` 不依赖 feature；目标状态不再依赖 `model`。
5. `contracts` 不依赖 CLI；目标状态不依赖 `capabilities` 的实现 package。
6. `download` / `selfupdate` 不依赖 `output`；progress interface 归使用方，output 提供实现。
7. `install` / `env` / `journal` 不自行获取 DataRoot 事务锁；由最外层协调者取得一次。
8. `selfupdate` 更新互斥与 DataRoot 事务锁分开设计，不得混用。
9. 网络策略必须分别表达 URL 接受范围与 redirect-chain 约束；官方控制平面与用户自定义来源的差异必须保留，不能把自定义来源当前 HTTP/HTTPS 行为意外收紧。
10. 文件写入统一的是语义清晰的 API，不是把所有底层实现替换成一种原子写。
11. 先在 package 内拆大文件 / 大函数，再基于独立数据所有权和调用数量决定是否拆 package。
12. 删除 dead code 前通过源码引用、测试、构建 tags 和外部数据兼容性复核；错误码常量可能是持久化/脚本契约的一部分。
13. 不为未来可能性增加公共接口、包或依赖。

## 14. 当前审计限制与重构前复核

本轮使用仓库可访问源码、`go.mod`、当前 `ARCHITECTURE.md` 和提供的审计报告进行文档校准；执行环境无法通过 Git 协议连接 GitHub，因此本轮没有在本地 clone 当前 `main` 并重新运行 Go 命令。包数量、行数、依赖矩阵、测试文件数量与覆盖率诊断数值继承自既有审计快照 `24a3f9f`，不能冒充为本轮在最新 HEAD 上重新测量。

因此第一个重构 PR 必须：

1. checkout 并记录完整 commit SHA；
2. 重新运行 `go list ./...` 与 package import 图提取；
3. 重新统计源文件、测试、复杂函数与 direct / indirect dependencies；
4. 运行 test / vet / build / cross-build / race / fuzz / E2E，记录实际结果；
5. 复核本文件和 ADR-0004 中的每个文件路径、调用点与依赖假设；
6. 确认 CLI contract freeze / golden / JSON schema tests 当前存在且通过。

**这份文档描述当前已核实的架构方向，并把旧快照数据显式标注为基线。它不是源码已完成重构的声明。**
