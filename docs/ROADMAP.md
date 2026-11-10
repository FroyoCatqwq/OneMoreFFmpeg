# 路线图

> 状态：Phase 0 计划
> 原则：先只读、后写入；先 CLI-first，后考虑 API/插件/包管理器

---

## Phase 0：文档与项目计划

状态：已完成文档初稿。

内容：

- 新建 GitHub repo；
- 写 README / 项目计划 / 架构 / 能力 / provider / 检测 / 环境 / 未来 API/插件 / 技术注意点；
- 新增 Go 依赖库评估；
- `go.mod` 锁定 Go 1.27.3；
- 不写 Go 业务代码。

---

## Phase 1：仓库骨架

内容：

- `cmd/one-more-ffmpeg`；
- `internal/cli`；
- `internal/app`；
- `internal/core`；
- `internal/platform`；
- `urfave/cli/v3`；
- `pelletier/go-toml/v2` strict mode；
- `log/slog` + `lmittmann/tint`；
- `version` 命令；
- `--help`；
- `detect --dry-run` 占位；
- fake ffmpeg 测试框架；
- `stretchr/testify`；
- Windows/Linux 交叉编译。

退出条件：

- `go test ./...`；
- `go vet ./...`；
- `CGO_ENABLED=0` 双平台编译；
- 不写 PATH / 不下载 / 不安装；
- 不引入包管理器适配；
- 不引入 Go API / 插件依赖。

---

## Phase 2：发现与校验

内容：

- PATH；
- 环境变量；
- 指定目录；
- 常见目录；
- 显式全盘扫描；
- 三件套校验；
- `detect --json`；
- `validate --json`。

当前不做：

```text
Scoop
Homebrew
winget
Chocolatey
apt / dnf / pacman
snap / flatpak 专门适配
```

退出条件：M0 前半通过。

---

## Phase 3：能力检查

内容：

- 用户列出的 13 个 FFmpeg 参数；
- 结构化 JSON；
- raw stdout hash；
- 能力缓存；
- `-h <topic>` allowlist；
- `capabilities --json`。

退出条件：M0 全通过。

---

## M0：只读硬门禁

```text
发现 + 校验 + 能力检查 + provider resolve（不下载）
```

通过前不做：

- download；
- install；
- PATH 写入；
- 环境变量写入；
- 包管理器识别；
- Go API；
- JSON-RPC / gRPC / C ABI；
- 公开 Release。

---

## Phase 4：Provider / Resolve / Download

内容：

- BtbN；
- BnqDzj nonfree；
- 自定义 URL；
- stable / daily；
- release pin；
- checksums；
- `net/http` + `sethvargo/go-retry`；
- `artifact.json` 记录。

退出条件：

- 下载产物可校验；
- 不自动安装；
- 不修改 PATH；
- nonfree 不进 public artifact。

---

## Phase 5：Install / Env / Rollback

内容：

- `klauspost/compress` + `ulikunitz/xz` 安全解压；
- 版本化目录；
- `current`；
- Windows user PATH；
- Linux shim；
- journal；
- rollback；
- `doctor`。

退出条件：

- dry-run 默认；
- 失败可回滚；
- 不删除非本项目文件；
- 不覆盖运行中 exe。

---

## Phase 6：CLI 外部调用契约

内容：

- CLI 子命令稳定化；
- `--json` schema 稳定化；
- stdout/stderr 分离；
- 退出码稳定化；
- 文件产物稳定化；
- `version --json`；
- `doctor --json`；
- LAE / VideoDamage 调用 CLI 的示例；
- 明确不提供 Go API / JSON-RPC / C ABI。

退出条件：

- 外部项目不需要 import OneMoreFFmpeg；
- 外部项目不需要链接 Go runtime；
- 所有机器结果在 stdout JSON 或明确文件中；
- CLI schema 主版本变化即拒绝。

---

## Phase 7：质量 / CI / Release

内容：

- GitHub Actions；
- Windows + Linux；
- Go 1.27.3；
- `gofmt` / `go vet` / `go test`；
- `govulncheck`；
- fuzz URL / parser；
- 真实资产集成测试；
- checksum / provenance；
- 公开 Release 不含 nonfree 二进制。

---

## 未来能力，明确不属于当前 Phase

- 包管理器识别：Scoop / Homebrew / winget / choco / apt / dnf / pacman；
- 对外开放 Go API；
- `pkg/onemore` 公共 SDK；
- JSON-RPC / gRPC / Connect RPC 插件；
- C ABI / c-shared；
- 私有制品库；
- 自动更新服务；
- GUI / TUI / CUI；
- FFmpeg 自构建；
- 生成 VideoDamage C1 锁文件的正式适配器可作为 CLI 产物扩展，但不是插件 API。
