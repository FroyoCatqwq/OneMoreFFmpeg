# 技术点与注意点

> 状态：Phase 0 设计
> Go：1.27.3
> 平台：Windows x64 + Linux x86_64
> 当前范围：CLI-first；不做包管理器识别；不做 Go API / 插件

---

## 1. Go 1.27.3

已核对 `golang/go` 标签，Go 1.27 稳定标签包含：

```text
go1.27.0
go1.27.1
go1.27.2
go1.27.3
```

当前文档与 `go.mod` 锁定 `go1.27.3`。
若后续发布 1.27.4+，需要写 ADR 后更新，不静默升级。

本阶段只提交文档和 `go.mod`，不提交 Go 代码；编译验证在 Phase 1 进行。

---

## 2. 可以大胆使用的 Go 特性

- 泛型；
- `iter.Seq` / range-over-func 风格迭代器；
- `log/slog` 结构化日志；
- `errors.Join`；
- `context.Context` / `context.AfterFunc`；
- `io/fs` 抽象；
- `slices` / `maps` / `cmp`；
- `debug/buildinfo`；
- `net/http` 客户端超时与重定向控制；
- `encoding/json` 稳定输出。

限制：

- 不用未 GA 的实验语言特性；
- CLI 输出 schema / TOML schema 不暴露实验泛型技巧；
- 不用反射做配置解析主路径；
- 不用 cgo，除非系统 API 确实无法替代，并写 ADR；
- 不用 Go `plugin` 包；
- 当前不提供公共 Go API，因此 `internal/*` 不对外承诺。

---

## 3. 依赖策略

原则：

> 标准库为主 + 少量 6 个月内活跃的第三方库。

硬标准：

```text
2026-05-10 之后有新版本
持续维护
纯 Go 优先
Windows/Linux 可用
许可证清晰
```

Phase 1 建议采用：

| 类别 | 库 |
|---|---|
| CLI | `github.com/urfave/cli/v3` |
| TOML 1.1 | `github.com/pelletier/go-toml/v2` |
| 日志 | `log/slog` + `github.com/lmittmann/tint` |
| 并发 | `golang.org/x/sync` |
| 系统 API | `golang.org/x/sys` / `golang.org/x/term` |
| TTY | `github.com/mattn/go-isatty` |
| 压缩 | `github.com/klauspost/compress` |
| xz | `github.com/ulikunitz/xz` |
| HTTP retry | `github.com/sethvargo/go-retry` |
| 限流 | `golang.org/x/time/rate` |
| 可读大小 | `github.com/dustin/go-humanize` |
| 测试 | `github.com/stretchr/testify` |

完整版本、时间与拒绝理由见 [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md)。

当前不引入：

- Go API 框架；
- JSON-RPC / gRPC / Connect RPC；
- C ABI / c-shared；
- `hashicorp/go-plugin`；
- 包管理器适配库；
- 消息系统 / Redis / NATS；
- 大型 Web 框架。

所有依赖写进 `go.sum` 并跑 `go vet` / `govulncheck` 评估。

---

## 4. 进程执行

统一 `ffexec`：

- `exec.CommandContext`；
- 参数数组，不拼字符串；
- 不调用 `cmd.exe` / `/bin/sh -c`；
- stdout/stderr 分别 capture；
- `io.LimitReader`；
- 超时取消；
- Windows 预留 Job Object 杀进程树；
- Linux 设置 process group，`kill(-pgid)`；
- 记录 exit code、duration、stdout hash、stderr hash；
- 不把 stderr 全部写入日志，防止噪声。

能力命令和校验命令都不得绕过此执行器。

---

## 5. 下载安全

- 只允许 `https://`；
- 禁止明文 HTTP；
- 禁止 `file://`；
- 限制重定向次数；
- 限制下载大小；
- 使用 checksums.sha256；
- direct URL 必须显式提供 SHA-256；
- 先下载到临时文件，再原子移动；
- 下载完成前不执行、不解压；
- token 不写日志；
- 不执行下载包里的 `.ps1` / `.sh` / installer；
- `net/http` + `sethvargo/go-retry` 做重试，不引入过旧 retry 库。

---

## 6. 解压安全

- zip：防 zip slip；
- tar.xz：防绝对路径、`..`、symlink 逃逸；
- 拒绝设备文件、FIFO、setuid 位；
- 限制总解压大小；
- 限制文件数量；
- 保留可执行位仅用于 Linux；
- Windows 不信任压缩包里的 ACL；
- 解压目标只能是本项目管理的 temp / apps 子目录；
- 解压后重新校验三件套 SHA-256。

---

## 7. 全盘扫描注意点

- 默认关闭；
- 显式 `--scan`；
- 显式 roots；
- 固定盘优先；
- 排除 `/proc` / `/sys` / `/dev` / 网络盘 / 可移动盘；
- symlink loop 检测；
- 权限错误聚合而不是终止；
- 最大深度、最大文件数、总超时；
- 只读，不打开文件内容，除非路径名匹配；
- 结果按 hash 去重；
- 日志显示扫描进度和取消原因；
- 当前不识别包管理器 shim / prefix，避免额外复杂度。

---

## 8. Windows 注意点

- `ffmpeg.exe` / `ffprobe.exe` / `ffplay.exe`；
- 路径可能有空格、UNC、长路径；
- 用户 PATH vs 系统 PATH；
- `REG_EXPAND_SZ` vs `REG_SZ`；
- 修改 PATH 后广播 `WM_SETTINGCHANGE`；
- 当前进程不会自动看到新 PATH；
- 正在运行的 exe 不能覆盖；
- 当前不解析 Scoop shim；
- 当前不调用 `scoop` / `winget` / `choco`；
- 不默认写 machine PATH；
- 不调用 UAC 静默提权；
- 包管理器识别属于未来 ADR。

---

## 9. Linux 注意点

- 可执行位；
- XDG 目录；
- `~/.local/bin` 是否已在 PATH；
- 不写 `/etc/profile`；
- 不写 root 拥有文件；
- `current` 用 symlink 或 directory pointer；
- 不覆盖 `/usr/bin/ffmpeg`；
- 当前不解析 Homebrew prefix；
- 当前不调用 `brew` / `apt` / `dnf` / `pacman`；
- 包管理器识别属于未来 ADR；
- snap / flatpak 路径未来再评估。

---

## 10. 能力解析注意点

- FFmpeg 输出不是稳定 API；
- 不同版本列宽、flags、描述可能变化；
- parser 必须容错；
- 保留 raw stdout hash；
- 不按 provider 名称推断能力；
- 不把 `-buildconf` 混入 v1 公开能力集合；
- `-h <topic>` 必须 allowlist；
- 未知行进入 warnings，不让整个命令失败；
- 缓存必须包含二进制 hash 与 schema 版本。

---

## 11. 环境变量注意点

- 默认 dry-run；
- 写入前 journal；
- 失败 rollback；
- 不覆盖未知值；
- 不删除非本项目创建项；
- 不静默改 PATH 顺序；
- Windows 与 Linux 用不同策略；
- `ONE_MORE_FFMPEG_DIR` 是本项目首选；
- `LAE_FFMPEG_DIR` 只在 `--lae-compat` 时写；
- 环境变量变化对已启动进程不生效，要如实报告；
- 当前不为包管理器写共存逻辑。

---

## 12. CLI 输出注意点

当前 CLI 是唯一对外接口：

- CLI JSON 契约见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) 与 [`schemas/`](schemas/)；
- 错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)；
- stdout 只能是机器结果；
- stderr 才能写日志/进度/错误；
- JSON 顶层必须有 `schema`；
- 错误必须有稳定 `code`；
- 不把 Go 结构体字段名当协议；
- 文件路径必须明确返回；
- 退出码稳定；
- `--json` 与人类可读输出不能混在 stdout；
- 版本不兼容直接报错，不静默迁移；
- 当前不提供 Go API / 插件协议。

---

## 13. 许可证注意点

- 本项目采用 GPL-3.0-only，见根目录 `LICENSE` 与 [`LICENSE_DECISION.md`](LICENSE_DECISION.md)；
- BtbN GPL/LGPL 与 BnqDzj nonfree 不是同一合规模型；
- BnqDzj nonfree 二进制不得进入 public GitHub Release；
- 不随源码仓库提交任何 FFmpeg 二进制；
- 不把 nonfree 资产缓存上传为 CI artifact；
- `install.json` 记录 `redistributable`；
- `license_hint` 只是提示，不构成法律意见。

---

## 14. 测试策略

- fake ffmpeg：覆盖正常/超时/非零退出/乱码/权限错误；
- golden parser：每种能力命令至少 2 个版本格式；
- TOML strict mode 测试；
- CLI JSON 快照；
- 指定目录、PATH、环境变量样例；
- 全盘扫描用临时目录树；
- 下载用 `httptest.Server`；
- checksum 故意错误测试；
- zip slip / tar escape 测试；
- PATH rollback 测试；
- 真机集成：BtbN 与 BnqDzj 各至少一次；
- CI 双平台 + Go 1.27.3。

当前不做：

- Scoop 模拟目录测试；
- Homebrew prefix 模拟测试；
- JSON-RPC 协议一致性测试；
- Go API 兼容性测试；
- C ABI 测试。

---

## 15. 性能注意点

- 能力命令默认串行，避免启动风暴；
- 探测源并发有上限；
- 全盘扫描用 `errgroup` + 有界 channel；
- 大文件 hash 用流式 SHA-256；
- 缓存避免重复启动 ffmpeg；
- 日志节流；
- 下载不做无上限并发；
- 所有长操作支持 context 取消；
- worker pool 库仅作为未来候选，M0 不引入。

---

## 16. 明确不做

- 不写 damage；
- 不自构建 FFmpeg；
- 不链接 libav*；
- 不做 GUI / TUI / CUI；
- 不用 Go `plugin` 包；
- 当前不识别包管理器；
- 当前不开放 Go API；
- 当前不开放 JSON-RPC / gRPC / C ABI 插件；
- 不静默改系统；
- 不公开分发 nonfree；
- 不兼容旧实验 schema；
- 不把 floating `latest` 当安装记录；
- 不在 Phase 0 写业务代码。
