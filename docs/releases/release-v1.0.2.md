# v1.0.2（2026-11-18）

> Bugfix 版本：全量审查每个包与每份文档后修复确认的缺陷。CLI 外部契约（18 命令 / 17 个 `/v1` schema / 退出码）不变。
> 产物：Windows x64 / Linux x64 二进制、RHEL 9+ x86_64 RPM、`SHA256SUMS.txt`、build provenance；Scoop bucket 同步更新至 v1.0.2。

## 修复要点

### 安全 / 契约
- `--json provider|env|journal|release help` 不再把无信封的纯文本 help 写到 stdout，统一返回 E1001 错误信封。
- 事务锁文件不再在加锁前被写入；锁路径为符号链接时显式拒绝，锁文件内容保持为空，正确性完全来自 OS 锁身份。
- `env rollback` 的 system scope 回滚强制校验提权状态；shim 目录白名单与 scope 一致性检查。

### 正确性
- `--max-bytes` CLI flag 加上与配置一致的 8 GiB 上限。
- `release list/install --days/--count` 非法值显式报错。
- `doctor --strict` human 模式输出失败检查项，不再丢失诊断。
- checksums 资产按 `checksums.sha256` → `*checksums*.sha256` → `*.sha256` 优先级查找。
- GitHub `ListReleases(nil, ...)` 不再 panic；`Retry-After` 溢出防护；releases 页重定向限制 10 次。
- `gpl-shared`/`nonfree-shared` 不再覆盖显式 `--linkage static`。
- 一次 detect 的 `--dir --recursive` 与 `--scan` 共享 `MaxFiles` 预算；重叠 roots 父目录优先。

### 文档
- 全量修正：journal schema 字段、shell rc 标记、删除 data-root 的影响、变量覆盖语义、锁错误码 E6009、latest pin、Go 工具链 1.27.x。
- 补齐 3 个新运行时依赖的第三方许可证声明与文本（flock / go-internal / jsonschema）。

## 安装

```powershell
# Windows — Scoop
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg          # 升级: scoop update one-more-ffmpeg
```

```bash
# RHEL 9 / AlmaLinux 9 / Rocky 9（x86_64）
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.2/one-more-ffmpeg-1.0.2-1.el9.x86_64.rpm
```

```bash
# Linux 通用
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.2/one-more-ffmpeg-v1.0.2-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.0.2/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

详见 [`安装指南`](../guides/INSTALL.md)。

## 真实网络测试

新增 `test_real_network_e2e.py`（marker `OMF_E2E_NETWORK=1`），直连真实 ffmpeg.org releases 页与 BtbN GitHub API 测 release list / resolve；本地实测 3/3 通过。CI 默认跳过以避免外网波动。

完整变更见 [`CHANGELOG.md`](CHANGELOG.md)；版本规则见 [`VERSIONING.md`](../reference/VERSIONING.md)。
