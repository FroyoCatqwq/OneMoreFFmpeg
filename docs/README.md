# 文档索引与权威顺序

> 项目：OneMoreFFmpeg
> 状态：Phase 1–5 已完成并通过全量审查与 Windows/Linux e2e
> 日期：2026-11-14
> 更新：用户视角 + 代码视角全量审查、缺陷修复、双机 e2e 与文档同步完成，见 `FULL_AUDIT_2026-11-14.md`

---

## 1. 权威顺序

| 优先级 | 文档 | 权威范围 |
|---|---|---|
| 1 | [`PROJECT_PLAN.md`](PROJECT_PLAN.md) | 当前阶段、范围、步骤、DoD |
| 2 | [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md) | Go 依赖选择、版本新鲜度、拒绝理由 |
| 3 | [`CLI_SCHEMA.md`](CLI_SCHEMA.md) | 当前唯一对外契约：CLI JSON v1 |
| 4 | [`ERROR_CODES.md`](ERROR_CODES.md) | 错误码、警告码、退出码 |
| 5 | [`ARCHITECTURE.md`](ARCHITECTURE.md) | 模块边界与依赖方向 |
| 6 | [`CAPABILITIES.md`](CAPABILITIES.md) | 13 个能力命令与输出模型 |
| 7 | [`PROVIDERS.md`](PROVIDERS.md) | 来源、channel、release pin、自定义 URL 总原则 |
| 8 | [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md) | GitHub API、限流、资产匹配、checksums、pin 细则 |
| 9 | [`DETECTION_AND_VALIDATION.md`](DETECTION_AND_VALIDATION.md) | 发现与校验规则 |
| 10 | [`ENVIRONMENT.md`](ENVIRONMENT.md) | PATH 与环境变量总原则 |
| 11 | [`PLATFORM_ENVIRONMENT.md`](PLATFORM_ENVIRONMENT.md) | Windows/Linux 平台级 PATH、current、journal 细则 |
| 12 | [`API_AND_PLUGIN.md`](API_AND_PLUGIN.md) | 未来 API/插件设计；当前不实现 |
| 13 | [`ROADMAP.md`](ROADMAP.md) | 阶段顺序 |
| 14 | [`TECHNICAL_NOTES.md`](TECHNICAL_NOTES.md) | 技术约束、风险、注意点 |
| 15 | [`LICENSE_DECISION.md`](LICENSE_DECISION.md) | 许可证决策记录（GPL-3.0-only） |
| 16 | [`DOCUMENTATION_AUDIT.md`](DOCUMENTATION_AUDIT.md) | 文档审查与改进记录 |
| 17 | [`PHASE1_STATUS.md`](PHASE1_STATUS.md) | Phase 1 实施状态与验证结果 |
| 18 | [`PHASE1_CODE_REVIEW.md`](PHASE1_CODE_REVIEW.md) | Phase 1 代码审查、P0/P1 修复与残留 tech debt |
| 19 | [`PHASE2_STATUS.md`](PHASE2_STATUS.md) | Phase 2 实施状态与验证结果 |
| 20 | [`PHASE2_CODE_REVIEW.md`](PHASE2_CODE_REVIEW.md) | Phase 2 代码审查、缺陷修复与残留 tech debt |
| 21 | [`PHASE3_STATUS.md`](PHASE3_STATUS.md) | Phase 3 实施状态与验证结果 |
| 22 | [`PHASE3_CODE_REVIEW.md`](PHASE3_CODE_REVIEW.md) | Phase 3 代码审查、P1 修复与残留 tech debt |
| 23 | [`PHASE4_READINESS.md`](PHASE4_READINESS.md) | Phase 4 准入清单、真实上游事实、决策项与缺口 |
| 24 | [`FULL_AUDIT_2026-11-11.md`](FULL_AUDIT_2026-11-11.md) | 全量审查、缺陷修复与残留风险 |
| 25 | [`PHASE4_STATUS.md`](PHASE4_STATUS.md) | Phase 4 实施状态、验证与限制 |
| 26 | [`PHASE4_CODE_REVIEW.md`](PHASE4_CODE_REVIEW.md) | Phase 4 审查、缺陷修复与残留限制 |
| 27 | [`PHASE5_STATUS.md`](PHASE5_STATUS.md) | Phase 5 安装/current/journal 状态与限制 |
| 28 | [`E2E_TESTING.md`](E2E_TESTING.md) | Windows/Linux e2e、远程服务器与 CI |
| 29 | [`USAGE.md`](USAGE.md) | 首次使用详细教程 |
| 30 | [`man/one-more-ffmpeg.1`](man/one-more-ffmpeg.1) | Linux man 页 |
| 31 | [`FULL_AUDIT_2026-11-13.md`](FULL_AUDIT_2026-11-13.md) | 2026-11-13 全量审查与二次修复 |
| 32 | [`VERSIONING.md`](VERSIONING.md) | SemVer 版本与发布规范 |
| 33 | [`releases/release-v0.6.0.md`](releases/release-v0.6.0.md) | v0.6.0 release notes |
| 34 | [`schemas/README.md`](schemas/README.md) | CLI JSON 示例索引 |
| 35 | `README.md` / `docs/README.md` | 入口与索引，不承担规则裁决 |
| 36 | [`FULL_AUDIT_2026-11-14.md`](FULL_AUDIT_2026-11-14.md) | 2026-11-14 用户/代码双视角全量审查与修复 |
| 37 | [`SYSTEM_REQUIREMENTS.md`](SYSTEM_REQUIREMENTS.md) | 最低/推荐系统要求、权限授权模型与事务锁 |
| 38 | [`releases/release-v0.7.0.md`](releases/release-v0.7.0.md) | v0.7.0 release notes |

冲突时：

- 范围与阶段：以 `PROJECT_PLAN.md` 为准；
- 依赖库：以 `GO_LIBRARY_EVALUATION.md` 为准；
- CLI 输出：以 `CLI_SCHEMA.md` 与 `schemas/` 为准；
- 错误码：以 `ERROR_CODES.md` 为准；
- 模块边界：以 `ARCHITECTURE.md` 为准；
- 能力命令：以 `CAPABILITIES.md` 为准；
- provider 总原则：以 `PROVIDERS.md` 为准；
- provider 解析细节：以 `PROVIDER_RESOLVER.md` 为准；
- 发现与校验：以 `DETECTION_AND_VALIDATION.md` 为准；
- 环境变量总原则：以 `ENVIRONMENT.md` 为准；
- 平台环境变量细节：以 `PLATFORM_ENVIRONMENT.md` 为准；
- 未来 API/插件：以 `API_AND_PLUGIN.md` 为准，但当前不实现。

---

## 2. 当前决策快照

| 项 | 值 |
|---|---|
| 语言 | Go only |
| Go 版本 | `go1.27.3` |
| 平台 | Windows x64 + Linux x86_64 |
| CLI | `one-more-ffmpeg` |
| Go module | `github.com/FroyoCatqwq/OneMoreFFmpeg` |
| 内置来源 | `FFmpeg-Builds/FFmpeg-Builds`、`BnqDzj/FFmpeg-Builds-nonfree` |
| 自定义来源 | 支持，但必须校验 SHA-256 |
| channel | `stable` / `daily` 抽象，解析为具体 release tag + asset SHA-256 |
| 能力命令 | 13 个 FFmpeg 参数 / 能力入口，见 `CAPABILITIES.md` |
| 非公开命令 | `-buildconf` 仅用于内部安装校验，v1 不进入公开 capability 集 |
| 包管理器识别 | 当前取消；Scoop / Homebrew / winget 等纳入未来 |
| 对外开放 Go API | 当前取消；纳入未来 |
| 插件 / JSON-RPC / C ABI | 当前取消；纳入未来 |
| 外部集成 | 其他项目只调用本项目的 CLI 可执行文件 |
| CLI 框架 | `urfave/cli/v3` |
| TOML | `pelletier/go-toml/v2`，支持 TOML 1.1.0 |
| CLI JSON | `CLI_SCHEMA.md` + `schemas/examples/` |
| 错误码 | `ERROR_CODES.md` |
| 日志 | `log/slog` + `lmittmann/tint` |
| 测试 | `stretchr/testify` |
| 许可证 | `GPL-3.0-only`，见根目录 `LICENSE` 与 `LICENSE_DECISION.md` |
| nonfree | 元数据可公开，二进制不可随公开 Release 分发 |
| Phase 1 | 骨架、契约、错误码、进程执行器、测试与交叉编译已完成 |
| Phase 2 | 真实发现源、三件套校验、`detect` / `validate` 已完成 |
| Phase 3 | 13 项能力检查、解析、缓存、`capabilities` CLI 已完成并通过代码审查 |
| Phase 4 | provider / resolve / download 已实现：真实 GitHub 解析与自定义资产下载通过；见 `PHASE4_STATUS.md` |
| Phase 5 | install（安全解压、版本化目录、install.json、current、journal）与只读 `env status` / `doctor` 已实现；见 `PHASE5_STATUS.md` |

---

## 3. 尚未完成

- `env set/unset/rollback`（写环境）；
- `uninstall` / 自动 rollback；
- Windows directory junction；
- GitHub Actions / 发布 CI（Phase 7）；
- 公开二进制发布与完整许可证复核；
- 包管理器识别；
- Go API / 插件接口。

## 4. 未来，明确不属于当前 Phase

- 包管理器识别；
- Go API；
- JSON-RPC / gRPC / Connect RPC 插件；
- C ABI / c-shared；
- 私有制品库；
- 自动更新服务；
- GUI / TUI / CUI。

## 5. 审查与状态报告

- [`DOCUMENTATION_AUDIT.md`](DOCUMENTATION_AUDIT.md)：文档审查、事实核实、P0/P1/P2 问题与修复记录。
- [`PHASE1_STATUS.md`](PHASE1_STATUS.md)：Phase 1 实施状态与验证结果。
- [`PHASE1_CODE_REVIEW.md`](PHASE1_CODE_REVIEW.md)：Phase 1 代码审查、P0/P1 修复与残留 tech debt。
- [`PHASE2_STATUS.md`](PHASE2_STATUS.md)：Phase 2 实施状态与验证结果。
- [`PHASE2_CODE_REVIEW.md`](PHASE2_CODE_REVIEW.md)：Phase 2 代码审查、缺陷修复与残留 tech debt。
- [`PHASE3_STATUS.md`](PHASE3_STATUS.md)：Phase 3 实施状态与验证结果。
- [`PHASE3_CODE_REVIEW.md`](PHASE3_CODE_REVIEW.md)：Phase 3 真实输出解析修复、错误语义与缓存写入审查。
- [`PHASE4_READINESS.md`](PHASE4_READINESS.md)：Phase 4 准入清单、真实 GitHub 资产事实与决策项。
- [`FULL_AUDIT_2026-11-11.md`](FULL_AUDIT_2026-11-11.md)：全量审查、缺陷修复与残留风险。
- [`PHASE4_STATUS.md`](PHASE4_STATUS.md)：Phase 4 实施状态、真实验证与已知限制。
- [`PHASE4_CODE_REVIEW.md`](PHASE4_CODE_REVIEW.md)：Phase 4 代码审查、缺陷修复与残留限制。
- [`PHASE5_STATUS.md`](PHASE5_STATUS.md)：Phase 5 安装、current 与 journal 实施状态。
- [`E2E_TESTING.md`](E2E_TESTING.md)：Windows/Linux e2e、远程服务器与 CI。
- [`FULL_AUDIT_2026-11-13.md`](FULL_AUDIT_2026-11-13.md)：本轮全量审查、二次检查与修复记录。
- [`VERSIONING.md`](VERSIONING.md)：SemVer 版本与发布规范。
- [`releases/release-v0.6.0.md`](releases/release-v0.6.0.md)：v0.6.0 release notes。
- [`USAGE.md`](USAGE.md)：首次使用详细教程。
- [`man/one-more-ffmpeg.1`](man/one-more-ffmpeg.1)：Linux man 页。
