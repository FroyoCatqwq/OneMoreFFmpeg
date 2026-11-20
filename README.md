# OneMoreFFmpeg

> 统一管理 FFmpeg 的发现、校验、下载、安装、环境变量与能力检查的 Go CLI。

仓库：<https://github.com/FroyoCatqwq/OneMoreFFmpeg> · 许可证：GPL-3.0-only

**状态：v1.1.2-pre4** — Phase 1–7 全部完成，CLI 外部契约冻结；全项目审计后 bugfix；Windows + Linux + Docker 全新环境 `go test` / e2e 全绿，
GitHub Actions 提供质量门禁与 tag 发布流水线。产物见 [Releases](https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases)。

> **仅 v1.1.2-pre4 受支持**：旧版本（v1.0.x 及更早）不再维护，请始终使用当前版本；仓库内 `docs/releases/` 的旧版说明仅作历史记录。

🟢 **第一次用？先看 [`新手上手指南`](docs/guides/HOWTO.md)。黄金顺序：`resolve`（默认官方 latest）→ `download/install --apply` → `env set --apply` → 开新终端 → `ffmpeg` 可用。**

默认下载 [FFmpeg-Builds](https://github.com/FFmpeg-Builds/FFmpeg-Builds) 的 **master 最新快照 / gpl / static / x86_64**；稳定版（n*）暂不支持；系统要求请查看[上游 README](https://github.com/FFmpeg-Builds/FFmpeg-Builds#readme)。

安装：从 [Releases](https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases) 下载对应平台二进制并校验 SHA-256（本工具不提供 Scoop/dnf/RPM 包）。

---

## 1. 它解决什么

机器上有哪些 `ffmpeg`/`ffprobe`/`ffplay`、来自哪里、能否用、支持哪些能力；如何从 FFmpeg-Builds 官方最新快照（或你自己的 URL）下载并校验 FFmpeg，
如何安装、切换、回滚、配置 PATH；以及让下游项目（LiveArchiveEncoder 等）通过**调用本 CLI 可执行文件**使用这些能力。

一句话：**OneMoreFFmpeg 管“FFmpeg 从哪里来、装到哪里、怎么验证、怎么被发现、支持什么能力”。不提供 Go SDK 或插件接口。**

## 2. 能力与边界

已实现：发现（PATH / 环境变量 / 指定目录 / 常见目录 / 显式 `--scan` 全盘扫描）、三件套校验、13 项能力检查（带缓存）、
默认 FFmpeg-Builds 官方 latest 快照 + 可选自定义 URL、下载（SHA-256 校验）、
安装（安全解压、版本化目录、`current`、journal）、`uninstall`、`env status|set|unset|rollback`、
`journal list|rollback`、`doctor`，共 18 个冻结命令（含 v1.1 新增的 `update`）。写入类命令默认 dry-run，需 `--apply`；`download` 与能力缓存会立即写文件。

明确不做：包管理器识别（Scoop/winget/apt 管理第三方 FFmpeg）、Go API / 插件 / JSON-RPC / C ABI、GUI/TUI、FFmpeg 自构建、私有制品库、**后台自动更新 / 静默替换 / 常驻更新服务**（v1.1 起已支持用户主动执行的 `update` 检查与 `update --apply` 自更新，二者不等同）。

对外契约只有 **CLI 参数 + stdout JSON（或文件产物）+ stderr 日志 + 退出码**：

- [`CLI_SCHEMA.md`](docs/reference/CLI_SCHEMA.md) — 唯一契约：JSON v1，18 命令 / 17 个 `/v1` schema，`freeze_test.go` 守护；无内置 provider，只支持自定义 URL；
- [`ERROR_CODES.md`](docs/reference/ERROR_CODES.md) — 错误码 / 警告码 / 退出码（0/1/2/130）注册表；
- 字段示例见 [`reference/schemas/`](docs/reference/schemas/README.md)。

## 3. 技术决策

| 项 | 决策 |
|---|---|
| 语言 / 工具链 | Go only；`go.mod` 声明 `go 1.27`，CI 采用 1.27.x 补丁版本，`CGO_ENABLED=0` 交叉编译 |
| 平台 | Windows x64 + Linux x86_64 |
| CLI | `one-more-ffmpeg`（不提供 `omf` 内置别名；HOWTO 中 `omf` 是用户自定义 alias） |
| 组件 | `urfave/cli/v3`、`pelletier/go-toml/v2`（TOML 1.1）、`log/slog` + `lmittmann/tint` |
| 安装源 | 默认 official（FFmpeg-Builds latest，gpl/static/x86_64）+ 可选自定义 URL（http/https）；nonfree 仅本地使用 |
| 外部集成 | 只调用 CLI 可执行文件，见 [`集成指南`](docs/guides/INTEGRATION.md) |
| 版本规范 | SemVer；schema/错误码/目录布局显式版本化，见 [`VERSIONING.md`](docs/reference/VERSIONING.md) |

## 4. 文档

完整索引见 [`docs/README.md`](docs/README.md)。常用入口：

- 上手：[`HOWTO.md`](docs/guides/HOWTO.md) · [`USAGE.md`](docs/guides/USAGE.md) · [`MIGRATION.md`](docs/guides/MIGRATION.md)
- 契约：[`CLI_SCHEMA.md`](docs/reference/CLI_SCHEMA.md) · [`ERROR_CODES.md`](docs/reference/ERROR_CODES.md) · [`CAPABILITIES.md`](docs/reference/CAPABILITIES.md)
- 开发：[`ARCHITECTURE.md`](docs/development/ARCHITECTURE.md) · [`E2E_TESTING.md`](docs/development/E2E_TESTING.md) · [`development/adr/`](docs/development/adr/)
- 发布：[`CHANGELOG.md`](docs/releases/CHANGELOG.md) · [`release-v1.1.2-pre4.md`](docs/releases/release-v1.1.2-pre4.md)（旧版本 Release 与历史快照已撤回）

## 5. 开发命令

需要 Go 1.27（本机可用 `mise exec -- ...`）：

```bash
go mod tidy && go build ./... && go test ./... && go vet ./...
go run ./cmd/one-more-ffmpeg version --json
go run ./cmd/one-more-ffmpeg detect --json --no-path --no-env --no-common
go run ./cmd/one-more-ffmpeg capabilities --dir C:/Tools/FFmpeg/bin --json --no-cache
```

本地质量门禁与 e2e：`mise run quality`、`mise run e2e-all`；Docker 全新环境：`docker compose up --build --abort-on-container-exit --exit-code-from e2e-fresh`。
双平台交叉编译：`CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...`（Linux 同理换 `GOOS=linux`）。

## 6. 许可证

GPL-3.0-only，见 [`LICENSE`](LICENSE) 与 [`许可证决策`](docs/development/adr/0003-license.md)。
该许可证只覆盖 OneMoreFFmpeg 自身代码与文档，不自动覆盖第三方 FFmpeg 资产；依赖清单见 [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)。
