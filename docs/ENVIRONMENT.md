# 环境变量与 PATH 管理

> 状态：Phase 0 设计
> 原则：默认只读、显式写入、先备份后修改、可回滚

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

不覆盖用户已有变量，除非命令/API 明确 `overwrite=true`。

---

## 2. 推荐模型：current 指针

安装目录使用版本化目录，不每次改 PATH：

```text
<data-root>/
├─ apps/<provider>/<release-tag>/<asset-hash>/
└─ current -> apps/<...>
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
- 不修改 machine PATH，除非 `--system` + 提权 + 明确 `--yes`；
- 不删除 PATH 中已有项；
- 不改变顺序，除非 `--prepend`；
- 写入前保存旧值到 journal；
- 写入后广播 `WM_SETTINGCHANGE`；
- 新进程才生效，当前进程不假装已生效。

### 3.2 Scoop 共存

- 如果检测到 Scoop shim，不覆盖 Scoop 的 PATH；
- 默认把 `current/bin` 追加到用户 PATH；
- `env status` 报告哪个 ffmpeg 会先被解析；
- 用户可用 `--prepend` 显式改变优先级。

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

### 4.3 Homebrew 共存

- 不覆盖 Homebrew link；
- `env status` 显示 `command -v ffmpeg` 的实际来源；
- 若用户要求 `--prepend`，才把 `current/bin` 放到 PATH 前面；
- 不执行 `brew unlink` / `brew link`。

---

## 5. 命令设计

```text
one-more-ffmpeg env status --json
one-more-ffmpeg env set --installation <id> --dry-run
one-more-ffmpeg env set --installation <id> --yes
one-more-ffmpeg env unset --yes
one-more-ffmpeg env rollback --journal <id>
one-more-ffmpeg doctor --json
```

`env set` 默认 dry-run；没有 `--yes` 不写。

`doctor` 检查：

- `current` 是否存在；
- PATH 是否包含 `current/bin`；
- `command -v ffmpeg` 是否解析到本项目；
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
- 没有 dry-run 就执行 env set。
