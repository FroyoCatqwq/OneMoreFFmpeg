# v1.1.2-pre1（2026-11-18，预发布）

> 预发布版本：全量源码/用户双视角审查后修复 16+ 项 bug/缺陷。CLI 契约保持 18 命令 / 17 个 `/v1` schema。
> 产物：Windows x64 / Linux x64 二进制、`SHA256SUMS.txt`、build provenance（无包管理器包）。
> **预发布版本，仅供测试，不建议生产使用。**

## 修复要点

- **自定义 URL 支持 http**：provider add / asset_url / checksums_url / download / resolve 接受 http 与 https（自建镜像常用 http）；仍拒绝 URL 内嵌凭据，禁止 https→http 降级。
- **自更新**：读取 `[network] github_api` / `max_attempts`；URL 校验拒绝 userinfo；digest 校验完整 hex。
- **checksums 抓取**：真正使用 `Resolver.Timeout` / `MaxRedirects`。
- **契约一致性**：`provider add --json` dry-run 输出规范化 ID；rollback 缺 id 返回 E6005；release pin 仅 `daily` channel。
- **并发安全**：进度终态读取加锁（消除 race）；journal ID 回退防碰撞。
- **清理**：移除 `sethvargo/go-retry` 死依赖与空导入保活；provider variants 统一为 gpl/lgpl/nonfree；清理已删 flag 的 argv 表与文档残留。
- **依赖**：全部更新到最新版本（含 `github.com/google/go-github/v93`）。

## 新增基础设施

- `github.com/google/go-github/v93`：用于自更新的 GitHub Releases 查询，为将来从 GitHub 获取 FFmpeg release 资产打基础。
- `github.com/rogpeppe/go-internal/testscript`：新增 `cmd/one-more-ffmpeg/testdata/*.txtar` 脚本化 CLI 测试（version / provider list / JSON 契约），比 Python e2e 更贴近 Go 工具链。

## 安装（手动二进制）

```bash
# Linux
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre1/one-more-ffmpeg-v1.1.2-pre1-linux-amd64
curl -LO https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.2-pre1/SHA256SUMS.txt
sha256sum -c SHA256SUMS.txt --ignore-missing
```

Windows 下载 `one-more-ffmpeg-v1.1.2-pre1-windows-amd64.exe` 并按 `SHA256SUMS.txt` 校验。

详见 [`安装指南`](../guides/INSTALL.md)。

## 验证

代码视角：gofmt / vet / golangci-lint(0 issues) / go test / fuzz / 三平台交叉编译 / Linux race（6 包）全绿。
用户视角：本地 HTTP 自建服务器跑通 `provider add(http) → provider list → resolve → download(SHA-256) → install(dry-run)`；Windows 47 passed、Linux 49 passed、testscript 3 脚本全过。
