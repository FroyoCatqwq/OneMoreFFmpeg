# 安全政策

## 支持范围

v1.0.1 为当前唯一受支持版本。安全问题按 SemVer 处理：修复进入 PATCH（或必要时 MAJOR）版本。

## 报告漏洞

请不要在公开 issue 中报告安全漏洞。通过 GitHub 私有渠道联系维护者：
<https://github.com/FroyoCatqwq/OneMoreFFmpeg/security/advisories/new>。

请提供：受影响版本、复现步骤、影响范围。确认后我们会协调修复与披露时间。

## 安全边界

OneMoreFFmpeg 是一个本机 CLI 工具，安全模型基于以下边界（详见 [`ARCHITECTURE.md`](docs/development/ARCHITECTURE.md) 与 [`SYSTEM_REQUIREMENTS.md`](docs/reference/SYSTEM_REQUIREMENTS.md)）：

- **信任 data-root**：`<data-root>` 目录视为用户私有空间。能写入它的人能影响 install/env 事务；不要放在共享可写目录。
- **写入默认 dry-run**：`install` / `uninstall` / `env set|unset|rollback` / `provider add` 默认只演练，需 `--apply` 才落盘。
- **写入类操作串行**：通过 `<data-root>/.transaction.lock` 的 OS 级文件锁跨进程互斥，不要手工删除该锁文件。
- **下载校验**：所有下载强制 SHA-256 校验与大小/重定向上限；自定义 URL 必须提供 pin。
- **解压防护**：拒绝绝对路径、`..`、符号链接、保留名；限制条目数、单文件大小、总大小与整体压缩比。
- **nonfree 边界**：nonfree FFmpeg 只保存元数据与下载入口，二进制不进入公开 Release。
- **环境变量**：`env set --apply` 会写用户 profile / 注册表（system scope 需显式提权授权）；回滚只恢复本项目管理的内容。
- **外部集成**：下游只调用 CLI 可执行文件，通过 JSON 信封与退出码交互，见 [`INTEGRATION.md`](docs/guides/INTEGRATION.md)。

## 已知限制（非漏洞但需知悉）

- journal 是 data-root 下的普通 JSON 文件，未做签名/加密；其路径字段在回滚前会做包含性校验，但 data-root 本身需由操作系统权限保护。
- `--json` 失败输出走 stdout（错误信封），成功输出仅一个信封；stderr 为日志。集成时应严格按 `schema` 字段分支。
- 自动化依赖许可证检查尚未接入 CI。
