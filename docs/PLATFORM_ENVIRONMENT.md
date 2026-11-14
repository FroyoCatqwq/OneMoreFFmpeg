# 平台环境变量与 current 指针细则

> 状态：Phase 0 契约补充
> 日期：2026-11-10
> 配套：[`ENVIRONMENT.md`](ENVIRONMENT.md)、[`ERROR_CODES.md`](ERROR_CODES.md)、[`CLI_SCHEMA.md`](CLI_SCHEMA.md)

---

## 1. 目标

解决审计中的 P1-11：Windows PATH / 环境变量写入细节不足，以及 `current` 指针不能只假设 symlink。

本文档定义：

- Windows user PATH / machine PATH；
- `REG_SZ` / `REG_EXPAND_SZ`；
- `WM_SETTINGCHANGE`；
- `--system` 提权；
- Linux shell rc 与 shim；
- current 指针类型；
- journal / rollback；
- `env status` / `doctor` 检查项。

---

## 2. current 指针

`current` 不能只写 symlink。

v1 支持三种实现：

| 类型 | Windows | Linux | 说明 |
|---|---|---|---|
| `directory_junction` | 优先 | 不常用 | Windows 目录 junction，普通用户通常可创建 |
| `symlink` | junction 失败时回退（开发者模式 / sudo / gsudo） | 默认 | 目录 symlink |
| `pointer_file` | 回退 | 回退 | 写 `current.pointer` 指向版本化目录 |

junction 限制：目标必须已存在；本机路径写入 `\??\<path>` 替换名，UNC 网络路径（`\\server\share`）写入 `\??\UNC\server\share`，因此支持本机跨卷与 UNC 目标。Windows 回退链为 junction → 目录 symlink → `pointer_file`；symlink 需要开发者模式（Win10 1809+）、Win11 24H2 `sudo` 或 `gsudo`（见 `scripts/elevate.ps1`、`scripts/enable-developer-mode.ps1`）。`ReplaceLink` 返回实际创建的 kind。

目录布局：

```text
<data-root>/
├─ apps/<provider>/<release-tag>/<sha256-16hex>/
├─ current                  # junction/symlink，或不存在
├─ current.pointer          # pointer_file 模式
└─ journal/
```

规则：

- 切换版本前先写 journal；
- 只切换本项目创建的 current；
- 遇到同名用户文件或链接，返回 `OMF-E6007`，不删除、不覆盖；
- junction 创建失败时回退到 `pointer_file`，并记录 warning；
- `pointer_file` 内容只允许一个相对路径，UTF-8，单行，结尾 LF；
- `env status` 必须报告 current 类型、目标路径、是否有效；
- `doctor` 必须检查 current 目标是否存在且包含三件套。

---

## 3. Windows PATH

### 3.1 作用域

| 作用域 | 注册表位置 | 默认 |
|---|---|---|
| 用户 PATH | `HKCU\Environment` | 是 |
| 系统 PATH | `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment` | 否 |

`--system` 与 `--apply` 同用时必须**真正提权**；`--authorize-system` 只记录用户同意，
在 dry-run 预览里可单独使用，但不能替代提权。

```text
--apply + --system   -> 需要已提权
--authorize-system   -> 仅记录同意（dry-run 预览可用）
```

未提权时返回 `OMF-E6003`（退出 2），不静默调用 UAC。

### 3.2 注册表类型

读取与写回必须保留：

- `REG_SZ`；
- `REG_EXPAND_SZ`；
- 原有 `%VAR%` 展开语义。

禁止：

- 把 `REG_EXPAND_SZ` 粗暴转成 `REG_SZ`；
- 按字符串替换整个 PATH；
- 删除非本项目路径；
- 追加重复 `current/bin`；
- 忽略大小写导致重复项。

PATH 项处理：

- Windows 路径大小写不敏感；
- 比较前规范化结尾反斜杠；
- 使用 `filepath.Clean` 等价规范化；
- PATH 顺序固定，CLI 不提供 `--prepend` 之类的调整参数；
- 默认 append。

### 3.3 写入流程

```text
journal begin
  → 读取旧 PATH + 类型
  → 生成新 PATH
  → 写入注册表
  → 广播 WM_SETTINGCHANGE
  → journal commit
```

失败：

```text
journal rollback
  → 恢复旧 PATH 与旧类型
  → 删除本次新增项
  → 返回错误
```

`SendMessageTimeout` / 广播失败：

- 返回 `OMF-E6008`；
- 但必须保留“已写入但可能未对已启动进程生效”的 warning；
- 提示用户重新登录或重启相关进程。

### 3.4 正在运行的 ffmpeg.exe

Windows 不能可靠覆盖正在运行的 exe。

规则：

- 切换 current 前检测目标三件套是否被占用；
- 若占用，`platform.RenamePath` 做有限次退避重试；`OMF-E5006` 为**保留编号，当前无使用点**；
- 不强制 kill 用户进程；
- 可把旧目录移动到 quarantine，记录 `OMF-E5007` 风险；
- quarantine 目录必须由 journal 管理，不能自动无限期删除；
- 目录 rename（install 提交、quarantine 恢复、current 切换）可能被 Defender/索引器等短暂占用而返回 `ACCESS_DENIED` / `SHARING_VIOLATION`；`platform.RenamePath` 对这两类错误做有限次退避重试，非占用类错误立即失败。

---

## 4. Windows 环境变量

除 PATH 外，可写：

```text
ONE_MORE_FFMPEG_DIR
FFMPEG_DIR
LAE_FFMPEG_DIR
```

规则：

- 默认写用户作用域；
- `--system` 规则同 PATH；
- 不覆盖已有值；
- 旧值写入 journal；
- 变量值指向 `<data-root>/current` 或 `<data-root>/current/bin`；
- 不写 token / 密码；
- `GITHUB_TOKEN`、`HTTPS_PROXY`、`HTTP_PROXY` 只读。

---

## 5. Linux shim

默认使用：

```text
~/.local/bin/ffmpeg
~/.local/bin/ffprobe
~/.local/bin/ffplay
```

规则：

- 写入带托管标记（`# OneMoreFFmpeg managed shim`）的 `#!/bin/sh` 脚本，`exec` 到 `<data-root>/current/bin/*`；目标路径用单引号包裹并转义其中的单引号，含空格/特殊字符的路径也安全；
- 若 `~/.local/bin` 不存在，可创建；
- 若 `~/.local/bin` 不在 PATH，只 warning，不改 shell rc；
- 不写 `/etc/profile`；
- 不写 `/etc/environment`；
- 不调用 root；
- 卸载时按 journal 删除本项目创建的 shim；
- 不删除同名但非本项目创建的文件，返回 `OMF-E6007`；
- 写入失败时按相反顺序撤销已应用动作：原本存在的 shim 恢复原内容，新建的 shim 删除。

---

## 6. Linux shell rc

`--no-shell-rc`（bool）可跳过修改 shell rc。**没有**「指定任意 rc 路径」的参数：
写哪一份 profile 由平台规则与 `internal/env/rollback_guard.go` 的白名单决定
（`~/.profile` / `/etc/profile.d/onemoreffmpeg.sh`），这是防止 journal 被改写到任意路径的安全边界。

限制：

- 必须是用户拥有的普通文件；
- 默认拒绝 `/etc/*`；
- 写入前备份到 journal；
- 使用受控块：

```text
# BEGIN OneMoreFFmpeg managed block
export PATH="..."
# END OneMoreFFmpeg managed block
```

- 已存在旧块则替换旧块，不追加第二份；
- 写入或备份这类 IO 失败返回 `OMF-E5009`（退出 1）；`OMF-E6004` 只表示「策略拒绝修改 shell rc」（退出 2，保留编号）；
- 回滚时恢复原文件；
- 当前进程不会自动获得新 PATH，记录 `OMF-W1005`。

---

## 7. journal

每次 env / install 写入必须生成 journal：

```json
{
  "schema": "onemoreffmpeg.journal/v1",
  "id": "...",
  "created_at": "...",
  "operations": [
    {
      "type": "user_path_append",
      "scope": "user",
      "old_value": "...",
      "new_value": "...",
      "old_registry_type": "REG_EXPAND_SZ",
      "status": "pending"
    }
  ],
  "rollback_supported": true
}
```

规则：

- 先写 journal，再改系统；
- 每步成功后更新 status；
- 全部成功才 commit；
- 失败自动 rollback；
- rollback 失败返回 `OMF-E5005` 并保留现场；
- journal 不保存 token / 密码；
- `doctor` 必须报告 pending journal。

---

## 8. `env status` 检查

至少输出：

| 字段 | 说明 |
|---|---|
| `current_type` | directory_junction / symlink / pointer_file / missing |
| `current_target` | 目标版本化目录 |
| `path_contains_current` | PATH 是否包含 current/bin |
| `resolved_ffmpeg` | 实际解析到的 ffmpeg |
| `variables` | 关键变量摘要 |
| `warnings` | 当前进程未生效、PATH 未包含等 |

---

## 9. `doctor` 检查

至少检查：

- data root 可写；
- current 存在且有效；
- current 三件套可启动；
- PATH 包含 current/bin 或 `ONE_MORE_FFMPEG_DIR` 正确；
- 无 pending journal；
- 无重复 managed block；
- 无非本项目文件被标记为可删除；
- Windows `REG_EXPAND_SZ` 未被破坏；
- Linux shim 指向当前版本；
- 若使用 nonfree，显示 `OMF-W1006` 合规提示。

---

## 10. 禁止

- 静默提权；
- 静默修改 machine PATH；
- 静默修改 shell rc；
- 删除用户自建 current / shim；
- 覆盖正在运行的 Windows exe；
- 把 token 写进 journal；
- 在 `env status` 中假装新 PATH 对当前进程已生效；
- 不写 journal 就修改环境变量。
