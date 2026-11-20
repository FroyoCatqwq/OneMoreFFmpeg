# 版本与发布规范

> 本项目遵循 [Semantic Versioning 2.0.0](https://semver.org/)；发布 tag 使用 `v` 前缀，例如 `v1.1.1`。

## 1. 版本号含义（v1.0 起适用标准 SemVer）

- **MAJOR**：不兼容的 CLI 参数、退出码、JSON schema 或安全默认变化；
- **MINOR**：向后兼容的新命令、新字段、新平台、新 provider 能力；
- **PATCH**：向后兼容的 bug 修复、文档修正、测试与构建修复。

`0.y.z` 阶段（历史）同一 minor 内保证 `onemoreffmpeg.*/v1` 字段只增不改；破坏性变化升 minor。
**v1.0 起**：不兼容变化一律升 MAJOR，`/v1` schema 主版本随 MAJOR 递增（见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) §5）。

## 2. 发布流程

```text
1. 全量测试：mise run quality（vet/test/fuzz，gofmt 与 golangci-lint/gosec 由 CI 门禁执行）、Windows+Linux e2e、Docker 全新环境
2. 更新 docs/releases/CHANGELOG.md 与 docs/releases/release-vX.Y.Z.md
3. 提交代码与文档；确认 GitHub Actions 质量门禁全绿
4. 创建 annotated tag：git tag -a vX.Y.Z -m "vX.Y.Z"
5. 推送 main 与 tag（按本机网络环境配置代理）
6. 打 tag 触发 .github/workflows/release.yml：构建双平台二进制、SHA256SUMS、provenance
7. 校验 release 资产与 checksum、确认无 nonfree 二进制
```

## 3. 产物命名

```text
one-more-ffmpeg-vX.Y.Z-windows-amd64.exe
one-more-ffmpeg-vX.Y.Z-linux-amd64
SHA256SUMS.txt
```

## 4. 兼容性

- CLI 人类输出可调整；`--json` 字段以 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) 为准；
- 错误码编号不复用、不改变语义；新增错误码必须更新注册表与文档；
- 数据目录布局与 journal schema 变化必须升 MAJOR，并提供迁移/回滚说明（见 [`MIGRATION.md`](../guides/MIGRATION.md)）；
- 外部集成只依赖 CLI 契约，见 [`INTEGRATION.md`](../guides/INTEGRATION.md)。

## 5. 历史

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.6.0 | 2026-11-13 | 首个公开 release：Phase 1–5、审查修复、e2e |
| v0.7.0 | 2026-11-14 | 全局事务锁、系统级作用域与显式提权授权、doctor --strict、Windows symlink 回退 |
| v0.8.0 | 2026-11-14 | Phase 6 契约冻结 + 外部调用示例、解压加固、shim 回滚/下载清理修复 |
| v0.9.0 | 2026-11-14 | Phase 1–6 收口：修完 P1-4a / P2-2 / P2-3 / P2-4 / P2-5（历史版本，Release 与文档已撤回） |
| v1.0.0 | 2026-11-16 | 首个稳定版：Phase 1–7 完成、完整 CI 门禁与 release 自动化（历史版本，Release 与文档已撤回） |
| v1.0.1 | 2026-11-17 | Bugfix：全项目源码审计后修复 CLI 契约绕过、token 泄露、重定向降级、GitHub 分页、doctor 假二进制、flag 互斥等缺陷（历史版本，Release 与文档已撤回） |
| v1.0.2 | 2026-11-17 | Bugfix：嵌套 help 契约绕过、锁文件符号链接风险、参数校验、checksums 查找、分页 nil ctx、文档错漏（历史版本） |
| v1.0.3 | 2026-11-18 | Bugfix：全量审查核心包后修复 4 个潜在 bug（emit-trio-lock reuse、能力缓存 W1001、days 溢出、Linux PATH 大小写）（历史版本） |
| v1.1.0 | 2026-11-18 | 首个 MINOR：新增 `update` 自更新命令与下载进度条；独立 output/cache 层；slog+tint 日志接入业务层；能力缓存跨进程锁；进度条 TTY/CI/JSON/Pipe 自适应（历史版本） |
| v1.1.1 | 2026-11-18 | Bugfix：自更新失败可靠恢复、进度四态语义、缓存统一存储、严格 SemVer 校验（历史版本） |
| v1.1.2-pre1 | 2026-11-18 | 预发布：全量审查后修复 16+ bug；自定义 URL 支持 http；引入 go-github/testscript；依赖全量更新（历史版本） |
| v1.1.2-pre2 | 2026-11-20 | 预发布：第二轮细致修复（update --recover 死代码/状态机、fetch panic、下载 symlink、锁一致性等）+ 文档/CI 一致性（历史版本） |
| v1.1.2-pre3 | 2026-11-21 | 预发布：新增默认 official provider（FFmpeg-Builds latest 快照，gpl/static/x86_64，数据驱动命名规则，README 链接提示）。详见 [`release-v1.1.2-pre3.md`](../releases/release-v1.1.2-pre3.md) |
