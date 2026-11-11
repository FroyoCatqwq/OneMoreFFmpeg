# Phase 2 实施状态

> 日期：2026-11-11
> 状态：Phase 2 已完成
> 工具链：Go 1.27.3 windows/amd64
> 适用：正式进入 Phase 3 前的发现/校验基线

---

## 1. 已完成内容

| 项 | 状态 |
|---|---|
| `internal/detect` 包 | 已建立 |
| 显式 `--dir` 发现 | 已实现，支持目录 / `bin/` / 可执行文件路径 |
| `ONE_MORE_FFMPEG_DIR` / `FFMPEG_DIR` / `LAE_FFMPEG_DIR` | 已实现 |
| PATH 发现 | 已实现，遍历 PATH 中全部目录 |
| 常见目录发现 | Windows / Linux 已实现 |
| `--scan` 全盘/指定根扫描 | 已实现 |
| Windows 默认根 | 固定盘，可选可移动盘 / 网络盘 |
| Linux 默认根 | `/`、`/home`、`/opt`、`/usr/local` |
| 扫描保护 | max-depth、max-files、timeout、排除目录、symlink dir 跳过 |
| 候选去重 | canonical root + 二进制 SHA-256，Windows 大小写不敏感 |
| 来源合并 | `explicit_dir` / `env` / `PATH` / `common_dir` / `scan` |
| 三件套路径查找 | `ffmpeg` / `ffprobe` / `ffplay`，Windows `.exe` |
| 执行权限判断 | 已实现 |
| `-version` 校验 | ffmpeg / ffprobe / ffplay |
| `-L` 校验 | ffmpeg |
| 核心版本一致性 | 已实现 |
| 二进制 SHA-256 | 已实现 |
| installation fingerprint / ID | 已实现 |
| 状态分类 | `healthy` / `degraded` / `unusable` / `unknown` |
| ffplay 缺失 warning | `OMF-W1001` |
| `detect --json` | 正式输出，不再有 Phase 1 `placeholder` |
| `validate --dir` | 已实现 |
| `validate --installation-id` | 已实现，重新发现并匹配 ID |
| 不下载 / 不安装 / 不改 PATH | 仍严格遵守 |

---

## 2. CLI 新增能力

### `detect`

```bash
one-more-ffmpeg detect --json
one-more-ffmpeg detect --dir C:/Tools/FFmpeg --json
one-more-ffmpeg detect --scan --roots /,/opt --json
one-more-ffmpeg detect --no-path --no-env --no-common --json
```

主要参数：

```text
--dir
--recursive
--scan
--roots
--max-depth
--max-files
--timeout
--probe-timeout
--include-removable
--include-network
--no-path
--no-env
--no-common
--no-validate
```

### `validate`

```bash
one-more-ffmpeg validate --dir C:/Tools/FFmpeg/bin --json
one-more-ffmpeg validate --installation-id inst-... --json
```

- `--dir` 与 `--installation-id` 必须二选一；
- `--installation-id` 不依赖持久化数据库，会重新做一次默认发现（不启用 `--scan`）；
- 输出 `onemoreffmpeg.validate/v1`。

---

## 3. 测试

新增 `internal/detect/detect_test.go` 覆盖：

- 显式目录 healthy；
- 环境变量发现；
- PATH 发现；
- 指定根扫描；
- max-files 限制；
- `validate --dir` healthy；
- ffplay 缺失 degraded；
- 版本不一致 unusable；
- 单个进程超时 unusable；
- POSIX 权限错误 unusable（Windows 跳过）；
- `validate --installation-id`。

更新：

- `internal/cli/root_test.go`：detect 不再是 placeholder；
- `internal/model/model_test.go`：validate nil slice 序列化为数组；
- `internal/testutil/fake_ffmpeg.go`：支持按程序名输出不同 version env。

验证结果：

```text
go build ./...                     通过
go test -count=1 ./...             通过
go vet ./...                       通过
go build -tags deps ./...          通过
GOOS=windows GOARCH=amd64 go build 通过
GOOS=linux   GOARCH=amd64 go build 通过
```

---

## 4. 明确未完成

- 13 项能力探测（Phase 3）；
- provider resolve / download（Phase 4）；
- install / env / rollback（Phase 5）；
- GitHub Actions / 发布 CI（Phase 7）；
- 包管理器识别；
- Go API / 插件接口；
- 自动选择安装（`--select`）。

---

## 5. 当前判定

> **Phase 2 完成，可以进入 Phase 3：能力检查。**

代码审查与残留 tech debt 见 [`PHASE2_CODE_REVIEW.md`](PHASE2_CODE_REVIEW.md)。

Phase 3 应复用 `internal/detect` 返回的 `Installation`，通过 `ffexec` 运行 13 个能力命令，不要绕过 Phase 1/2 的进程树、超时、输出上限与错误码契约。
