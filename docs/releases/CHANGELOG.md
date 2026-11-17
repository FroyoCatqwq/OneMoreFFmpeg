# 更新日志

本项目遵循 [Semantic Versioning 2.0.0](../reference/VERSIONING.md)。格式基于 [Keep a Changelog](https://keepachangelog.com/)。
仅保留当前受支持版本 v1.0.1 的记录；更早的 Release 与历史快照已撤回。

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
