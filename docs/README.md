# 文档索引与权威顺序

> 项目：OneMoreFFmpeg
> 状态：**Phase 1–6 已完成**，CLI 外部契约已冻结；Windows / Fedora / Win2025 三机 e2e 通过
> 日期：2026-11-14

**历史文档已全部移入 [`archive/`](archive/README.md)**：阶段状态、代码审查、全量审计记录保留为当时的快照，
不再维护；它们记录的是「当时发生了什么」，**不代表当前状态**。需要现行规则时只看下面 §1。

---

## 1. 现行文档（权威顺序）

冲突时按优先级裁决，数字越小越权威。

| 优先级 | 文档 | 权威范围 |
|---|---|---|
| 1 | [`PROJECT_PLAN.md`](PROJECT_PLAN.md) | 阶段划分、范围、步骤、DoD |
| 2 | [`CLI_SCHEMA.md`](CLI_SCHEMA.md) | 当前唯一对外契约：CLI JSON v1（命令集 / schema / 字段） |
| 3 | [`ERROR_CODES.md`](ERROR_CODES.md) | 错误码、警告码、退出码映射 |
| 4 | [`ARCHITECTURE.md`](ARCHITECTURE.md) | 模块边界与依赖方向 |
| 5 | [`CAPABILITIES.md`](CAPABILITIES.md) | 13 项能力检查与输出模型 |
| 6 | [`PROVIDERS.md`](PROVIDERS.md) | 来源、channel、release pin、自定义 URL 总原则 |
| 7 | [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md) | GitHub API、限流、资产匹配、checksums、pin 细则 |
| 8 | [`DETECTION_AND_VALIDATION.md`](DETECTION_AND_VALIDATION.md) | 发现与校验规则 |
| 9 | [`ENVIRONMENT.md`](ENVIRONMENT.md) | PATH 与环境变量总原则 |
| 10 | [`PLATFORM_ENVIRONMENT.md`](PLATFORM_ENVIRONMENT.md) | Windows / Linux 的 PATH、current、journal 细则 |
| 11 | [`SYSTEM_REQUIREMENTS.md`](SYSTEM_REQUIREMENTS.md) | 系统要求、权限与提权模型、事务锁 |
| 12 | [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md) | 依赖选择、版本新鲜度、拒绝理由 |
| 13 | [`ROADMAP.md`](ROADMAP.md) | 阶段顺序与各阶段退出条件 |
| 14 | [`TECHNICAL_NOTES.md`](TECHNICAL_NOTES.md) | 技术约束、风险、注意点 |
| 15 | [`VERSIONING.md`](VERSIONING.md) | SemVer 版本与发布规范 |
| 16 | [`API_AND_PLUGIN.md`](API_AND_PLUGIN.md) | 外部集成方式（只调用 CLI）；明确不提供 Go API/插件 |
| 17 | [`LICENSE_DECISION.md`](LICENSE_DECISION.md) | 许可证决策（GPL-3.0-only） |
| 18 | [`E2E_TESTING.md`](E2E_TESTING.md) | Windows / Linux e2e、远程服务器与 CI |
| 19 | [`HOWTO.md`](HOWTO.md) | 新手上手指南（第一次用先读这个） |
| 20 | [`USAGE.md`](USAGE.md) | 详细使用教程 |
| 21 | [`schemas/README.md`](schemas/README.md) | CLI JSON 示例索引（17 个 data schema 全覆盖） |
| 22 | [`man/one-more-ffmpeg.1`](man/one-more-ffmpeg.1) | Linux man 页 |
| 23 | `README.md` / `docs/README.md` | 入口与索引，不承担规则裁决 |

冲突时：

- 范围与阶段 → `PROJECT_PLAN.md`；CLI 输出 → `CLI_SCHEMA.md` + `schemas/`；
- 错误码 → `ERROR_CODES.md`；模块边界 → `ARCHITECTURE.md`；能力命令 → `CAPABILITIES.md`；
- provider 总原则 → `PROVIDERS.md`；解析细节 → `PROVIDER_RESOLVER.md`；
- 发现/校验 → `DETECTION_AND_VALIDATION.md`；环境变量 → `ENVIRONMENT.md` + `PLATFORM_ENVIRONMENT.md`。

## 2. 归档文档（历史快照，不维护）

见 [`archive/README.md`](archive/README.md)。包含 18 份：Phase 1–6 的 `*_STATUS.md` 与 `*_CODE_REVIEW.md`、
`PHASE4_READINESS.md`、6 份 `FULL_AUDIT_*`、`DOCUMENTATION_AUDIT.md`。

**引用它们时要注意**：里面的「未实现」「计划中」「待办」都是**当时**的结论，多数已在后续 Phase 完成。
判断当前能力只看 §1 与代码。

发布说明单独放在 [`releases/`](releases/)（v0.6.0 / v0.7.0 / v0.8.0）：它们按版本记录一次发布的内容，
同样不随代码更新；当前版本与 SemVer 规则见 [`VERSIONING.md`](VERSIONING.md)。

## 3. 当前决策快照

| 项 | 值 |
|---|---|
| 语言 | Go only |
| Go 版本 | `go1.27.4`（`mise.toml` 声明 `1.27`） |
| 平台 | Windows x64 + Linux x86_64 |
| CLI | `one-more-ffmpeg` |
| Go module | `github.com/FroyoCatqwq/OneMoreFFmpeg` |
| 内置来源 | `FFmpeg-Builds/FFmpeg-Builds`、`BnqDzj/FFmpeg-Builds-nonfree` |
| 自定义来源 | 支持，但必须校验 SHA-256 |
| channel | `stable` / `daily` 抽象，解析为具体 release tag + asset SHA-256 |
| 能力命令 | 13 个 FFmpeg 参数 / 能力入口，见 `CAPABILITIES.md` |
| 非公开命令 | `-buildconf` 仅用于内部安装校验，v1 不进入公开 capability 集 |
| 包管理器识别 | 不做（明确推迟） |
| 对外开放 Go API / 插件 / JSON-RPC / C ABI | 不做（明确推迟） |
| 外部集成 | 其他项目只调用本项目的 CLI 可执行文件 |
| CLI 框架 | `urfave/cli/v3` |
| TOML | `pelletier/go-toml/v2`，支持 TOML 1.1.0 |
| CLI JSON | `CLI_SCHEMA.md` + `schemas/examples/`（18 命令 / 17 个 `/v1` data schema，已冻结） |
| 错误码 | `ERROR_CODES.md`（含 `OMF-E5009 IO_FAILED`） |
| 日志 | `log/slog` + `lmittmann/tint` |
| 测试 | `stretchr/testify`；e2e 用 pytest + Robot + Bats |
| 许可证 | `GPL-3.0-only`，见根目录 `LICENSE` 与 `LICENSE_DECISION.md` |
| nonfree | 元数据可公开，二进制不可随公开 Release 分发 |
| Phase 1 | 骨架、契约、错误码、进程执行器、测试与交叉编译 |
| Phase 2 | 真实发现源、三件套校验、`detect` / `validate` |
| Phase 3 | 13 项能力检查、解析、缓存、`capabilities` |
| Phase 4 | provider / resolve / download（真实 GitHub 解析 + 自定义资产） |
| Phase 5 | install（安全解压、版本化目录、install.json、current、journal）、`env` 全套、`uninstall`、`doctor` |
| Phase 6 | CLI 契约冻结（18 命令 / 17 schema / 退出码）、外部调用示例、解压安全加固 |
| Phase 7 | 未开始：CI 门禁（gofmt / govulncheck）、fuzz、真实上游资产集成、release 自动化与签名 |

## 4. 仍保留的限制

- **P1-4a**：`install --apply` 在锁外下载（有意取舍：持锁会阻塞其它进程；失败清理已只动本次文件）。
- **P2-2**：`journal list` / `doctor` 看不到损坏的 journal 记录（需要给 `model.JournalListData` 加 `warnings`）。
- **P2-3**：`env` 事务的撤销失败被丢弃，只报原始错误。
- **P2-4**：压缩炸弹比例检查依赖 `os.Stat`，失败时静默失效；tar 路径没有磁盘空间预检。
- **P2-5**：`install --json` 的 dry-run 输出 `journal_id: ""`，而 `env-set` / `uninstall` 省略该字段。
- `env set --apply` 会写 `~/.profile` 与 `HKCU\Environment`，无法在无人值守 CI 隔离，故 e2e 只覆盖 dry-run。
- 多 title 的 DVD 只读第一个 title（这与本项目无关，属 FFmpeg 的 dvdvideo demuxer 行为，见 LiveArchiveEncoder 文档）。

## 5. 未来，明确不属于当前 Phase

- 包管理器识别（Scoop / Homebrew / winget / choco / apt / dnf / pacman）；
- 开放式 Go API、`pkg/onemore` SDK；
- JSON-RPC / gRPC / Connect RPC 插件；
- C ABI / c-shared；
- 私有制品库、自动更新服务；
- GUI / TUI / CUI；
- FFmpeg 自构建。
