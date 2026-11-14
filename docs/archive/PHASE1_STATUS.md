# Phase 1 实施状态

> 日期：2026-11-10
> 状态：Phase 1 骨架已完成
> 工具链：Go 1.27.3 windows/amd64
> 适用：正式进入 Phase 2 前的最小骨架

---

## 1. 已完成内容

| 项 | 状态 |
|---|---|
| `go.mod` | 锁定 `go 1.27.3` 与建议依赖版本 |
| `go.sum` | 已生成 |
| `cmd/one-more-ffmpeg` | 已建立 |
| `internal/model` | CLI JSON 内部模型 |
| `internal/errors` | 66 个错误/警告码注册表 |
| `internal/config` | TOML 1.1 strict mode |
| `internal/logging` | `log/slog` + `tint` + TTY 判断；`Before` 已接线 debug 日志 |
| `internal/ffexec` | `exec.CommandContext`、超时、WaitDelay、进程组/进程树取消、输出上限、SHA-256、env 覆盖/清理 |
| `internal/contracts` | JSON 示例与错误码文档校验 |
| `internal/testutil` | 跨平台 fake ffmpeg 测试框架 |
| `version --json` | 已实现 |
| `detect --json` | Phase 1 只读占位，`placeholder=true`，真实发现留到 Phase 2 |
| `--help` | 已验证 |
| 错误 JSON | `OMF-E7007` 等路径已验证 |
| 单元测试 | `go test ./...` 通过 |
| `go vet` | 通过 |
| Windows 交叉编译 | `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` 通过 |
| Linux 交叉编译 | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` 通过 |
| `THIRD_PARTY_NOTICES.md` | 初稿完成，许可证文本已复制到 `third_party/licenses/` |

---

## 2. 验证输出

`go test ./...`：

```text
ok internal/buildinfo
ok internal/cli
ok internal/config
ok internal/contracts
ok internal/errors
ok internal/ffexec
ok internal/logging
ok internal/model
```

`go vet ./...`：通过。

CLI 冒烟：

```text
one-more-ffmpeg version --json
one-more-ffmpeg detect --json
one-more-ffmpeg --help
one-more-ffmpeg --config does-not-exist.toml version --json
```

结果：

- `version --json` 输出 `onemoreffmpeg.result/v1` + `onemoreffmpeg.version/v1`；
- `detect --json` 输出 `onemoreffmpeg.result/v1` + `onemoreffmpeg.detect/v1`，`placeholder=true`，空 installations/errors；
- `--help` 正常；
- 缺失配置返回 `OMF-E7007`，退出码 2。

---

## 3. 目录结构

```text
cmd/one-more-ffmpeg/
internal/
  buildinfo/
  cli/
  config/
  contracts/
  deps/
  errors/
  ffexec/
  logging/
  model/
  testutil/
third_party/licenses/
config.example.toml
THIRD_PARTY_NOTICES.md
```

---

## 4. 明确未完成

- 真实 FFmpeg 发现与校验（Phase 2）；
- 13 项能力探测（Phase 3）；
- provider resolve / download（Phase 4）；
- install / env / rollback（Phase 5）；
- 完整 CI / Release（Phase 7）；
- Go API / 插件 / 包管理器识别（未来）；
- 公开二进制发布与完整许可证复核。

---

## 5. 当前判定

> **Phase 1 完成，P0/P1 代码审查问题已修复，可以进入 Phase 2：发现与可用性校验。**

代码审查与残留 tech debt 见 [`PHASE1_CODE_REVIEW.md`](PHASE1_CODE_REVIEW.md)。

进入 Phase 2 前不再需要补文档契约；但需要保持 `docs/schemas/examples/` 与 Go 结构体同步。
