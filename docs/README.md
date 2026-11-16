# 文档索引

> OneMoreFFmpeg — 统一管理 FFmpeg 的发现、下载、安装、环境与能力检查的 CLI。
> 当前版本 **v1.0.0**；Phase 1–7 全部完成，CLI 外部契约冻结。
> 仓库：<https://github.com/FroyoCatqwq/OneMoreFFmpeg>

本目录按用途分为五类；`archive/` 中的历史快照不代表当前状态。

## 1. 上手与使用（guides/）

| 文档 | 用途 |
|---|---|
| [`guides/INSTALL.md`](guides/INSTALL.md) | 安装：Scoop bucket / RHEL 9+ dnf RPM / 手动二进制与校验 |
| [`guides/HOWTO.md`](guides/HOWTO.md) | 新手上手：下载、首次安装、开新终端、`ffmpeg` 可用、排错 |
| [`guides/USAGE.md`](guides/USAGE.md) | 详细教程：provider、release、install、env、journal 的完整流程 |
| [`guides/INTEGRATION.md`](guides/INTEGRATION.md) | 外部项目如何调用 CLI（Python / PowerShell 示例） |
| [`guides/MIGRATION.md`](guides/MIGRATION.md) | 版本间数据目录 / journal / 配置的迁移与回滚说明 |

## 2. 对外契约与参考（reference/）

冲突时以本节文档为准；实现细节以代码为准。

| 文档 | 权威范围 |
|---|---|
| [`reference/CLI_SCHEMA.md`](reference/CLI_SCHEMA.md) | 唯一对外契约：CLI JSON v1（命令集 / schema / 字段） |
| [`reference/ERROR_CODES.md`](reference/ERROR_CODES.md) | 错误码 / 警告码 / 退出码映射 |
| [`reference/CAPABILITIES.md`](reference/CAPABILITIES.md) | 13 项能力检查与输出模型 |
| [`reference/PROVIDERS.md`](reference/PROVIDERS.md) / [`PROVIDER_RESOLVER.md`](reference/PROVIDER_RESOLVER.md) | 来源、channel、release pin、GitHub 解析与校验 |
| [`reference/DETECTION_AND_VALIDATION.md`](reference/DETECTION_AND_VALIDATION.md) | 发现与校验规则 |
| [`reference/ENVIRONMENT.md`](reference/ENVIRONMENT.md) / [`PLATFORM_ENVIRONMENT.md`](reference/PLATFORM_ENVIRONMENT.md) | PATH / 环境变量 / current / journal 的跨平台细则 |
| [`reference/SYSTEM_REQUIREMENTS.md`](reference/SYSTEM_REQUIREMENTS.md) | 系统要求、权限与提权、事务锁 |
| [`reference/VERSIONING.md`](reference/VERSIONING.md) | SemVer 版本与发布规范 |
| [`reference/schemas/`](reference/schemas/README.md) | 18 个 data schema 的 golden 示例 |
| [`reference/man/one-more-ffmpeg.1`](reference/man/one-more-ffmpeg.1) | Linux man 页 |

## 3. 开发与决策（development/）

| 文档 | 用途 |
|---|---|
| [`development/ARCHITECTURE.md`](development/ARCHITECTURE.md) | 模块划分、依赖方向、CLI-first 架构 |
| [`development/TECHNICAL_NOTES.md`](development/TECHNICAL_NOTES.md) | 并发、安全、测试与已知取舍 |
| [`development/E2E_TESTING.md`](development/E2E_TESTING.md) | 本机 / 远程 / Docker / CI 测试方式 |
| [`development/adr/`](development/adr/) | 架构决策记录（依赖、许可证、provider pin 与 nonfree） |

## 4. 发布与历史

- [`releases/`](releases/) — 各版本 release notes 与 [`CHANGELOG.md`](releases/CHANGELOG.md)；
- [`archive/`](archive/README.md) — 阶段状态、审计与历史计划快照，**不维护**，仅作历史证据。

## 5. 决策快照

| 项 | 值 |
|---|---|
| 语言 / 工具链 | Go only；`go.mod` 声明 `go 1.27`，工具链 `go1.27.4` |
| 平台 | Windows x64 + Linux x86_64；`CGO_ENABLED=0` 纯 Go 交叉编译 |
| CLI | `one-more-ffmpeg`（不提供 `omf` 内置别名） |
| 内置来源 | BtbN + BnqDzj nonfree；支持自定义 URL（强制 SHA-256 校验） |
| channel | `stable` / `daily`，解析为 release tag + asset SHA-256 |
| 能力命令 | 13 个 FFmpeg 参数 / 能力入口 |
| 对外契约 | 18 命令 / 17 个 `/v1` data schema / 退出码 0/1/2/130，`internal/contracts/freeze_test.go` 守护 |
| 外部集成 | 只调用 CLI 可执行文件；不提供 Go API / 插件 / JSON-RPC / C ABI |
| 明确不做 | 包管理器识别、GUI/TUI、FFmpeg 自构建、私有制品库、自动更新服务 |
| 许可证 | GPL-3.0-only；nonfree FFmpeg 元数据可公开，二进制不可随公开 Release 分发 |
| 质量门禁 | gofmt / vet / test / govulncheck / fuzz + Windows/Linux e2e + Docker 全新环境（见 CI） |
