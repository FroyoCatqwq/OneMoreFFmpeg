# CLI JSON 契约 v1

> 状态：Phase 0 契约
> 日期：2026-11-10
> 适用：当前唯一外部集成方式：调用 `one-more-ffmpeg` 可执行文件
> 配套：[`ERROR_CODES.md`](ERROR_CODES.md)、[`schemas/`](schemas/)

---

## 1. 原则

其他项目只调用 CLI，因此：

> **stdout JSON 就是对外 API。**

规则：

- stdout 只写机器结果；
- stderr 只写日志、进度、人类错误；
- `--json` 成功/失败都必须返回 JSON；
- 例外：`--help` 是交互帮助，始终写人类可读文本；机器消费者不要同时使用 `--json` 与 `--help`；
- 无 `--json` 时输出人类可读文本，但不能混入 stdout JSON；
- JSON 顶层必须有 `schema`；
- 错误对象必须使用 `ERROR_CODES.md` 的 `code`；
- 主版本变化即不兼容，不静默迁移；
- 本契约当前只有 v1，不承诺旧实验输出。

---

## 2. 通用成功对象

```json
{
  "schema": "onemoreffmpeg.result/v1",
  "ok": true,
  "command": "detect",
  "data": {}
}
```

字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `schema` | string | 是 | `onemoreffmpeg.result/v1` |
| `ok` | boolean | 是 | 固定 `true` |
| `command` | string | 是 | 子命令名；二级命令用点连接，如 `provider.list` / `env.status` |
| `data` | object | 是 | 命令专属数据；必须含 `schema`，不能为空对象 |

---

## 3. 通用错误对象

见 [`ERROR_CODES.md`](ERROR_CODES.md)。

```json
{
  "schema": "onemoreffmpeg.error/v1",
  "ok": false,
  "code": "OMF-E7001",
  "message": "invalid command usage",
  "details": {}
}
```

示例：[`schemas/examples/error.example.json`](schemas/examples/error.example.json)

---

## 4. 命令专属 data

> 以下每张表都隐含第一行 `schema` 字段；具体值以示例或章节标题中的 schema 名为准，统一规则见 §2/§5。

### 4.1 `version`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `schema` | string | `onemoreffmpeg.version/v1` |
| `name` | string | `OneMoreFFmpeg` |
| `version` | string | 语义版本 |
| `go_version` | string | 构建 Go 版本 |
| `git_commit` | string | 提交 hash |
| `build_time` | string | RFC 3339 UTC |
| `os` | string | `windows` / `linux` |
| `arch` | string | `amd64` |

> 开发构建允许 `git_commit` / `build_time` 为 `"unknown"`；发布构建应通过 `-ldflags` 注入真实提交与 RFC 3339 构建时间。

示例：[`schemas/examples/version.example.json`](schemas/examples/version.example.json)

---

### 4.2 `detect`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `schema` | string | `onemoreffmpeg.detect/v1` |
| `placeholder` | boolean | 已废弃；仅 Phase 1 占位使用，Phase 2 起省略 |
| `installations` | array | `Installation[]` |
| `errors` | array | 非致命来源错误 |

`detect` 命令选项：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--dir <PATH>` | 无 | 显式 trio 目录或 ffmpeg 可执行文件路径 |
| `--recursive` | false | 在 `--dir` 下递归查找 |
| `--scan` | false | 显式扫描默认根或 `--roots` |
| `--roots <CSV>` | 无 | 逗号分隔扫描根；必须配合 `--scan` |
| `--max-depth <N>` | 8 | 扫描最大深度 |
| `--max-files <N>` | 200000 | 每个扫描根最大文件数 |
| `--timeout <SEC>` | 10 | 整体 detection 超时 |
| `--probe-timeout <SEC>` | 5 | 单个 ffmpeg 进程超时 |
| `--include-removable` | false | Windows 默认根包含可移动盘 |
| `--include-network` | false | Windows 默认根包含网络盘 |
| `--no-path` | false | 跳过 PATH 发现 |
| `--no-env` | false | 跳过环境变量发现 |
| `--no-common` | false | 跳过常见目录发现 |
| `--no-validate` | false | 跳过运行 ffmpeg 校验命令 |

`Installation`：

```json
{
  "id": "inst-...",
  "root_dir": "...",
  "ffmpeg": "...",
  "ffprobe": "...",
  "ffplay": "...",
  "sources": ["explicit_dir", "PATH"],
  "status": "healthy",
  "version": "n9.0.2",
  "fingerprint": "...",
  "warnings": []
}
```

示例：[`schemas/examples/detect.example.json`](schemas/examples/detect.example.json)

---

### 4.3 `validate`

命令必须二选一：

```text
one-more-ffmpeg validate --dir <PATH>
one-more-ffmpeg validate --installation-id <ID>
```

- `--dir` 只校验该目录；
- `--installation-id` 会重新做一次默认发现（不启用 `--scan`）并匹配 ID；
- `validate` 继承 detect 的发现参数；`--no-validate` 对 validate 命令无意义，不建议使用。

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `installation_id` | string | 校验对象 |
| `status` | string | `healthy` / `degraded` / `unusable` / `unknown` |
| `checks` | array | 每项含 `name` / `ok` / `detail` |
| `warnings` | array | `OMF-W*` 列表 |

示例：[`schemas/examples/validate.example.json`](schemas/examples/validate.example.json)

---

### 4.4 `capabilities`

命令必须二选一：

```text
one-more-ffmpeg capabilities --dir <PATH>
one-more-ffmpeg capabilities --installation-id <ID>
```

主要参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--help-topic` | `full` | `full` 或 `decoder=<name>` / `encoder=<name>` / `filter=<name>` / `muxer=<name>` / `demuxer=<name>` |
| `--cache-dir` | 用户缓存目录 | capability cache 目录 |
| `--refresh` | false | 忽略旧缓存并重新执行 |
| `--no-cache` | false | 不读不写缓存 |
| `--no-path` / `--no-env` / `--no-common` | false | 控制 `--installation-id` 的重新发现范围 |
| `--scan` / `--roots` | false / 无 | 可选重新发现范围 |

`--no-validate` 在 `capabilities` 中禁止。

`data` 遵循 [`CAPABILITIES.md`](CAPABILITIES.md) 中的 `onemoreffmpeg.capabilities/v1`，必须包含：

| 字段 | 类型 | 说明 |
|---|---|---|
| `schema` | string | `onemoreffmpeg.capabilities/v1` |
| `installation_id` | string | Phase 2 installation id |
| `fingerprint` | string | capabilities cache fingerprint |
| `installation_fingerprint` | string | 可选，Phase 2 installation fingerprint |
| `created_at` | string | 可选，RFC 3339 UTC |
| `cache_hit` | boolean | 可选，true 表示来自缓存 |
| `commands` | object | 固定 13 个命令结果 |
| `version` | object | raw / version / configuration |
| `license` | object | raw / summary |
| `muxers` / `demuxers` / `devices` / `decoders` / `encoders` / `filters` / `pixel_formats` / `channel_layouts` / `sample_formats` | array | `CapabilityEntry[]` |
| `help` | object | key 为 help topic |
| `warnings` | array | `OMF-W*` 列表 |

`CapabilityEntry`：

```json
{
  "name": "h264",
  "flags": "V....D",
  "description": "H.264 / AVC",
  "raw": " V....D h264 ..."
}
```

示例：[`schemas/examples/capabilities.example.json`](schemas/examples/capabilities.example.json)

---

### 4.5 `provider list`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `providers` | array | provider 摘要列表 |

provider 摘要：

```json
{
  "id": "btbn",
  "repository": "FFmpeg-Builds/FFmpeg-Builds",
  "channels": ["stable", "daily"],
  "variants": ["gpl", "lgpl", "gpl-shared", "lgpl-shared"],
  "linkages": ["static", "shared"],
  "redistributable": true
}
```

示例：[`schemas/examples/provider-list.example.json`](schemas/examples/provider-list.example.json)

---

### 4.6 `provider add`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 新 provider id |
| `url` | string | HTTPS URL |
| `sha256` | string | 必填时记录 |
| `status` | string | `added` / `dry-run` |

示例：[`schemas/examples/provider-add.example.json`](schemas/examples/provider-add.example.json)

---

### 4.7 `resolve`

`data` 字段遵循 `onemoreffmpeg.release/v1`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `provider` | string | provider id |
| `repository` | string | `owner/repo` 或自定义来源标识 |
| `channel` | string | `stable` / `daily` |
| `release_tag` | string | 具体 tag，不能是 `latest` |
| `published_at` | string | RFC 3339 UTC |
| `asset_name` | string | 资产名 |
| `asset_url` | string | HTTPS URL |
| `asset_sha256` | string | SHA-256 |
| `checksums_url` | string | 可为空 |
| `checksums_sha256` | string | 可为空 |
| `variant` | string | gpl / lgpl / nonfree / ... |
| `linkage` | string | static / shared / unknown |
| `target` | string | win64 / linux64 |
| `license_hint` | string | gpl / lgpl / nonfree / custom |
| `redistributable` | boolean | nonfree 必须为 false |

示例：[`schemas/examples/provider-resolve.example.json`](schemas/examples/provider-resolve.example.json)

详见 [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)。

---

### 4.8 `download`

`data` 字段遵循 `onemoreffmpeg.artifact/v1`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `artifact_id` | string | 本地标识 |
| `provider` | string | provider id |
| `release_tag` | string | 具体 tag |
| `asset_name` | string | 资产名 |
| `asset_sha256` | string | SHA-256 |
| `local_path` | string | 本地文件路径 |
| `verified` | boolean | 是否校验通过 |
| `downloaded_at` | string | RFC 3339 UTC |

示例：[`schemas/examples/artifact.example.json`](schemas/examples/artifact.example.json)

---

### 4.9 `install`

`data` 字段遵循 `onemoreffmpeg.install/v1`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `installation_id` | string | 安装 ID |
| `root_dir` | string | 版本化安装目录 |
| `current_pointer` | string | `directory_junction` / `symlink` / `pointer_file` / `missing` |
| `provider` | string | provider id |
| `repository` | string | `owner/repo` 或自定义来源标识 |
| `release_tag` | string | 具体 tag |
| `asset_sha256` | string | SHA-256 |
| `binaries` | object | 三件套 hash |
| `license_hint` | string | gpl / lgpl / nonfree / custom |
| `redistributable` | boolean | nonfree 必须为 false |
| `lock_path` | string | 由 `--emit-trio-lock` 触发时的可选输出 |
| `journal_id` | string | 安装事务 ID |
| `dry_run` | boolean | 是否只预览 |

示例：[`schemas/examples/install.example.json`](schemas/examples/install.example.json)

---

### 4.10 `env status`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `current_type` | string | directory_junction / symlink / pointer_file / missing |
| `current_target` | string | current 指向的版本化目录 |
| `path_contains_current` | boolean | PATH 是否包含 current/bin |
| `resolved_ffmpeg` | string | 实际解析到的 ffmpeg |
| `variables` | object | 关键变量当前值摘要 |
| `warnings` | array | `OMF-W*` |

示例：[`schemas/examples/env-status.example.json`](schemas/examples/env-status.example.json)

---

### 4.11 `env set` / `env unset` / `env rollback`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `dry_run` | boolean | 是否只预览 |
| `applied` | boolean | 是否实际写入 |
| `operations` | array | 操作摘要 |
| `journal_id` | string | 事务 ID |
| `rollback_supported` | boolean | 是否可回滚 |

示例：[`schemas/examples/env-set.example.json`](schemas/examples/env-set.example.json)

---

### 4.12 `doctor`

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `healthy` | boolean | 总体健康 |
| `checks` | array | 每项含 `name` / `ok` / `detail` |
| `warnings` | array | `OMF-W*` |
| `journal_pending` | array | 未提交/未回滚 journal |

示例：[`schemas/examples/doctor.example.json`](schemas/examples/doctor.example.json)

---

## 5. 版本与兼容性

- 每个命令的 `data` 对象必须包含 `schema` 字段并带版本；顶层 `onemoreffmpeg.result/v1` 只表示信封；
- 新增可选字段允许；
- 删除字段、改变字段语义、改变类型、改变 `code` 语义都必须升主版本；
- 消费者必须忽略未知可选字段；
- `ERROR_CODES.md` 与本文档必须同步更新；
- 未来重新开放 Go API / 插件时，不得直接复用 Go 内部结构体作为协议。

---

## 6. 示例文件

示例位于：

```text
docs/schemas/examples/
```

它们是 v1 形状基准，不是完整 JSON Schema Draft 文件。
Phase 1 可以基于这些示例生成或手写 JSON Schema、Go struct validator 与 golden test。
