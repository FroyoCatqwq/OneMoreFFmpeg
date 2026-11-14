# OneMoreFFmpeg

> 实验性技术项目：统一管理 FFmpeg 的发现、下载、安装、部署、环境变量与能力检查。

仓库：<https://github.com/FroyoCatqwq/OneMoreFFmpeg>

状态：**Phase 1–6 已完成；已通过全量审查、二次修复与 Windows/Linux e2e；CLI 外部契约已冻结。**

> 已实现：`version` / `detect` / `validate` / `capabilities [--only]` / `provider list|add` / `resolve` / `download` / `install` / `uninstall` / `release list|install` / `journal list|rollback` / `env status|set|unset|rollback` / `doctor`。
> install/uninstall/rollback/provider add 默认 dry-run；download 与能力缓存会立即写文件。

> 🟢 **第一次用？先看 [`docs/HOWTO.md`](docs/HOWTO.md) 新手上手指南。**
> 黄金顺序：`install --apply` → `env set --apply` → **开新终端** → `ffmpeg` 可用。
> 「装完了却 `ffmpeg: command not found`」的解决办法也在里面。

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

> **OneMoreFFmpeg 管“FFmpeg 从哪里来、装到哪里、怎么验证、怎么被发现、支持什么能力”；当前不提供 Go SDK，也不提供插件接口。**

---

## 2. 当前范围

| 能力 | 状态 |
|---|---|
| 管理 FFmpeg 路径 | 已实现：发现、校验、安装、切换（`current`） |
| 下载/安装/部署 FFmpeg | 已实现（`download` / `install`，默认 dry-run） |
| 配置环境变量 / PATH | 已实现（`env set\|unset\|status\|rollback`，默认 dry-run） |
| 检查支持的能力列表 | 已实现（13 项，带缓存） |
| Windows + Linux | 已完成本地构建/测试/交叉编译与双平台 e2e |
| 自定义 FFmpeg URL | 已实现（`--url` + `provider add`） |
| 内置 BtbN 与 BnqDzj 来源 | 已实现（`provider list` / `resolve`） |
| 每日构建版 / 稳定版 | 已实现（channel + release pin） |
| 识别系统已有 FFmpeg | 已实现 |
| 全盘扫描 | 已实现，必须显式 `--scan` |
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

已实现 13 个 FFmpeg 能力查看命令：

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
| Go 版本 | `go 1.27`（允许 1.27.x 补丁版本） |
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
| [`docs/archive/README.md`](docs/README.md) | 文档索引与权威顺序 |
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
| [`docs/archive/DOCUMENTATION_AUDIT.md`](docs/archive/DOCUMENTATION_AUDIT.md) | 文档审查、事实核实、P1/P2 问题与改进顺序 |
| [`docs/archive/PHASE1_STATUS.md`](docs/archive/PHASE1_STATUS.md) | Phase 1 实施状态与验证结果 |
| [`docs/archive/PHASE1_CODE_REVIEW.md`](docs/archive/PHASE1_CODE_REVIEW.md) | Phase 1 代码审查、P0/P1 修复与残留 tech debt |
| [`docs/archive/PHASE2_STATUS.md`](docs/archive/PHASE2_STATUS.md) | Phase 2 实施状态与验证结果 |
| [`docs/archive/PHASE2_CODE_REVIEW.md`](docs/archive/PHASE2_CODE_REVIEW.md) | Phase 2 代码审查、缺陷修复与残留 tech debt |
| [`docs/archive/PHASE3_STATUS.md`](docs/archive/PHASE3_STATUS.md) | Phase 3 实施状态与验证结果 |
| [`docs/archive/PHASE3_CODE_REVIEW.md`](docs/archive/PHASE3_CODE_REVIEW.md) | Phase 3 代码审查、真实输出解析修复与残留 tech debt |
| [`docs/archive/PHASE4_READINESS.md`](docs/archive/PHASE4_READINESS.md) | Phase 4 准入清单、真实上游事实与决策项 |
| [`docs/archive/FULL_AUDIT_2026-11-11.md`](docs/archive/FULL_AUDIT_2026-11-11.md) | 全量审查、缺陷修复与残留风险 |
| [`docs/archive/PHASE4_STATUS.md`](docs/archive/PHASE4_STATUS.md) | Phase 4 实施状态、真实验证与已知限制 |
| [`docs/archive/PHASE4_CODE_REVIEW.md`](docs/archive/PHASE4_CODE_REVIEW.md) | Phase 4 代码审查、缺陷修复与残留限制 |
| [`docs/archive/PHASE5_STATUS.md`](docs/archive/PHASE5_STATUS.md) | Phase 5 安装、current 与 journal 实施状态 |
| [`docs/E2E_TESTING.md`](docs/E2E_TESTING.md) | Windows/Linux e2e、远程服务器与 CI 说明 |
| [`docs/USAGE.md`](docs/USAGE.md) | 首次使用详细教程 |
| [`docs/man/one-more-ffmpeg.1`](docs/man/one-more-ffmpeg.1) | Linux man 页（`man ./docs/man/one-more-ffmpeg.1`） |
| [`docs/archive/FULL_AUDIT_2026-11-13.md`](docs/archive/FULL_AUDIT_2026-11-13.md) | 2026-11-13 全量审查与二次修复记录 |
| [`docs/VERSIONING.md`](docs/VERSIONING.md) | SemVer 版本与发布规范 |
| [`docs/releases/release-v0.6.0.md`](docs/releases/release-v0.6.0.md) | v0.6.0 release notes |
| [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) | 第三方依赖许可证清单初稿 |

---

> 表中带 `docs/archive/` 的是**历史快照**（阶段状态、代码审查、全量审计记录），只记录当时发生了什么，不代表当前状态；判断当前能力请看 [`docs/README.md`](docs/README.md) §1 与代码。

## 5. 当前进度

- GitHub 仓库：已创建；
- `go.mod`：已创建，限制 Go 大版本 1.27；
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
- Phase 3 13 项能力检查、解析、缓存、`capabilities`：已完成；
- Phase 3 代码审查与真实 FFmpeg 输出解析修复：已完成；
- Phase 4 契约核对与准入清单：已完成；
- Phase 4 基础件 `internal/paths` / `internal/checksum` / `model.Release` / 配置网络段 / 契约字段校验：已完成；
- Phase 4 provider 注册表、GitHub Releases 客户端、`provider list`：已完成；
- Phase 4 决策 D1（stable pin）/ D5（nonfree 同意）/ D7（Job Object 推迟）：已定并落文；
- Phase 4 `provider add` / `resolve` / `download`：已完成，状态见 [`docs/archive/PHASE4_STATUS.md`](docs/archive/PHASE4_STATUS.md)；
- Phase 5 `install`（安全解压、版本化目录、install.json、current、journal）：已完成，状态见 [`docs/archive/PHASE5_STATUS.md`](docs/archive/PHASE5_STATUS.md)；
- 全量审查与缺陷修复：见 [`docs/archive/FULL_AUDIT_2026-11-11.md`](docs/archive/FULL_AUDIT_2026-11-11.md)；
- CLI：`version` / `detect` / `validate` / `capabilities` 已实现；
- 本地单测 / `go vet` / Windows+Linux 交叉编译：通过；
- 第三方许可证清单 / `THIRD_PARTY_NOTICES`：初稿完成，许可证文本已复制；
- 下载/安装/环境变量修改：已实现（Phase 4/5，写入路径默认 dry-run）；
- provider resolve / download：已实现；
- 包管理器识别（Scoop / winget / apt …）：未实现，明确推迟；
- Go API / 插件接口：未实现，明确推迟。

## 6. 开发命令

需要 Go 1.27：

```bash
go mod tidy
go build ./...
go test ./...
go vet ./...
go run ./cmd/one-more-ffmpeg version --json
go run ./cmd/one-more-ffmpeg detect --json --no-path --no-env --no-common
go run ./cmd/one-more-ffmpeg validate --dir C:/Tools/FFmpeg/bin --json
go run ./cmd/one-more-ffmpeg capabilities --dir C:/Tools/FFmpeg/bin --json --no-cache
```

双平台交叉编译：

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...
```

## 7. 许可证

GPL-3.0-only，见根目录 [`LICENSE`](LICENSE) 与 [`docs/LICENSE_DECISION.md`](docs/LICENSE_DECISION.md)。

GPL-3.0-only 只覆盖 OneMoreFFmpeg 自己的代码与文档，不自动覆盖第三方 FFmpeg 资产。nonfree FFmpeg 二进制仍不得进入 public GitHub Release。
