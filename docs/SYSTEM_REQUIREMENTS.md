# 系统要求与权限模型

> 状态：v1
> 日期：2026-11-14
> 适用：OneMoreFFmpeg CLI（Go 1.27，`CGO_ENABLED=0` 静态二进制）

---

## 1. 操作系统

### Windows

| 级别 | 版本 | 说明 |
|---|---|---|
| 最低 | Windows 10 1809（build 17763） | x64；可运行静态二进制 |
| 推荐 | Windows 11 24H2 及以上 | 内置 `sudo`，开发者模式完善 |

current 指针优先使用目录 **junction**（普通用户通常无需特权）。junction 创建被拒绝时回退到目录 **symlink**，再回退到 `current.pointer` 文件。symlink 在以下任一条件下可用：

- 开启**开发者模式**（Windows 10 1809+ / Windows 11）；
- Windows 11 24H2+ 内置 `sudo`（在开发者选项中启用）；
- 旧版本 Windows 安装 `gsudo`。

辅助脚本（见 `scripts/`）：

- `scripts/elevate.ps1 <command> [args...]`：用 `sudo` → `gsudo` → RunAs（UAC）依次尝试提权运行命令；这是沙箱受限时的**手动授权**步骤。
- `scripts/enable-developer-mode.ps1`：启用开发者模式（写 HKLM，需提权运行），使 symlink 无需提权即可创建。

### Linux

| 级别 | 要求 | 说明 |
|---|---|---|
| 最低 | glibc ≥ 2.28（RHEL 8 系列） | Go 与 FFmpeg 共同支持的发行版 |
| 推荐 | glibc ≥ 2.39（RHEL 10、Debian 13 及以上） | 更新的内核与工具链 |

二进制为 `CGO_ENABLED=0` 静态链接，原则上只要 Go 与 FFmpeg 共同支持的系统即可运行；上表给出经过验证的最低/推荐基线。

---

## 2. 权限与授权模型

OneMoreFFmpeg **绝不自动提权**（不会静默调用 `sudo` / `runas`）。

- 默认所有写入都在用户级 data root 与用户环境（Windows `HKCU\Environment`、Linux `~/.local/bin` + `~/.profile`），普通用户即可完成。
- 系统级操作（`env set/unset --system`：Windows `HKLM` 系统环境、Linux `/usr/local/bin` + `/etc/profile.d/onemoreffmpeg.sh`）需要授权：
  - 进程**已提权**（Linux `root` / euid 0；Windows Administrator 提升令牌）→ 视为已授权，可直接操作；
  - 其他情况必须由用户**显式授权**（传 `--authorize-system`）；未授权返回 `OMF-E6003`（退出 2）并告知用户用 `sudo` / 管理员身份运行。
- 非管理员请求系统级操作时明确拒绝并告知；具备管理员权限但未直接以超户运行的用户，同样需要显式授权。
- 例：把 FFmpeg 全局安装到 `/root` 等特权目录，必须用 `sudo`（此时进程为 root，直接放行）。

---

## 3. 事务并发

`install` / `uninstall` / `env set|unset|rollback` / `journal rollback` 的 `--apply` 路径持有全局 data root 事务锁 `<data-root>/.transaction.lock`，避免并发写互相交错。锁超过 30 秒视为陈旧（进程崩溃）自动回收；活跃锁等待超过 5 秒返回 `OMF-E6009`（退出 1）。dry-run 预览不加锁。