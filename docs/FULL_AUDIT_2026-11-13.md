# 全量审查、二次检查与修复记录（2026-11-13）

> 环境：PowerShell 7 + `C:\Program Files\coreutils`；Linux 使用 `Linux-Server-Info.txt` 指向的服务器。
> 方法：用户视角只读审计 + 代码视角只读审计 + 逐条核实 + 回归测试 + Windows/Linux e2e + 文档/契约一致性检查。

---

## 1. 结论

本轮对 Phase 1–5 全部 CLI 与内部包做了用户视角和代码视角两轮审计，共核实 30 项高置信度问题；本轮修复了其中的主要正确性、事务、契约、平台与文档问题，并把剩余设计边界明确记录。

项目当前状态：Phase 1–5 核心闭环可用，写入路径默认 dry-run，install/uninstall/env 事务可回滚，Windows 与 Linux 服务器 e2e 通过。

---

## 2. 关键修复

### 2.1 事务与回滚

- install journal 现在记录 `quarantine` / `quarantine_original` / `current_target` / `current_kind` / `current_existed`；成功的 `--force` 保留旧版本隔离目录，回滚可恢复旧版本与 current。
- `--force` 现在会隔离同一 provider/tag 下的旧版本目录，而不只是同 SHA 的目标目录。
- install 幂等快路径会重新校验归档 SHA、验证三件套、刷新 current 类型与 install.json，不再返回陈旧状态。
- env set/unset 改为写前 `pending` journal；每一步后刷新 details；失败自动撤销已写内容并写 `failed` journal。
- env rollback 恢复 Windows 变量后广播 `WM_SETTINGCHANGE`；Linux 恢复 shim 与 `~/.profile` 备份。
- `journal rollback` / `env rollback` 默认 dry-run，需 `--apply` 才恢复。
- journal ID 拒绝 `..` 与路径分隔符；journal / install metadata / current link 临时文件改用随机名，避免并发互相删除。

### 2.2 用户视角与 CLI

- 首次使用提示以前因 `configPath` 总非空而从不打印；现在无配置文件时写 stderr，并用 `<data-root>/.first-use` 保证只提示一次。
- `--in-detail-usage` 不再把 `--config --in-detail-usage` 这类值参数误判为 usage flag。
- `--release` 接受 UTF-8（可带 BOM）与 UTF-16LE/BE，兼容 PowerShell 5 `>` 输出。
- `release list --channel stable` 的 `--count` 真正截断，stable 的 `days` 输出 0；`--days` 只用于 daily。
- daily 资产名无法解析版本时回退为 snapshot tag，覆盖 `master` / `nX.Y-latest` 资产。
- `release install`：`--version` / `--snapshot` 互斥；显式选择时忽略 `--count` 并扩大查找窗口；bnqdzj-nonfree 未指定 variant 时自动使用 `nonfree`。
- `provider add` 默认 dry-run，需 `--apply`；全链路 `Load → AddCustom → Save` 持有跨进程锁。
- `provider add` 与 archive 共用格式白名单：支持 `.zip` / `.tar.xz` / `.tar.gz` / `.tgz` / `.tar.zst`，拒绝 `.7z`。
- runtime `writeResult` 校验 command 白名单并补齐 EnvStatus / EnvSet / Doctor / Uninstall / JournalList / JournalAction / Version / Detect / Validate 等字段级校验。
- `uninstall` 只隔离 `<data-root>/apps/` 下带 `install.json` 的目录，回滚恢复目录与 current。

### 2.3 能力探测与发现

- `capabilities --only` 支持 `--cache-dir` / `--no-cache` / `--refresh`，缓存键按 kind 独立计算；补齐 Linux 可执行位检查与取消/超时映射。
- Linux 显式 `--recursive --dir /tmp/...` 或 `--scan --roots /tmp/...` 不再因 `/tmp` 排除规则跳过所有子目录。

### 2.4 环境写入

- Linux `--no-path` / `--no-variables` / `--lae-compat` 实际控制 shim 与 `~/.profile` 托管块内容。
- `~/.profile` 托管块使用原子替换；journal 记录备份；失败与 rollback 可恢复。
- shim 增加托管标记，非 OneMoreFFmpeg 文件返回 `E6007`，不再覆盖/删除用户文件。
- Windows `REG_EXPAND_SZ` PATH 保持原类型写回。

### 2.5 二次检查（修复后复核）

只读二次复核又确认 4 个 P1，已修：

- Linux 首次创建的 shim 在 undo / rollback 时被 `managed=0` 误判为第三方文件而残留：改为仅跳过“原本存在且非托管”的 shim，新建 shim 正常删除；
- `shell_rc_previous` 在无实际 shell rc 修改时被 rollback 无条件覆写，可能抹掉用户后续编辑：只有存在 `shell_rc_backup` 时才恢复；
- install `--force` 在 quarantine 之后任一步失败只写 failed journal，不恢复旧安装：新增 `restoreOnFailure` + `committed` 标志，覆盖所有 fail 路径，恢复旧目录与 current；
- 新增 `TestInstallFailureRestoresQuarantinedVersion` 覆盖上述失败补偿。

### 2.6 CI / 远程 e2e

- `remote_linux.py` 合并 stdout/stderr 避免 SSH 通道死锁；Go 单测与 Bats 失败不再被 `check=False` 吞掉； secret 缺失时 CI 明确失败。
- `.github/workflows/e2e.yml` 保持 Windows/Ubuntu matrix，并增加配置化 Linux 服务器任务。

---

## 3. 新增/更新的回归测试

- `internal/install`：force 升级 + journal rollback 恢复旧版本。
- `internal/env`：写前 pending、失败撤销、failed journal、rollback 注入 store。
- `internal/provider`：stable `--count`、daily master 资产版本回退。
- `internal/capabilities`：`--only` 缓存命中。
- `internal/detect`：Linux 显式 `/tmp` root 递归扫描。
- `internal/cli`：UTF-16 release JSON 输入。
- `internal/contracts`：运行时 command 白名单；示例契约字段。
- e2e：新增 `--in-detail-usage`、`--only`、动态版本目录、uninstall → journal rollback 恢复。

---

## 4. 验证

实际结果：

- Windows：Go 单测全绿；pytest e2e `12 passed`；Robot `3 passed`
- Linux 服务器：Go 单测全绿；pytest e2e `12 passed`；Robot `3 passed`；Bats `3 passed`
- `go vet ./...`、`go build -tags deps ./...`、Windows/Linux 交叉编译通过
- 文档链接、尾随空格、JSON 示例校验通过

验证命令：

```text
go vet ./...
go test -count=1 ./...
go build -tags deps ./...
GOOS=windows GOARCH=amd64 go build ./...
GOOS=linux GOARCH=amd64 go build ./...
Windows: pytest e2e + Robot Framework
Linux server: go test + pytest e2e + Robot Framework + Bats
```

---

## 5. 仍保留的边界

1. GitHub releases 只取前 100 条；`--days` 很大时可能仍受第一页限制，未实现 Link 头分页。
2. `provider add` 锁文件超过 30 秒会被视为陈旧锁清理；极长事务仍可能竞争。
3. install / env / uninstall 没有全局 data-root 事务锁；并发写仍然遵循“后写覆盖”语义，但临时文件与 current 备份已随机化。
4. Linux 只维护 `~/.profile`，不处理 fish/zsh 专属 rc；不改系统级环境、不提权。
5. `doctor` 默认退出 0，即使 `healthy=false`；脚本必须读取 `data.healthy`。
6. `download` 与 capabilities 缓存会立即写文件，不属于默认 dry-run 命令。
