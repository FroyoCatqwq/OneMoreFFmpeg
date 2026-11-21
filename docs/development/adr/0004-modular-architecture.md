# ADR-0004：轻量化 Modular Monolith 架构重构方案

> **设计状态**：Proposed  
> **实施授权**：Awaiting Implementation Approval（本 ADR 不授权修改源码）  
> **本轮修订**：2026-11-21（基于仓库源码与既有全项目审计材料复核）  
> **当前架构**：[`../ARCHITECTURE.md`](../ARCHITECTURE.md)  
> **既有详细审计记录**：[`../ARCHITECTURE_AUDIT.md`](../ARCHITECTURE_AUDIT.md)（如该文件已纳入仓库）

> 状态说明：`Proposed` 只表示目标设计已形成；不代表已获实施批准。只有用户明确批准后，才可把实施授权改为 `Approved for Implementation` 并开始迁移。**本 ADR 当前不允许以“重构准备”为由顺手修改 Go 源码、测试、依赖或 CLI 契约。**

---

## 0. 本轮修订说明

本轮的目的不是延续旧版 ADR 的所有 API 细节，而是把方案重新对齐当前仓库代码，并删除未经证实的实现承诺。

本轮明确修正以下问题：

1. **架构图按真实包依赖描述**：`internal/platform` 已经存在；当前没有 `internal/app`、`netpolicy`、`httpfetch`、`fsatomic`。这些只能出现在 To-Be 方案中。
2. **`provider/fetch.go` 的策略归属修正**：该实现允许用户自定义来源使用任意 host 的 HTTP/HTTPS。将它迁入“仅 HTTPS 或 loopback HTTP”的严格 URL 策略会破坏既有行为。它不能照旧 ADR 的示例映射到 `StrictPolicy`。
3. **URL 接受规则与 redirect-chain 规则分开**：是否允许普通 HTTP 与是否要求整条 redirect chain 不降级，是两个不同维度。旧 ADR 将它们压成 Strict / Custom 两档，表达力不足。
4. **`app` 继续保持条件创建**：当前 CLI 同时承担组合与跨模块编排，但不代表必须立即创建 App；单模块命令不得经过空转发层。
5. **文件操作不预先统一实现**：普通原子替换、lockedfile 写入、下载 `.bak` 提交、自更新 FileOps、symlink/junction 操作必须保留各自语义。
6. **缩减未经验证的跨平台保证**：不在设计阶段预设 Win32 API、所有文件系统原子性、目录 fsync 或 reservation 的绝对防竞态能力；这些需要先由现有调用点和平台测试证明必要性。
7. **取消固定 17/18 个 PR 的承诺**：先锁定源码 commit 和质量基线，再按依赖关系拆分小 PR。PR 数不是架构目标。
8. **质量记录只作为历史基线**：既有 ADR 对 `24a3f9f` 的 test/vet/build 结果不是本轮最新 HEAD 的测试结果。本轮没有在本地重新运行 Go 全套测试。

## 1. Context（现状与问题）

### 1.1 当前架构事实

基于本轮核对的仓库源码及既有审计快照：

- 项目是 Go 1.27 CLI-first 工具，CLI 可执行文件是唯一稳定集成边界。
- `cmd/one-more-ffmpeg/main.go` 是薄入口，负责 signal context、buildinfo 注入、调用 `cli.Run` 和退出。
- 当前 Composition Root 分散在 `internal/cli.Run` / `New` 与命令 Action；CLI package 聚合依赖约 22 个内部包。
- 当前没有 `internal/app`；install、env、uninstall、journal rollback 等跨 feature 编排由 CLI 命令 Action 承担。
- `internal/platform` 是现有包，当前仍依赖 `model` 中的链接/current 表示常量。
- `contracts` 依赖 `capabilities`、`errors`、`model`；其中纯能力元数据不应由契约层反向依赖 feature 实现。
- `download` 与 `selfupdate` 当前依赖 `output` 的进度实现。
- `provider` 汇集自定义 provider 配置存储、resolver 和多项基础能力；官方 FFmpeg-Builds resolver 独立位于 `provider/official`，由 CLI 显式选择且是默认来源。`provider.go` 中“没有内置 provider”的旧注释与实际 CLI 默认路径冲突，应作为后续文档/注释修正项。
- `provider/fetch.go` 的网络语义与 GitHub API 控制平面不同；自定义来源当前允许 HTTP/HTTPS。
- `model.KnownDataSchemas`、`contracts.CommandDataSchemas` 是导出可变 map。
- 旧审计快照记录 34 个 package、约 119 个 Go 测试文件；这些统计必须在实际重构 checkout 上重跑。

### 1.2 核心结构问题

| 优先级 | 问题 | 证据 / 现状 | 目标 |
|---|---|---|---|
| P1 | CLI 扇出过高 | CLI package 直接依赖约 22 个内部包，并负责事务编排 | 降低跨模块编排职责；不要求 CLI 只 import 一个 App |
| P1 | Provider 职责聚集 | provider 同时处理配置存储、文件锁、解析、checksum/archive/customurl 协调 | 先同 package 文件级拆分；子包以数据所有权和多个调用方为创建条件 |
| P1 | Contracts 反向依赖 | `contracts → capabilities` | 将纯数据定义移出 feature 实现依赖 |
| P1 | Platform 反向依赖 DTO | `platform → model` | platform 自有平台类型，DTO 边界做显式转换 |
| P1 | Feature 依赖输出实现 | `download/selfupdate → output` | 接口由使用方定义，CLI 注入 output 实现 |
| P2 | 可变全局契约 map | `KnownDataSchemas`、`CommandDataSchemas` 导出可变 | 私有数据 + 防御性副本 |
| P2 | 大文件 / 长函数 | contracts、download、env、selfupdate、archive 等 | 先在 package 内按职责拆文件和函数 |
| P2 | 未接入抽象 | `deps`、`cache.Store`、`selfupdate.Options.Now` 等 | 逐项验证生产引用、测试用途和兼容性后决定删除或接入 |
| P2 | 重复实现 | SHA-256、hex 校验、URL 校验、文件查找、原子写等 | 只抽行为与错误语义真正相同的部分 |

### 1.3 明确不作为本 ADR 目标的事项

- 不引入 DDD / Clean Architecture / Hexagonal 全套结构。
- 不创建 `common`、`shared`、`utils` 超级包。
- 不强制一函数一 package、不为抽象而引入接口。
- 不改变 CLI JSON v1、错误码、退出码、文件格式、安装目录布局。
- 不在本 ADR 中实现 Ed25519 签名、完整 SSRF、Journal WAL、安装事务重写、自更新健康验证等独立安全项目。
- 不因架构调整改变用户自定义 HTTP 源的现有接受范围；若产品要收紧，必须另立安全/兼容提案。

## 2. Decision（架构决策）

采用**轻量化 Modular Monolith**，以 package 依赖规则约束模块边界，不采用机械化的固定层级。

### 2.1 决策摘要

1. 保持 `cmd → cli` 入口结构；`main` 继续薄。
2. 暂不创建 `app`；先识别真实跨模块用例，达到第 4 节条件后再创建。
3. 不强制将所有 package 塞进统一 Feature / Shared / Platform 层；按职责与实际依赖治理。
4. 扩展 `checksum`、`platform`、`ffexec` 等已有包时，只加入职责内的真实共用能力。
5. `netpolicy`、`httpfetch`、`fsatomic` 是候选共享基础设施，必须先证明至少两个真实调用点、调用语义可表达且迁移不会扩大行为差异，再实施。
6. `clock`、`retry` 不创建；日期格式化与不同领域的退避策略不值得先抽公共包。
7. `platform → model` 与 `contracts → capabilities` 作为优先依赖方向修正项。
8. `download/selfupdate → output` 通过依赖倒置解除。
9. DataRoot 事务锁由最外层协调者持有一次；迁移 App 时同时迁移锁所有权。
10. Self-update 更新锁与 DataRoot 事务锁分离；锁路径、硬链接处理和目录不可写行为需要单独验证，不在此预设未实现的强保证。
11. 不把不同文件持久化语义收敛为同一个底层实现。
12. 所有行为变更与结构迁移分开；安全强化必须有独立测试和明确批准。

### 2.2 当前状态 / 迁移态 / 目标态

所有 package 依赖表必须标注三种状态：

- **As-Is**：固定 commit 上 `go list` / 源码实际测得；
- **Transitional**：迁移期间允许、且已在对应任务中说明的临时依赖；
- **To-Be**：完成该项重构后希望维持的依赖集合。

不能把 To-Be 白名单画成当前架构，也不能用分组名称替代实际 import 图。

## 3. 依赖规则

### 3.1 永久黑名单

1. feature、shared、platform 不得依赖 `internal/cli`。
2. `errors`、`model`、`output`、`buildinfo` 保持零内部依赖。
3. `archive` 不得依赖 `install`。
4. `checksum` 不得依赖 `download`、`output` 或 `cli`。
5. `platform` 不得依赖 feature；目标状态也不依赖 `model`。
6. `contracts` 不得依赖 `cli` / `app`；目标状态不依赖 `capabilities` 的实现 package。
7. `output` 不得依赖 feature。
8. `download` / `selfupdate` 不得依赖 `output`。
9. `selfupdate` 不得使用 DataRoot 事务锁。
10. 不创建 `common` / `util` / `utils` 超级包。
11. 禁止循环依赖；每个迁移 PR 使用 `go list` / 编译检查。

### 3.2 当前已知依赖异常

| 依赖 | 当前 | 目标 | 修正方式 |
|---|---|---|---|
| `platform → model` | 存在 | 移除 | platform 定义自己的 LinkKind / current 指针枚举；边界显式映射到 DTO 字符串 |
| `contracts → capabilities` | 存在 | 移除 | 将纯能力 kind、命令键等 metadata 放到中立定义；DTO 层不承载采集实现 |
| `download → output` | 存在 | 移除 | `download.Progress` 在 download 包定义；CLI 注入 output 具体实现 |
| `selfupdate → output` | 存在 | 移除 | `selfupdate.Progress` 在 selfupdate 包定义；CLI 注入 output 具体实现 |
| `cli` 扇出 22 包 | 存在 | 只保留参数入口、单模块调用和真正需要的组合 | 如 app 条件成立，把跨模块用例移入 app |

### 3.3 CLI 单模块与跨模块命令

**单模块命令**只调用一个 feature，应直接在命令 Action 中使用该 feature，不经过 App：

- `version`、`detect`、`validate`、`capabilities`；
- `provider list/add`、`resolve`、单独的 `download`；
- `update --check` 等单一 selfupdate 查询操作。

**跨模块命令**在同一用例中协调两个或更多 feature、事务或锁，才考虑移入 App：

- `install` 路径中的 resolve/download/install 组合；
- `env` 写入与 journal；
- `uninstall`、journal rollback 等需要多个 feature 协作的写操作。

具体命令是否跨模块，以实现中的真实调用图为准。`install` 可接受 release/artifact 输入，也可能在 `--apply` 路径中获取归档；不能在设计中假设所有 install 命令都只消费已经存在的下载文件。

## 4. Application Layer 条件创建规则

**当前不创建 `internal/app`。** 只有同时满足以下条件，才在独立 PR 中引入：

1. 至少两个稳定用例有真实跨 feature 编排，而不是仅包装一个 feature；
2. 这些用例共享事务所有权、锁定边界或具有明确的业务流程；
3. App 能消除 CLI 中的真实复杂度，而不是简单转发 Options；
4. 输入 / 输出不依赖 urfave CLI 类型、stdout writer 或 `output` 实现；
5. App 的测试可以直接验证用例事务行为，并且不需要启动整个 CLI。

禁止：

```go
func (a *App) Detect(ctx context.Context, opts detect.Options) error {
    // 只有一次转发，不构成 Application Layer
    return detect.New().Detect(ctx, opts)
}
```

允许的职责包括：协调 resolve/download/install、持有一次 DataRoot 锁、协调 journal 与 environment 回滚等。

若创建 App：

- `cli` 负责参数验证、调用 App、把结果写到 JSON/human 输出；
- `app` 不 import `cli` / `output`；
- Progress / logger / 文件 / HTTP 等依赖按使用方定义接口或传入具体配置，不能把 CLI 类型带进 App；
- 迁移某个用例时，DataRoot 锁必须同时从 CLI 移到 App，禁止双重锁或出现未持锁窗口。

## 5. 锁所有权与事务边界

### 5.1 DataRoot 事务锁

`lock.DataRoot(dataRoot)` 应由某一业务事务的最外层协调者持有一次。

| 操作 | 当前锁所有者 | 若创建 app 的目标所有者 | 约束 |
|---|---|---|---|
| `install --apply` | CLI Action | App install use case | `install.Run` 本身不加 DataRoot 锁 |
| `uninstall --apply` | CLI Action | App uninstall use case | 多阶段 install/journal 协作在同一事务范围 |
| `env set/unset --apply` | CLI Action | App environment use case | feature `env` 不自己再加 DataRoot 锁 |
| `env rollback --apply` | CLI Action | App environment use case | 与 journal 状态一致 |
| `journal rollback --apply` | CLI Action | App journal use case | rollback 调用链不重复加锁 |
| `download` | 不持 DataRoot 事务锁 | 不变 | 下载目标提交使用自身并发保护 |
| `resolve/detect/capabilities/doctor` | 不持 DataRoot 事务锁 | 不变 | 纯读路径不无故锁住 data-root |

PR 必须检索全部 `lock.DataRoot` 调用点，确认锁范围与唯一所有者；App 与 CLI 不得同时锁同一 DataRoot。

### 5.2 其他锁不得混用

| 锁语义 | 目的 | 不能替代 |
|---|---|---|
| DataRoot 事务锁 | 同一 data-root 的跨命令写事务互斥 | 不替代 journal 文件锁或下载提交保护 |
| 配置文件锁 | provider/cache 等单文件 read-modify-write 串行化 | 不代表整个 install/env 事务锁 |
| Journal 文件锁 | journal 记录文件并发写保护 | 不应因 App 持 DataRoot 锁而删除 |
| Download 提交并发保护 | 目标文件、唯一备份和恢复逻辑 | 不等价于普通 atomic rename |
| Self-update 更新互斥 | 同一 CLI 可执行文件更新 / recover | 不得使用 `lock.DataRoot` 替代 |

### 5.3 Self-update 更新锁

当前 `selfupdate` 有独立文件操作注入，但更新互斥路径需要以当前实现重新核实。目标是：

- `update --check` 不应持有写锁；
- `update --apply` 与 `update --recover` 必须互斥；
- 更新锁身份必须对应正在更新的可执行文件，而不是仅以可被用户切换的 DataRoot 区分；
- 锁竞争、锁路径不可写、符号链接和文件系统别名必须有明确错误语义；
- 锁失败不能静默回退到另一个互不相同的锁路径。

**本 ADR 不把“可执行文件同目录锁 + Canonicalize + 硬链接计数检查”视为已批准的唯一实现。** 该设计会影响系统目录安装、Scoop shim、符号链接和 Windows 文件身份处理；实施前应单独验证：

1. `os.Executable()` 在各平台的实际返回语义；
2. 通过 symlink 启动时锁路径是否统一；
3. 目录不可写时 update 的预期产品行为；
4. Windows / Unix 文件 link count 查询能力与测试可行性；
5. 检查 link count 与替换之间是否仍存在竞态；
6. bind mount、短文件名、大小写别名等残余边界。

若无法提供可靠文件身份保证，应如实记录边界，而不是声称路径规范化覆盖所有别名。

## 6. Shared Infrastructure 方案

> 本节是候选设计，不是当前 package 清单。具体签名需在迁移 PR 中通过调用点审查、测试和 API review 后确定。

### 6.1 `internal/checksum`（扩展现有 package）

适合集中真实重复的纯 SHA-256 / hex 基础能力：

```go
func SumHex(data []byte) string
func IsLowerHex64(s string) bool
func IsHex64(s string) bool
```

并优先让 `detect` 的文件 hash 使用现有 `checksum.StreamSHA256`，避免保留第二套手写读取循环。

必须保留不同 checksum 文件解析器的语义：

- 宽松解析允许路径型文件名、对重复项的处理方式按当前实现；
- Custom URL 严格解析只接受裸文件名、重复项处理更严格；
- 可以共享行扫描内核，但不能把两套 parser 的容错级别合并。

`checksum` 不得依赖 HTTP、download、output、CLI。

### 6.2 URL 策略：接受规则与 redirect-chain 规则分开设计

旧设计把 URL 策略简单归并成 `StrictPolicy` / `CustomPolicy`，这不足以表达当前代码的真实差异。至少存在两个独立维度：

**A. URL 接受范围**

| 类型 | 当前用途 | 接受规则（以调用点复核为准） |
|---|---|---|
| Control-plane / strict | GitHub API、自更新等受控控制平面 | HTTPS 为主；若有 loopback HTTP 测试例外，必须明确限制 host；拒绝 userinfo |
| User-supplied / custom | provider direct / directory、用户自建镜像 | 当前接受 HTTP/HTTPS 任意 host；拒绝其他 scheme 和 userinfo |

**B. Redirect 行为**

| 现有调用族 | 现有差异 | 迁移要求 |
|---|---|---|
| `customurl/urlpolicy` | 审计记录指出该处具有整条 redirect chain 的 HTTPS 降级检查 | 必须保留链状态语义 |
| `provider/fetch.go` | 允许 HTTP/HTTPS 任意 host；当前检查目标 scheme，若初始源是 HTTPS 则拒绝降级到 HTTP | 保留其自定义源接受范围和既有降级语义；不得改成 strict host policy |
| `download` | 下载自官方或自定义 release；校验应由来源 provenance 决定 | 先确认每类 `Release.Provider` / URL 来源，再决定策略；不允许一刀切 |
| `provider/github` | GitHub API 客户端有自己的 redirect / token 语义 | 保留 API client 的 transport、auth、rate/retry 和脱敏行为 |
| `selfupdate` | 自更新下载与替换 | 保留其现有 host / scheme / redirect 限制和错误映射 |

因此，迁移前必须形成**逐调用点策略矩阵**，明确初始 URL 规则、redirect URL 规则、是否 sticky HTTPS、回调顺序、重定向上限、错误码和 HTTP client 来源。没有矩阵不得把所有闭包替换成同一个策略函数。

若创建 `internal/netpolicy`：

- 只放 URL / redirect policy；不得依赖 `httpfetch` 或 feature；
- 外部调用者不能构造“跳过校验”的策略；零值必须 fail closed；
- policy 类型使用未导出字段和明确构造函数；
- 回调前先校验 redirect URL；调用方回调返回 nil 后，必须再校验最终 `req.URL`，防止 callback 改写 URL 绕过；
- 若调用方 callback 返回 `http.ErrUseLastResponse`，原样返回，不得包装；
- `req == nil`、`req.URL == nil` 不得 panic；
- `maxRedirects` 必须定义为最大跳数而非请求数，并测试 0、1、最大值及越界；
- `Strict` / `Custom` 的命名只有在真实行为维度已被清楚分离后才允许使用；不能用名称掩盖不同 redirect chain 语义。

**禁止将 `provider/fetch.go` 直接迁入旧 ADR 所写的 `StrictPolicy`。** 这会把任意 host HTTP 自定义源收紧为 HTTPS/loopback，属于行为变更。

### 6.3 `internal/httpfetch`（满足条件后再新建）

只适合小型、有界文本读取，例如 checksums、latest 文本或受限 release 元数据；不负责归档下载。

候选职责：

- 校验并读取 bounded response body，超限时明确失败而非静默截断；
- 复用调用方 HTTP client 的 Transport、TLS、Proxy、Jar、Timeout 等配置；
- 不改变调用方传入 client 的原对象；
- 统一 UA / 非 2xx / Body.Close / maxBytes+1 检测等真正共同行为；
- redirect policy 显式注入，不能由 helper 静默决定 Strict / Custom。

不应该吸收：

- 大归档流式下载、进度、文件提交、校验和重试；
- GitHub client 的认证、rate limit、API 错误映射和业务重试；
- TLS CA 配置、token、代理等属于各调用方的配置来源。

`MaxBytesCeiling`（例如 8 MiB）只有在重新核对所有真实文本调用点后才能定案；旧 ADR 的数值是设计建议，不是现有系统保证。超过文本上限需要单独设计流式 API，而不是无上限提升 `[]byte` 读取限制。

### 6.4 `internal/fsatomic`（候选；不能统一底层语义）

只有在现有至少两个调用点确实共享相同持久化语义时才新建。

可以考虑的低层能力：

```go
func WriteFile(path string, payload []byte, mode os.FileMode) error
func WriteLockedFile(path string, payload []byte, mode os.FileMode) error
```

其中：

- `WriteFile`：普通文件临时写入与替换；必须定义 mode、短写、Sync、Close、rename、临时文件清理和平台失败行为；
- `WriteLockedFile`：仅封装现有 lockedfile 写入语义，表示跨进程文件锁写；**不代表** temp+rename 原子替换；
- `ReserveFile`：只有确认 download 与其他普通文件占位点具有同样生命周期时才抽取；需记录文件句柄 / 所有权 / release 失败语义；
- `ReserveLink` 不放入 fsatomic，链接和 junction 操作归 `platform`。

必须维持的调用方行为：

| 调用点 | 现在的关键语义 | 迁移约束 |
|---|---|---|
| Journal 写入 | 文件级跨进程锁 | 不得移除 lock / sidecar 语义 |
| Cache JSON 写入 | 文件级锁或其当前持久化策略 | 保持原错误码 / cache 失效行为 |
| Download commit | 唯一 `.bak`、并发提交判断、失败恢复 | 不迁为简单 `WriteFile` |
| Self-update state / executable 操作 | `FileOps` 注入、阶段状态和 rollback | 保留注入边界与状态机语义 |
| current pointer | symlink / junction / fallback pointer | 使用 platform 适配，不作为普通文件写入 |

**平台保证必须审慎描述**：Unix `rename` 的同文件系统目录项替换、Windows `os.Rename` 的运行时实现、网络文件系统行为、权限元数据和掉电持久性并不等价。不要在 ADR 中笼统声称“所有 Windows 文件系统都原子”或“rename 成功即崩溃持久”。

`ReserveFile` / `ReserveLink` 的 Lstat + SameFile + Remove 存在竞态窗口。它们最多在可信、隔离目录下提供常见误删防护，不得宣传为不可信共享目录下的绝对 TOCTOU 防护。优先使用权限隔离、随机文件名和句柄生命周期，必要时单独研究 `unlinkat` / Windows handle-based delete。

### 6.5 `internal/platform`（扩展现有 package）

候选收敛内容：

- `LinkKindAt(path)`：统一查询链接 / junction / 普通文件类型，错误语义需要明确；
- 当前 `install` / `platform` 重复的链接类型判断；
- `install` 的路径包含判断改为委托现有 `platform.LexicallyInside`；
- `Canonicalize` 只有在调用方确实需要完整路径解析时才新增，并明确不存在路径、symlink、权限、硬链接和 Windows junction 的边界。

`platform` 自己定义链接类型，避免 `platform → model`。Windows `os.Symlink`、junction 和 dangling link 语义需基于现有实际代码、目标是否存在和 CI 环境决定；不在本 ADR 预先要求使用某个 Win32 API，也不把 junction→symlink fallback 写成已批准行为。

### 6.6 `internal/errors`（按需扩展）

`download`、GitHub、checksum、ffexec 中的 context cancellation / timeout 映射具有相似结构，但领域错误码并不相同。

只有在重复逻辑确实稳定时，才考虑：

```go
func ContextError(err error, timeoutCode, canceledCode Code) error
```

不得为了抽取四行包装函数而增加泛化错误框架；映射结果、`errors.Is` / `errors.As` 和既有 typed details 必须由调用点测试锁定。

### 6.7 `internal/ffexec`（按需扩展）

`install` 与 `doctor` 的 `findBinary` 若确认逐字符且语义完全相同，可选择放入 `ffexec` 或更合适的现有基础 package。提取必须保留 Windows 扩展名、可执行文件判定、错误吞吐方式等现有语义。

不提取 `HasBanner`：detect 扫描完整版本输出与 doctor 只检查首行存在语义差异，不为单个共享调用扩大 API。

### 6.8 不创建 `clock` / `retry`

- `time.Now().UTC().Format(time.RFC3339)` 很简单，但调用方时间语义与注入需求不同；需要可测时优先在原 package 里提供最小 Now 注入。
- GitHub API 重试与 Self-update 传输重试可能有不同的重试条件、退避、Retry-After 和错误映射；不能只因都有 sleep 就合并。
- `ffexec` / `lock` 内部用于超时与单调时钟的 `time.Now()` 不应为了测试可注入而全局替换。

## 7. DTO、Contracts 与可变全局状态

### 7.1 `model`

继续保持轻量 DTO / JSON contract 数据，不引入完整 Domain Model。

- `CurrentSymlink` / `CurrentJunction` 等平台状态常量应从 model 下沉到 platform 自己的类型；在输出边界映射回现有 JSON 字符串，JSON 值不变。
- `KnownDataSchemas` 不应以导出可变 map 暴露；可以改为私有 map + 返回副本的 getter。
- 不把 provider 规则、网络策略、文件事务逻辑放到 model。

### 7.2 `contracts`

- 按验证对象拆分文件（result / error / release / artifact / capability / provider / env / doctor / update）；
- 初期仅拆文件，不改变导出 API 和验证顺序；
- `CommandDataSchemas` 返回深拷贝，包含内部 `[]string` 的复制，不能只复制 map；
- `knownCommands` 等包内只读 map 可以保留。

### 7.3 `contracts → capabilities`

把 `IsCapabilityKind`、`CommandKeys` 等纯 metadata 从具体 capabilities 实现包移到中立定义位置。若放入 `model`，需确认其语义是 JSON / DTO 元数据，而非采集业务。保留 `contracts` 对错误类型和 DTO 的依赖，不引入循环。

## 8. 大型 package / 函数拆分

原则：优先 package 内按职责拆文件与私有函数，只有独立数据所有权或独立生命周期才拆 package。

| 位置（旧审计快照） | 问题 | 第一选择 |
|---|---|---|
| `internal/contracts/contracts.go` | 约 746 行，多个数据 schema 校验混在一个文件 | 拆成多个同 package 文件，保持外部 API |
| `internal/download/download.go` | 下载、redirect、校验、进度、commit 逻辑较集中 | 同 package 拆 client / transfer / commit helpers；不拆成许多小包 |
| `internal/env/apply.go` | env plan / transaction 逻辑较长 | 按路径、变量、shell rc plan 拆私有函数 |
| `internal/selfupdate/selfupdate.go` | `Apply` 复杂状态机 | 拆 download/verify/stage/swap/commit 私有步骤，保持状态顺序和 FileOps |
| `internal/archive/archive.go` | ZIP / TAR 分支 | 分文件拆格式实现，共用安全路径 helpers |
| `internal/provider/custom.go` | provider schema、读写、校验、锁混合 | 先拆文件：store IO 与 schema/validation；子包需要重新评估 |
| `internal/cli` | 多命令、命令 Action 和编排集中 | 每命令保留独立文件；只把真实跨 feature use case 移入 app |

行数只是定位线索。函数拆分应按事务边界、错误路径和资源生命周期拆，不以“函数少于 N 行”为验收目标。

## 9. 重复代码迁移原则

### 9.1 候选真重复

旧审计指出以下重复项，实施前必须重新按当前 HEAD 复核：

| 重复项 | 候选归属 | 约束 |
|---|---|---|
| `sha256Hex(data []byte)` 多处重复 | `checksum.SumHex` | 纯计算，无副作用 |
| `isLowerHex` / SHA-256 长度校验 | `checksum` | 保留 lowercase 与 mixed-case 两种语义 |
| `detect.hashFile` / `install.checksumFile` | 委托 `checksum.StreamSHA256` | 保留不同错误码映射 |
| `isLoopback` | `netpolicy` 或原使用方 | loopback 识别必须覆盖 IPv4 / IPv6 / localhost 的真实定义 |
| URL 校验 / redirect policy | `netpolicy`（候选） | 先完成第 6.2 节矩阵；不能按函数名合并 |
| 有界文本抓取 | `httpfetch`（候选） | 不合并 Archive download 和 GitHub client 专属逻辑 |
| `findBinary` | `ffexec` 或现有合适 package | 必须逐项相同，包括平台扩展名和可执行位语义 |
| link type / path containment | `platform` | `currentType` 语义可能比简单 LinkKind 更丰富，不得降级 |
| context 错误包装 | `errors`（候选） | 领域错误码保持在调用方 |
| `toReleaseInfo` | `selfupdate` package 内私有函数 | 不需要新 package |

### 9.2 必须保持独立的伪重复

- stdout `WriteJSON` 与写文件 `WriteJSON`；
- 本地文件复制与 HTTP 下载；
- 普通文件提交与 symlink/junction replace；
- detect 的可执行文件查找与 install 的查找（若语义不同）；
- 两套 checksum parser 的严格/宽松差异；
- `install.currentType` 与单纯 `LinkKindAt`（前者可能额外验证 target 和外来文件）；
- download `.bak` 提交、journal locked write、selfupdate FileOps 状态提交；
- detect 完整版本输出 banner 与 doctor 首行 banner；
- 每个领域各自的退避 / Retry-After / 超时处理。

## 10. 未接入 / 疑似死代码处理

删除前必须针对当前 commit 执行引用搜索和测试，不可只根据旧审计结果直接删除。

| 项目 | 旧审计结论 | 处理原则 |
|---|---|---|
| `internal/deps` | 占位 package，无实际功能 | 确认无引用后删除 |
| `cache.Store[K,V]` 与 TTL/LRU/Prune | 测试使用、无生产调用 | 如果能力缓存仍用 JSON helpers，则删除泛型容器或补充真实产品调用理由 |
| `selfupdate.Options.Now` | normalized 时设置但后续未使用 | 逐调用点确认后删除；不要与 provider Resolver.Now 混淆 |
| `platform.LinkKind()` | 无生产调用的无参函数 | 查明是否与 LinkKindAt 重叠后删除 |
| `github.LatestRelease` / `ReleaseByTag` | 生产路径可能已由 go-github 替代 | 保留测试注入等真实使用路径，确认后再删 |
| `provider.Provider` / `Registry.Add` | 当前注册表可能没有实际外部实现者 | 保留或收敛，不能据 future 注释直接删除 |
| `errors.E5006` / `E6004` | 目前无已知发出路径 | 错误码属于兼容数据，先保留，不强行补实现 |
| `KnownCodes` / `defaultWait` | 测试辅助 | 标注用途，按引用决定 |
| `install.TargetDir` | 导出薄壳 | 确认调用方后考虑改由更适合 package 提供 |
| `testutil/fake_ffmpeg.go` | 普通 `.go` 测试辅助 package | 已无生产 import 时可保留，不误称它一定进入主可执行文件；不要因加 build tag 破坏测试 import |

## 11. 测试基线与实施前质量门禁

### 11.1 既有历史记录

旧 ADR 对基准 `24a3f9f` 记载：

- `go test ./...`、`go vet ./...`、`go build ./...` 曾通过；
- Linux / Windows amd64 `CGO_ENABLED=0` 交叉编译曾通过；
- `go test -coverprofile=... ./...` 曾因 testscript 在覆盖模式下 stderr 断言失败；
- 失败运行得到的部分覆盖率（例如 67.3%）不能作为正式基线；
- race、fuzz、E2E 当时未在该轮执行。

以上是**历史记录**，本轮未在当前最新 HEAD 重新运行。不能在 PR 描述中复制这些结果充当本 PR 测试记录。

### 11.2 PR-0 必须重新建立基线

1. 固定并记录当前 checkout 的完整 commit SHA；
2. 修复或解释覆盖率模式下 testscript 的 stderr 断言问题；
3. 成功运行 `go test -coverprofile` 并确认覆盖率文件涵盖所有包；
4. 执行 `go test ./...`、`go vet ./...`、`go build ./...`；
5. 执行 Windows / Linux 交叉编译；
6. 对并发 / 安全敏感包执行 race / fuzz；
7. 按环境条件执行 pytest / Robot / Bats / remote Linux / real network E2E；
8. 检查 JSON golden、schema freeze、error code freeze 和 CLI 行为测试。

覆盖率不设单一百分比目标；关键是 `customurl/latest`、`platform`、`env`、`selfupdate`、CLI composition 等核心错误路径的测试完整性。

### 11.3 测试注入策略

- 保留现有 HTTP client / fetch 注入；
- 保留 `selfupdate.FileOps`；
- 保留 `platform.EnvStore`、rollback 和 Progress 测试边界；
- 不为每个 `os.*` 调用创建 filesystem interface；
- `ffexec.Runner` 只有在真实测试需求无法覆盖时再增加小接口；
- 每个重构先补旧行为的 characterization tests，再移动代码。

## 12. 安全范围和安全行为控制

### 12.1 本 ADR 可包含的行为修复

若某个现有问题已被审计明确定位，且修复范围有限、测试能证明预期，可单独拆 PR 并标记安全行为变化。例如：

- redirect 回调执行后再次校验最终 URL，防止 callback 改写绕过；
- 修正 `maxRedirects` 的跳数 / 请求数 off-by-one；
- 修正 Self-update 固定 `.tmp` 名的并发覆盖风险；
- 保留 typed error，不以裸 `fmt.Errorf` 丢失稳定错误码。

这些都应是独立可审查的改动，不得与大规模目录移动混在同一 PR。

### 12.2 明确不做的安全项目

| 项目 | 本 ADR 状态 | 说明 |
|---|---|---|
| Ed25519 / release 签名验证 | 不做 | 需密钥管理、轮换、信任根与发布流程设计 |
| 完整 SSRF 防护 | 不做 | 需 DNS/IP、metadata、rebinding、proxy 与自托管场景策略 |
| Journal WAL / 全事务重写 | 不做 | 会改变跨进程一致性语义，需独立 ADR |
| 安装事务重写 | 不做 | 只允许行为等价整理，不改状态机 |
| Self-update 新版本健康探针后提交 | 不做 | 需定义探针、失败判定和回滚协议 |
| 自定义 URL 禁止 HTTP | 不做 | 这是产品安全与兼容决策，不得混入结构重构 |

### 12.3 安全测试底线

- Strict 与 Custom 的 scheme / host / userinfo 行为分别测试；
- 测试 `http → https → http`、`https → http`、loopback 例外与自定义来源行为；
- 测试 callback 修改 `req.URL`、修改 `via`、返回 `http.ErrUseLastResponse`、nil request；
- 归档解压验证路径逃逸、symlink、硬链接和平台差异；
- 下载提交验证并发、备份和失败恢复；
- journal / DataRoot / Self-update 锁独立测试，不能因锁 API 重构降低互斥范围。

## 13. 分阶段迁移计划

按依赖顺序分阶段实施。每项可拆成多个小 PR；不固定 PR 数量。

### Phase 0：锁定现状和基线（必须先做）

- 固定 commit SHA；重跑 `go list ./...`、import graph、`go.mod` / `go.sum`、文件和测试统计；
- 建立 test / vet / build / cross-build / race / fuzz / E2E 基线；
- 修正 testscript coverage 失败，生成可信覆盖率基线；
- 建立 CLI 命令 / schema / error code freeze 清单；
- 复核当前架构文档、技术文档与源码冲突。

**退出条件**：可重复运行的基线记录已提交；审计快照有固定 commit SHA。

### Phase 1：低风险叶子包和契约安全

- 清理 `deps`、未接入 Store、死注入等已证实无用项；
- 合并纯 SHA-256 / hex 校验重复；
- 修复可变全局 map 的暴露方式；
- `contracts.go` 仅拆同 package 文件，不改校验行为；
- 移除 `platform → model` 和 `contracts → capabilities` 反向依赖。

**退出条件**：依赖图无环；JSON golden / schema freeze 不变。

### Phase 2：网络策略边界

- 逐调用点建立 URL 与 redirect 策略矩阵；
- 识别 control-plane strict 与 user-supplied custom 的差异；
- 单独设计策略构造器和有限文本获取工具；
- `provider/fetch` 不得迁移到 strict host acceptance；
- 保留 caller 的 HTTP client TLS / proxy / auth / retry 设置；
- callback 前后 URL 复核和 redirect hop 规则用专门测试覆盖。

**退出条件**：所有调用点行为矩阵与代码一致；没有用户自定义源行为意外收紧。

### Phase 3：文件系统原语

- 先给 Journal、Cache、Download、Install、Env、Self-update 补行为测试；
- 明确各写入的并发、原子替换、backup、mode、Sync、Close 与失败回滚语义；
- 只有重复的底层原语才迁入 fsatomic；
- Link / Junction 保留在 platform；
- 不删除 DataRoot / journal / download / selfupdate 各自所需的锁和注入。

**退出条件**：各写入路径的行为与错误语义不变，跨平台测试有证据。

### Phase 4：大函数与 provider / contracts 收敛

- 按事务阶段拆 selfupdate.Apply、env plan、download transfer、archive format 等函数；
- provider store 先做文件级拆分；只有多个调用方和独立所有权成立才拆子包；
- 复核 `Provider` interface / `Registry` 的真实用途；
- 检查所有新增 export 是否真正跨 package 调用。

**退出条件**：外部 API 无意外扩大；每个 package 有清晰单一职责。

### Phase 5：Composition Root / App 评估

- 以真实调用图评估跨 feature use case；
- 若满足第 4 节门槛，再创建 `internal/app`；否则明确决定继续由 CLI 编排；
- 如创建 App，锁所有权在同 PR 转移；
- 单模块命令保持直连 feature。

**退出条件**：App 如创建则不含转发 wrapper；锁所有权唯一；CLI fan-out 有实质下降。

### Phase 6：完整回归和架构验收

- 重新生成依赖图、包清单、职责清单；
- 跑完整质量门禁和双平台 E2E；
- 检查文档、help、CLI schemas、错误码和示例；
- 输出迁移前后 diff、风险清单与尚未解决项。

**退出条件**：第 14 节验收标准全部满足或逐项说明豁免理由。

## 14. 验收标准

1. 固定 commit 的实际 import graph 与文档一致；无循环依赖。
2. `errors`、`model`、`output`、`buildinfo` 保持零内部依赖。
3. `platform` 不依赖 feature；目标状态不再依赖 `model`。
4. `contracts` 不依赖 CLI；目标状态不再依赖 `capabilities` 实现包。
5. `download` / `selfupdate` 不依赖 `output`；progress 接口归使用方。
6. App（若创建）只承接真实跨模块 use case，不包装单模块方法。
7. DataRoot 锁由最外层协调者唯一持有；Self-update、Journal、Download 锁按语义独立。
8. 重复提取均有调用点证据；伪重复保持独立。
9. 网络规则对所有调用点显式，URL 接受范围与 redirect-chain 约束两个维度不互相混淆。
10. 文件写入 API 不改变原有锁、backup、atomic replace、FileOps 和回滚语义。
11. CLI 命令、flags、JSON schema、错误码、exit code、安装布局和文件格式零意外变化。
12. Go tests、vet、build、双平台交叉编译、必要 race/fuzz/E2E 全通过。
13. docs / README / reference / help 文本与实现一致；没有把历史数据写成当前状态。
14. 无未解释的新增依赖、未使用导出 API 或死代码。

## 15. 风险与回滚

| 风险 | 缓解 | 回滚 |
|---|---|---|
| CLI JSON 或退出码变化 | 每 PR 跑 freeze / golden / schema tests | 单 PR revert |
| URL 策略被错误收紧 | 建立逐调用点策略矩阵；Custom 允许行为单独测试 | 单 PR revert；策略增强另立 PR |
| 文件原子性或锁语义退化 | 分别测试普通写、locked write、download commit、selfupdate FileOps | 保留旧实现至新路径验证完成 |
| DataRoot 重复 / 漏锁 | 同 PR 转移所有权；检索所有锁调用点 | 单 PR revert |
| App 变成转发层 | 按第 4 节门槛评审；未达门槛不创建 | 删除该 App PR |
| 跨平台链接差异 | Windows/Linux 实测；不在设计阶段夸大保证 | 回退 platform 改动 |
| 死代码误删 | 搜索生产引用、测试引用、build tags 和数据兼容性 | 单 PR revert |
| 文档先于代码 | 文档中的 As-Is / To-Be 分区；每 PR 同步更新 | 文档独立 revert |

**回滚原则**：结构迁移逐 PR 进行；不要把多个高风险改动打包成无法单独回滚的大 PR。

## 16. 未决事项（实施前必须确认）

| # | 事项 | 默认处理 | 影响 |
|---|---|---|---|
| 1 | 是否创建 `internal/app` | 暂不创建，Phase 5 评估 | 影响 cli fan-out 与 DataRoot 锁所有权 |
| 2 | 自定义来源 HTTP 是否维持 | 维持当前 HTTP/HTTPS 行为 | 改为 HTTPS-only 属安全 / 产品兼容变更，不属于本 ADR |
| 3 | URL policy 是否抽成一个公共包 | 先完成调用点策略矩阵 | 若链路行为不同，可能需要多个策略构造器或保留现有调用点 |
| 4 | 文本抓取上限 | 按真实现存上限逐调用点定值 | 不把归档 `Download.MaxBytes` 用作文本 body 限制 |
| 5 | `fsatomic` 是否创建 | 只有真实共享调用点及语义得到验证才创建 | 避免过度抽象 |
| 6 | ReserveFile / ReserveLink 的目录信任模型 | 默认只允许应用私有可信目录 | 不得声称对不可信共享目录绝对防 TOCTOU |
| 7 | Self-update 锁身份 | 单独评估 executable path / file identity / hardlink 等边界 | 关系到系统目录安装、symlink 与并发更新 |
| 8 | Windows 链接策略 | 以现有实现和测试为基线再定 | 不提前强制 Junction / symlink fallback 行为 |
| 9 | 目录 fsync 与崩溃持久性 | 默认不额外承诺 | 若产品需要，须另外设计和测量 |
| 10 | Runner 是否接口化 | 暂不接口化 | 只有存在具体测试缺口才引入窄接口 |

## 17. 批准状态

设计状态：**Proposed**  
实施授权：**Awaiting Implementation Approval**

- [ ] 用户批准目标架构与迁移原则；
- [ ] Phase 0 完成，当前 HEAD 和质量基线锁定；
- [ ] 用户批准开始源码实施；
- [ ] 每阶段完成后更新 As-Is / To-Be 依赖图和审计结论。

**未批准前不得启动源码重构。** 本轮只准备架构文档与 ADR，没有修改 Go 源码、测试、go.mod、go.sum 或 CLI 契约。
