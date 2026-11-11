# OneMoreFFmpeg

> 实验性技术项目：统一管理 FFmpeg 的发现、下载、安装、部署、环境变量与能力检查。

仓库：<https://github.com/FroyoCatqwq/OneMoreFFmpeg>

状态：**Phase 2 真实发现与校验已完成，Phase 3 尚未开始。**

> `version` / `detect` / `validate` 已实现；其余 `one-more-ffmpeg ...` 命令仍是计划命令，尚未实现。

---

## 1. 项目定位

`OneMoreFFmpeg` 是一个独立的 Go CLI 工具，用来回答和解决下面几个问题：

- 机器上到底有哪些 `ffmpeg` / `ffprobe` / `ffplay`？
- 它们来自哪里：PATH、环境变量、指定目录、常见目录，还是全盘扫描？
- 它们能不能用？三件套是否同目录、同版本、可启动？
- 它们支持哪些 muxer / demuxer / decoder / encoder / filter / pix_fmt / layout / sample_fmt？
- 如何从 BtbN 或 BnqDzj 下载每日构建或稳定版？
- 如何安装、切换、回滚、配置 PATH / 环境变量？
- 如何让 `LiveArchiveEncoder` 与 `LiveArchiveEncoder-VideoDamage` 通过**调用本项目产物可执行文件**使用这些能力？

一句话：

> **OneMoreFFmpeg 管“FFmpeg 从哪里来、装到哪里、怎么验证、怎么被发现”；当前不提供 Go SDK，也不提供插件接口。**

---

## 2. 当前范围

首批只做简单但完整的闭环：

| 能力 | 状态 |
|---|---|
| 管理 FFmpeg 路径 | 计划中 |
| 下载/安装/部署 FFmpeg | 计划中 |
| 配置环境变量 / PATH | 计划中 |
| 检查支持的能力列表 | 计划中 |
| Windows + Linux | 计划中 |
| 自定义 FFmpeg URL | 计划中 |
| 内置 BtbN 与 BnqDzj 来源 | 计划中 |
| 每日构建版 / 稳定版 | 计划中 |
| 识别系统已有 FFmpeg | 计划中 |
| 全盘扫描 | 计划中，但必须显式启用 |
| 其他项目调用本项目产物可执行文件 | 当前唯一外部集成方式 |
| 写视频损伤代码 | 不做 |
| 自构建 FFmpeg | 不做，属于另一条未来路线 |

### 2.1 明确推迟到未来

| 能力 | 原因 |
|---|---|
| Scoop / Homebrew / winget / choco / apt 等包管理器识别 | 当前会显著增加平台差异和复杂度 |
| 对外开放 Go API | 程序功能尚不完善，API 会过早冻结 |
| 插件接口 / JSON-RPC / gRPC / C ABI | 当前会增加架构复杂度 |
| 其他项目直接 import Go module | 当前不开放；只允许调用 CLI |
| GUI / TUI / CUI | 不做 |

当前对外契约只有：

```text
CLI 参数 + stdout JSON / 文件产物 + stderr 日志 + 退出码
```

详见：

- [`docs/CLI_SCHEMA.md`](docs/CLI_SCHEMA.md)
- [`docs/ERROR_CODES.md`](docs/ERROR_CODES.md)
- [`docs/schemas/README.md`](docs/schemas/README.md)

当前支持的 FFmpeg 能力查看命令范围：

```text
-L
-license
-h <topic>
-version
-muxers
-demuxers
-devices
-decoders
-encoders
-filters
-pix_fmts
-layouts
-sample_fmts
```

详见 [`docs/CAPABILITIES.md`](docs/CAPABILITIES.md)。

---

## 3. 技术决策

| 项 | 决策 |
|---|---|
| 语言 | Go，仅 Go |
| Go 版本 | `go1.27.3`，CI 精确锁定 |
| 平台 | Windows x64 + Linux x86_64 |
| 构建 | `CGO_ENABLED=0` 优先，纯 Go 交叉编译 |
| CLI | `one-more-ffmpeg`；`omf` 为未来候选，当前未提供 |
| Go module | `github.com/FroyoCatqwq/OneMoreFFmpeg` |
| CLI 框架 | `github.com/urfave/cli/v3` |
| 配置 | TOML 1.1，`github.com/pelletier/go-toml/v2` |
| 日志 | `log/slog` + `github.com/lmittmann/tint` |
| CLI 输出 | `CLI_SCHEMA.md` + `docs/schemas/` |
| 错误码 | `ERROR_CODES.md` |
| 外部集成 | 只调用 CLI 可执行文件 |
| Go API / 插件 | 未来考虑，当前不实现 |
| 包管理器识别 | 未来考虑，当前不实现 |
| 安装源 | 内置 BtbN + BnqDzj，支持自定义 URL |
| nonfree | 只保存元数据与下载入口，不在本仓库/公开 Release 分发二进制 |
| 兼容性 | 实验项目，不为旧配置/旧 CLI 输出做长期兼容层 |

依赖评估详见 [`docs/GO_LIBRARY_EVALUATION.md`](docs/GO_LIBRARY_EVALUATION.md)。

本项目是实验性技术项目，允许大胆使用 Go 1.27 已 GA 的语法特性与架构实验；但**CLI 输出 schema、TOML 配置 schema、错误码、安装目录布局必须显式版本化**。

---

## 4. 文档

| 文档 | 内容 |
|---|---|
| [`docs/README.md`](docs/README.md) | 文档索引与权威顺序 |
| [`docs/PROJECT_PLAN.md`](docs/PROJECT_PLAN.md) | 项目计划、阶段、步骤、DoD、技术点 |
| [`docs/GO_LIBRARY_EVALUATION.md`](docs/GO_LIBRARY_EVALUATION.md) | Go 依赖库评估与采用/候选/拒绝清单 |
| [`docs/CLI_SCHEMA.md`](docs/CLI_SCHEMA.md) | 当前唯一外部契约：CLI JSON v1 |
| [`docs/schemas/README.md`](docs/schemas/README.md) | CLI JSON 示例索引 |
| [`docs/ERROR_CODES.md`](docs/ERROR_CODES.md) | 错误码、警告码、退出码注册表 |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | 模块划分、依赖方向、CLI-first 架构 |
| [`docs/CAPABILITIES.md`](docs/CAPABILITIES.md) | FFmpeg 能力命令、解析与缓存 |
| [`docs/PROVIDERS.md`](docs/PROVIDERS.md) | BtbN / BnqDzj / 自定义 URL / daily / stable |
| [`docs/PROVIDER_RESOLVER.md`](docs/PROVIDER_RESOLVER.md) | GitHub API、限流、重试、资产匹配、checksums、release pin |
| [`docs/DETECTION_AND_VALIDATION.md`](docs/DETECTION_AND_VALIDATION.md) | PATH / 环境变量 / 指定目录 / 常见目录 / 全盘扫描 / 可用性校验 |
| [`docs/ENVIRONMENT.md`](docs/ENVIRONMENT.md) | PATH、环境变量、current 链接、回滚 |
| [`docs/PLATFORM_ENVIRONMENT.md`](docs/PLATFORM_ENVIRONMENT.md) | Windows 注册表 / UAC / junction、Linux shim / shell rc / journal |
| [`docs/API_AND_PLUGIN.md`](docs/API_AND_PLUGIN.md) | 未来 API/插件设计；当前不实现 |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Phase 0–7 与 M0 门禁 |
| [`docs/TECHNICAL_NOTES.md`](docs/TECHNICAL_NOTES.md) | Go 1.27、并发、安全、测试、注意点 |
| [`docs/LICENSE_DECISION.md`](docs/LICENSE_DECISION.md) | 许可证决策记录（GPL-3.0-only） |
| [`docs/DOCUMENTATION_AUDIT.md`](docs/DOCUMENTATION_AUDIT.md) | 文档审查、事实核实、P1/P2 问题与改进顺序 |
| [`docs/PHASE1_STATUS.md`](docs/PHASE1_STATUS.md) | Phase 1 实施状态与验证结果 |
| [`docs/PHASE1_CODE_REVIEW.md`](docs/PHASE1_CODE_REVIEW.md) | Phase 1 代码审查、P0/P1 修复与残留 tech debt |
| [`docs/PHASE2_STATUS.md`](docs/PHASE2_STATUS.md) | Phase 2 实施状态与验证结果 |
| [`docs/PHASE2_CODE_REVIEW.md`](docs/PHASE2_CODE_REVIEW.md) | Phase 2 代码审查、缺陷修复与残留 tech debt |
| [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) | 第三方依赖许可证清单初稿 |

---

## 5. 当前进度

- GitHub 仓库：已创建；
- `go.mod`：已创建，锁定 Go 1.27.3；
- `go.sum`：已生成；
- 项目计划与文档：已写入；
- Go 依赖库评估：已完成；
- CLI JSON 契约与示例：已写入；
- 错误码注册表：已写入，并在 Go 代码中实现；
- 平台环境变量细则：已写入；
- Provider resolver 细则：已写入；
- 许可证：GPL-3.0-only，根目录 `LICENSE` 已添加；
- Phase 1 Go 骨架：已完成；
- Phase 1 代码审查与 P0/P1 修复：已完成；
- Phase 2 真实 FFmpeg 发现、三件套校验、`detect` / `validate`：已完成；
- CLI：`version` / `detect` / `validate` 已实现；
- 本地单测 / `go vet` / Windows+Linux 交叉编译：通过；
- 第三方许可证清单 / `THIRD_PARTY_NOTICES`：初稿完成，许可证文本已复制；
- 下载/安装/环境变量修改：未实现；
- 能力探测、provider resolve：未实现；
- 包管理器识别：未实现，未来考虑；
- Go API / 插件接口：未实现，未来考虑；
- GitHub Actions / 发布 CI：未建立。

## 6. Phase 1 开发命令

需要 Go 1.27.3：

```bash
go mod tidy
go build ./...
go test ./...
go vet ./...
go run ./cmd/one-more-ffmpeg version --json
go run ./cmd/one-more-ffmpeg detect --json --no-path --no-env --no-common
go run ./cmd/one-more-ffmpeg validate --dir C:/Tools/FFmpeg/bin --json
```

双平台交叉编译：

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...
```

## 7. 许可证

GPL-3.0-only，见根目录 [`LICENSE`](LICENSE) 与 [`docs/LICENSE_DECISION.md`](docs/LICENSE_DECISION.md)。

GPL-3.0-only 只覆盖 OneMoreFFmpeg 自己的代码与文档，不自动覆盖第三方 FFmpeg 资产。nonfree FFmpeg 二进制仍不得进入 public GitHub Release。
