# OneMoreFFmpeg v0.7.0

> 2026-11-14 · SemVer 0.7.0
> 并发事务锁、系统级作用域与显式提权授权、doctor 严格模式、Windows symlink 回退。

## 亮点

- **全局事务锁**：`install` / `uninstall` / `env set|unset|rollback` / `journal rollback` 的 `--apply` 路径持有 `<data-root>/.transaction.lock`，避免并发写交错；陈旧锁（>30s）自动回收，活跃锁等待超时返回新错误码 `OMF-E6009`（退出 1）。
- **系统级作用域**：`env set/unset --system` 写 Windows HKLM 系统环境或 Linux `/usr/local/bin` + `/etc/profile.d/onemoreffmpeg.sh`。
- **显式提权授权**：系统级操作需授权——以 root/Administrator 直接运行放行；否则须显式 `--authorize-system`，未授权返回 `OMF-E6003`。工具绝不自动 sudo/提权；非管理员被明确拒绝并告知。
- **doctor 严格模式**：`doctor --strict` 在 `healthy=false` 时返回 `OMF-E6006`（退出 1），报告置于 `details.report`；默认行为不变（退出 0）。
- **Windows symlink 回退**：current 指针回退链 junction → 目录 symlink → pointer_file；symlink 在开发者模式 / Win11 24H2 sudo / gsudo 下可用，`ReplaceLink` 返回真实 kind。新增 `scripts/elevate.ps1` 与 `scripts/enable-developer-mode.ps1`。
- **系统要求文档**：新增 [`SYSTEM_REQUIREMENTS.md`](../SYSTEM_REQUIREMENTS.md)——Windows 最低 Win10 1809 / 推荐 Win11 24H2；Linux 最低 glibc 2.28（RHEL8）/ 推荐 glibc 2.39（RHEL10、Debian 13+）。

## 修复与健壮性

- junction 替换名修正 UNC 目标（`\??\UNC\`）。
- env 失败撤销仅回滚已应用动作；shim 目标单引号转义；`env status` Windows 变量名大小写归一。
- 下载改用连接阶段超时（不再整体 cap body）；tar 解压器正确关闭。

## 测试

- 新增单元/CLI 回归：事务锁、授权门、env 系统作用域、doctor --strict、symlink 回退。
- e2e 扩充：pytest 17 项、Robot 5 项、Bats 5 项。
- 本地 Windows、fedora45、win2025server 三处 `go test` + e2e 全绿。

## 产物

- `one-more-ffmpeg-v0.7.0-windows-amd64.exe`
- `one-more-ffmpeg-v0.7.0-linux-amd64`
- `SHA256SUMS.txt`

校验：

```bash
sha256sum -c SHA256SUMS.txt
```

Windows PowerShell：

```powershell
Get-FileHash .\one-more-ffmpeg-v0.7.0-windows-amd64.exe -Algorithm SHA256
```

## 兼容性

- JSON schema 仍为 `onemoreffmpeg.*/v1`，字段只增不改。
- 新增错误码 `OMF-E6009`；`OMF-E6003` / `OMF-E6006` 语义在文档中明确（编号不变）。
- `doctor` 默认退出码不变；`--strict` 为可选新行为。