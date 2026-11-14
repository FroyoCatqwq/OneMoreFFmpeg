# 全量审查与修复记录（2026-11-14 · 第二轮）

> 承接 [`FULL_AUDIT_2026-11-14.md`](FULL_AUDIT_2026-11-14.md)（F1–F14）。那一轮之后代码新增了
> `internal/lock` 事务锁与 system scope；本轮的审计发现该锁的语义有洞，并修掉三处更早遗留的问题。
> 环境：Windows 本地 `go1.27.4`；`go build ./...`、`go vet ./...`、`go test ./...` 全绿（22 个包）。

## 1. 已修复

### 1.1 事务锁：持有者标识 + 心跳续期 + 校验式释放（P1）

`internal/lock/lock.go`。原实现有两个缺陷：

* 锁文件的 mtime 在持有期间**从不续期**，而 `staleAfter` 只有 30 s。`install` 的临界区包含解压
  ≤512 MiB、复制三个可执行文件、两次 `-version` 探测、`rename` 与多次 journal 写盘，正常机器上
  很容易超过 30 s —— 第二个进程会判定「锁过期」并删掉它进入临界区。
* `unlock` 是 `func() { os.Remove(path) }`：原持有者退出时删掉的是**此刻属于别人**的锁，两个进程
  于是同时写 `apps/`、`current`、`quarantine/` 与 `journal/`。

现在：

* 锁文件写入 owner nonce（`<pid>-<8 字节 hex>`）；
* 持有期间每 5 s 心跳（`Chtimes` 续期），因此活锁**不会**被判过期；只有崩溃进程才停止续期；
* `release` 先读回内容、nonce 匹配才删除；
* 心跳发现文件已不是自己的 nonce 就退出，绝不触碰别人的锁。

回归测试：`internal/lock/lock_owner_test.go`（owner 写入、外来锁不被删除、重复释放不炸）。

### 1.2 journal 路径白名单（P1，安全）

`internal/env/apply.go` 在回滚时直接使用 `record.Details["shim_dir"]` 与
`record.Details["shell_rc_path"]` 拼路径。journal 是 data root 下的普通文件，**任何能写它的人
都能让 `--apply` 往任意目录写文件、把任意文件当成 shell profile 重写**。

新增 `internal/env/rollback_guard.go`：

* shim 目录必须落在 data root 内；
* shell profile 只接受 `~/.profile` 与 `/etc/profile.d/onemoreffmpeg.sh`（后者仅 Linux）；
* 其余一律 `OMF-E7001`，`--apply` 与 dry-run 预览共用同一判断。

回归测试：`internal/env/rollback_guard_test.go`（root 内放行、越界/相对/空白拒绝、Windows 上
`/etc/...` 因非绝对路径被拒）。

### 1.3 `switchCurrent` 报错了错误的变量（P2）

`internal/install/current.go`：`current.pointer` 写失败时 `WithCause(err.Error())` 用的是
`ReplaceLink` 的旧错误，诊断指向了错误的操作。改为 `writeErr`。

### 1.4 只读命令不再创建 data root（P2）

`internal/cli/root.go` 的 `showFirstUseNotice` 无条件 `MkdirAll` + 写 `.first-use`，因此
`version`、`detect`、`doctor` 以及所有 `--dry-run` 都会写盘，与「dry-run 不写文件」的契约矛盾。

现在只有带 `--apply` 的调用才落标记（`writesSomething`），提示本身照常打印。
回归测试：`internal/cli/firstuse_test.go`。

### 1.5 journal 写失败会把旧安装留在 quarantine（P1）

`internal/install/install.go`：`quarantineExistingInstalls` 先把旧版本移进 `quarantine/`，随后写 journal；
写失败时 `restoreOnFailure` 这个闭包**还没赋值**（它在下一段才构建），于是旧版本永久滞留在 quarantine，
`current` 指向一个已经不存在的目录。

现在这条失败路径先把隔离目录搬回 `quarantine_original`，再返回错误。

### 1.6 下载失败会删掉并发进程的共享目录（P1）

`internal/download/download.go`：下载目录按 `sha[:16]` 命名，**两个进程下载同一资产会共享它**；
原清理逻辑 `os.Remove(targetDir)` 会把对方正在写的文件一起删掉。

现在只删本次写出的文件，再对目录做一次 `os.Remove` —— 非空时安全失败，空目录才顺带清掉。

### 1.7 `--authorize-system` 不能替代提权（P2）

`internal/platform/authz.go` 原来是 `elevated || consented` 就放行，于是「未提权 + `--authorize-system`
+ `--apply`」会走进写入流程，写 HKLM / `/usr/local/bin` 失败后以退出码 1 + failed journal 收场，
而不是参数错误 E6003。

现在按是否落盘区分：**dry-run** 允许 consent 预览（`TestEnvSystemDryRunWithConsent` 保持通过），
**`--apply`** 必须真正提权，否则 E6003 退出 2。`authz_test.go` 覆盖四种组合。

### 1.8 第二校验源缺失时不再静默降级（P2）

`internal/download` 的 `Options` 新增 `Warn` 回调：release 没有 checksums 文件时发一条警告
（`warning: no independent checksums file ... only the pinned sha256 was verified`），
`download` 与 `install` 两条命令都接上（stderr，不影响 JSON 与退出码）。

### 1.9 两处顺手修掉的小洞（P2-6 的一部分）

* `env.writeFileAtomic` 的最终替换改用 `platform.RenamePath`（Windows 上带共享冲突重试），
  不再用裸 `os.Rename`。
* `env` 的 apply / rollback 事务开头检查 `ctx.Err()`，取消不再被 `_ = ctx` 吞掉。

## 2. 文档修正

* `README.md` §2「当前范围」仍写着安装/下载/环境变量「计划中」、§5 又写「Phase 5 已完成」、
  段尾再写「未实现」——三处互相矛盾。§2 与 §5 已统一到「已实现」，只保留包管理器识别与
  Go API/插件为「明确推迟」。
* `docs/API_AND_PLUGIN.md` 的命令速查里 `--installation`（真名 `--installation-id`）与
  `--dry-run|--yes`（真名 `--apply`，默认即 dry-run）会直接触发 `OMF-E7001`，已改正。
* 本文件补记：`FULL_AUDIT_2026-11-14.md` §6.3 说「install / env / uninstall 没有全局
  data-root 事务锁」——该条已被 `internal/lock` 推翻（且它当时带着 §1.1 的缺陷）。以本文件为准。

## 3. 仍未修复（下一轮）

| 编号 | 问题 | 位置 |
|---|---|---|
| P2-5 | 解压边界：NTFS 备用数据流（ADS）、磁盘空间预检、压缩比上限、解压期 TOCTOU | `internal/archive` |
| P1-4a | `install --apply` 仍在**锁外**下载（下载耗时长，持锁会阻塞其它进程，这是有意的取舍；但失败清理已改为只动本次文件，见 §1.6） | `internal/cli/install.go` |

> P2-6a 核实后不成立：`restoreShellRC` 本来就走 `writeFileAtomic`，而它现在用
> `platform.RenamePath`（带 Windows 共享冲突重试），所以 `.bak` 恢复也是原子的。

其余已修：P1-3（§1.5）、P1-4 的共享目录清理（§1.6）、P2-1（§1.7）、P2-4（§1.8）、
P2-6 的 `RenamePath` 与 `context`（§1.9）。

P2-5 需要针对 zip/tar 的恶意样本（ADS 名、超大解压比、符号链接、`..`）建一组负向测试，
以及一次真实磁盘空间不足的模拟，建议单开一轮。
