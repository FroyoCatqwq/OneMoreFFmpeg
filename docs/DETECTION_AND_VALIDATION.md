# FFmpeg 发现与可用性校验

> 状态：Phase 0 设计
> 原则：先只读、显式扫描、结果可审计、不误删

---

## 1. 发现来源优先级

默认顺序：

```text
1. 显式 --dir
2. ONE_MORE_FFMPEG_DIR
3. FFMPEG_DIR
4. LAE_FFMPEG_DIR
5. PATH
6. 包管理器：Scoop / Homebrew
7. 常见目录
8. 全盘扫描（仅 --scan 显式触发）
```

结果按“可信度 + 路径 + 二进制 hash”排序；不自动选择最高优先级安装，除非 API/CLI 明确要求 `--select`。

---

## 2. 显式目录

```text
one-more-ffmpeg detect --dir C:/Tools/FFmpeg --json
```

要求：

- 目录必须存在；
- 查找 `ffmpeg` / `ffprobe` / `ffplay`；
- Windows 自动尝试 `.exe`；
- Linux 检查可执行位；
- 缺失 ffplay 允许 warning，不直接判失败；
- 不递归默认进入子目录，除非 `--recursive`。

---

## 3. 环境变量

识别：

```text
ONE_MORE_FFMPEG_DIR
FFMPEG_DIR
LAE_FFMPEG_DIR
```

语义：

- 可以指向 trio 根目录；
- 可以指向 `bin/` 目录；
- 可以指向 `ffmpeg` 可执行文件；
- 路径不存在时记录 warning，不中断其他来源；
- 不做 PATH 回退拼接猜测。

另外扫描 PATH：

- Windows：`where.exe ffmpeg` / `exec.LookPath`；
- Linux：`exec.LookPath("ffmpeg")`。

---

## 4. Windows / Scoop

识别路径：

```text
%USERPROFILE%\scoop\apps\ffmpeg\current\bin
%USERPROFILE%\scoop\shims\ffmpeg.exe
```

策略：

- 先检查 `scoop` 命令是否存在；存在则调用 `scoop prefix ffmpeg`；
- 不解析 Scoop 内部 JSON 作为唯一依据；
- shim 需要解析到真实 current 目录；
- 记录 `source = "scoop"`；
- 不调用 `scoop update` / `scoop install`；
- 不修改 Scoop 管理的目录。

其他 Windows 常见目录：

```text
C:\ffmpeg\bin
C:\Program Files\ffmpeg\bin
%LOCALAPPDATA%\Programs\ffmpeg\bin
%LOCALAPPDATA%\Microsoft\WinGet\Packages\
```

---

## 5. Linux / Homebrew

识别路径：

```text
/home/linuxbrew/.linuxbrew/bin/ffmpeg
/home/linuxbrew/.linuxbrew/opt/ffmpeg/bin/ffmpeg
~/.linuxbrew/bin/ffmpeg
/opt/homebrew/bin/ffmpeg
```

策略：

- 若 `brew` 存在，调用 `brew --prefix ffmpeg`；
- 不执行 `brew install` / `brew upgrade`；
- 不修改 Homebrew prefix；
- 记录 `source = "homebrew"`。

其他常见目录：

```text
/usr/bin
/usr/local/bin
/opt/ffmpeg/bin
/snap/bin
/var/lib/flatpak/exports/bin
~/.local/bin
```

---

## 6. 全盘扫描

必须显式启用：

```text
one-more-ffmpeg detect --scan --roots C:/ --json
one-more-ffmpeg detect --scan --roots /,/home,/opt --json
```

默认不扫描全盘。

### Windows 默认根

- 所有固定 NTFS 盘；
- 排除可移动盘、网络盘，除非显式 `--include-removable` / `--include-network`。

### Linux 默认根

```text
/
/home
/opt
/usr/local
/snap
```

排除：

```text
/proc
/sys
/dev
/run
/tmp
/var/lib/docker
```

以及：

- symlink loop；
- 无权限目录；
- 特殊文件系统；
- 超过 `--max-depth` 的目录；
- 超过 `--max-files` 后停止并 warning；
- 单次扫描超过 `--timeout` 后取消。

只匹配文件名：

```text
ffmpeg / ffmpeg.exe
ffprobe / ffprobe.exe
ffplay / ffplay.exe
```

匹配后按目录聚合 trio，不把散落在不同目录的二进制拼成一套。

---

## 7. 去重

去重 key：

```text
canonical(root_dir) + ffmpeg_sha256 + ffprobe_sha256 + ffplay_sha256
```

规则：

- symlink 解析为真实路径；
- Windows 大小写不敏感；
- 同一目录多个硬链接只保留一个；
- 同一 trio 被多个 source 发现时合并 `sources[]`；
- 不删除任何文件。

---

## 8. 可用性校验

每个候选执行：

1. 三个路径存在（ffplay 可 warning）；
2. 文件可执行；
3. `ffmpeg -version`；
4. `ffprobe -version`；
5. `ffplay -version`（若存在）；
6. `ffmpeg -L`；
7. 核心版本一致；
8. 记录二进制 SHA-256；
9. 记录 stdout/stderr 摘要；
10. 生成 `Installation.fingerprint`；
11. 输出状态。

状态：

| 状态 | 含义 |
|---|---|
| `healthy` | 三件套完整、版本一致、命令成功 |
| `degraded` | ffmpeg/ffprobe 可用，但 ffplay 缺失或某项 warning |
| `unusable` | 无法启动、版本不一致、权限错误、损坏 |
| `unknown` | 信息不足，需要进一步验证 |

---

## 9. 输出示例

```json
{
  "schema": "onemoreffmpeg.detect/v1",
  "installations": [
    {
      "id": "inst-...",
      "root_dir": "C:/Users/.../scoop/apps/ffmpeg/current/bin",
      "ffmpeg": "C:/.../ffmpeg.exe",
      "ffprobe": "C:/.../ffprobe.exe",
      "ffplay": "C:/.../ffplay.exe",
      "sources": ["scoop", "PATH"],
      "status": "healthy",
      "version": "n9.0.2",
      "warnings": []
    }
  ],
  "errors": []
}
```

---

## 10. 安全边界

- `detect` 默认只读；
- 不写 PATH；
- 不写注册表；
- 不写 shell profile；
- 不下载；
- 不安装；
- 不删除；
- 不执行下载包内脚本；
- 不执行任意用户传入命令；
- 全盘扫描只读且有超时；
- 日志不打印 token / 凭据；
- 对 UNC、网络盘、符号链接保持显式警告。
