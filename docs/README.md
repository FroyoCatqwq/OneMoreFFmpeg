# 文档索引与权威顺序

> 项目：OneMoreFFmpeg
> 状态：Phase 0 计划文档，尚未开始实现
> 日期：2026-11-10

---

## 1. 权威顺序

| 优先级 | 文档 | 权威范围 |
|---|---|---|
| 1 | [`PROJECT_PLAN.md`](PROJECT_PLAN.md) | 当前阶段、范围、步骤、DoD |
| 2 | [`ARCHITECTURE.md`](ARCHITECTURE.md) | 模块边界与依赖方向 |
| 3 | [`CAPABILITIES.md`](CAPABILITIES.md) | 能力命令与输出模型 |
| 4 | [`PROVIDERS.md`](PROVIDERS.md) | 来源、channel、release pin、自定义 URL |
| 5 | [`DETECTION_AND_VALIDATION.md`](DETECTION_AND_VALIDATION.md) | 发现与校验规则 |
| 6 | [`ENVIRONMENT.md`](ENVIRONMENT.md) | PATH 与环境变量规则 |
| 7 | [`API_AND_PLUGIN.md`](API_AND_PLUGIN.md) | Go API / CLI / 插件协议 |
| 8 | [`ROADMAP.md`](ROADMAP.md) | 阶段顺序 |
| 9 | [`TECHNICAL_NOTES.md`](TECHNICAL_NOTES.md) | 技术约束、风险、注意点 |
| 10 | `README.md` / `docs/README.md` | 入口与索引，不承担规则裁决 |

冲突时：

- 范围与阶段：以 `PROJECT_PLAN.md` 为准；
- 模块边界：以 `ARCHITECTURE.md` 为准；
- 具体能力命令：以 `CAPABILITIES.md` 为准；
- 下载源与 pin：以 `PROVIDERS.md` 为准；
- 发现与校验：以 `DETECTION_AND_VALIDATION.md` 为准；
- 环境变量：以 `ENVIRONMENT.md` 为准；
- 对外接口：以 `API_AND_PLUGIN.md` 为准。

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
| 插件 | JSON-RPC 2.0 over stdio 优先；C ABI 后续 |
| Go plugin 包 | 不使用；Windows 不支持且 ABI 脆弱 |
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
- JSON-RPC 插件；
- 单元测试 / 集成测试 / CI；
- LICENSE 选择。
