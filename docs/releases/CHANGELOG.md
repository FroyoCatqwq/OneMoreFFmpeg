# 更新日志

本项目遵循 [Semantic Versioning 2.0.0](../reference/VERSIONING.md)。格式基于 [Keep a Changelog](https://keepachangelog.com/)。

## [1.0.1] — 2026-11-17

Bugfix 版本。对全项目做源码级审计（CLI/契约、网络/provider、检测/进程、CI/测试基础设施），修复全部确认缺陷。

### Security
- `--json help [cmd]` 不再绕过 stdout JSON-only 契约：内置 `help` 子命令纳入纯文本请求拦截，返回 E1001 错误信封。
- `network.github_api` 与 GitHub 客户端拒绝 URL userinfo（如 `https://api.github.com@evil.example`），防止 Bearer token 发往错误主机；错误信息不再回显含凭据的原始 URL。
- 下载/GitHub API/catalog 重定向禁止 HTTPS→HTTP 降级；下载 URL 拒绝 userinfo。
- 回滚路径符号链接防护（上一轮）：journal 路径解析后包含性校验、逐级 symlink 拒绝；shim body 模板校验；system scope 提权检查；archive 预置 symlink 拒绝。
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
- 源码拆分提升可维护性：`contracts`→`validation.go`、`env`→`shell.go`、`capabilities`→`cache.go`（上一轮）；本轮行为与 CLI 契约不变。

## [1.0.0] — 2026-11-16

首个稳定版。Phase 1–7 全部完成，CLI 外部契约冻结。

### Added
- Phase 7 质量门禁：gofmt / go vet / go test / govulncheck / fuzz（parser、URL/asset、JSON 输入）；
- tag 触发的 release 流水线：双平台二进制、SHA256SUMS、build provenance、nonfree 资产守卫；
- 扩展 e2e：`test_cli_full_e2e.py`（全部 18 命令信封、capabilities 字段级解析、9 种 `--only`、缓存、参数互斥）；
- Docker 全新环境 e2e 镜像（干净容器复现构建与测试）；
- 发布产物新增 RHEL 9+ x86_64 RPM 包（纯静态、无运行时依赖）与 Scoop bucket 自动更新（独立仓库 `OneMoreFFmpeg-bucket`）；
- fuzz 目标：`FuzzParseVersion` / `FuzzParseList` / `FuzzValidateHelpTopic` / `FuzzIsHTTPS` / `FuzzSafeAssetName` / `FuzzIsAbsoluteLike` / `FuzzNormalizeJSONInput`；
- 文档按 `guides/`、`reference/`、`development/`、`releases/`、`archive/` 重整，新增集成与迁移指南、SECURITY.md。

### Fixed
- **Critical**：数据根事务锁改用 OS 级 flock/LockFileEx，消除 stale 回收与新锁获取之间的 TOCTOU（可导致两个写事务并发破坏安装目录）；
- **High**：`env rollback` 现校验 journal 中的 `shell_rc_backup` 必须位于 data root，杜绝借伪造 journal 任意写文件；
- **High**：`uninstall` 提交阶段重构——quarantine 先落 journal，current 指针删除失败时回滚，最终 journal 写失败提供明确恢复指引；
- **Medium**：压缩包压缩比防护改为全局生效（多条目聚合不再绕过 `compressed × maxRatio`）；
- **Medium**：`providers.toml` 复用 OS 级文件锁，备份文件改唯一名，消除并发覆盖；
- **Medium**：下载共享目录用唯一 owner 标记保护首次创建的清理竞态；
- **Low**：Linux PATH 去重改为大小写敏感（原 EqualFold 会误判不同目录）；
- `--json` 下 help/usage 文本一律以错误信封拒绝；`--release` 与 provider 解析 flag 互斥；短 SHA-256 拒绝；`--only` 与 `--help-topic` 冲突拒绝。

### Changed
- CLI 契约：18 命令 / 17 个 `/v1` schema / 退出码 0/1/2/130 冻结，`internal/contracts/freeze_test.go` 守护；
- 命令→data.schema 映射提升为生产代码单一来源（运行时校验 + golden 示例 + 冻结测试共用）；
- Windows Job Object 决策定为永久不实现（taskkill + Process.Kill 兜底）。
- 从 v1.0 起，不兼容变化一律升 MAJOR。

## [0.9.0] — 2026-11-14

Phase 1–6 收口：修完 P1-4a / P2-2 / P2-3 / P2-4 / P2-5（旧版 Release 与文档已撤回）。

## [0.8.0] — 2026-11-14

Phase 6 契约冻结 + 外部调用示例、解压加固、shim 回滚与下载清理修复。

## [0.7.0] — 2026-11-14

全局事务锁、系统级作用域与显式提权授权、`doctor --strict`、Windows symlink 回退。

## [0.6.0] — 2026-11-13

首个公开 release：Phase 1–5、审查修复、e2e。
