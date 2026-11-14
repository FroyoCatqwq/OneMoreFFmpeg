# OneMoreFFmpeg 使用教程

> 面向第一次使用的用户；内置版可用 `one-more-ffmpeg --in-detail-usage` 查看；Linux 可用 `man ./docs/man/one-more-ffmpeg.1` 查看 man 页。

---

## 1. 第一次使用注意

1. 写操作默认 dry-run：`install`、`uninstall`、`env set/unset/rollback`、`journal rollback`、`provider add` 都需要 `--apply` 才修改文件；`download` 与能力缓存会立即写文件。
2. stdout 只输出 JSON；诊断、首次使用提示写 stderr。脚本请始终加 `--json`。
3. 没有配置文件时使用内置默认值；首次运行会在 stderr 打印一次注意事项，并写入 `<data-root>/.first-use` 标记。
4. data root 由 `ONE_MORE_FFMPEG_HOME` 决定；未设置时 Windows 为 `%LOCALAPPDATA%\OneMoreFFmpeg`，Linux 为 `~/.local/share/onemoreffmpeg`。
5. nonfree 构建必须显式 `--allow-nonfree`，且只在本机使用；不会随公开产物分发。
6. 写操作会写 journal；出问题先用 `journal list --json` 找 ID，再 `journal rollback --journal <id> --apply --json`。

## 2. 快速开始（Windows）

```powershell
.\one-more-ffmpeg-windows-amd64.exe version --json
.\one-more-ffmpeg-windows-amd64.exe detect --json
.\one-more-ffmpeg-windows-amd64.exe capabilities --dir C:\ffmpeg --json
```

只查一类能力（真正只运行对应 FFmpeg 命令）：

```powershell
.\one-more-ffmpeg-windows-amd64.exe capabilities --dir C:\ffmpeg --only encoders --json
.\one-more-ffmpeg-windows-amd64.exe capabilities --dir C:\ffmpeg --only muxers --json
```

支持的 `--only`：`muxers`、`demuxers`、`devices`、`decoders`、`encoders`、`filters`、`pixel_formats`、`channel_layouts`、`sample_formats`。单类探测也支持 `--cache-dir` / `--no-cache` / `--refresh`，缓存键按 kind 独立计算。

## 3. 版本目录

稳定版版本信息来自 `https://ffmpeg.org/download.html#releases`，下载链接来自 provider release 资产。大版本线 `9.0` 会展示该线最后一个补丁版本（例如 `9.0.2`），`8.1` 展示 `8.1.3`；版本号不写死。

```powershell
.\one-more-ffmpeg-windows-amd64.exe release list --channel stable --target win64 --json
.\one-more-ffmpeg-windows-amd64.exe release list --channel daily --days 7 --json
.\one-more-ffmpeg-windows-amd64.exe release list --channel daily --days 14 --count 3 --json
```

每日/快照版默认只展示最近 7 天。可指定 `--days` 调整窗口、`--count` 限制展示数量。stable 的 `days` 固定输出 0，但 `--count` 会截断版本列表；daily 资产名无法解析版本时回退为 snapshot tag。选择安装：

```powershell
.\one-more-ffmpeg-windows-amd64.exe release install --channel stable --version 9.0.2 --json
.\one-more-ffmpeg-windows-amd64.exe release install --channel stable --version 9.0.2 --apply --json
.\one-more-ffmpeg-windows-amd64.exe release install --channel daily --snapshot autobuild-2026-11-12-12-00 --apply --json
```

不传 `--version` / `--snapshot` 时安装当前列表里的第一个（最新）版本；二者不能同时使用。`--provider bnqdzj-nonfree` 且未显式指定 `--variant` 时自动使用 `nonfree`，但仍需 `--allow-nonfree`。`release install` 与 `install` 一样默认 dry-run。

## 4. 解析、下载、安装

```powershell
.\one-more-ffmpeg-windows-amd64.exe resolve --provider btbn --channel stable --target win64 --json |
  Out-File -Encoding utf8 .\release.json
.\one-more-ffmpeg-windows-amd64.exe download --release .\release.json --json
.\one-more-ffmpeg-windows-amd64.exe install --release .\release.json --json
.\one-more-ffmpeg-windows-amd64.exe install --release .\release.json --apply --emit-trio-lock --json
```

PowerShell 7 推荐 `Out-File -Encoding utf8`；PowerShell 5 的 `>` 会生成 UTF-16LE，程序也能读取，但建议显式指定编码。

`install` 会：校验归档 SHA-256 → 安全解压 → 保留 `bin/` 与 LICENSE/README → 运行 `ffmpeg -version` / `ffprobe -version` → 写 `install.json` → 切换 `current`（Windows junction，Linux symlink）→ 写 journal。

## 5. 环境与健康检查

```powershell
.\one-more-ffmpeg-windows-amd64.exe env status --json
.\one-more-ffmpeg-windows-amd64.exe env set --json            # dry-run
.\one-more-ffmpeg-windows-amd64.exe env set --apply --json
.\one-more-ffmpeg-windows-amd64.exe doctor --json
.\one-more-ffmpeg-windows-amd64.exe doctor --strict --json    # 不健康时退出 1
```

Windows `env set` 写 HKCU\Environment 并广播 `WM_SETTINGCHANGE`；Linux 写 `~/.local/bin` shim 与 `~/.profile` 托管块（`--no-shell-rc`、`--no-path`、`--no-variables`、`--lae-compat` 可控制内容）。当前进程不会立即看到新的 PATH，请打开新终端。`doctor` 默认退出 0，即使 `healthy=false`，脚本必须读取 `data.healthy`；加 `--strict` 后不健康会返回 `OMF-E6006` 并退出 1（报告在 `details.report`）。

系统级（全局）环境：`env set --system --apply` 写 Windows HKLM 系统环境或 Linux `/usr/local/bin` + `/etc/profile.d/onemoreffmpeg.sh`。需要授权——以 Administrator/root 直接运行可放行；否则须显式 `--authorize-system` 表示同意，未授权返回 `OMF-E6003`。工具不会自动 sudo/提权；例如全局安装到 `/root` 必须自行用 `sudo` 运行。详见 [`SYSTEM_REQUIREMENTS.md`](SYSTEM_REQUIREMENTS.md)。

## 6. 卸载与回滚

```powershell
.\one-more-ffmpeg-windows-amd64.exe uninstall --root C:\...\apps\btbn\latest\<sha16> --apply --json
.\one-more-ffmpeg-windows-amd64.exe journal list --json
.\one-more-ffmpeg-windows-amd64.exe journal rollback --journal journal-... --apply --json
```

`uninstall` 不直接删除，而是把安装目录移到 `<data-root>/quarantine/<journal-id>/`，因此可完整恢复；不会删除下载归档、provider 或非本项目管理目录。`journal rollback` 支持 install、uninstall、env-set、env-unset。

## 7. Linux man 页

```bash
man ./docs/man/one-more-ffmpeg.1
# 或安装后
sudo cp docs/man/one-more-ffmpeg.1 /usr/local/share/man/man1/
mandb 2>/dev/null || true
man one-more-ffmpeg
```

## 8. 故障排查

| 现象 | 处理 |
|---|---|
| `E7001` | 参数组合非法，看 `--help` |
| `E7003` | JSON/schema 不匹配，检查 `--release` 文件 |
| `E3010` | `latest` 缺少 release_id/published_at/asset_sha256 |
| `E4002` | 归档 SHA-256 与 pin 不一致 |
| `E5008` | 安装后三件套无法运行 |
| `E6007` | 试图删除/覆盖非本项目管理文件 |
| `doctor healthy=false` | 查看 `checks[]` 中失败的项 |
| `journal_pending` 非空 | `journal list --json` 后 rollback 或人工确认 |
