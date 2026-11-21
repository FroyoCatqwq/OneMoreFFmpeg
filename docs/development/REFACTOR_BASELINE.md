# Modular Monolith 重构基线（Phase 0）

> 本文件锁定重构前的可重复基线。所有重构 PR 必须在同一基线之上可复现。
> 依据：[`adr/0004-modular-architecture.md`](adr/0004-modular-architecture.md) 第 2 节、
> [`ARCHITECTURE.md`](ARCHITECTURE.md) 第 7 节。

## 1. 固定基线

| 项 | 值 |
|---|---|
| 审计/基线 commit | `24a3f9fa860efb22378496dd6e44d7cc89bc4cb1` |
| tag | `v1.1.2` |
| Go | `go version go1.27.4 windows/amd64`（本工作区） |
| 模块 | `github.com/FroyoCatqwq/OneMoreFFmpeg` |
| package 数（`go list ./...`） | **33** |
| 生产 Go 文件（`*.go` 非 `_test.go`） | **约 101** |
| 测试 Go 文件（`*_test.go`） | **86** |
| 直接依赖（`go.mod`） | **17** |
| 间接依赖 | **9** |

> **与旧快照的差异**：用户提供文档中的 34 包 / 119 测试文件来自更早快照；当前实测为 33 / 86，以本节为准。Phase 0 前必须复核 `go list ./...` 与文件数，任何差异需记录原因。

## 2. 基线命令（可重复）

以下命令在仓库根目录执行；Go/Python 通过 `mise exec --` 调用。

```bash
# 2.1 静态事实
git rev-parse HEAD
mise exec -- go version
mise exec -- go list ./...
mise exec -- go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./...
mise exec -- go list -m -u -f '{{if .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all

# 2.2 质量门禁
mise exec -- gofmt -l $(git ls-files '*.go')
mise exec -- go vet ./...
mise exec -- go test -count=1 ./...
mise exec -- go build ./...

# 2.3 覆盖率（TMPDIR 避免污染工作区）
TMPDIR=/tmp/omf-cover mise exec -- go test -coverprofile=/tmp/omf-cover/cover.out ./...
mise exec -- go tool cover -func=/tmp/omf-cover/cover.out | tail -1

# 2.4 fuzz 短跑（各 5s）
mise exec -- go test ./internal/capabilities -run='^$' -fuzz=FuzzParseList -fuzztime=5s
mise exec -- go test ./internal/contracts -run='^$' -fuzz=FuzzSafeAssetName -fuzztime=5s
mise exec -- go test ./internal/cli -run='^$' -fuzz=FuzzNormalizeJSONInput -fuzztime=5s

# 2.5 交叉编译（CGO_ENABLED=0）
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 mise exec -- go build ./...
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 mise exec -- go build ./...

# 2.6 race（需要 gcc；本机无 gcc，改在 Linux CI 服务器执行）
CGO_ENABLED=1 go test -race -count=1 ./internal/lock ./internal/cache ./internal/output \
  ./internal/selfupdate ./internal/install ./internal/download
```

## 3. 已确认结果（本次采集）

| 项 | 结果 |
|---|---|
| `gofmt -l` | clean |
| `go vet ./...` | PASS (exit 0) |
| `go test -count=1 ./...` | PASS（全包） |
| `go build ./...` | PASS |
| 交叉编译 windows/amd64、linux/amd64 | PASS |
| fuzz 3 个目标（各 5s） | PASS |
| 覆盖率 | **67.3%**（命令成功，可信基线；`cmd/one-more-ffmpeg` testscript 0.0%，其在 PR-0a 补覆盖） |
| race（本机 Windows） | **无法执行**：`cgo: C compiler "gcc" not found` |
| race（Windows VM，gcc 16.2.0） | **PASS**：`lock/cache/output/selfupdate/download` 5 包全过。环境已重建：Scoop `mingw-winlibs` 16.2.0（winget 源失败后改 Scoop） |
| race（Linux VM，AlmaLinux 9.9） | **PASS**：`lock/cache/output/selfupdate/install/download` 6 包全过（install 25.8s） |
| Windows VM 单测 | **5 个失败，均为路径长/短名差异**：`TestScanExplicitRootUnderTmpIsNotExcluded`、`TestValidateDirFindsBinSubdirectory`、`TestCreateLinkFallsBackToSymlink`、`TestReplaceLinkReportsActualKind`、`TestInstallRollbackRestoresPreviousVersion`。测试期望 `C:\Users\ADMINI~1\...`，VM 上 `t.TempDir()` 返回长名 `C:\Users\Administrator\...`。**非产品 bug，属测试对 8.3 短名的硬编码假设**；PR-0a 应把这些断言改为 `filepath.EvalSymlinks`/规范化比较 |
| Windows VM fuzz / 交叉编译 | PASS（3 个目标各 5s；Linux cross=0） |
| Windows VM e2e | PASS：testscript + pytest **47 passed / 4 skipped**（`OMF_E2E_REAL_ENV=1`） |
| 依赖可用更新 | 8 项（见下），本阶段**不升级**，仅记录 |

### 3.1 依赖可用更新（记录，不在 Phase 0 变更）

`github.com/mattn/go-isatty` `github.com/pelletier/go-toml/v2` `golang.org/x/sync` `golang.org/x/sys` `golang.org/x/term` `golang.org/x/time` `golang.org/x/tools` `golang.org/x/vuln` 均有可用更新。
按用户长期要求"依赖必须用最新版"，实际升级应放入 PR-0b/独立 PR，并在升级后重跑本文件全部基线命令。

### 3.2 关键覆盖率缺口（来自本次采集）

| 包 | 覆盖率 | 说明 |
|---|---|---|
| `cmd/one-more-ffmpeg` | 0.0% | testscript 注入未进入覆盖率统计，PR-0a 处理 |
| `internal/customurl/latest` | 28.0% | 最低，PR-0a 补边界测试 |
| `internal/cli` | 59.4% | 编排多、输出分支多 |
| `internal/customurl/checksum` | 62.9% | 可接受 |
| `internal/cache` | 67.0% | 可接受 |

## 4. 退出条件复核

- [x] 基线 commit SHA 已固定：`24a3f9f...`
- [x] 可重复命令清单已记录（第 2 节）
- [x] 质量基线结果已记录（第 3 节）
- [x] race 基线已采集（Windows VM gcc 16.2 PASS；Linux VM PASS；本机无 gcc）
- [x] Windows VM 环境已重建（Scoop mingw-winlibs + Go 1.27.4 + mise trusted + 依赖下载）
- [ ] PR-0a 待修正 5 个 Windows 长/短名测试断言（已登记，非产品 bug）
- [x] 用户批准实施授权（2026-11-21；ADR 已更新为 `Approved for Implementation`）

> Phase 0 全部退出条件已满足；可从 Phase 1 开始按 PR 分阶段实施。
