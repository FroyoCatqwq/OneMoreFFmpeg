# FFmpeg 能力检查契约

> 状态：Phase 0 设计
> 范围：v1 只支持用户列出的 13 类命令
> 原则：原样调用 FFmpeg、结构化解析、保留原文哈希

---

## 1. v1 公开能力命令

| 参数 | 输出主题 | 结构化字段 | 说明 |
|---|---|---|---|
| `-L` | license | `license.raw` / `license.summary` | 短 license |
| `-license` | license | `license.raw` / `license.summary` | 完整 license |
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

注意：表里是 13 行，因为 `-L` 与 `-license` 是两个独立命令；用户列出的能力组共 13 类，v1 按 13 个可调用参数实现。

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

`capabilities --json` 顶层：

```json
{
  "schema": "onemoreffmpeg.capabilities/v1",
  "installation_id": "...",
  "fingerprint": "...",
  "commands": {
    "-decoders": {
      "ok": true,
      "exit_code": 0,
      "stdout_sha256": "...",
      "stderr_sha256": "...",
      "duration_ms": 42
    }
  },
  "version": {},
  "license": {},
  "muxers": [],
  "demuxers": [],
  "devices": [],
  "decoders": [],
  "encoders": [],
  "filters": [],
  "pixel_formats": [],
  "channel_layouts": [],
  "sample_formats": [],
  "help": {},
  "warnings": []
}
```

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
7. **不硬编码 provider 差异**：BtbN 与 BnqDzj 的能力差异以实际输出为准。

---

## 5. 缓存

缓存路径：

```text
<data-root>/cache/capabilities/<installation_fingerprint>.json
```

`installation_fingerprint` 由以下字段生成：

```text
ffmpeg_sha256 + ffprobe_sha256 + ffplay_sha256
+ version stdout hash
+ -L stdout hash
+ OS + ARCH
+ OneMoreFFmpeg schema version
```

缓存失效：

- 任一二进制 hash 变化；
- version / license 输出变化；
- CLI 强制 `--refresh`；
- schema 主版本变化；
- 缓存文件 JSON 损坏。

---

## 6. 内部校验命令与公开能力的边界

v1 公开能力集合以本文件第 1 节为准。

安装器内部为了生成 `install.json` / `ffmpeg-trio.lock` 可以额外调用：

```text
ffmpeg -hide_banner -buildconf
```

但它不进入公开 `capabilities` 输出，也不算入用户要求的 13 类能力命令；后续若要开放，需要提升 `onemoreffmpeg.capabilities` schema 版本。

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

---

## 8. 错误语义

| 场景 | 行为 |
|---|---|
| ffmpeg 不存在 | 整体失败 |
| ffmpeg 无执行权限 | 整体失败 |
| 单个命令超时 | 该命令 `ok=false`，其他继续 |
| 单个命令非零退出 | 该命令 `ok=false`，继续 |
| 解析异常 | warning，保留 raw hash |
| 缓存损坏 | 重新探测，记录 warning |

`capabilities` 不修改任何文件，除了缓存目录；`--no-cache` 可完全只读。
