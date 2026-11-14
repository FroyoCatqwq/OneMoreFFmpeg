# Phase 3 实施状态

> 日期：2026-11-11
> 状态：Phase 3 已完成并通过代码审查
> 工具链：Go 1.27.4 windows/amd64（`go.mod` 要求 1.27.3）
> 审查与修复记录：[`PHASE3_CODE_REVIEW.md`](PHASE3_CODE_REVIEW.md)
> 适用：正式进入 Phase 4 前的能力检查基线

---

## 1. 已完成内容

| 项 | 状态 |
|---|---|
| `internal/capabilities` 包 | 已建立 |
| 13 个能力入口 | 已全部执行并纳入 `commands` |
| `-L` / `-license` | 已解析 license.raw / summary |
| `-h <topic>` | 已实现 allowlist 与按 topic key 输出 |
| `-version` | 已解析 raw / version / configuration |
| `-muxers` / `-demuxers` / `-devices` / `-decoders` / `-encoders` / `-filters` | 已解析 `CapabilityEntry[]` |
| `-pix_fmts` / `-layouts` / `-sample_fmts` | 已解析 `CapabilityEntry[]` |
| 宽松解析 | 已实现，未知行计数并写 `OMF-W1002` |
| `-pix_fmts` 表头跳过 | 已实现（不会产生 `NAME` 条目） |
| `-layouts` 分节解析 | 已实现（`channel_layouts[]` 只含标准布局） |
| `-h <topic>` 未命中识别 | 已实现，`found=false` + `OMF-W1002` |
| 去重排序 | 已实现 |
| stdout / stderr SHA-256 | 已记录 |
| stdout / stderr 字节数与 stderr 摘要 | 已记录 |
| stdout 截断标记 | 已记录 |
| 每命令独立 timeout / context | 已通过 `ffexec.Runner` 实现 |
| 单命令失败不拖垮整体 | 已实现 |
| 能力缓存 | 已实现 |
| `--refresh` | 已实现 |
| `--no-cache` | 已实现 |
| `--cache-dir` | 已实现 |
| ffplay 缺失 `OMF-W1001` | 已实现 |
| 缓存损坏/过期 `OMF-W1003` | 已实现 |
| `capabilities --json` | 已实现 |
| golden / parser / cache / CLI 测试 | 已实现 |

---

## 2. 13 个命令键

```text
-L
-license
-h full
-version
-muxers
-demuxers
-devices
-decoders
-encoders
-filters
-pix_fmts
-layouts
-sample_fmts
```

`-h <topic>` 的 key 随 `--help-topic` 变化，例如：

```text
-h decoder=h264
-h encoder=libx264
-h filter=scale
-h muxer=mp4
-h demuxer=mov,mp4
```

help topic 只允许：

```text
full
decoder=<name>
encoder=<name>
filter=<name>
muxer=<name>
demuxer=<name>
```

`<name>` 仅允许 `A-Z a-z 0-9 . _ + , = -`，不允许空格、路径分隔符、shell 元字符。

---

## 3. CLI

```bash
one-more-ffmpeg capabilities --dir C:/Tools/FFmpeg/bin --json
one-more-ffmpeg capabilities --installation-id inst-... --json
one-more-ffmpeg capabilities --dir C:/Tools/FFmpeg/bin --help-topic decoder=h264 --json
one-more-ffmpeg capabilities --dir C:/Tools/FFmpeg/bin --refresh --json
one-more-ffmpeg capabilities --dir C:/Tools/FFmpeg/bin --no-cache --json
```

规则：

- `--dir` / `--installation-id` 必须二选一；
- `--installation-id` 会重新做 Phase 2 发现，可配合 `--no-path` / `--no-env` / `--no-common` / `--scan`；
- `--no-validate` 禁止，因为缓存 fingerprint 需要 version/license stdout hash；
- `--dir` 强制只校验该目录；
- 输出 `onemoreffmpeg.result/v1` + `onemoreffmpeg.capabilities/v1`。

---

## 4. 缓存

默认：

```text
<data-root>/cache/capabilities/<capabilities_fingerprint>.json
```

Windows 默认 `%LOCALAPPDATA%\OneMoreFFmpeg\cache\capabilities`；Linux 默认 `~/.local/share/onemoreffmpeg/cache/capabilities`（见 `ENVIRONMENT.md` §1.1）。

- `--cache-dir` 覆盖；
- `--no-cache` 不读不写；
- `--refresh` 忽略旧缓存但写入新结果；
- 缓存命中输出 `cache_hit=true`；
- 缓存损坏、过期、缺命令键时重新探测并输出 `OMF-W1003`。

`capabilities_fingerprint` 包含：

```text
schema + command spec version
+ OS + ARCH
+ installation_id / installation_fingerprint / version
+ ffmpeg / ffprobe / ffplay SHA-256
+ version stdout SHA-256 + -L stdout SHA-256
+ help topic
+ 13 个 command key
```

---

## 5. 测试

新增：

- `internal/capabilities/parser_test.go`
- `internal/capabilities/capabilities_test.go`

覆盖：

- 13 个命令键全部存在且成功；
- version / license / list / help 解析；
- 真实 `-pix_fmts` 表头、`-layouts` 分节、GPL / nonfree `-L` 文本；
- 未知行 warning；
- 非法 help topic `OMF-E2005`；
- help topic 未命中 `found=false` + `OMF-W1002`；
- ffplay 缺失 `OMF-W1001`；
- cache hit；
- `--refresh` 重新执行；
- 缓存损坏后重跑；
- `--no-cache` 不写缓存；
- 单命令失败只记录 `ok=false`；
- 13 个命令全部失败 → `OMF-E2001` 且不写缓存；
- context 取消 → `OMF-E9004` 且不写缓存；
- 显式 `--probe-timeout` 覆盖所有能力命令超时；
- 缓存缺少 `help[topic]` 条目时拒绝命中并重新采集；
- 缓存目录默认派生自 data root（`internal/paths`）；
- CLI 缺输入 / `--no-validate` / `--dir` + `--scan` 冲突 / happy path；
- `CapabilitySet` nil slice/map 序列化为 `[]` / `{}`。

---

## 6. 验证结果

```text
go build ./...                      通过
go test -count=1 ./...              全部通过
go vet ./...                        通过
go build -tags deps ./...           通过
GOOS=windows GOARCH=amd64 go build  通过
GOOS=linux   GOARCH=amd64 go build  通过
Markdown 相对链接                   OK
Markdown/JSON/Go 尾随空格           OK
JSON 示例解析                       OK
Linter 诊断                        0
```

CLI 冒烟：

- `capabilities --json` 无输入 → `OMF-E7001`，退出码 2；
- `capabilities --dir does-not-exist --json` → `OMF-E1001`，退出码 1；
- `capabilities --dir <dir> --scan --json` → `OMF-E7001`，退出码 2；
- `--help` 已列出 `version` / `detect` / `validate` / `capabilities`。

真实 FFmpeg 冒烟（`N-128713-g0f89d936b3-20261108` win64 nonfree static）：

- 13 个命令全部 `ok=true`，`warnings=[]`；
- `pixel_formats=270`，无 `NAME` 表头条目；
- `channel_layouts=42`，无 `FL` / `FR` / `LFE`；
- `help.full.found=true`；
- `--help-topic decoder=nosuch` → 该命令 `ok=false` + `OMF-W1002`；
- 连续两次执行，第二次 `cache_hit=true`，缓存目录只有 1 个文件。

---

## 7. 明确未完成

- ~~provider resolve / download~~（Phase 4 已完成，见 [`PHASE4_STATUS.md`](PHASE4_STATUS.md)）；
- install / env / rollback（Phase 5）；
- GitHub Actions / 发布 CI（Phase 7）；
- 包管理器识别；
- Go API / 插件接口；
- 公开二进制发布与完整许可证复核。

---

## 8. 当前判定

> **Phase 3 完成，可以进入 Phase 4：Provider / Resolve / Download。**

Phase 4 应复用 Phase 2 `Installation` 与 Phase 1 `ffexec`；不要绕过已有错误码、进程边界、超时和只读发现原则。
