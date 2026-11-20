# OneMoreFFmpeg 使用教程

> 面向第一次使用的用户；内置版可用 `one-more-ffmpeg --in-detail-usage` 查看；Linux 可用 `man ./docs/reference/man/one-more-ffmpeg.1` 查看 man 页。

---

## 1. 第一次使用注意

1. 写操作默认 dry-run：`install`、`uninstall`、`env set/unset/rollback`、`journal rollback`、`provider add` 都需要 `--apply` 才修改文件；`download` 与能力缓存会立即写文件。
2. stdout 只输出 JSON；诊断写 stderr。脚本请始终加 `--json`。
3. 没有配置文件时使用内置默认值；启动时**不打印任何提示**，也不写标记文件。需要完整说明用 `--in-detail-usage` 或读本文档。
4. data root 由 `ONE_MORE_FFMPEG_HOME` 决定；未设置时 Windows 为 `%LOCALAPPDATA%\OneMoreFFmpeg`，Linux 为 `~/.local/share/onemoreffmpeg`。
5. nonfree 构建必须显式 `--allow-nonfree`，且只在本机使用；不会随公开产物分发。
6. 写操作会写 journal；出问题先用 `journal list --json` 找 ID，再 `journal rollback --journal <id> --apply --json`。
7. `update` 用于自更新 CLI 本身（默认只检查，`--apply` 才替换）；能力缓存按 FFmpeg 二进制内容指纹失效，替换二进制后下次采集自动重建。
8. 下载进度按环境自适应：TTY 显示动画进度条，CI / 输出重定向显示阶段日志，`--json` 不渲染；进度信息只走 stderr。
9. 调试用 `--log-level debug|trace` 输出诊断日志到 stderr，`--quiet` 只留错误；JSON 模式下日志记录也走 stderr，不污染 stdout。

## 2. 快速开始（Windows）

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe version --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe detect --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe capabilities --dir C:\ffmpeg --json
```

只查一类能力（真正只运行对应 FFmpeg 命令）：

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe capabilities --dir C:\ffmpeg --only encoders --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe capabilities --dir C:\ffmpeg --only muxers --json
```

支持的 `--only`：`muxers`、`demuxers`、`devices`、`decoders`、`encoders`、`filters`、`pixel_formats`、`channel_layouts`、`sample_formats`。单类探测也支持 `--cache-dir` / `--no-cache` / `--refresh`，缓存键按 kind 独立计算。

## 3. 默认官方 provider + 自定义 provider

`resolve` 不带 `--provider` 时默认使用内置 **official** provider：查询 FFmpeg-Builds 的 `latest` release，选 master 最新快照 / gpl / static / x86_64 资产。稳定版（n*）暂不支持；系统要求请查看上游 README（warning 会给出链接）。

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe resolve --target win64 --json > release.json
```

也可以注册自己的 URL + SHA-256（支持 http/https）：

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe provider add --id my-ffmpeg --url https://example.invalid/ffmpeg.zip --sha256 <64位SHA256> --apply --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe provider list --json
```

自定义来源的版本与资产完全由你提供的 URL 决定。nonfree 变体需 `--allow-nonfree`，且只能本地使用、不可再分发。

## 4. 解析、下载、安装

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe resolve --provider my-ffmpeg --target win64 --json |
  Out-File -Encoding utf8 .\release.json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe download --release .\release.json --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe install --release .\release.json --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe install --release .\release.json --apply --emit-trio-lock --json
```

PowerShell 7 推荐 `Out-File -Encoding utf8`；PowerShell 5 的 `>` 会生成 UTF-16LE，程序也能读取，但建议显式指定编码。

`install` 会：校验归档 SHA-256 → 安全解压 → 保留 `bin/` 与 LICENSE/README → 运行 `ffmpeg -version` / `ffprobe -version` → 写 `install.json` → 切换 `current`（Windows junction，Linux symlink）→ 写 journal。

## 5. 环境与健康检查

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe env status --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe env set --json            # dry-run
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe env set --apply --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe doctor --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe doctor --strict --json    # 不健康时退出 1
```

Windows `env set` 写 HKCU\Environment 并广播 `WM_SETTINGCHANGE`；Linux 写 `~/.local/bin` shim 与 `~/.profile` 托管块（`--no-shell-rc`、`--no-path`、`--no-variables`、`--lae-compat` 可控制内容）。当前进程不会立即看到新的 PATH，请打开新终端。`doctor` 默认退出 0，即使 `healthy=false`，脚本必须读取 `data.healthy`；加 `--strict` 后不健康会返回 `OMF-E6006` 并退出 1（报告在 `details.report`）。

系统级（全局）环境：`env set --system --apply` 写 Windows HKLM 系统环境或 Linux `/usr/local/bin` + `/etc/profile.d/onemoreffmpeg.sh`。授权——以 Administrator/root 直接运行可放行；未提权时 dry-run 可加 `--authorize-system` 预览，但 `--apply` 必须真正提权（`--authorize-system` 只记录同意、不能替代提权），否则 `OMF-E6003`。工具不会自动 sudo/提权；例如全局安装到 `/root` 必须自行用 `sudo` 运行。详见 [`SYSTEM_REQUIREMENTS.md`](../reference/SYSTEM_REQUIREMENTS.md)。

## 6. 卸载与回滚

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe uninstall --root C:\...\apps\my-ffmpeg\latest\<sha16> --apply --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe journal list --json
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe journal rollback --journal journal-... --apply --json
```

`uninstall` 不直接删除，而是把安装目录移到 `<data-root>/quarantine/<journal-id>/`，因此可完整恢复；不会删除下载归档、provider 或非本项目管理目录。`journal rollback` 支持 install、uninstall、env-set、env-unset。

## 7. 自更新与日志

```powershell
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe update --json            # 检查新版，不改动
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe update --apply --json    # 下载+校验+替换
.\one-more-ffmpeg-v1.1.2-pre4-windows-amd64.exe download --release .\release.json --log-level debug --json
```

`update` 默认 dry-run；`--apply` 时按平台选产物、校验 release asset 的 SHA-256，成功后原子替换当前可执行文件（Windows 用旧文件改名策略，失败自动回滚）。`--check` 与 `--apply` 互斥；非 amd64 平台不支持自更新（`OMF-E3007`）。返回 `onemoreffmpeg.update/v1` data schema，字段见 [`CLI_SCHEMA.md`](../reference/CLI_SCHEMA.md) §4.17。

日志：全局 `--log-level error|warn|info|debug|trace`（`trace` 比 `debug` 更详细）、`--quiet`（只留错误，优先于 `--log-level`）。日志走 stderr，JSON 模式下为结构化 JSON handler，stdout 仍只有结果信封。

进度条：`download` 与 `update --apply` 输出进度。TTY 显示 go-pretty 动画条（百分比 + ETA）；CI（`CI` / `GITHUB_ACTIONS`）或输出重定向时改为阶段日志（`label: 25%` … `100%`，无控制字符）；`--json` 不渲染。进度只写 stderr。

## 8. Linux man 页

```bash
man ./docs/reference/man/one-more-ffmpeg.1
# 或安装后
sudo cp docs/reference/man/one-more-ffmpeg.1 /usr/local/share/man/man1/
mandb 2>/dev/null || true
man one-more-ffmpeg
```

## 9. 故障排查

| 现象 | 处理 |
|---|---|
| `E7001` | 参数组合非法，看 `--help` |
| `E7003` | JSON/schema 不匹配，检查 `--release` 文件 |
| `E3010` | `latest` 缺少 release_id/published_at/asset_sha256 |
| `E4002` | 归档 SHA-256 与 pin 不一致；`update --apply` 也会校验资产 digest |
| `E5008` | 安装后三件套无法运行 |
| `E6007` | 试图删除/覆盖非本项目管理文件 |
| `E6009` | data-root 或能力缓存正被另一进程加锁，稍后重试 |
| `update` 显示无更新但版本确实落后 | 检查网络与 stderr 日志（`--log-level debug`）；非 amd64 平台不支持自更新 |
| `OMF-E6003` on `detect --scan` | 全盘扫描需提权：Linux 用 sudo/root，Windows 用 Administrator/SYSTEM；或改用 `--scan --roots <具体目录>` |
| `doctor healthy=false` | 查看 `checks[]` 中失败的项 |
| 能力检测结果疑似过时 | 确认是否替换过 FFmpeg 二进制；缓存按内容指纹自动失效，必要时 `--refresh` |
| `journal_pending` 非空 | `journal list --json` 后 rollback 或人工确认 |
