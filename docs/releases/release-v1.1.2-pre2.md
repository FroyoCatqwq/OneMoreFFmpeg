# v1.1.2-pre2（2026-11-20，预发布）

> 第二轮细致排查后修复更深层 bug。CLI 契约保持 18 命令 / 17 个 `/v1` schema。
> 产物：Windows x64 / Linux x64 二进制、`SHA256SUMS.txt`、build provenance。
> **预发布版本，仅供测试。**

## 修复要点

- **自更新恢复可用**：`update --recover` 接线此前是死代码的 `Recover()`；修正状态机（`replaced` 提交新版本而非回滚，`backup_created` 才回滚）；崩溃后不再永久卡死。
- **panic 修复**：`fetchURL` 对畸形 URL（含控制字符/坏转义）先判 `url.Parse` 错误再解引用。
- **安全**：下载缓存版本目录新增符号链接防护。
- **配置真正生效**：`network.max_attempts` 通过 retry transport 作用于 go-github 自更新查询。
- **并发**：progress terminal state 读写锁；cache 读写统一 `lockedfile` 锁。
- **测试隔离**：testscript 重定向 Windows `LOCALAPPDATA` 到临时目录。

## 文档/CI

- 依赖清单与版本全面核对（THIRD_PARTY_NOTICES、ADR-0002）；`govulncheck` pin 到 v1.10.0。
- 修正 channel（`daily` 唯一）、命令数、`update.example.json`、schema `$id`、E1001 语义、`doctor --strict` 描述、E2E 测试描述。

## 安装（手动二进制）

```bash
# Linux
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre2/one-more-ffmpeg-v1.1.2-pre2-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre2/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

Windows 下载 `one-more-ffmpeg-v1.1.2-pre2-windows-amd64.exe` 并按 `SHA256SUMS.txt` 校验。

详见 [`安装指南`](../guides/INSTALL.md)。

## 验证

gofmt / vet / golangci-lint(0 issues) / go test / fuzz / 三平台交叉编译 / Linux race 全绿；Windows 与 Linux e2e、testscript、用户视角自建 HTTP 流程均通过。
