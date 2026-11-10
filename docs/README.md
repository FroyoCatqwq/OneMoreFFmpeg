# 文档索引与权威顺序

> 项目：OneMoreFFmpeg
> 状态：Phase 0 计划文档，尚未开始实现
> 日期：2026-11-10

---

## 1. 权威顺序

| 优先级 | 文档 | 权威范围 |
|---|---|---|
| 1 | [`PROJECT_PLAN.md`](PROJECT_PLAN.md) | 当前阶段、范围、步骤、DoD |
| 2 | [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md) | Go 依赖选择、版本新鲜度、拒绝理由 |
| 3 | [`ARCHITECTURE.md`](ARCHITECTURE.md) | 模块边界与依赖方向 |
| 4 | [`CAPABILITIES.md`](CAPABILITIES.md) | 能力命令与输出模型 |
| 5 | [`PROVIDERS.md`](PROVIDERS.md) | 来源、channel、release pin、自定义 URL |
| 6 | [`DETECTION_AND_VALIDATION.md`](DETECTION_AND_VALIDATION.md) | 发现与校验规则 |
| 7 | [`ENVIRONMENT.md`](ENVIRONMENT.md) | PATH 与环境变量规则 |
| 8 | [`API_AND_PLUGIN.md`](API_AND_PLUGIN.md) | 未来 API/插件设计；当前不实现 |
| 9 | [`ROADMAP.md`](ROADMAP.md) | 阶段顺序 |
| 10 | [`TECHNICAL_NOTES.md`](TECHNICAL_NOTES.md) | 技术约束、风险、注意点 |
| 11 | `README.md` / `docs/README.md` | 入口与索引，不承担规则裁决 |

冲突时：

- 范围与阶段：以 `PROJECT_PLAN.md` 为准；
- 依赖库：以 `GO_LIBRARY_EVALUATION.md` 为准；
- 模块边界：以 `ARCHITECTURE.md` 为准；
- 具体能力命令：以 `CAPABILITIES.md` 为准；
- 下载源与 pin：以 `PROVIDERS.md` 为准；
- 发现与校验：以 `DETECTION_AND_VALIDATION.md` 为准；
- 环境变量：以 `ENVIRONMENT.md` 为准；
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
| 能力命令 | 13 项，见 `CAPABILITIES.md` |
| 非公开命令 | `-buildconf` 仅用于内部安装校验，v1 不进入公开 capability 集 |
| 包管理器识别 | 当前取消；Scoop / Homebrew / winget 等纳入未来 |
| 对外开放 Go API | 当前取消；纳入未来 |
| 插件 / JSON-RPC / C ABI | 当前取消；纳入未来 |
| 外部集成 | 其他项目只调用本项目的 CLI 可执行文件 |
| CLI 框架 | `urfave/cli/v3` |
| TOML | `pelletier/go-toml/v2`，支持 TOML 1.1.0 |
| 日志 | `log/slog` + `lmittmann/tint` |
| 测试 | `stretchr/testify` |
| nonfree | 元数据可公开，二进制不可随公开 Release 分发 |

---

## 3. 尚未完成

- Go 代码；
- CLI；
- config schema 实现；
- provider resolver；
- downloader；
- installer；
- PATH / 环境变量写入；
- 单元测试 / 集成测试 / CI；
- LICENSE 选择。

## 4. 未来，明确不属于当前 Phase

- 包管理器识别；
- Go API；
- JSON-RPC / gRPC / Connect RPC 插件；
- C ABI / c-shared；
- 私有制品库；
- 自动更新服务；
- GUI / TUI / CUI。
