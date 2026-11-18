# 更新日志

本项目遵循 [Semantic Versioning 2.0.0](../reference/VERSIONING.md)。格式基于 [Keep a Changelog](https://keepachangelog.com/)。
仅保留当前及上一版本的记录；更早的 Release 与历史快照已撤回。

## [1.1.1] — 2026-11-18

v1.1.0 bugfix：聚焦自更新可靠性、进度语义与缓存正确性。本版同时移除了内置 provider 与 Scoop/dnf 支持，CLI 契约调整为 18 命令 / 17 schema。

### Fixed
- **自更新回滚**：失败时不再忽略恢复错误；`errors.Join` 同时报告"新程序安装失败"与"旧程序恢复失败"，保留唯一可恢复备份及其路径；新增 staged/verified/backup_created/replaced/committed 阶段 journal 与 `Recover()` 崩溃恢复；保留原可执行文件权限；备份清理失败记录诊断。
- **进度语义**：`Progress` 拆为 `Update/Complete/Fail/Abort/Close` 四态互斥幂等；失败/取消/校验失败/提交失败不再显示 100%/completed；未知 Content-Length 不伪造百分比；TTY 与非 TTY 语义一致；进度只写 stderr。
- **缓存**：key 文件名改为 SHA-256 稳定摘要（消除 `a/b` 与 `a_b` 碰撞），entry 保存 key digest 与 schema 并校验；读写错误真实传播；新增 `Prune()` 过期/损坏文件清理（限制在 cache root、不跟随 symlink）；`InvalidateErr` 不吞 I/O 错误。
- **capabilities**：缓存读写统一走 `internal/cache` 的 JSON/锁/原子写辅助，业务层只保留 fingerprint 与有效性判断。
- **版本校验**：自更新只接受严格 `X.Y.Z`（拒绝缺/多/空字段、非数字、前导零、溢出、prerelease），数值语义比较；`v` 前缀由 Check 规范化。
- **Windows 测试**：自更新替换/恢复用可注入 `FileOps` 在临时文件上测试，Windows CI 不再跳过核心替换路径。

### Changed
- 文档：README 澄清"用户主动 update"与"后台自动更新"的区别；CHANGELOG/VERSIONING/安装指南/第三方声明同步到 v1.1.1。
- 新增：全盘扫描（`detect --scan` / `--dir --recursive`）要求 elevated 权限（Linux root/sudo，Windows Administrator/SYSTEM），防止普通权限下无效的全盘遍历。

## [1.1.0] — 2026-11-18

首个 MINOR 版本：新增自更新与下载进度条两个功能，并保持 CLI 契约稳定扩充。

### Added
- **`update` 命令（自更新）**：默认 dry-run 检查 GitHub Releases 最新稳定版，`--apply` 下载并校验 SHA-256 后原子替换当前可执行文件；Windows 采用旧文件改名策略，失败自动回滚。新增 `onemoreffmpeg.update/v1` schema。
- **下载进度条**：TTY 显示进度条+百分比+ETA；CI/重定向输出阶段日志（无动画字符）；`--json` 仅 stdout JSON；进度走 stderr 不污染数据。
- **日志框架接入业务层**：`log/slog` + `lmittmann/tint`，install/download/capabilities 等通过 `logging.FromContext(ctx)` 输出调试日志，`--log-level` / `--quiet` 控制。
- **独立 `internal/output` 层**：CLI 与下载层不再直接使用 JSON/表格/进度第三方库（encoding/json/v2、go-pretty、tidwall/pretty 封装在 output 包内）。
- **独立 `internal/cache` 层**：基于 otter 的内存缓存 + 磁盘持久化（TTL、原子写），调用方不直接接触缓存库。

### Changed
- 能力缓存写入改为原子写 + `internal/lock` 的 OS 级跨进程锁；fingerprint 含 FFmpeg 二进制内容 hash，替换二进制自动失效。
- 引入依赖：`jedib0t/go-pretty/v6`、`tidwall/pretty`、`maypok86/otter/v2`、标准库 `encoding/json/v2`；`gofrs/flock` 继续作为文件并发锁。

## [1.0.3] — 2026-11-19

Bugfix 版本。全量审查核心包（provider/download/capabilities/install/env）后修复潜在 bug。

### Fixed
- 重装同一 SHA 并加 `--emit-trio-lock` 时不再静默忽略该 flag（reuse 分支补写 lock 并回写 `install.json` 的 `LockPath`）。
- `capabilities list <kind>` 缓存命中时按当前 trio 状态补 `W1001 ffplay is missing`，与全量收集结果一致。
- `--days` 极大正数不再因 `time.Duration` 溢出导致 daily catalog 过滤为空（clamp 100 年上界 + `AddDate`）。
- Linux 下 `env status` 的 PATH 变量名比较改为平台敏感（`envNameKey`），小写 `path` 不再被误判为 PATH。

四项均附回归测试；CLI 外部契约不变。

## [1.0.2] — 2026-11-18

Bugfix 版本。全量审查每个包与每份文档后修复确认的缺陷。

### Security
- `--json provider|env|journal|release help`（嵌套内置 help 子命令）不再绕过 stdout JSON-only 契约，统一返回 E1001 错误信封。
- 事务锁文件不再在加锁前被写入（避免符号链接锁路径把 data-root 外的文件截断）；锁路径为符号链接时显式拒绝；锁文件内容保持为空，正确性完全来自 OS 锁身份。
- `env rollback` 的 shim 目录白名单增加 scope/提权一致性校验的说明；`--system` scope 回滚强制提权。

### Fixed
- `--max-bytes` CLI flag 加上与 config 一致的 8 GiB 上限，防止 flag 绕过配置硬限制。
- `doctor --strict` 在 human 模式的错误里输出失败检查项，不再丢失诊断信息。
- checksums 资产按文档优先 `checksums.sha256` → `*checksums*.sha256` → `*.sha256` 查找，不再只认精确文件名。
- GitHub `ListReleases(nil, ...)` 在满页时不再 panic；`Retry-After` 超大整数不再溢出绕过 60s 上限；officialVersions 重定向限制 10 次。
- `provider add` 的 `gpl-shared`/`nonfree-shared` 不再覆盖用户显式 `--linkage static`。
- 一次 detect 中 `--dir --recursive` 与 `--scan` 共享 `MaxFiles` 预算；重叠 roots 按父目录优先扫描避免重复遍历。
- 文档修正：journal schema 字段、shell rc 托管块标记、删除 data-root 的影响、变量覆盖语义、锁错误码 E6009、latest pin 表述、Go 工具链 1.27.x 等。

### Added

## [1.0.1] — 2026-11-17

Bugfix 版本。对全项目做源码级审计（CLI/契约、网络/provider、检测/进程、CI/测试基础设施），修复全部确认缺陷。

### Security
- `--json help [cmd]` 不再绕过 stdout JSON-only 契约：内置 `help` 子命令纳入纯文本请求拦截，返回 E1001 错误信封。
- `network.github_api` 与 GitHub 客户端拒绝 URL userinfo（如 `https://api.github.com@evil.example`），防止 Bearer token 发往错误主机；错误信息不再回显含凭据的原始 URL。
- 下载/GitHub API/catalog 重定向禁止 HTTPS→HTTP 降级；下载 URL 拒绝 userinfo。
- 回滚路径符号链接防护：journal 路径解析后包含性校验、逐级 symlink 拒绝；shim body 模板校验；system scope 提权检查；archive 预置 symlink 拒绝。
- 远程 e2e 优先使用 known_hosts 严格校验 SSH 主机密钥（缺失时告警回退）；远程命令加空闲/总体看门狗。
- `officialVersions` 校验 releases 页 URL scheme、禁止降级跳转、限制响应体 4 MiB。

### Fixed
- `uninstall` 多个定位 flag（`--root`/`--release`/`--provider`）同时给出时改为显式报错，不再静默选优先级。
- `provider add` 的 `--apply` 与 `--dry-run` 同时给出时显式报错。
- GitHub Releases 列表支持最多 5 页抓取（此前只取第一页，静默漏掉旧版本）。
- `doctor` 的 `trio_valid` 现在校验 `-version` 的 banner，exit-0 假二进制（如 `/bin/true`）不再被判为健康。
- ffexec 中进程已成功退出后才到达的 context 取消不再把成功结果改写成取消错误。
- 全盘扫描把目录也计入 `MaxFiles` 预算，防止大量空目录绕过资源上限。
- `-h <topic>` 缺失检测扫描全部输出行，版本 banner 不再掩盖 `Unknown ...` 诊断。
- Windows 下 `binaryNames` 增加 `.com`，与扫描器的可执行名识别一致。
- `commitProvidersFile` 回滚失败时报告备份路径，不再静默丢失 providers.toml。
- config：`timeout` 上限 5 分钟、`max_bytes` 上限 8 GiB、配置文件读取上限 1 MiB。
- release：commit 元数据改用实际 checkout 的 ref；quality gate 补齐 gofmt/vet/fuzz；严格 SemVer tag 校验；provenance 覆盖 SHA256SUMS.txt。

### Added
- 环境变量真实修改（`--apply`）的完整 e2e：Linux 临时 HOME 隔离、Windows marker 控制的 HKCU 注册表；CI 的 Windows job 现在执行注册表 e2e。
- 针对本轮全部修复的回归测试；新增 `.dockerignore` 避免测试产物进入镜像。

### Changed
- 源码拆分提升可维护性：`contracts`→`validation.go`、`env`→`shell.go`、`capabilities`→`cache.go`；本轮行为与 CLI 契约不变。
- 文档全量审查：删除 archive 历史快照与旧版 release notes，余下文档与 v1.0.1 源码对齐。
