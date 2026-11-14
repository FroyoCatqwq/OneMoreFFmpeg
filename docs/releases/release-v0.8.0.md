# OneMoreFFmpeg v0.8.0

> 2026-11-14 · SemVer 0.8.0
> Phase 6 对外契约冻结 + 解压安全加固 + 第三轮审查修复（shim 回滚 / 下载清理）。

## 亮点

- **Phase 6 契约冻结**：18 个命令、17 个 `/v1` data schema、退出码映射由 `internal/contracts/freeze_test.go` 守护，静默改动即 CI 失败。
- **外部调用示例**：`examples/external-caller/`（Python + PowerShell）仅消费 stdout JSON，不 import Go / 不链接 Go runtime。
- **解压安全加固（P2-5）**：拒绝 NTFS ADS/冒号、Windows 保留设备名、尾随点/空格别名、`.`/空路径分量；1000:1 压缩比上限防 zip/tar bomb；zip 解压前磁盘空间预检；齐全负向测试。

## 修复

- **Linux shim 回滚**（P1 回归）：白名单误把 `~/.local/bin` / `/usr/local/bin`（均在 data root 之外）判为越界，导致带 shim 的 `env rollback` 全部 E7001 失败；新增 `safeShimDir` 精确放行合法 shim 目录。
- **下载清理**（P1）：checksums/下载失败不再误删并发进程已校验的同名资产；清理 record 失败分支的重复代码。
- **首次使用标记**：`download` 与 `capabilities`（非 `--no-cache`）会写盘，现在正确落 `.first-use` 标记。
- 文档 reconcile：系统级授权（`--apply` 必须真正提权）、锁 owner/心跳、回滚白名单、只读不写盘等与代码对齐。

## 测试

- 单元/CLI 回归：shim 回滚（Linux 真实 apply→rollback）、下载共享目录保护、`safeShimDir`、`writesSomething`、archive 负向、契约冻结。
- e2e：pytest 17 / Robot 5 / Bats 5。
- 本地 Windows、fedora45、win2025server 三处 `go test` + e2e 全绿。

## 产物

- `one-more-ffmpeg-v0.8.0-windows-amd64.exe`
- `one-more-ffmpeg-v0.8.0-linux-amd64`
- `SHA256SUMS.txt`

校验：

```bash
sha256sum -c SHA256SUMS.txt
```

Windows PowerShell：

```powershell
Get-FileHash .\one-more-ffmpeg-v0.8.0-windows-amd64.exe -Algorithm SHA256
```

## 兼容性

- JSON schema 仍为 `onemoreffmpeg.*/v1`，字段只增不改；命令集与退出码已冻结。
- 无新增错误码；`OMF-E7001` 现也用于回滚路径白名单拒绝（语义不变：用法错误）。
- `doctor` 默认退出 0；`--strict` 不健康退出 1（v0.7.0 引入）。