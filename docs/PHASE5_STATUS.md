# Phase 5 实施状态：安装与 current 切换

> 日期：2026-11-13
> 范围：`install`（解压、版本化目录、install.json、current、journal）+ 只读 `env status` / `doctor`
> 不在本轮：`env set/unset/rollback`、`uninstall`、Windows directory junction

---

## 1. 完成内容

| 能力 | 位置 | 说明 |
|---|---|---|
| 安全解压 | `internal/archive` | zip / tar.xz / tar.gz / tar.zst；拒绝绝对路径、`..`、盘符、symlink、非普通 tar 条目；条目/大小上限 |
| 版本化安装 | `internal/install` | `<data-root>/apps/<provider>/<tag>/<sha256[:16]>/bin/`；`install.json`；`ffmpeg-trio.lock`（可选） |
| 安装后校验 | `internal/install` | `ffmpeg -version` / `ffprobe -version`，失败 `E5008` 并清理 |
| current 指针 | `internal/install` | 优先 symlink；Windows 无权限时写 `current.pointer`；真实目录拒绝覆盖 `E5004` |
| 事务 journal | `internal/journal` | pending → committed / failed；`<data-root>/journal/<id>.json` |
| 冲突与替换 | `internal/install` | 同 asset 幂等；不同 asset / 已存在目录需 `--force`，旧目录隔离后替换 |
| CLI | `internal/cli/install.go` | `install --release|--provider...`、`--artifact`、`--apply`（默认 dry-run）、`--force`、`--no-current`、`--emit-trio-lock` |
| 环境只读检查 | `internal/env` | `env status`：current 类型/目标、PATH、关键变量、resolved ffmpeg |
| 健康检查 | `internal/doctor` | data root、current、三件套可运行、PATH、pending journal |

---

## 2. 安全与失败语义

```text
unsupported archive        E5001
zip slip / symlink         E5002
target already exists      E5003
current is a real dir      E5004
quarantine failed          E5007
post-install validation    E5008
archive file missing       E7007
archive sha mismatch       E4002
```

- dry-run 只做路径/冲突/格式检查，不写文件、不下载；
- apply 失败会删除临时目录、回滚已移动的目标目录与隔离目录，并写 failed journal；
- 归档 SHA-256 在解压前重新计算，必须与 release pin 一致；
- nonfree release 仍需 `--allow-nonfree`；
- 不修改 PATH，不修改系统环境变量（Phase 6）。

---

## 3. 验证

```text
go test ./internal/archive ./internal/journal ./internal/install ./internal/cli
  通过：zip/tar.xz 解压、zip slip、symlink、unsupported、dry-run、
        幂等安装、force 替换、journal 状态、current symlink/pointer
go build -tags deps ./... / windows / linux
```

测试使用 `internal/testutil` 的真实 fake 可执行文件打包成 zip，安装后实际执行 `-version`。

---

## 4. 模块拆分

| 文件 | 职责 |
|---|---|
| `internal/install/install.go` | 安装编排、冲突/回滚、journal 状态迁移 |
| `internal/install/files.go` | 目录布局、组件名清洗、二进制查找/复制/哈希/校验 |
| `internal/install/current.go` | current 类型判定、切换、临时 symlink 替换 |
| `internal/install/record.go` | install.json / ffmpeg-trio.lock 原子写入 |
| `internal/provider/resolve.go` | Request 归一化、GitHub 解析主流程 |
| `internal/provider/checksums.go` | digest/checksums 双源、HTTP fetch 与错误映射 |
| `internal/provider/custom_resolve.go` | 自定义 provider 解析与严格 URL 校验 |

## 5. 后续能力（本轮补齐）

- Windows `current` 使用 directory junction（reparse point），无需 symlink 权限；不可用时回退 `current.pointer`；
- 切换 current 使用“新链接 → 备份旧链接 → 替换 → 删备份”，失败恢复旧指针；
- 安装额外复制归档根部的 `LICENSE*` / `COPYING*` / `README*` / `NOTICE*`（best effort）；
- `env set/unset`：Windows 写 HKCU\Environment（PATH / `ONE_MORE_FFMPEG_DIR` / `FFMPEG_DIR`，可选 `LAE_FFMPEG_DIR`），Linux 写 `~/.local/bin` shim；默认 dry-run，写前 journal 备份旧值；
- `env rollback --journal <id>` 恢复旧 PATH/变量；
- `uninstall --root <abs> --apply`：仅允许删除 `<data-root>/apps/` 下的目录，并清理指向它的 current。

## 6. 已补齐

- Windows 写入环境后广播 `WM_SETTINGCHANGE`（失败 `E6008`）；
- `uninstall` 支持 `--release <json>` 或 `--provider` + `--release-tag` 定位，仍可用 `--root`；
- Linux `env set/unset` 维护 `~/.profile` 中的 `# >>> OneMoreFFmpeg >>>` 块，并备份到 journal 目录；`--no-shell-rc` 可关闭；
- `providers.toml` 写入使用跨进程锁文件（5s 超时，30s 视为陈旧锁）；
- Windows `current` 使用 junction reparse point；`WM_SETTINGCHANGE` 只影响新启动的进程。

## 7. 全量审查修复（2026-11-13）

- Windows junction 以前用 `CREATE_NEW` 创建空文件，reparse point 无法挂到目录：改为先建目录、再 `OPEN_EXISTING` + `FSCTL_SET_REPARSE_POINT`，失败清理；名称长度排除 NUL；
- `env set/unset` dry-run 在 Linux 曾真的改写 `~/.profile`：dry-run 现在只生成操作计划；
- journal id 未校验导致 `env rollback --journal ../../x` 可越界读文件：加入 ID 白名单并拒绝 `..`；
- `WM_SETTINGCHANGE` 广播失败时 `callErr` 可能为 nil 导致 panic：先判空；
- Windows `REG_EXPAND_SZ` PATH 曾被写成 `REG_SZ`：按原类型写回 `SetExpandStringValue`；
- uninstall 对 junction 的 current 解析失败并可能误删任意目录：改用 reparse 属性识别、解析真实路径、要求 `install.json`、只删托管链接；
- `install.json` 最终刷新错误被吞：改为失败时回滚并写 failed journal；
- Linux shim 曾覆盖/删除用户文件：加入托管标记，非托管文件返回 `E6007`，路径带空格正确引用；
- 契约校验把 `latest + 非法 sha` 误报 `E3010`：改为空 pin 才 `E3010`，非法 sha 为 `E7003`；
- Linux `go test` 暴露：非可执行 ffmpeg 被当成“不存在”，改为发现后报告 `status=unusable`。

## 8. 回滚/卸载强化（2026-11-13）

- `journal list` / `journal rollback` 成为正式 CLI；
- uninstall 改为「移到 quarantine」而非删除，journal 记录 `quarantine` / `current_target` / `current_kind`，回滚恢复安装目录与 current；
- install/uninstall/env 事务统一可通过 `journal rollback --journal <id>` 恢复；
- 能力探测增加 `capabilities --only <kind>`，只执行目标 FFmpeg 命令并输出 `onemoreffmpeg.capability-list/v1`；
- `release list` 动态读取 FFmpeg 官网版本，稳定版按大版本线显示最新补丁，daily 默认只展示近 7 天快照；`release install` 可按 `version` / `snapshot` 安装；
- 内置 `--in-detail-usage`、首次使用 stderr 提示、`docs/USAGE.md` 与 `docs/man/one-more-ffmpeg.1`。

## 9. 2026-11-13 二次审查强化

- install `--force` 隔离同 provider/tag 旧版本并保留 quarantine，journal rollback 可恢复旧版本与 current；
- env set/unset 写前 pending journal、失败自动撤销、成功 committed；回滚恢复 shim / `~/.profile` / Windows 变量并广播；
- journal rollback / env rollback 默认 dry-run，需 `--apply`；
- install/uninstall/current/journal 临时文件随机化；provider add 全链路锁；
- 首次使用提示只显示一次；`--release` 支持 UTF-16；运行时 command/data 契约校验；
- release catalog、capabilities 单类缓存、Linux `/tmp` 显式扫描、远程 e2e 等细节见 [`FULL_AUDIT_2026-11-13.md`](FULL_AUDIT_2026-11-13.md)；
- 二次复核修复了 Linux 新建 shim 回滚残留、`~/.profile` 无变更回滚误覆写、`--force` 中途失败未恢复 quarantine、非托管 shim 导致 dry-run 失败等问题。

## 10. 已知限制

1. `uninstall` 仍需用户显式选择安装（`--root` / `--release` / `--provider`+`--release-tag`），不支持交互选择；
2. Linux 只改 `~/.profile`，不处理 fish/zsh 专属配置；
3. 安装只保留 `bin/` 与许可证/README，不保留完整归档结构；
4. 环境写入不做系统级（machine）修改，也不提权。
