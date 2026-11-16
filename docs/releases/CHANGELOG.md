# 更新日志

本项目遵循 [Semantic Versioning 2.0.0](../reference/VERSIONING.md)。格式基于 [Keep a Changelog](https://keepachangelog.com/)。

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

Phase 1–6 收口：修完 P1-4a / P2-2 / P2-3 / P2-4 / P2-5，详见 [release-v0.9.0.md](release-v0.9.0.md)。

## [0.8.0] — 2026-11-14

Phase 6 契约冻结 + 外部调用示例、解压加固、shim 回滚与下载清理修复。

## [0.7.0] — 2026-11-14

全局事务锁、系统级作用域与显式提权授权、`doctor --strict`、Windows symlink 回退。

## [0.6.0] — 2026-11-13

首个公开 release：Phase 1–5、审查修复、e2e。
