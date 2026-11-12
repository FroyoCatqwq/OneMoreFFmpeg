# FFmpeg 能力检查契约

> 状态：Phase 3 已实现
> 范围：v1 只支持用户列出的 13 个 FFmpeg 参数 / 能力入口
> 原则：原样调用 FFmpeg、结构化解析、保留原文哈希
> 实现入口：`internal/capabilities/`；CLI：`capabilities`
> 输出契约：[`CLI_SCHEMA.md`](CLI_SCHEMA.md)、[`schemas/examples/capabilities.example.json`](schemas/examples/capabilities.example.json)

---

## 1. v1 公开能力命令

| 参数 | 输出主题 | 结构化字段 | 说明 |
|---|---|---|---|
| `-L` | license | `license.raw` | show license |
| `-license` | license | `license.raw` | show license |
| `-h <topic>` | help topic | `help[topic].raw` | topic 必须 allowlist |
| `-version` | version | `version.*` | 版本与 build 信息 |
| `-muxers` | muxers | `muxers[]` | 可用封装器 |
| `-demuxers` | demuxers | `demuxers[]` | 可用解封装器 |
| `-devices` | devices | `devices[]` | 可用设备 |
| `-decoders` | decoders | `decoders[]` | 可用解码器 |
| `-encoders` | encoders | `encoders[]` | 可用编码器 |
| `-filters` | filters | `filters[]` | 可用滤镜 |
| `-pix_fmts` | pix_fmts | `pixel_formats[]` | 像素格式 |
| `-layouts` | layouts | `channel_layouts[]` | 标准声道布局 |
| `-sample_fmts` | sample_fmts | `sample_formats[]` | 音频采样格式 |

注意：表里是 13 行，因为 `-L` 与 `-license` 是两个独立命令；用户列出的能力入口共 13 个，v1 按 13 个可调用参数实现。

---

## 2. 调用规则

- 调用目标固定为已验证的 `ffmpeg` 可执行文件；
- 不通过 shell；
- 参数数组直接传递；
- 每个命令独立 timeout；
- 每个命令独立 context；
- stdout/stderr 分离；
- stdout 有最大字节限制；
- stderr 保存摘要与 hash；
- 退出码非 0 记为 `command_failed`，不让整个 capabilities 调用崩溃；
- `-h <topic>` 的 topic 只允许：
  - `full`
  - `decoder=<name>`
  - `encoder=<name>`
  - `filter=<name>`
  - `muxer=<name>`
  - `demuxer=<name>`
- 不在 allowlist 的 topic 直接拒绝，避免参数注入。

---

## 3. 公开输出模型

`capabilities --json` 使用 `CLI_SCHEMA.md` 的统一成功信封，以下对象位于 `data` 字段：

```json
{
  "schema": "onemoreffmpeg.result/v1",
  "ok": true,
  "command": "capabilities",
  "data": {
    "schema": "onemoreffmpeg.capabilities/v1",
    "installation_id": "...",
    "fingerprint": "...",
    "installation_fingerprint": "...",
    "created_at": "...",
    "cache_hit": false,
    "commands": {
      "-L": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-license": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-h full": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-version": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-muxers": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-demuxers": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-devices": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-decoders": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 42 },
      "-encoders": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-filters": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-pix_fmts": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-layouts": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 },
      "-sample_fmts": { "ok": true, "exit_code": 0, "stdout_sha256": "...", "stderr_sha256": "...", "duration_ms": 10 }
    },
    "version": { "raw": "...", "version": "...", "configuration": "..." },
    "license": { "raw": "...", "summary": "..." },
    "muxers": [],
    "demuxers": [],
    "devices": [],
    "decoders": [],
    "encoders": [],
    "filters": [],
    "pixel_formats": [],
    "channel_layouts": [],
    "sample_formats": [],
    "help": { "full": { "topic": "full", "raw": "...", "stdout_sha256": "...", "found": true } },
    "warnings": []
  }
}
```

说明：

- `commands` 固定包含 13 个键；`-h <topic>` 的键由 `--help-topic` 决定，例如 `-h full`、`-h decoder=h264`。
- 每个 command result 还可选包含 `stdout_bytes` / `stderr_bytes` / `truncated` / `stderr_summary` / `error`。
- `help[topic].found` 表示该 topic 是否真的被 FFmpeg 命中；未命中时 `help[topic].found=false`、对应 command `ok=false`、`error="help topic not found"`，并记录 `OMF-W1002`。
- `cache_hit` 为 `true` 时表示结果来自缓存，未重新执行 13 个命令。
- `created_at` 为 RFC 3339 UTC；缓存命中时保留首次生成时间。
- `fingerprint` 为 capabilities cache fingerprint；`installation_fingerprint` 为 Phase 2 `Installation.fingerprint`。
- 数组字段的元素均为 `CapabilityEntry`：`name` / `flags` / `description` / `raw`。

条目示例：

```json
{
  "name": "h264",
  "flags": "D.V.L.",
  "description": "H.264 / AVC",
  "raw": " V....D h264 ..."
}
```

`raw` 保留原行，解析失败时仍可审计。

---

## 4. 解析原则

1. **宽松解析**：FFmpeg 不同版本输出列宽/前缀会变，解析器不能因为一行异常直接整体失败；
2. **保留原文**：每个命令保存 stdout SHA-256，必要时保存受限原文；
3. **结构化优先**：常见格式解析为 name / flags / description；
4. **未知行警告**：无法解析的行计数并进入 `warnings`；
5. **去重排序**：同名能力稳定排序，避免缓存漂移；
6. **不猜能力**：没出现就是没有，不按 provider 名称推断；
7. **不硬编码 provider 差异**：BtbN 与 BnqDzj 的能力差异以实际输出为准；
8. **跳过表格表头**：`-pix_fmts` 的 `FLAGS NAME NB_COMPONENTS ...` 表头不是能力条目，必须跳过；
9. **分节解析**：`-layouts` 同时输出 `Individual channels` 与 `Standard channel layouts`，`channel_layouts[]` 只取后者，避免把 `FL` / `FR` 当成布局；
10. **不把错误文本当帮助**：`-h` 未命中时 FFmpeg 仍以 0 退出并输出 `Codec '...' is not recognized by FFmpeg.` / `Unknown filter|format '...'`，必须识别为未命中。

---

## 5. 缓存

data root 由 [`ENVIRONMENT.md`](ENVIRONMENT.md) §1.1 定义，实际缓存路径：

```text
<data-root>/cache/capabilities/<capabilities_fingerprint>.json
```

Windows 默认 `<LOCALAPPDATA>\\OneMoreFFmpeg\\cache\\capabilities`；Linux 默认 `~/.local/share/onemoreffmpeg/cache/capabilities`。可用 `--cache-dir <PATH>` 覆盖目录。

`capabilities_fingerprint` 由以下字段生成：

```text
onemoreffmpeg.capabilities/v1
+ capabilities command spec version
+ OS + ARCH
+ installation_id / installation_fingerprint / version
+ ffmpeg_sha256 + ffprobe_sha256 + ffplay_sha256
+ version stdout hash
+ -L stdout hash
+ help topic
+ 13 个 command key
```

缓存失效：

- 任一二进制 hash 变化；
- version / license 输出变化；
- CLI 强制 `--refresh`；
- schema 主版本变化；
- 缓存文件 JSON 损坏；
- 缓存缺少 13 个命令键；
- 缓存缺少当前 help topic 的 `help[topic]` 条目；
- 缓存记录的 `installation_id` 与目标安装不符。

`--no-cache` 完全不读不写缓存；`--refresh` 忽略旧缓存但会写入新结果。缓存损坏或过期时输出 `OMF-W1003`，缓存命中时 `cache_hit=true`。

缓存只在结果可用时写入：

- 父 context 被取消或超时（`OMF-E9004` / `OMF-E1006`）不写缓存；
- 13 个命令全部失败（`OMF-E2001`）不写缓存，避免把损坏安装的空结果长期缓存；
- 部分命令失败仍会写缓存，失败项以 `ok=false` + `OMF-W1002` 记录，供下次命中时审计。

命令规格版本从 `capability-commands-v1` 升为 `capability-commands-v3`（表头/分节解析、`help.found`、生效超时/输出上限进入缓存键），旧缓存按 fingerprint 不匹配自动失效。

---

## 6. 内部校验命令与公开能力的边界

v1 公开能力集合以本文件第 1 节为准。

安装器内部为了生成 `install.json` / `ffmpeg-trio.lock` 可以额外调用：

```text
ffmpeg -hide_banner -buildconf
```

但它不进入公开 `capabilities` 输出，也不算入用户要求的 13 个能力入口；后续若要开放，需要提升 `onemoreffmpeg.capabilities` schema 版本。

---

## 7. 超时与资源限制

默认值：

| 命令 | 超时 |
|---|---:|
| `-version` / `-L` / `-license` | 5s |
| `-h <topic>` | 5s |
| 列表类命令 | 15s |

默认 stdout 上限：8 MiB。
超过上限：截断、记录 warning、命令仍返回可解析结果。

并发：13 个命令默认以最多 4 个并发进程执行（`Collector.MaxParallel`），每个命令仍有独立 timeout 与 context；并发上限只影响墙钟时间，不影响输出内容和缓存键。

超时覆盖：显式传入 `--probe-timeout <秒>` 时，该值会覆盖所有能力命令的单命令超时（`Options.CommandTimeout`）；未显式传入时使用上表默认值。`--timeout` 只约束 detect 的整体发现，不改变能力命令超时。

---

## 8. 错误语义

| 场景 | 行为 |
|---|---|
| ffmpeg 不存在 | `OMF-E1002`，整体失败 |
| ffmpeg 无执行权限 | `OMF-E1005`，整体失败 |
| `--dir` / `--installation-id` 缺失或同时给出 | `OMF-E7001` |
| help topic 非法 | `OMF-E2005` |
| help topic 未命中 | 该命令 `ok=false`，`found=false`，记录 `OMF-W1002`，整体仍成功 |
| 单个命令超时 | 该命令 `ok=false`，其他继续，记录 `OMF-W1002` |
| 单个命令非零退出 | 该命令 `ok=false`，继续，记录 `OMF-W1002` |
| 13 个命令全部失败 | 整体 `OMF-E2001`，不写缓存 |
| 父 context 取消/超时 | `OMF-E9004` / `OMF-E1006`，不写缓存 |
| stdout 截断 | warning `OMF-W1002`，仍尝试解析 |
| 解析异常 | `OMF-W1002`，保留 raw hash |
| 缓存损坏/过期/不可读 | 重新探测，记录 `OMF-W1003` |
| 缓存写入失败 | 仍返回结果，记录 `OMF-W1007` |
| ffplay 缺失 | `OMF-W1001`，不误判 ffmpeg 不可用 |

`capabilities` 不修改任何文件，除了缓存目录；`--no-cache` 可完全只读。

---

## 9. Phase 3 CLI 映射

```text
one-more-ffmpeg capabilities --dir <PATH> --json
one-more-ffmpeg capabilities --installation-id <ID> --json
one-more-ffmpeg capabilities --dir <PATH> --help-topic decoder=h264 --json
one-more-ffmpeg capabilities --dir <PATH> --refresh --json
one-more-ffmpeg capabilities --dir <PATH> --no-cache --json
```

规则：

- `--dir` 与 `--installation-id` 必须二选一；
- `--installation-id` 会重新做发现以得到 `Installation`，可配合 `--no-path` / `--no-env` / `--no-common` / `--scan`；
- `--dir` 表示“只使用该目录”，不接受 `--scan` / `--recursive` / `--roots`，冲突时返回 `OMF-E7001`；
- `--no-validate` 在 capabilities 中禁用，必须得到 version/license stdout hash 才能算缓存指纹；
- 输出 `onemoreffmpeg.result/v1` + `onemoreffmpeg.capabilities/v1`；
- 所有能力命令都复用 `ffexec` 的 timeout、WaitDelay、进程树与输出上限。
