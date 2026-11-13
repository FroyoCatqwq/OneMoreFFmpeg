# OneMoreFFmpeg v0.6.0

> 2026-11-13 · SemVer 0.6.0
> 首个公开 release：Phase 1–5 闭环 + 全量审查修复 + Windows/Linux e2e。

## 亮点

- **发现与校验**：`detect` / `validate` / `capabilities`，支持 `capabilities --only <kind>` 单类探测。
- **Provider / 下载**：`provider list|add`、`resolve`、`download`，支持 BtbN、BnqDzj nonfree 与自定义直链。
- **版本目录**：`release list|install` 动态读取 FFmpeg 官网；stable 按大版本线展示最新补丁（如 9.0.2），daily 默认最近 7 天快照。
- **安装**：安全解压 zip/tar.xz/tar.gz/tar.zst，版本化目录、install.json、trio 校验、LICENSE/README 保留。
- **current 与环境**：Windows junction / Linux symlink，`env status|set|unset|rollback`，`doctor`。
- **可回滚**：install/uninstall/env 事务 journal；`journal list|rollback`，uninstall 进入 quarantine 而非直接删除。
- **CLI 体验**：默认 dry-run，`--in-detail-usage` 内置教程，首次使用提示只显示一次。

## 产物

- `one-more-ffmpeg-v0.6.0-windows-amd64.exe`
- `one-more-ffmpeg-v0.6.0-linux-amd64`
- `SHA256SUMS.txt`

校验：

```bash
sha256sum -c SHA256SUMS.txt
```

Windows PowerShell：

```powershell
Get-FileHash .\one-more-ffmpeg-v0.6.0-windows-amd64.exe -Algorithm SHA256
```

## 使用

```bash
./one-more-ffmpeg-v0.6.0-linux-amd64 --in-detail-usage
./one-more-ffmpeg-v0.6.0-linux-amd64 release list --channel stable --json
./one-more-ffmpeg-v0.6.0-linux-amd64 release install --channel stable --version 9.0.2 --json
```

Windows 将可执行文件替换为 `.exe` 即可。详细教程见 `docs/USAGE.md`，man 页见 `docs/man/one-more-ffmpeg.1`。

## 已知边界

- GitHub releases 最多读取前 100 条，未实现分页；
- `provider add` 使用 30 秒陈旧锁策略；
- Linux 只维护 `~/.profile`，不处理 fish/zsh 专属 rc；
- `doctor` 默认退出 0，脚本需读取 `data.healthy`；
- nonfree 构建需 `--allow-nonfree`，仅限本机使用。
