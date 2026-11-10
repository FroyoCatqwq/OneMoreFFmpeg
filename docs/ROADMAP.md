# 路线图

> 状态：Phase 0 计划
> 原则：先只读、后写入；先简单闭环、后扩展生态

---

## Phase 0：文档与项目计划

状态：已完成文档初稿。

内容：

- 新建 GitHub repo；
- 写 README / 项目计划 / 架构 / 能力 / provider / 检测 / 环境 / API / 技术注意点；
- `go.mod` 锁定 Go 1.27.3；
- 不写 Go 业务代码。

---

## Phase 1：仓库骨架

内容：

- `cmd/one-more-ffmpeg`；
- `pkg/onemore`；
- `internal/core`；
- `internal/app`；
- `internal/platform`；
- `version` 命令；
- `--help`；
- `detect --dry-run` 占位；
- fake ffmpeg 测试框架；
- Windows/Linux 交叉编译。

退出条件：

- `go test ./...`；
- `go vet ./...`；
- `CGO_ENABLED=0` 双平台编译；
- 不写 PATH / 不下载 / 不安装。

---

## Phase 2：发现与校验

内容：

- PATH；
- 环境变量；
- Scoop；
- Homebrew；
- 指定目录；
- 显式全盘扫描；
- 三件套校验；
- `detect --json`；
- `validate --json`。

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
- C ABI；
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
- 安全下载；
- `artifact` 记录。

退出条件：

- 下载产物可校验；
- 不自动安装；
- 不修改 PATH；
- nonfree 不进 public artifact。

---

## Phase 5：Install / Env / Rollback

内容：

- zip / tar.xz 安全解压；
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

## Phase 6：API / Plugin

内容：

- Go API v1；
- CLI JSON v1；
- JSON-RPC stdio；
- plugin manifest；
- 权限；
- LAE / VideoDamage 集成示例；
- C ABI ADR。

退出条件：

- 跨语言调用不依赖 Go 内部类型；
- 插件能力/权限可审计；
- C ABI 明确采用或拒绝。

---

## Phase 7：质量 / CI / Release

内容：

- GitHub Actions；
- Windows + Linux；
- Go 1.27.3；
- `gofmt` / `go vet` / `go test`；
- fuzz URL / parser；
- 真实资产集成测试；
- checksum / provenance；
- 公开 Release 不含 nonfree 二进制。

---

## 未来能力

- C ABI / c-shared；
- GUI/TUI 明确不做；
- FFmpeg 自构建明确不做；
- 自动更新服务；
- 私有制品库；
- 更多包管理器：winget、choco、apt、dnf、pacman；
- 生成 VideoDamage C1 锁文件的正式适配器；
- 生成 LAE v8 provider 配置。
