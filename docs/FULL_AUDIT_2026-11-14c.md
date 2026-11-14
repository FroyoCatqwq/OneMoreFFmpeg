# 全量审查与修复记录（2026-11-14 · 第三轮）

> 承接 [`FULL_AUDIT_2026-11-14.md`](FULL_AUDIT_2026-11-14.md)（F1–F14）与 [`FULL_AUDIT_2026-11-14b.md`](FULL_AUDIT_2026-11-14b.md)（deepseek 第二轮）。
> 本轮在 Phase 6 之后再做代码 + 文档全量审查，重点复核第二轮引入的并发/回滚/下载改动，修掉两处真实 bug 并补回归测试。
> 环境：Windows 本地 go1.27.4；`go build`/`go vet`/`go test ./...` 全绿（22 包）；fedora45 + win2025server e2e 全绿。

## 1. 已修复

### 1.1 Linux shim 回滚被白名单误拒（P1，回归）

`internal/env/rollback_guard.go` + `internal/env/apply.go`。第二轮 §1.2 的 `safeRollbackPath` 假设「用户级 shim 落在
`<data-root>/bin`」，**前提错误**：用户级 shim 实际在 `~/.local/bin`、系统级在 `/usr/local/bin`，两者都在 data root 之外。
于是回滚 env-set 记录时 `safeRollbackPath(shimDir, dataRoot)` 恒为 false，**所有带 shim 的 Linux `env rollback` /
`journal rollback` 都以 `OMF-E7001` 失败**（e2e 此前未真正回滚过用户级 shim，故未暴露）。

修复：新增 `safeShimDir`，仅接受 `shimDirectory()` 会产生的两个目录（`~/.local/bin`、`/usr/local/bin`）；
回滚 shim 改用 `safeShimDir`，profile 仍用 `safeShellProfile`/`safeRollbackPath`。既放行合法 shim，又拒绝篡改 journal 指向任意目录。
回归测试：`internal/env/rollback_shim_test.go`（真实 apply→rollback 用户级 shim，Linux）、`rollback_guard_test.go` 增 `safeShimDir` 用例。

### 1.2 下载失败清理会误删并发进程的资产（P1）

`internal/download/download.go`。`Fetch` 的清理有三处问题：
- checksums 校验失败发生在**下载之前**，本次调用尚未写任何文件，却 `os.Remove(finalPath)`——会删掉并发进程已校验好的同名资产；
- 下载失败时 `download` 只写 `.part` 临时文件、成功才 commit 到 `finalPath`，失败路径同样不应删 `finalPath`；
- record 失败分支的清理代码被重复粘贴（`finalPath`/`targetDir` 各删两次、注释交错）。

修复：checksums/下载失败路径只 tidy 空目录（`os.Remove(targetDir)` 非空安全失败），不动 `finalPath`；
record 失败仅在 `downloaded=true`（本次确实写出）时删 `finalPath` + tidy 目录，去掉重复块。
回归测试：`TestFetchChecksumsFailureKeepsExistingAsset`（预置并发资产，checksums 失败后仍在）。

### 1.3 首次使用标记漏记会写盘的命令（P3）

`internal/cli/root.go`。`writesSomething` 只看 `--apply`，但 `download` 立即下载、`capabilities` 默认写缓存，
二者不带 `--apply` 也写盘，却不落 `.first-use` 标记（提示会重复打印）。修复：`download` 恒为 true，
`capabilities` 除 `--no-cache` 外为 true。`firstuse_test.go` 增对应用例。

## 2. 复核确认无问题

- `internal/lock`：owner nonce + 5s 心跳 + 校验式释放；回收竞争由 `O_CREATE|O_EXCL` 兜底，仅一个进程能抢到；
  释放/心跳都先校验 nonce，不碰别人的锁。keepFresh 与 release 之间存在极小的 Chtimes 竞态，仅可能短暂延长他人活锁，无正确性危害。
- `internal/install`：quarantine 后 journal 写失败的内联回搬（第二轮 §1.5）正确；`restoreOnFailure` 由 `committed` 门控且幂等，defer 重复调用无害。
- 本轮 Phase 6 的 archive 加固（ADS/保留名/压缩比/磁盘预检）与契约冻结测试均通过。

## 3. 文档同步

- `SYSTEM_REQUIREMENTS.md` §4：shim 白名单改为「`~/.local/bin` / `/usr/local/bin`（在 data root 之外）」；first-use 标记补 download/capabilities。
- 本文件补记：第二轮 §1.2 的 shim 白名单描述以本轮 §1.1 为准。

## 4. 仍保留（下一轮 / Phase 7）

- 解压期 TOCTOU 仅靠 `O_EXCL`+空目录+拒 symlink 缓解，未做 openat2 级硬化。
- P1-4a：install --apply 锁外下载（有意取舍）。
- Phase 7：gofmt/govulncheck 门禁、fuzz、真实上游集成、release 自动化 + 签名/provenance。