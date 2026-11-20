# 新手上手指南（HOWTO）

> 面向第一次使用 OneMoreFFmpeg 的你。照着「最短正确流程」走一遍即可用上 `ffmpeg`。
> 如果你已经遇到「装完了却 `ffmpeg: command not found`」，直接跳到 [§5 排错](#5-为什么装完了-ffmpeg-还是找不到)。

---

## 1. 先搞清楚：这个工具是什么

OneMoreFFmpeg **不是 FFmpeg 本身**，而是一个「FFmpeg 安装管理器」。它帮你：

1. 从网上**下载** FFmpeg 压缩包并校验；
2. **安装**到一个带版本号的目录；
3. 把 `ffmpeg` / `ffprobe` / `ffplay` 三个命令**接进你的 PATH**；
4. 以后可以**切换版本、卸载、回滚**。

装好之后，你敲的 `ffmpeg` 才是真正的 FFmpeg。

### 必须先懂的 5 个词

| 词 | 含义 |
|---|---|
| **data-root** | 所有下载/安装/记录的「家目录」。Linux：`~/.local/share/onemoreffmpeg`；Windows：`%LOCALAPPDATA%\OneMoreFFmpeg`。 |
| **current** | 一个指向「当前启用版本」的链接。`ffmpeg` 命令最终就是指向 `current/bin/ffmpeg`。**没有 current，ffmpeg 就用不了。** |
| **PATH / shim** | 让终端能找到 `ffmpeg` 的机制。Linux 在 `~/.local/bin` 放转发脚本（shim）；Windows 改用户 PATH。**这一步由 `env set` 完成，`install` 不会自动做。** |
| **dry-run** | 所有写操作（install/uninstall/env/provider add）**默认只预览、不改动**，必须加 `--apply` 才真正执行。 |
| **journal** | 每次写操作的流水记录，出了问题可以 `journal rollback` 回滚。 |

> 一句话记住正确顺序：**install --apply（装好，建立 current）→ env set --apply（接进 PATH）→ 开新终端 → ffmpeg 可用**。

---

## 2. 最短正确流程（Linux）

假设你已经从 [Releases](https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases) 下载了 `one-more-ffmpeg-v1.1.2-pre2-linux-amd64`。下面用 `omf` 代指这个文件。

```bash
# 0) 赋予执行权限，改个短名字（可选）
chmod +x one-more-ffmpeg-v1.1.2-pre2-linux-amd64
alias omf=./one-more-ffmpeg-v1.1.2-pre2-linux-amd64

# 1) 确认能跑
omf version
# 期望：OneMoreFFmpeg 1.1.1 (...) linux/amd64 ...

# 2) 注册你自己的 FFmpeg 下载地址（URL + SHA-256），然后下载安装。
#    项目没有内置 provider；所有来源都由你自己登记。
omf provider add --id my-ffmpeg --url https://example.invalid/ffmpeg.zip --sha256 <64位SHA256> --apply
omf resolve --provider my-ffmpeg --json > release.json
omf download --release release.json --apply
omf install --release release.json --apply
# 期望最后几行：
#   root: /home/you/.local/share/onemoreffmpeg/apps/my-ffmpeg/...
#   current: symlink          <-- 看到 current 不是 missing 就对了
#   installation: inst-...

# 3) 体检：此时 current 应已存在，但 PATH 还没接
omf doctor
# current_exists: true ... 但 path_contains_current 可能还是 false（正常，下一步解决）

# 4) 把 ffmpeg 接进 PATH（创建 shim + 写 ~/.profile）
omf env set --apply
# 期望：set: applied=true journal=journal-...

# 5) 关键：打开一个【新终端】（或执行 source ~/.profile）
#    当前这个终端不会立刻看到新 PATH！

# 6) 在新终端里验证
ffmpeg -version
# 期望：ffmpeg version 9.0 ... （成功！）

# 7) 再体检，应全绿
omf doctor
# healthy: true
```

想装**稳定版**而不是每日快照？把第 2 步换成：

```bash
omf install --channel stable --apply
```

---

## 3. 最短正确流程（Windows）

下载 `one-more-ffmpeg-v1.1.2-pre2-windows-amd64.exe`，在 PowerShell 里：

```powershell
# 改个短名字（可选）
Rename-Item .\one-more-ffmpeg-v1.1.2-pre2-windows-amd64.exe omf.exe

.\omf.exe version
.\omf.exe install --channel stable --apply   # 下载并安装，建立 current（junction）
.\omf.exe env set --apply                    # 写 HKCU 用户环境并广播
# 关键：【关掉当前 PowerShell，开一个新的】
ffmpeg -version                              # 新窗口里验证
.\omf.exe doctor                             # healthy: true
```

> Windows 不需要 `source`，但**必须开新终端**才能看到新 PATH。

---

## 4. 每个命令到底干了什么

| 命令 | 作用 | 会改东西吗 |
|---|---|---|
| `version` | 看版本 | 否 |
| `install --apply` | 下载+解压+建立 current | 是（默认 dry-run） |
| `env set --apply` | 接 PATH / 写 shim / 写环境变量 | 是（默认 dry-run） |
| `doctor` | 体检，看哪里没接好 | 否 |
| `env status` | 看 current / PATH / 变量现状 | 否 |
| `detect` | 扫描系统里**已有的** FFmpeg（PATH/常见目录） | 否 |
| `uninstall --apply` | 把某版本移入 quarantine（可恢复） | 是（默认 dry-run） |
| `journal list` / `journal rollback --apply` | 看流水 / 回滚 | rollback 是 |
| `provider list` / `provider add --apply` | 看 / 添加 provider | add 是（默认 dry-run） |
| `capabilities --dir <root>` / `--only <kind>` | 检查 FFmpeg 能力（默认走缓存） | 否（只读，写缓存文件） |
| `update` / `update --apply` | 检查新版 / 自更新本程序 | apply 是（默认 dry-run） |

> `detect` 扫的是「系统里已经能找到的」FFmpeg。你刚 `install` 完、还没 `env set` 时，它不在 PATH 里，`detect` 可能显示 0——这**不代表安装失败**，用 `doctor` 看托管安装才准。

---

## 5. 为什么装完了 `ffmpeg` 还是找不到？

对照常见踩坑（很多人第一次都会遇到）：

### 坑 1：`install` 后直接敲 `ffmpeg` → command not found
**原因**：`install` 只把 FFmpeg 放进 data-root 并建立 current，**没有接 PATH**。
**解决**：还要跑 `omf env set --apply`，然后开新终端。

### 坑 2：跑了 `env set` 还是不行
**原因**：没加 `--apply`，那只是预览（输出 `applied=false`）。
**解决**：`omf env set --apply`，确认输出 `applied=true`。

### 坑 3：`env set --apply` 成功了，当前终端还是找不到
**原因**：当前终端的 PATH 是启动时快照，不会自动更新。
**解决**：**开新终端**；Linux 也可 `source ~/.profile`。

### 坑 4：`ffmpeg` 报「没有那个文件或目录 / No such file or directory」
例如：
```
/home/you/.local/bin/ffmpeg: 行 3: /home/you/.local/share/onemoreffmpeg/current/bin/ffmpeg: 没有那个文件或目录
```
**原因**：shim 在，但它指向的 `current/bin/ffmpeg` 不存在——也就是 **current 缺失或损坏**（通常因为安装没真正完成）。
**解决**：
```bash
omf doctor                 # 若 current_exists: false，说明没装好
omf install --channel stable --apply --force   # 重新装一遍，建立 current
omf env set --apply        # 确保 PATH 接好
# 开新终端再试
```

### 坑 5：`install --apply` 好像「卡住」了
**原因**：它在**下载**几十 MB 的压缩包，网速慢时要等一会儿，不是死机。
**解决**：耐心等。**不要 Ctrl+C / Ctrl+Z**——中途打断会留下「pending journal」（未完成事务），让状态变乱。

### 坑 6：doctor 显示 `1 pending journals`
**原因**：有一次写操作被中断（比如坑 5 里 Ctrl+Z 停掉了安装）。
**解决**：
```bash
omf journal list                       # 看是哪条
omf journal rollback --journal <id> --apply   # 回滚它
omf install --channel stable --apply --force  # 再干净地装一次
```

---

## 6. 常见错误码速查

| 报错 | 含义 | 怎么办 |
|---|---|---|
| `OMF-E3001: provider id is not registered` | provider 未登记 | 先用 `provider add --id ... --url ... --sha256 ... --apply` 注册，再用 `omf provider list` 确认 ID。 |
| `OMF-E7001: unsupported variant` | variant 不支持 | 用 `gpl` / `lgpl` / `nonfree`；nonfree 需加 `--allow-nonfree`。 |
| `OMF-E6003: system scope requires elevation` | 用了 `--system` 但未提权 | 普通用户别用 `--system`；确需全局安装请用 sudo/管理员运行。`--authorize-system` 只能用于 dry-run 预览，不能替代提权。 |
| `OMF-E6009: data root is locked` | 另一个 install/env 正在跑 | 等它结束再试。 |
| `doctor healthy=false` | 有检查项没过 | 看具体哪行 false，对照 §5。 |

### provider 怎么选

项目**没有内置 provider**（不再有 btbn / bnqdzj-nonfree，也没有 ffmpeg.org 版本目录）。所有 FFmpeg 来源都由一个自定义 URL + SHA-256 定义：

```bash
# 1. 登记（一次即可）
omf provider add --id my-ffmpeg --url https://example.invalid/ffmpeg.zip --sha256 <64位SHA256> --apply
omf provider list

# 2. 解析成 pin → 下载 → 安装
omf resolve --provider my-ffmpeg --json > release.json
omf download --release release.json --apply
omf install --release release.json --apply

# nonfree 变体需显式同意（仅本地使用，不可再分发）
omf provider add --id my-nonfree --url https://example.invalid/ffmpeg-nonfree.zip --sha256 <64位SHA256> --apply
omf resolve --provider my-nonfree --variant nonfree --allow-nonfree --json > release.json
```

没有 `release list` / `release install`：版本选择由你提供的 URL 决定。

---

## 7. 切换 / 卸载 / 回滚

```bash
# 装新版本会自动把 current 切过去
omf install --channel stable --apply

# 卸载某个版本（移入 quarantine，不是真删，可恢复）
omf uninstall --root /home/you/.local/share/onemoreffmpeg/apps/my-ffmpeg/latest/<sha> --apply

# 看历史事务
omf journal list

# 回滚某次操作（install/uninstall/env 都行）
omf journal rollback --journal <id> --apply

# 环境接错了想重来
omf env unset --apply
omf env set --apply
```

---

## 8. 彻底搞乱了的「自救」流程

```bash
omf doctor                 # 1. 先看现状
omf journal list           # 2. 看有没有 pending
# 3. 有 pending 就回滚掉
omf journal rollback --journal <id> --apply
# 4. 干净重装 + 重接环境
omf install --channel stable --apply --force
omf env unset --apply
omf env set --apply
# 5. 开新终端
ffmpeg -version
omf doctor                 # healthy: true
```

> 实在不行可以删除 data-root 重来，但**先执行 `env unset --apply`**：`env set --apply` 还写过 `~/.local/bin` shim、`~/.profile` 托管块或注册表变量，仅删目录会留下失效条目（见上一节）。清空 data-root 本身：
> Linux `rm -rf ~/.local/share/onemoreffmpeg`，然后从 §2 重新走一遍。

---

## 9. 更新 OneMoreFFmpeg 自己（v1.1 新功能）

工具能**自己更新自己**（只更新 CLI 程序，不影响已安装的 FFmpeg）：

```bash
omf update            # 只检查有没有新版，不改动（默认 dry-run）
omf update --apply    # 下载新版、校验 SHA-256、原子替换当前程序
omf update --json     # 机器可读输出；外部程序据此决定是否提示用户
```

- 替换程序时若失败会自动回滚到旧版，不会把工具弄坏；
- Windows 上正在运行的程序不能直接覆盖，`--apply` 会用"旧文件改名 → 新文件就位"策略完成。

> 下载 FFmpeg 时，如果终端是交互式的会显示进度条（含百分比与预计剩余时间）；在 CI、`--json` 或输出被重定向（pipe）时改为纯文本阶段日志或完全不输出动画字符，stdout 始终只有 JSON。

## 10. 下一步

- 内置完整教程：`omf --in-detail-usage`
- 进阶教程：[`USAGE.md`](USAGE.md)
- 系统要求与权限模型：[`SYSTEM_REQUIREMENTS.md`](../reference/SYSTEM_REQUIREMENTS.md)
- 全部错误码：[`ERROR_CODES.md`](../reference/ERROR_CODES.md)
- 命令契约：[`CLI_SCHEMA.md`](../reference/CLI_SCHEMA.md)

记住那条黄金顺序就不会迷路：**`install --apply` → `env set --apply` → 开新终端 → `ffmpeg`**。