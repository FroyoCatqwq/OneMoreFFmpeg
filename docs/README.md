# 文档索引与权威顺序

> 项目：OneMoreFFmpeg
> 状态：Phase 2 发现与校验已完成，Phase 3 尚未开始
> 日期：2026-11-11
> 更新：完成 Phase 1 代码审查修复与 Phase 2 真实发现/校验实现

---

## 1. 权威顺序

| 优先级 | 文档 | 权威范围 |
|---|---|---|
| 1 | [`PROJECT_PLAN.md`](PROJECT_PLAN.md) | 当前阶段、范围、步骤、DoD |
| 2 | [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md) | Go 依赖选择、版本新鲜度、拒绝理由 |
| 3 | [`CLI_SCHEMA.md`](CLI_SCHEMA.md) | 当前唯一对外契约：CLI JSON v1 |
| 4 | [`ERROR_CODES.md`](ERROR_CODES.md) | 错误码、警告码、退出码 |
| 5 | [`ARCHITECTURE.md`](ARCHITECTURE.md) | 模块边界与依赖方向 |
| 6 | [`CAPABILITIES.md`](CAPABILITIES.md) | FFmpeg 能力命令与输出模型 |
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
| 21 | [`schemas/README.md`](schemas/README.md) | CLI JSON 示例索引 |
| 22 | `README.md` / `docs/README.md` | 入口与索引，不承担规则裁决 |

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

---

## 3. 尚未完成

- 13 项能力探测（Phase 3）；
- provider resolver / download 实现（Phase 4）；
- install / env / rollback 实现（Phase 5）；
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
