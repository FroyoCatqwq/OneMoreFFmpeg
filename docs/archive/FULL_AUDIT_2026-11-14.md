# 全量审查与修复记录（2026-11-14）

> 环境：本地 Windows（PowerShell 7 + coreutils + mise go 1.27 / python 3.14 + uv）；
> 远程 CI：`Linux-Server-Info.txt` 指向的 fedora45 与 `Windows-Server-Info.txt` 指向的 win2025server。
> 方法：用户运行视角审计 + 代码视角审计 + 逐条核实 + 回归测试 + 本地/双机 e2e + 文档/契约一致性检查。

---

## 1. 结论

本轮从「用户运行视角」和「代码视角」两条线对 Phase 1–5 全部 CLI 与内部包做全量审查，核实并修复 14 项缺陷（F1–F14），覆盖事务回滚、错误码契约、Windows 平台健壮性、环境写入、下载/解压、诊断提示六个方面，并补齐回归测试与文档。

项目当前状态：Phase 1–5 核心闭环可用，写入路径默认 dry-run，install/uninstall/env 事务可回滚且回滚预览也先做校验；本地 Windows、fedora45、win2025server 三处 Go 单测与 e2e 全绿。

---

## 2. 关键修复

### 2.1 事务与回滚（F1 / F1b / F5）

- `journal rollback` 的 dry-run 预览以前跳过校验、直接构造空动作：现在 dry-run 与 `--apply` 都先解析 data root、读取记录、检查 `rolled-back` 状态并校验 action 白名单（`install` / `uninstall` / `env-set` / `env-unset`）。预览的 `action` 取自记录本身。
  - journal id 非法（含 `..` / 路径分隔符）、记录不存在或读取失败 → `OMF-E6005`（退出 1）；
  - 记录已回滚或 action 不在支持列表 → `OMF-E7001`（退出 2）。
- `env rollback` 同样先读记录：指向非 env 记录或已回滚记录 → `OMF-E7001`，id 非法/缺失 → `OMF-E6005`；`env.RollbackWithStore` 拒绝已回滚记录，避免重复撤销。
- install 幂等快路径的 `writeRecord` 错误以前被 `_ =` 吞掉：现在向上传播，避免 install.json 写失败却报告成功。

### 2.2 错误码契约（F2）

- `capabilities --only <kind>` 的 kind 不在支持列表时，以前误用 help topic 的 `OMF-E2005`：现在返回 `OMF-E7005`（非法枚举值，退出 2）。`OMF-E2005` 语义收窄为仅 `-h <topic>` 校验。

### 2.3 Windows 平台健壮性（F3 / F7 / F13）

- 新增 `platform.RenamePath`：Windows 上对 `ERROR_ACCESS_DENIED` / `ERROR_SHARING_VIOLATION`（Defender/索引器短暂占用目录）做最多 5 次退避重试，其它错误立即失败；非 Windows 直接 `os.Rename`。install 提交、quarantine 隔离/恢复、current 切换、install/uninstall rollback 的全部目录 rename 改用该函数，消除 win2025server 上偶发的 `OMF-E5005` / `OMF-E5004` rename 失败。
- junction 替换名修正：UNC 目标（`\\server\share`）写入 `\??\UNC\server\share`，本机路径写入 `\??\<path>`；此前 UNC 会生成非法的 `\??\\server\share`。
- `uninstall` 判断 current 是否指向被删目录改用 `samePath`：Windows 大小写不敏感比较，避免路径大小写差异导致漏判。

### 2.4 环境写入（F10 / F11 / F12）

- Linux shim 的目标路径用单引号包裹并转义其中的单引号（`shellQuote`），含空格/特殊字符的安装路径不再破坏 shim 脚本。
- env 计划中每个动作携带 `undo` 闭包；写入失败时按相反顺序撤销「已应用」的动作（PATH/变量恢复原值、新建 shim 删除、原有 shim 恢复原内容、shell rc 恢复备份），不再残留半完成状态。
- `env status` 变量名在 Windows 大小写不敏感归一（`envNameKey`）：进程环境中的 `ffmpeg_dir` 等小写写法也按规范名 `FFMPEG_DIR` 报告。

### 2.5 下载与解压（F14 / F6）

- `download` 默认 HTTP 客户端改用阶段超时（dial / TLS 握手 / 响应头各受 `Timeout` 约束），去掉整体 `http.Client.Timeout`：大资产在慢链路上持续读取 body 不再被整体超时中断；保留 `ProxyFromEnvironment` 与「重定向目标必须 https」校验。
- `extractTar` 现在关闭 gzip/zstd 解压器（`defer closeReader()`），避免 reader 资源泄漏。

### 2.6 诊断与提示（F8 / F9）

- `doctor` 的 trio 检查新增 ffplay 探测：缺失或不可运行记录 `OMF-W1001`（不影响 ffmpeg/ffprobe 判定），三者均健康时 detail 为 `ffmpeg, ffprobe, ffplay healthy`；报告合并 `status.Warnings` 与 trio 警告。
- `provider add` 自定义 URL 格式错误消息明确列出受支持的归档格式（zip / tar.xz / tar.gz / tar.zst）。

### 2.7 测试稳定性（F4）

- `TestCommandTimeoutOverrideAppliesToEveryCommand` 移除墙钟耗时断言（在慢 CI 上 flaky），保留「超时覆盖 applied 到每个命令 → `OMF-E2001`」的语义断言。

---

## 3. 新增/更新的回归测试

- `internal/cli/rollback_dryrun_test.go`（新增）：journal/env rollback 的未知 id、`..`  traversal → `E6005` 退出 1；预览 action 取自记录；已回滚/不支持 action/非 env 记录 → `E7001` 退出 2。
- `internal/capabilities`：`--only` 不支持 kind → `E7005`；移除 flaky 墙钟断言。
- `internal/doctor`：trio 健康（含 ffplay）与 ffplay 缺失告警（`W1001`）两用例。
- `internal/env`：`shellQuote` 单引号/展开转义、含单引号目标的 shim；失败撤销「无动作应用」与「仅撤销已应用动作」；Windows 变量名大小写不敏感。
- `internal/download`：慢 body（约 480ms）在 250ms 阶段超时下仍完成。
- `internal/platform`：`junctionSubstituteName` 本机/UNC 替换名（Windows）。

---

## 4. 验证

实际结果：

- 本地 Windows：`go build ./...` / `go vet ./...` 通过；`go test ./...` 全绿；pytest e2e `12 passed`。
- fedora45：`go test -count=1 ./...` 全绿；pytest e2e `12 passed`；Robot `3 passed`；Bats `3 passed`。
- win2025server：`go build` / `go vet` 通过；`go test -count=1 ./...` 全绿（此前 flaky 的 `TestCommandTimeoutOverrideAppliesToEveryCommand` 与受 Defender 占用影响的 `TestInstallRollbackRestoresPreviousVersion` 均通过）；pytest e2e `12 passed`；Robot `3 passed`。
- 四个用户视角复现确认修复：journal rollback 未知 id / `../evil` → `OMF-E6005` 退出 1；env rollback 未知 id → `OMF-E6005` 退出 1；`capabilities --only bogus` → `OMF-E7005` 退出 2。
- Windows/Linux 交叉编译通过；`internal/contracts` 示例契约校验通过。

验证命令：

```text
go vet ./...
go test -count=1 ./...
GOOS=windows GOARCH=amd64 go build ./...
GOOS=linux GOARCH=amd64 go build ./...
本地 Windows: pytest e2e
fedora45: go test + pytest e2e + Robot Framework + Bats
win2025server: go build/vet/test + pytest e2e + Robot Framework
```

---

## 5. 文档同步

- `ERROR_CODES.md`：`W1001` 扩为「ffplay 缺失或不可运行」；补充 `E2005` 仅限 help topic、`--only` 非法 kind 用 `E7005`；补充 journal id 校验 → `E6005`、已回滚/非适用记录 → `E7001`。
- `CLI_SCHEMA.md`：`capabilities --only` 非法 kind → `E7005`；`journal rollback` / `env rollback` 的 dry-run 先校验语义与错误码。
- `CAPABILITIES.md`：错误语义表新增 `--only` 非法 kind 行。
- `PLATFORM_ENVIRONMENT.md`：junction 支持本机跨卷与 UNC（替换名前缀）；pointer 文件名统一为 `current.pointer`；Linux shim 改为带托管标记的 `#!/bin/sh` 脚本（单引号转义）与失败撤销；Windows 目录 rename 退避重试。
- `ENVIRONMENT.md`：Windows 变量名大小写不敏感归一。
- `TECHNICAL_NOTES.md`：下载阶段超时语义（不设整体 body 超时）。
- `schemas/examples/doctor.example.json`：补 `data_root` 检查、path detail 与代码一致（`current/bin found in PATH`）、trio detail 含 ffplay。
- `man/one-more-ffmpeg.1`：journal rollback 校验语义。
- `docs/README.md`：索引新增本报告。

---

## 6. 仍保留的边界

1. 本地 Windows 沙箱创建 junction 受限（开发者模式下 symlink 可用）：doctor 测试用 `ensureCurrentLink` 兜底重建链接，主机两者都不支持时跳过；属环境限制，非代码缺陷。
2. GitHub releases 只取前 100 条；`--days` 很大时仍可能受第一页限制，未实现 Link 头分页。
3. ~~install / env / uninstall 没有全局 data-root 事务锁~~ —— 该条已被后续提交的 `internal/lock` 推翻；
   该锁本身的语义缺陷（30 s 被判定过期、释放时删除他人锁）见
   [`FULL_AUDIT_2026-11-14b.md`](FULL_AUDIT_2026-11-14b.md) §1.1。
4. Linux 只维护 `~/.profile`，不处理 fish/zsh 专属 rc；不改系统级环境、不提权。
5. `doctor` 默认退出 0，即使 `healthy=false`；脚本必须读取 `data.healthy`。
6. `download` 与 capabilities 缓存会立即写文件，不属于默认 dry-run 命令。