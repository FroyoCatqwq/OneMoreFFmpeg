# 系统要求与权限模型

> 状态：v1
> 日期：2026-11-14
> 适用：OneMoreFFmpeg CLI（Go 1.27，`CGO_ENABLED=0` 静态二进制）

---

## 0. 全盘扫描权限（v1.1.2 起）

`detect --scan` 在未指定 `--roots` 时使用默认全盘根（Linux `/`、`/home`、`/opt`、`/usr/local`；Windows 所有固定盘符），`detect --dir <volume-root> --recursive` 从文件系统根递归遍历。这类**全盘扫描要求 elevated 进程**：

| 平台 | 可接受的运行身份 |
|---|---|
| Linux | `root`（euid 0），或 `sudo one-more-ffmpeg detect --scan ...` |
| Windows | 提升的 Administrator 令牌，或 `SYSTEM` 账户 |

普通权限下执行全盘扫描返回 `OMF-E6003`（`SYSTEM_SCOPE_DENIED`），JSON `details` 含 `elevated:false` 与 `scan:true`，并在消息中提示用 sudo/管理员重新运行。这样可以避免普通用户得到大量"权限拒绝"导致的**不完整且误导**的扫描结果。

**不受影响**：指定具体目录的扫描（`--dir <普通目录>`、`--scan --roots <dir1,dir2>`）不需要提权，普通用户可正常使用。

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
  - **未提权 + dry-run**：传 `--authorize-system` 可预览将发生的改动；未传则 `OMF-E6003`（退出 2）；
  - **未提权 + `--apply`**：一律 `OMF-E6003`（退出 2）。`--authorize-system` 只记录用户同意，**不能替代提权**——写 HKLM / `/usr/local/bin` 必须由真正提权的进程执行，否则会产生半写入的系统作用域。
- 非管理员请求系统级操作时明确拒绝并告知；具备管理员权限但未直接以超户运行的用户，`--apply` 同样必须先提权（sudo/管理员）。
- 例：把 FFmpeg 全局安装到 `/root` 等特权目录，必须用 `sudo`（此时进程为 root，直接放行）。

---

## 3. 事务并发

`install` / `uninstall` / `env set|unset|rollback` / `journal rollback` 的 `--apply` 路径持有全局 data root 事务锁 `<data-root>/.transaction.lock`，避免并发写互相交错。锁是 **OS 级 advisory lock**（Linux `flock` / Windows `LockFileEx`），绑定在打开的文件描述符上：持有进程崩溃时由内核自动释放，不存在"陈旧锁回收"的竞态，锁文件也不会被删除。活跃锁等待超过 5 秒返回 `OMF-E6009`（退出 1）。dry-run 预览不加锁。

## 4. 回滚路径白名单与只读不写盘

- `env rollback` 从 journal 读取 shim 目录与 shell profile 路径时会做白名单校验（`internal/env/rollback_guard.go`）：shim 目录仅接受 `~/.local/bin`（用户级）与 `/usr/local/bin`（系统级）——两者都在 data root **之外**，故不能用 data-root 包含判断；shell profile 仅接受 `~/.profile` 与 `/etc/profile.d/onemoreffmpeg.sh`（后者仅 Linux）；其余/相对/空白一律 `OMF-E7001`。这防止有人篡改 journal 文件后让回滚往任意目录写文件。
- 程序启动**不打印任何提示**，也不写任何标记文件（`.first-use` 已移除）。只读命令（`version` / `detect` / `doctor` 等）与 install/env/uninstall 的 dry-run 完全不写盘；只有带 `--apply` 的命令、`download` 与 `capabilities`（写缓存，`--no-cache` 除外）才会落文件。