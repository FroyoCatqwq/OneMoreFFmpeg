# v1.0.0（2026-11-16）

> 首个稳定版：Phase 1–7 全部完成，CLI 外部契约冻结；文档重整，全量审查修复锁/回滚/解压等缺陷。
> 产物：Windows x64 / Linux x64 可执行文件（纯 Go，`CGO_ENABLED=0`）、RHEL 9+ x86_64 RPM 包、`SHA256SUMS.txt`、build provenance；
> Scoop bucket 自动更新至 <https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket>。

## 安装

```powershell
# Windows — Scoop
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg
```

```bash
# RHEL 9 / AlmaLinux 9 / Rocky 9（x86_64）
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.0/one-more-ffmpeg-1.0.0-1.el9.x86_64.rpm
```

```bash
# Linux 通用：下载二进制 + SHA256SUMS 校验
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.0/one-more-ffmpeg-v1.0.0-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.0/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
chmod +x one-more-ffmpeg-v1.0.0-linux-amd64
```

## 亮点

- **契约稳定**：18 个命令、17 个 `/v1` data schema、退出码 0/1/2/130 冻结，由 `internal/contracts/freeze_test.go` 在 CI 守护。外部项目只调用 CLI，见 [`集成指南`](../guides/INTEGRATION.md)。
- **质量门禁**：gofmt / go vet / go test / govulncheck / fuzz 短跑；Windows + Linux e2e 矩阵（pytest 45 例 / Robot 5 例 / Bats 5 例）；Docker 全新环境验证。
- **发布自动化**：tag 触发 `.github/workflows/release.yml`，构建双平台二进制、校验产物只含 CLI、生成 checksums 与 provenance attestation。
- **文档重整**：`guides/`（上手/使用/集成/迁移）、`reference/`（对外契约）、`development/`（架构/技术/ADR）、`releases/`、`archive/`（历史快照）。

## 修复（安全与数据完整性）

| 严重度 | 问题 | 修复 |
|---|---|---|
| Critical | 数据根事务锁用 mtime 判断 stale 再删除，与新锁获取存在 TOCTOU，可能让两个写事务并发操作安装目录 | 改用 OS 级 `flock` / `LockFileEx`，锁绑定文件描述符；进程崩溃由内核释放，不再有 stale 回收竞态 |
| High | 伪造 journal 的 `shell_rc_backup` 可让 `env rollback` 读任意文件写入 shell profile | 回滚前校验 backup 路径必须位于 data root |
| High | `uninstall` 在隔离目录/删 current 后写 journal 失败，无法自动恢复 | quarantine 先落 journal；current 指针删除失败时回滚；最终写失败给出明确 rollback 指引 |
| Medium | 压缩比防护只对单条目生效，多条目聚合可绕过 | 引入全局 `min(compressed×maxRatio, maxTotal)`，流式写入按剩余预算限制 |
| Medium | `providers.toml` 自实现锁有相同 stale 竞态、固定 `.bak` 可被并发覆盖 | 复用 OS 级锁 + 唯一备份名 |
| Low | Linux PATH 去重用 EqualFold，误判大小写不同的真实目录 | 按平台选择大小写敏感性 |
| Low | 下载共享目录首次创建清理可能删掉并发任务的目录 | 唯一 owner 标记保护清理 |

## 验证

| 项目 | Windows 本机 | Linux CI 服务器 | Docker 全新环境 |
|---|---|---|---|
| `go test ./...` | 全绿 | 全绿 | 全绿 |
| `go vet` / 交叉编译 | 通过 | 通过 | 构建即测试 |
| pytest e2e（两文件） | 45 passed | 45 passed | 45 passed |
| Robot / Bats | 5 / — | 5 / 5 | 5 / 5 |
| fuzz 冒烟 | 通过 | 通过 | — |

## 校验和验证

Release 提供 `SHA256SUMS.txt` 与 Sigstore build provenance：

```bash
# Linux
sha256sum -c SHA256SUMS.txt --ignore-missing

# Windows PowerShell
Get-FileHash .\one-more-ffmpeg-v1.0.0-windows-amd64.exe -Algorithm SHA256
```

## 已知限制

- GitHub release 列表只读最多 100 条（无 Link 分页回溯）；
- `resolve` 无 `--select` / provider 优先级；
- `env set --apply` 写真实 profile / 注册表，e2e 只覆盖 dry-run；
- 自动化许可证检查（go-licenses）尚未接入 CI（非发布阻断）。

完整变更见 [`CHANGELOG.md`](CHANGELOG.md)；版本规则见 [`VERSIONING.md`](../reference/VERSIONING.md)。
