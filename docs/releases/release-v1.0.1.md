# v1.0.1（2026-11-17）

> Bugfix 版本：全项目源码级审计后修复全部确认缺陷。CLI 外部契约（18 命令 / 17 个 `/v1` schema / 退出码）不变。
> 产物：Windows x64 / Linux x64 二进制、RHEL 9+ x86_64 RPM、`SHA256SUMS.txt`、build provenance；Scoop bucket 同步更新。

## 修复要点

### 安全
- `--json help [cmd]` 不再向 stdout 输出无信封的纯文本（返回 E1001）。
- `github_api`/GitHub 客户端拒绝带 userinfo 的 URL，防止 Bearer token 发往错误主机；错误不再回显含凭据 URL。
- 下载、GitHub API、catalog 的重定向禁止 HTTPS→HTTP 降级；URL 拒绝 userinfo。
- 远程 e2e 优先严格校验 SSH 主机密钥；远程命令加看门狗防无限挂起。

### 正确性
- `uninstall` / `provider add` 的互斥 flag 组合改为显式报错。
- GitHub Releases 支持分页（最多 5 页），不再静默漏版本。
- `doctor` 校验 `-version` banner，假二进制不再误判健康。
- `ffexec` 成功退出不被迟到的 context 取消改写；扫描目录计入 `MaxFiles`；`-h` topic 缺失检测覆盖全部行；Windows `.com` 识别一致。
- config 数值/大小上限；release commit 元数据、严格 SemVer、门禁补齐、provenance 覆盖校验清单。

## 安装

```powershell
# Windows — Scoop
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg          # 升级: scoop update one-more-ffmpeg
```

```bash
# RHEL 9 / AlmaLinux 9 / Rocky 9（x86_64）
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.1/one-more-ffmpeg-1.0.1-1.el9.x86_64.rpm
```

```bash
# Linux 通用（手动二进制 + 校验）
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.1/one-more-ffmpeg-v1.0.1-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.1/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

详见 [`安装指南`](../guides/INSTALL.md)。

## 验证

| 项目 | Windows 本机 | Linux CI 服务器 | Docker 全新环境 |
|---|---|---|---|
| `go test ./...` | 全绿 | 全绿 | 全绿 |
| `gofmt` / `go vet` / 三平台交叉编译 | 通过 | — | — |
| `go test -race`（7 核心包） | — | 全绿 | — |
| fuzz（7 目标） | 全绿 | — | — |
| pytest e2e | 45 passed | 48 passed | 全绿 |
| Robot / Bats | 5 / — | 5 / 5 | 5 / 5 |

完整变更见 [`CHANGELOG.md`](CHANGELOG.md)；版本规则见 [`VERSIONING.md`](../reference/VERSIONING.md)。
