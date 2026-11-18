# v1.0.3（2026-11-19）

> Bugfix 版本：全量审查核心包后修复 4 个潜在 bug。CLI 外部契约（18 命令 / 17 个 `/v1` schema / 退出码）不变。
> 产物：Windows x64 / Linux x64 二进制、RHEL 9+ x86_64 RPM、`SHA256SUMS.txt`、build provenance；Scoop bucket 同步更新。

## 修复

| 严重度 | 问题 | 修复 |
|---|---|---|
| 中 | 重装同一 SHA 并加 `--emit-trio-lock` 时静默忽略该 flag，下游拿不到 `ffmpeg-trio.lock` | reuse 分支在 `--apply` 时补写 lock 并回写 `install.json` 的 `LockPath` |
| 低-中 | `capabilities list <kind>` 缓存命中时丢失 `W1001 ffplay is missing` 告警，与全量收集不一致 | 缓存命中路径按当前 trio 状态补 W1001（与 `collect()` 一致） |
| 低 | `--days` 极大正数使 `time.Duration(days)*24h` 溢出为负，daily catalog 过滤出空列表 | clamp 到 100 年上界并改用 `AddDate` |
| 低 | Linux 下 `env status` 对环境变量名做大小写折叠，小写 `path` 被当成 `PATH` | 按平台使用 `envNameKey` 比较（Windows 折叠、Unix 精确） |

四项均附回归测试。

## 安装

```powershell
# Windows — Scoop
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg          # 升级: scoop update one-more-ffmpeg
```

```bash
# RHEL 9 / AlmaLinux 9 / Rocky 9（x86_64）
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.3/one-more-ffmpeg-1.0.3-1.el9.x86_64.rpm
```

```bash
# Linux 通用
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.3/one-more-ffmpeg-v1.0.3-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.3/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

详见 [`安装指南`](../guides/INSTALL.md)。

完整变更见 [`CHANGELOG.md`](CHANGELOG.md)；版本规则见 [`VERSIONING.md`](../reference/VERSIONING.md)。
