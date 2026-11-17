# 迁移与回滚指南

OneMoreFFmpeg v1.0.2 为唯一受支持版本，CLI 外部契约已冻结：人类输出可调整，但 `--json` 契约、错误码、数据目录布局与 journal schema 按 [`VERSIONING.md`](../reference/VERSIONING.md) 管理。
本指南汇总版本间需要注意的迁移点与「环境搞乱后」的恢复流程。

## 1. 数据目录布局

data-root 解析优先级：`ONE_MORE_FFMPEG_HOME`（须绝对路径）→ Windows `LOCALAPPDATA/OneMoreFFmpeg` → Unix `XDG_DATA_HOME/onemoreffmpeg` 或 `~/.local/share/onemoreffmpeg`。

```text
<data-root>/
├─ apps/<provider>/<tag>/<sha16>/   # 版本化安装 + install.json
├─ current（symlink / Windows junction / current.pointer 兜底）
├─ downloads/  cache/  journal/
├─ providers.toml
└─ config.toml
```

布局在 v0.6–v1.0.2 保持一致；遇到不认识的字段，CLI 按严格模式处理配置（未知字段会报错），请按 [`config.example.toml`](../../config.example.toml) 对照。

## 2. 配置迁移

- TOML 1.1，`schema = "onemoreffmpeg.config/v1"`；未知字段会被严格拒绝。
- 修改配置后先 `doctor --json` 确认健康；网络段（`network.github_api`）必须是 https（测试回环 http 除外）。
- 自定义 provider 存于 `providers.toml`，`provider add` 默认 dry-run，需 `--apply`。

## 3. journal 与事务迁移

- 每个写操作（install / uninstall / env set|unset|rollback）都有 journal 记录；`journal list --json` 查看，`journal rollback --journal <id> --apply` 回滚。
- 有 pending 记录时 `doctor` 会显示失败检查项；先回滚 pending 再做其他写操作。
- 跨进程互斥由 `<data-root>/.transaction.lock` 提供；不要手工删除该锁文件，除非确认没有正在运行的写命令。

## 4. 环境搞乱后的恢复流程

```bash
omf doctor                       # 1. 看现状
omf journal list                 # 2. 看是否有 pending
omf journal rollback --journal <id> --apply   # 3. 有 pending 就回滚
omf install --channel stable --apply --force  # 4. 干净重装
omf env unset --apply
omf env set --apply
# 5. 开新终端
ffmpeg -version
omf doctor                       # healthy: true
```

兜底：删除 data-root 前请先 `env unset --apply`（或 `env rollback`）清理 shim/profile/注册表，否则会留下失效条目；之后整个 data-root 删除只会移除本项目自己的文件：
Linux `rm -rf ~/.local/share/onemoreffmpeg`，Windows 删除 `%LOCALAPPDATA%\OneMoreFFmpeg`，然后从 [`HOWTO.md`](HOWTO.md) §2 重走。

## 5. 版本兼容

- `0.x` 阶段：同一 minor 内 JSON `data.schema` 字段只增不改；破坏性变化升 minor。
- `v1.0` 起遵循标准 SemVer：不兼容的 CLI 参数 / 退出码 / schema / 安全默认变化升 **MAJOR**。
- 外部消费者只依赖 [`CLI_SCHEMA.md`](../reference/CLI_SCHEMA.md) 与 [`ERROR_CODES.md`](../reference/ERROR_CODES.md)，不依赖内部 Go 结构体。
