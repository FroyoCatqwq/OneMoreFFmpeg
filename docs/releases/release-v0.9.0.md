# v0.9.0（2026-11-14）

> Phase 1–6 收口轮：按阶段 DoD 全量复核，修掉前几轮登记的全部 5 项缺陷，并补上契约与压缩边界的缺口。
> 产物：Windows x64 / Linux x64 可执行文件（纯 Go，无 CGO）。

## 修复

| 编号 | 问题 | 修复 |
|---|---|---|
| P2-4 | 压缩炸弹比例检查依赖 `os.Stat`，失败时 `compressed=0` 让检查永久空转；tar 无磁盘预检 | `Stat` 失败即拒绝（`OMF-E5009`）；tar 每个条目写前做 best-effort 空间预检；zip 仍用 central directory 精确预检 |
| P2-5 | `install --json` dry-run 输出 `journal_id: ""`，`env-set`/`uninstall` 省略；复用分支还带上次 apply 的过期 id | `model.InstallData.JournalID` 加 `omitempty`；复用分支清空；`contracts.validateInstall` 要求 apply 后必须有 id、且 `dry_run` 必须是 bool |
| P2-3 | `env` 事务的撤销失败被 `_ =` 丢弃，CLI 只报原始错误，用户不知道 PATH/shim/profile 半改 | `errors.Join` 合并全部撤销错误；写进 journal 的 `undo_error`、追加到 `Cause`；原始错误仍排第一，`Code` / 退出码 / `Details` 不变 |
| P2-2 | `journal.List` 对读失败与 JSON 解析失败都 `continue`，损坏记录从 `journal list` / `doctor` 消失 | 返回 `OMF-E6005` + 文件名；`doctor` 把它转成 `journals_readable: false` 检查项，`--json` 仍是完整可解析报告 |
| P1-4a | 下载仍在锁外（有意取舍），但共享缓存文件在「校验」与「解压」之间有 TOCTOU 窗口 | 锁内把 archive 复制到 `journalID` 私有的临时文件，从副本解压；下载保持在锁外，不阻塞其它写命令 |

另外：`contracts.validateInstall` 现在会拒绝非布尔 `dry_run`；新增 3 组回归测试（journal 损坏可见性、install `journal_id` 省略、契约类型）。

## 一轮中修正过的判断

- 首次尝试用 `compressed * maxRatio` 作为磁盘预留预算，在 Linux 上把普通几 MB 档案误判为需要数 GB → **撤回**，改为逐条目声明大小预检。
- 「目录名 `*.json`」被 `List` 的 `IsDir()` 正确跳过，不是缺陷，删掉那条误写的测试。

## 验证

| | 本地 Windows | Fedora 45 | Win2025 Server |
|---|---|---|---|
| `go build` / `go vet` | 0 / 0 | 0 / 0 | 0 / 0 |
| `go test ./...` | 22 包全绿 | 22 包全绿 | 22 包全绿 |
| pytest e2e | — | 17 passed | 17 passed |
| Robot / Bats | — | 5 / 5 | — |

## 仍未做

`contracts.validateCapabilities` 只校验字段存在不校验类型；GitHub release 列表仍不分页（最多 100 条）；
没有 `--select` / provider 优先级；`env set --apply` 因会写真实 profile / 注册表，e2e 只覆盖 dry-run；
Phase 7 的质量门禁、fuzz、真实上游资产集成与 release 签名。
