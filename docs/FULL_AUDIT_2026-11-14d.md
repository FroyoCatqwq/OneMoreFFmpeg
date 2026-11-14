# 第三轮深度审查与修复（2026-11-14）

> 承接 [`FULL_AUDIT_2026-11-14.md`](FULL_AUDIT_2026-11-14.md)（F1–F14）与
> [`FULL_AUDIT_2026-11-14b.md`](FULL_AUDIT_2026-11-14b.md)（锁/回滚/提权/校验源）。
> 本轮先按要求移除首次使用提示，再针对更隐蔽的缺陷做一轮审计与修复。
> 环境：Windows 本地 `go1.27.4`；`go build` / `go vet` / `go test ./...` 全绿（22 包）。

## 1. 首次使用提示已移除

`internal/cli/root.go` 的 `showFirstUseNotice`、`writesSomething` 与 `internal/cli/usage.go` 的
`firstUseNotice` 全部删除（含 `internal/cli/firstuse_test.go`）：

* 程序启动**不打印任何提示**，也不再写 `<data-root>/.first-use`；
* 同样的内容仍在 `--in-detail-usage` 与 `docs/HOWTO.md`，需要时主动查看；
* 顺带修掉一个真问题：`--json --in-detail-usage` 会把纯文本写进 stdout，任何调用方都无法解析。
  现在二者互斥，冲突时输出 `OMF-E1001` 的**合法 error envelope**（stdout 仍是 JSON）。

文档同步：`docs/CLI_SCHEMA.md`、`docs/SYSTEM_REQUIREMENTS.md` §4、`docs/USAGE.md` 的对应段落已改。

## 2. P1：`--force` 安装失败会删掉刚恢复的旧版本（数据丢失）

`internal/install/install.go` 的恢复路径被**执行两次**：`fail()` 主动调用 `restoreOnFailure()`，
而 `defer func(){ restoreOnFailure() }()` 在任何 return 时再跑一次。第二次 `committed` 仍为 false，
于是：

1. 第一次：`RemoveAll(targetDir)`（空）→ `RenamePath(quarantine → restoreTarget)`，旧安装**恢复成功**；
2. 第二次：`RemoveAll(targetDir)` → **删掉刚恢复的旧安装**，随后 `Stat(quarantine)` 已 ENOENT，不再恢复。

净效果是用户的上一版被静默删除，而 journal 仍指向一个空 quarantine，连 `journal rollback` 也救不回。
触发条件很普通：`--force` 覆盖安装时解压失败、缺 ffprobe、校验超时或 journal 写失败。

修复：恢复闭包加 `restored` 幂等标记（`if committed || restored { return }`），两半各执行一次。

> 现有 `TestInstallFailureRestoresQuarantinedVersion` 覆盖的是「新旧 target 目录不同」的路径，
> 同目录 `--force` 的失败场景难以在单测里构造（同一 zip 不会在解压阶段失败），
> 该修复由代码审查确认，已在本文档记录。

## 3. P1：锁心跳刷新失败被静默忽略

`internal/lock/lock.go` 的 `keepFresh` 只写 `_ = os.Chtimes(path, now, now)`。当杀毒/索引器短暂
独占锁文件时 `Chtimes` 会失败，mtime 停止前进 → 30 秒后另一个进程判定「过期」并回收 →
**互斥被静默打破**，两个进程同时写 `apps/` / `current` / `journal/`。

修复：`Chtimes` 失败时回退为「重写同样的 nonce 内容」（同样刷新 mtime，且走的是能绕开
`Chtimes` 限制的路径）；两者都失败说明文件已不可写，心跳退出交给下一轮 `owner` 检查。

## 4. P2：`OMF-E5005` 被当成通用写失败复用

`E5005` 的注册名是 `ROLLBACK_FAILED`，实际有 **22 处**非回滚场景（解压、复制、写 `install.json`、
建目录）在用它，调用方无法从码值区分「回滚失败」和「磁盘满了」。

修复：新增 `OMF-E5009 IO_FAILED`（exit 1），把 4 个文件里的非回滚用途全部改过去；
`E5005` 只在 `internal/install/rollback.go` 的 4 处真正回滚失败处保留。

同时清理两个码的语义：

* `E6004 SHELL_RC_DENIED` 原本把「备份/读写 shell rc 失败」也归进来（exit 2，像用法错误）。
  这 7 处 IO 失败改用 `E5009`；`E6004` 只表示「策略拒绝」，退出码 2 保持。
* `E5006 EXECUTABLE_BUSY` 在全仓**没有任何使用点**（`RenamePath` 改为退避重试后不再需要）。
  编号按「不得改变已有编号语义」保留，注册表与文档都标注「保留、当前无使用点」。

`internal/contracts/freeze_test.go` 的退出码冻结集已加入 `E5009`。

## 5. 仍未修（下一轮）

| 编号 | 问题 |
|---|---|
| P2-2 | `journal.List` 对读失败与 JSON 解析失败都 `continue`，`journal list` / `doctor` 看不到损坏记录；修它要动 `model.JournalListData`（加 warnings），属于 Phase 6 契约 |
| P2-3 | env 事务撤销阶段的失败被丢弃，只报原始错误 |
| P2-4 | 压缩炸弹比例检查依赖 `os.Stat`，失败时静默失效；tar 路径没有磁盘空间预检 |
| P2-5 | `install --json` 的 dry-run 输出 `journal_id: ""`，而 `env-set` / `uninstall` 直接省略该字段 |

这几项都不影响默认 dry-run 路径；P2-2/P2-5 建议与 Phase 6 的 schema 稳定化一起做。
