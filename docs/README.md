# 文档索引与权威顺序

> 项目：OneMoreFFmpeg
> 状态：Phase 0 计划与契约文档，尚未开始实现
> 日期：2026-11-10
> 更新：依据 `DOCUMENTATION_AUDIT.md` 完成第二轮补充

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
| 17 | [`schemas/README.md`](schemas/README.md) | CLI JSON 示例索引 |
| 18 | `README.md` / `docs/README.md` | 入口与索引，不承担规则裁决 |

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

---

## 3. 尚未完成

- Go 代码；
- CLI 实现；
- `go.sum`；
- config schema 实现；
- provider resolver 实现；
- downloader；
- installer；
- PATH / 环境变量写入实现；
- 单元测试 / 集成测试 / CI；
- `THIRD_PARTY_NOTICES`；
- 第三方依赖许可证清单。

## 4. 未来，明确不属于当前 Phase

- 包管理器识别；
- Go API；
- JSON-RPC / gRPC / Connect RPC 插件；
- C ABI / c-shared；
- 私有制品库；
- 自动更新服务；
- GUI / TUI / CUI。

## 5. 审查报告

- [`DOCUMENTATION_AUDIT.md`](DOCUMENTATION_AUDIT.md)：第一轮审查、事实核实、P1/P2 问题与第二轮更新。
