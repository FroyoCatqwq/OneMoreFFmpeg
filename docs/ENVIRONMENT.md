# 环境变量与 PATH 管理

> 状态：Phase 0 设计
> 原则：默认只读、显式写入、先备份后修改、可回滚
> 当前不做包管理器识别与包管理器共存逻辑

---

## 1. 管理的变量

| 变量 | 用途 | 写入方 |
|---|---|---|
| `PATH` | 让 `ffmpeg` 在 shell 中可见 | 仅显式 `env set` |
| `ONE_MORE_FFMPEG_DIR` | 本项目首选 trio 根目录 | 仅显式 `env set` |
| `FFMPEG_DIR` | 通用 FFmpeg 根目录 | 可选写入，需显式 |
| `LAE_FFMPEG_DIR` | 与 LAE / VideoDamage 兼容 | 默认不写，除非 `--lae-compat` |
| `GITHUB_TOKEN` | GitHub API 可选 token | 只读，不持久化 |
| `HTTPS_PROXY` / `HTTP_PROXY` | 网络代理 | 只读，不持久化 |
| `ONE_MORE_FFMPEG_HOME` | data root 覆盖 | 只读，本项目不写 |

不覆盖用户已有变量。

Windows 环境变量名不区分大小写：`env status` 按规范名（`FFMPEG_DIR` / `ONE_MORE_FFMPEG_DIR` / `LAE_FFMPEG_DIR`）报告，进程环境中诸如 `ffmpeg_dir` 的小写写法也会被识别并归一到规范名。

默认作用域为用户级。`env set/unset --system` 切换到系统级（Windows HKLM 系统环境；Linux `/usr/local/bin` + `/etc/profile.d/onemoreffmpeg.sh`）。授权：已提权（root/Administrator）直接放行；未提权时 dry-run 可加 `--authorize-system` 预览，但 `--apply` 必须真正提权（consent 不能替代提权），否则 `OMF-E6003`。工具绝不自动提权。权限模型与系统要求见 [`SYSTEM_REQUIREMENTS.md`](SYSTEM_REQUIREMENTS.md)。

### 1.1 data root 解析

所有下载、缓存、安装、journal 与默认配置都位于同一个 data root，解析顺序固定：

1. `ONE_MORE_FFMPEG_HOME`：必须为绝对路径，相对路径直接 `OMF-E7007`，不做“相对当前目录”猜测；
2. Windows：`%LOCALAPPDATA%\OneMoreFFmpeg`；
3. Linux：`$XDG_DATA_HOME/onemoreffmpeg`（仅接受绝对路径），否则 `~/.local/share/onemoreffmpeg`。

目录布局：

```text
<data-root>/
├─ downloads/
├─ cache/
│  └─ capabilities/
├─ apps/<provider>/<release-tag>/<sha256[:16]>/
├─ current -> apps/<...>
├─ current.pointer          # Windows 无 symlink 权限时的回退
├─ journal/<id>.json
├─ tmp/<journal-id>/        # 安装临时目录
├─ config.toml
└─ providers.toml
```

规则：

- `internal/paths` 是唯一解析入口，命令不得自行拼接平台路径；
- `--cache-dir`（CLI）优先于默认值；下载目录由 `[download] dir` 覆盖 `<data-root>/downloads`；
- data root 不可判定时返回 `OMF-E9003`，不静默回退到当前工作目录；
- Windows 不使用 `XDG_*`；Linux 不读 `LOCALAPPDATA`；
- 配置：显式 `--config` > `<data-root>/config.toml`（存在时）> 内置默认值，详见 `CLI_SCHEMA.md` §3.1；
- 自定义 provider：`<data-root>/providers.toml`，由 `provider add` 写入，`provider list` / `resolve` 读取；
- `[network] github_api` 仅允许 https；例外：`localhost` / `127.0.0.1` / `::1` 的 http 允许用于测试与自建镜像。

---

## 2. 推荐模型：current 指针

安装目录使用版本化目录，不每次改 PATH：

```text
<data-root>/
├─ apps/<provider>/<release-tag>/<sha256-16hex>/
└─ current（symlink / Windows junction / directory pointer） -> apps/<...>
```

PATH 只加入一次：

```text
<data-root>/current/bin
```

或环境变量只指向一次：

```text
ONE_MORE_FFMPEG_DIR=<data-root>/current
```

切换版本时只改 `current`，不改 PATH。

优点：

- 回滚简单；
- 不污染系统环境变量；
- 不会 PATH 越来越长；
- 多版本共存；
- 与 LAE / VideoDamage 的 `--ffmpeg-dir` 兼容。

---

## 3. Windows

### 3.1 默认行为

- 只修改当前用户 PATH；
- 不修改 machine PATH，除非 `--system --apply` 且已提权；
- 不删除 PATH 中已有项；
- 不改变 PATH 顺序，固定追加 `current/bin`；
- 写入前保存旧值到 journal；
- 写入后广播 `WM_SETTINGCHANGE`；
- 新进程才生效，当前进程不假装已生效。

### 3.2 包管理器共存（未来）

当前不识别也不处理：

```text
Scoop
winget
Chocolatey
```

未来如果重新开放包管理器适配，需要单独 ADR，并明确：

- 不覆盖 shim；
- 不调用 `scoop update` / `scoop install`；
- 不改包管理器内部目录；
- `env status` 报告实际解析来源；
- PATH 顺序固定，CLI 不提供调整参数。

### 3.3 风险

- Windows PATH 有长度和用户/系统作用域问题；
- 修改注册表需要正确处理 UTF-16 / REG_EXPAND_SZ；
- 不能按字符串粗暴替换；
- `ffmpeg.exe` 正在运行时不能直接覆盖，需要 quarantine 与提示。

---

## 4. Linux

### 4.1 默认行为

- 不改 `/etc/profile`；
- 不改 `/etc/environment`；
- 不改 shell rc，除非 `--shell-rc` 显式指定；
- 不要求 root；
- 优先在 `~/.local/bin` 创建 shim/symlink；
- 若 `~/.local/bin` 不在 PATH，报告 warning 并给出添加命令，但默认不自动改 rc。

### 4.2 shim

示例：

```text
~/.local/bin/ffmpeg  -> <data-root>/current/bin/ffmpeg
~/.local/bin/ffprobe -> <data-root>/current/bin/ffprobe
~/.local/bin/ffplay  -> <data-root>/current/bin/ffplay
```

shim 由本项目管理，卸载时按 journal 删除；不删除同名但非本项目创建的文件。

### 4.3 包管理器共存（未来）

当前不识别也不处理：

```text
Homebrew
apt / dnf / pacman
snap / flatpak
```

未来如果重新开放，需要单独 ADR，并明确：

- 不覆盖 `/usr/bin/ffmpeg`；
- 不执行 `brew link` / `brew unlink`；
- 不调用系统包管理器安装/升级；
- `env status` 报告 `command -v ffmpeg` 的实际来源；
- 是否 prepend PATH 必须由用户显式指定。

---

## 5. 命令设计

```text
one-more-ffmpeg env status --json
one-more-ffmpeg env set --json                    # 预览（默认 dry-run）
one-more-ffmpeg env set --apply --json            # 实际写入
one-more-ffmpeg env unset --apply --json
one-more-ffmpeg env rollback --journal <id> --apply --json
one-more-ffmpeg doctor --json
```

所有写操作**默认 dry-run**，只有 `--apply` 才落盘。`--apply` + `--system` 必须真正提权；
`--authorize-system` 只记录系统级同意，不能替代提权（见 `SYSTEM_REQUIREMENTS.md`）。

| 参数 | 作用 |
|---|---|
| `--apply` | 实际写入；不带则只预览 |
| `--no-path` | 不改 PATH |
| `--no-variables` | 不改环境变量 |
| `--no-shell-rc` | 不改 Linux shell profile |
| `--system` | 系统级作用域；`--apply` 时需要提权 |
| `--authorize-system` | 记录系统级同意（dry-run 预览可只用这个） |
| `--lae-compat` | 额外写入 `LAE_FFMPEG_DIR` |

完整参数与 JSON 字段以 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) §4.11 为准。

Windows 注册表、UAC、`REG_EXPAND_SZ`、`WM_SETTINGCHANGE`、junction / pointer_file、Linux shim / shell rc 与 journal 细则见 [`PLATFORM_ENVIRONMENT.md`](PLATFORM_ENVIRONMENT.md)。错误码见 [`ERROR_CODES.md`](ERROR_CODES.md)，`env status` / `doctor` 输出形状见 [`CLI_SCHEMA.md`](CLI_SCHEMA.md)。

`doctor` 检查：

- `current` 是否存在；
- PATH 是否包含 `current/bin`；
- `command -v ffmpeg` / `where.exe ffmpeg` 是否解析到本项目；
- `ONE_MORE_FFMPEG_DIR` 是否一致；
- 三件套是否仍通过校验；
- 是否存在旧 journal 未提交/未回滚。

---

## 6. Journal

每次写入生成：

```json
{
  "schema": "onemoreffmpeg.journal/v1",
  "id": "...",
  "created_at": "...",
  "operations": [
    {
      "type": "user_path_append",
      "old_value": "...",
      "new_value": "...",
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
- rollback 失败也要保留记录并报错；
- journal 不保存 token / 密码。

---

## 7. 明确禁止

- 静默修改 PATH；
- 静默覆盖 `FFMPEG_DIR`；
- 静默修改系统 PATH；
- 静默改 shell rc；
- 删除非本项目创建的 ffmpeg；
- 覆盖正在运行的 Windows exe；
- 把 nonfree 路径写进公开 CI 的默认环境变量；
- 没有 dry-run 就执行 env set；
- 当前阶段为包管理器写专门兼容逻辑。
