# v1.1.0（2026-11-18）

> 首个 MINOR 版本：新增自更新与下载进度条两个功能；CLI 外部契约扩充为 19 命令 / 18 个 `/v1` schema（含 `update`），退出码不变。
> 产物：Windows x64 / Linux x64 二进制、RHEL 9+ x86_64 RPM、`SHA256SUMS.txt`、build provenance；Scoop bucket 同步更新。

## 新功能

### `update` 自更新命令

```bash
one-more-ffmpeg update            # 默认 dry-run：只检查
one-more-ffmpeg update --json     # JSON 信封输出
one-more-ffmpeg update --apply    # 下载 + SHA-256 校验 + 原子替换
```

- 通过 `internal/provider/github` 查询最新稳定版，按平台选产物；
- Apply 时校验 release asset 的 SHA-256 digest，写临时文件后原子替换；
- Windows 采用"旧 exe 改名 → 新 exe 就位 → 删除旧文件"，任一步失败自动回滚；
- 新增 `onemoreffmpeg.update/v1` schema；`--check` 与 `--apply` 互斥（E1001）。

### 下载进度条（环境自适应）

| 环境 | 表现 |
|---|---|
| TTY | 进度条 + 百分比 + ETA（go-pretty 动画） |
| CI（`CI` / `GITHUB_ACTIONS` / `TERM=dumb`） | 普通阶段日志（0/25/50/75/100%），无动画字符 |
| JSON（`--json`） | stdout 仅 JSON；进度信息不渲染 |
| Pipe（输出重定向） | 纯文本阶段日志，无控制字符 |

## 内部改进

- **日志框架**：`log/slog` + `lmittmann/tint`，`--log-level` / `--quiet` 控制，JSON 模式走 stderr 的结构化 handler；业务层（install/download/capabilities/selfupdate）通过 `logging.FromContext(ctx)` 接入调试日志，stdout 始终保持 JSON 纯净。
- **能力缓存**：fingerprint 已包含 FFmpeg 二进制内容 hash（替换二进制即失效）；缓存写入改为原子写 + `internal/lock` 的 OS 级跨进程锁（锁文件置于缓存目录外的 `.locks` 兄弟目录）。
- **架构解耦**：`internal/output` 封装 go-pretty / tidwall/pretty / encoding/json-v2，`internal/cache` 封装 otter；业务层只依赖注入的 `Progress` 接口，不直接 import 表现层第三方库；`urfave/cli` 仅存在于 `internal/cli`。
- 新增依赖：`go-pretty/v6`、`tidwall/pretty`、`maypok86/otter/v2`、标准库 `encoding/json/v2`；许可证见 `THIRD_PARTY_NOTICES.md`。

## 安装

```powershell
# Windows — Scoop
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg          # 升级: scoop update one-more-ffmpeg
```

```bash
# RHEL 9 / AlmaLinux 9 / Rocky 9（x86_64）
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.0/one-more-ffmpeg-1.1.0-1.el9.x86_64.rpm
```

```bash
# Linux 通用
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.0/one-more-ffmpeg-v1.1.0-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.0/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

详见 [`安装指南`](../guides/INSTALL.md)。

## 验证

gofmt / vet / golangci-lint（含 gosec）/ go test / fuzz / 三平台交叉编译全绿；本机 Windows 与 Linux CI 服务器完整 e2e（含真实注册表、真实网络、update 命令）通过；Linux race 检测覆盖 output/cache/selfupdate/install/download。

完整变更见 [`CHANGELOG.md`](CHANGELOG.md)；版本规则见 [`VERSIONING.md`](../reference/VERSIONING.md)。
