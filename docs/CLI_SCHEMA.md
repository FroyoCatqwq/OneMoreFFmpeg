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

## 3.1 全局参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `--json` | false | stdout 输出 v1 JSON；错误也输出错误对象 |
| `--config <PATH>` / `-c` | `<data-root>/config.toml`（存在时） | 显式指定 TOML；显式路径不存在、或 `ONE_MORE_FFMPEG_HOME` 非绝对路径 → `OMF-E7007` |
| `--log-level <LEVEL>` | 配置值或 `info` | 覆盖 `[log] level`；当前 `trace` 与 `debug` 输出相同（保留级别，写入类命令实现后再区分） |
| `--quiet` | false | 强制 `error` 级；与 `--log-level` 同时出现时 `--quiet` 优先 |
| `--in-detail-usage` | false | 打印内置完整使用指南并退出 0；与 `--json` 互斥（stdout 只留 JSON，冲突返回 `OMF-E1001`）。程序启动不再打印任何提示 |

配置加载顺序（已实现）：

1. 显式 `--config`（`--config ""` 表示只用内置默认值）；
2. `<data-root>/config.toml` 存在则加载，不存在则使用默认值；
3. 内置默认值。

未知子命令、未知 flag、多余位置参数都会返回 `OMF-E7001` 且退出码 2，不会退化为“打印帮助并退出 0”。

`--release <FILE>` 接受 UTF-8（可带 BOM）以及 UTF-16LE/BE（PowerShell 5 `>` 默认编码）；仍建议使用 PowerShell 7 的 `Out-File -Encoding utf8`。写入类命令中 `install` / `uninstall` / `env set|unset|rollback` / `journal rollback` / `provider add` 默认 dry-run；`download` 与 capabilities 缓存会立即写文件，脚本可用 `--no-cache` 关闭能力缓存。stdout 输出前会校验 `command` 与 `data` 契约，非法输出返回 `OMF-E7003`。

---

## 4. 命令专属 data

> 以下每张表都隐含第一行 `schema` 字段；具体值以示例或章节标题中的 schema 名为准，统一规则见 §2/§5。

实现状态：

| 命令 | 状态 |
|---|---|
| `version` / `detect` / `validate` / `capabilities` | 已实现 |
| `provider list` / `provider add` / `resolve` / `download` | 已实现（Phase 4） |
| `install` | 已实现（Phase 5，默认 dry-run） |
| `env status` / `doctor` | 已实现（只读） |
| `env set` / `env unset` / `env rollback` | 已实现（默认 dry-run） |
| `uninstall` | 已实现（默认 dry-run，隔离到 quarantine，可回滚） |
| `journal list` / `journal rollback` | 已实现（rollback 默认 dry-run） |
| `release list` / `release install` | 已实现（动态官方版本 + 近 7 天快照） |
| `capabilities --only <kind>` | 已实现（单命令、单列表输出） |

18 个命令的 data 表与示例**均已实现**，并由 `internal/contracts` 做字段级校验；`data.schema` 冻结为
`/v1`，未注册的 schema 或主版本升级会得到 `OMF-E7001`（见 [`PHASE6_STATUS.md`](PHASE6_STATUS.md)）。

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
| `--max-files <N>` | 200000 | 一次扫描（所有 root 合计）的最大文件数，超限记 `OMF-E1008` |
| `--timeout <SEC>` | 10 | 扫描/发现预算；仅 `detect` 要求 ≥ `--probe-timeout`，`validate` / `capabilities` 不使用该预算 |
| `--probe-timeout <SEC>` | 5 | 单个 ffmpeg 进程超时；不占用扫描预算 |
| `--include-removable` | false | Windows 默认根包含可移动盘；必须配合 `--scan` |
| `--include-network` | false | Windows 默认根包含网络盘；必须配合 `--scan` |
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

- `--dir` 只校验该目录，且与 `--scan` / `--recursive` / `--roots` 互斥（`OMF-E7001`）；父目录下的 `bin/` 会被自动发现；
- `--installation-id` 会重新做一次默认发现并匹配 ID，与 `--scan` / `--recursive` / 非空 `--roots` 互斥（`OMF-E7001`）；纯空白的 `--roots` 视为未提供；
- `--no-validate` 对 validate 无意义，使用即返回 `OMF-E7001`；
- 只要生成了报告就退出 0；判定结果在 `data.status`，脚本必须显式检查；
- `status=unusable` 时报告仍完整返回，不映射成整体错误码。

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
| `--cache-dir` | `<data-root>/cache/capabilities` | capability cache 目录；显式传入时必须为绝对路径，否则 `OMF-E7001` |
| `--refresh` | false | 忽略旧缓存并重新执行 |
| `--no-cache` | false | 不读不写缓存 |
| `--probe-timeout` | 5 | 显式传入时覆盖每个能力命令的单命令超时（秒） |
| `--no-path` / `--no-env` / `--no-common` | false | 控制 `--installation-id` 的重新发现范围 |
| `--scan` / `--roots` | false / 无 | 可选重新发现范围；与 `--dir` 互斥 |

`capabilities` 不提供 `--no-validate`（help 中也不显示），因为能力采集必须先完成校验。`--dir` 只使用该目录，与 `--scan` / `--recursive` / `--roots` 同时出现时返回 `OMF-E7001`。`--timeout` 只约束 detect 的发现预算；能力命令超时由 `--probe-timeout` 显式覆盖，可以大于 `--timeout`。未显式传入时使用 `CAPABILITIES.md` §7 的默认值。生效的单命令超时与输出上限参与缓存键。输出前会校验 13 个命令键与必需字段，缓存命中同样受此约束。

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
| `help` | object | key 为 help topic；值为 `{topic, raw, stdout_sha256, found}` |
| `warnings` | array | `OMF-W*` 列表 |

`help[topic].found=false` 表示 FFmpeg 未识别该 topic：对应命令 `ok=false`、`error="help topic not found"`、`warnings` 含 `OMF-W1002`。13 个命令全部失败时命令整体返回 `OMF-E2001`，不写缓存。

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

### 4.5 `provider list`（已实现，离线）

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `providers` | array | 内置 provider 在前（btbn、bnqdzj-nonfree），自定义 provider 按 `providers.toml` 顺序追加 |

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

### 4.6 `provider add`（已实现）

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 新 provider id（小写，保留 id 不可用） |
| `url` | string | HTTPS 直链 |
| `sha256` | string | 必填；64 位 hex |
| `status` | string | `added` / `dry-run` |

参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--id` | 无 | 必填；`^[a-z0-9][a-z0-9-]{0,63}$`，不能是 `btbn` / `bnqdzj-nonfree` / `custom` |
| `--url` | 无 | 必填；仅 https，格式必须与 `archive.FormatFor` 支持的一致：`.zip` / `.tar.xz` / `.tar.gz` / `.tgz` / `.tar.zst` |
| `--sha256` | 无 | 必填；缺失 `OMF-E3009`，格式错误同码 |
| `--checksums-url` | 无 | 可选；仅 https |
| `--license-hint` | `custom` | `gpl` / `lgpl` / `nonfree` / `custom` |
| `--apply` | false | 实际写入；不带时默认 dry-run |
| `--dry-run` | false | 强制预览；与 `--apply` 同时出现时以 dry-run 为准 |

整个 `LoadCustomFile → AddCustom → Save` 链路持有 `providers.toml.lock`；锁超时 `OMF-E7005`。

存储：`<data-root>/providers.toml`，schema `onemoreffmpeg.providers/v1`；重复 id 返回 `OMF-E7005`，不覆盖旧 pin。文件写入为临时文件 + rename。

示例：[`schemas/examples/provider-add.example.json`](schemas/examples/provider-add.example.json)

---

### 4.7 `resolve`（已实现）

`data` 字段遵循 `onemoreffmpeg.release/v1`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `provider` | string | provider id |
| `repository` | string | `owner/repo` 或自定义来源标识 |
| `channel` | string | `stable` / `daily` |
| `release_tag` | string | 具体 tag；`latest` 仅允许同时携带 `release_id` + `published_at` + `asset_sha256`（D1） |
| `release_id` | string | 可选；GitHub release 不可变 id，`release_tag=latest` 时必填 |
| `published_at` | string | RFC 3339 UTC |
| `pinned_at` | string | 可选；本次 pin 的 RFC 3339 时间 |
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
| `warnings` | array | 可选；`OMF-W1006` 等 |

`resolve` 选项：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--provider <ID>` | `btbn` | 内置或已登记 provider |
| `--channel <CH>` | `daily` | `stable` / `daily` |
| `--target <T>` | 当前平台 | `win64` / `linux64`；其他平台 `OMF-E3003` |
| `--variant <V>` | `gpl` | `gpl` / `lgpl` / `nonfree` / `nonfree-shared` |
| `--linkage <L>` | `static` | `static` / `shared` |
| `--release-line <X>` | 最高 `nX.Y` | 仅 btbn stable；`master` 只用于 daily |
| `--release-tag <TAG>` | 无 | 显式 pin；与 `--release-line` 互斥；daily 不接受 `latest` |
| `--allow-nonfree` | false | nonfree provider 必填，缺失返回 `OMF-E7001`；成功时输出 `OMF-W1006` |

`release_tag=latest` 的合法条件是同时具备 `release_id`、`published_at` 与非空 `asset_sha256`；缺少任一字段返回 `OMF-E3010`。其他 channel/tag 组合按 `PROVIDER_RESOLVER.md` §7。

实现行为：

- `stable` + btbn：解析 `latest` release，在 `nX.Y` 资产中取最高版本（可用 `--release-line` 指定）；`release_tag` 记录 `latest`，同时写 `release_id` / `published_at` / `asset_sha256`；
- `daily` + btbn/bnqdzj：列出 releases，取最新 `autobuild-*`，记录具体 tag；daily 必须匹配 master 资产，缺失返回 `E3003`，**不回退** stable-line 资产；
- 资产格式按 target 优先：win64 选 `.zip`，linux64 选 `.tar.xz`；同一 release line 仍有多个匹配时返回 `E3003`；
- `--release-tag` 显式指定时直接按 tag 解析；custom provider 不接受 `--release-tag` / `--release-line`（`E7001`）；
- 校验来源优先级：GitHub asset `digest` > release 内 `checksums.sha256` > `OMF-E3005`；两者同时存在但不一致返回 `OMF-E4002`；digest 存在而 checksums 拉取失败时降级为仅 digest（不写 `checksums_url` / `checksums_sha256`）；
- custom provider 的 `checksums_url` 会在 resolve 时拉取并计算 `checksums_sha256`，同时校验目标行；
- `resolve` 只产出 pin，不下载资产；`asset_url` 必须为 https；
- `[network] timeout` 与 `[download] max_redirects` 同时作用于 GitHub API 与 checksums 请求；`max_redirects=0` 表示禁止重定向；
- 输出前按 `onemoreffmpeg.release/v1` 做运行时字段级校验，不满足契约返回 `OMF-E7003` 而不是输出非法 pin；
- nonfree-only provider 只提供 `nonfree*` variant；`--variant gpl` 返回 `E7001`，且无论 variant 为何都要求 `--allow-nonfree`。

示例：[`schemas/examples/provider-resolve.example.json`](schemas/examples/provider-resolve.example.json)

详见 [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)。

---

### 4.8 `download`（已实现）

`data` 字段遵循 `onemoreffmpeg.artifact/v1`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `artifact_id` | string | `art-` + sha256 前 16 位 |
| `provider` | string | provider id |
| `release_tag` | string | resolve 记录的 tag（可能为 `latest` + pin） |
| `asset_name` | string | 资产名 |
| `asset_sha256` | string | SHA-256 |
| `local_path` | string | 本地文件路径 |
| `verified` | boolean | 固定 true；失败不会返回 artifact |
| `downloaded_at` | string | RFC 3339 UTC |
| `warnings` | array | 可选；nonfree 下载带 `OMF-W1006` |

参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--release <PATH>` | 无 | 读取 `resolve --json` 输出或裸 release 对象；给定时不再解析 |
| `--dir <PATH>` | `[download] dir` 或 `<data-root>/downloads` | 必须为绝对路径 |
| `--max-bytes <N>` | `[download] max_bytes`（512 MiB） | 覆盖单资产上限 |
| provider 参数 | 同 `resolve` | 未给 `--release` 时先执行 resolve |

落盘规则：

```text
<dir>/<asset_sha256[0:16]>/<asset_name>
<dir>/<asset_sha256[0:16]>/artifact.json
```

- 先校验 `checksums_url`（含其自身 SHA-256 与目标行），再下载资产；校验失败不会留下半成品；
- 已有同名文件且摘要匹配时复用，不重新下载资产；但 checksums 第二源每次仍重新校验；
- 先写 `*.part`，流式 SHA-256 通过后再原子提交；提交失败会备份旧文件并回滚；
- `artifact.json` 写入失败时删除本次新下载的资产，不保留“有资产无记录”的目录；复用已有资产时不会删除它；
- 失败删除临时文件与空目录；
- `license_hint=nonfree` 或 `variant=nonfree*` 仍需 `--allow-nonfree`，离线 release 文件同样适用；
- `--release` 文件按完整 `onemoreffmpeg.release/v1` 契约校验；不存在的路径 `E7007`，字段非法 `E7003`，未 pin 的 latest `E3010`。

示例：[`schemas/examples/artifact.example.json`](schemas/examples/artifact.example.json)

---

### 4.9 `install`（已实现，默认 dry-run）

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

参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--release <PATH>` | 无 | 离线 release JSON；未提供时按 provider 参数在线 resolve |
| `--artifact <PATH>` | 自动下载 | 已下载归档；必须绝对路径且 SHA-256 与 pin 一致（`E4002`） |
| `--apply` | false | 实际安装；不带该参数为 dry-run，不写任何文件 |
| `--force` | false | 目标目录已存在时先隔离旧目录再安装 |
| `--no-current` | false | 不更新 `current` 指针 |
| `--emit-trio-lock` | false | 在安装目录写 `ffmpeg-trio.lock` |
| `--allow-nonfree` | false | nonfree release 必填 |

落盘与事务：

```text
<data-root>/apps/<provider>/<release-tag>/<sha256[:16]>/
  ├─ bin/ffmpeg[.exe]
  ├─ bin/ffprobe[.exe]
  ├─ bin/ffplay[.exe]（若归档包含）
  ├─ install.json
  └─ ffmpeg-trio.lock（可选）
<data-root>/current                  symlink -> apps/...（Windows 无权限时 current.pointer）
<data-root>/journal/<id>.json        pending -> committed / failed
<data-root>/tmp/<journal-id>/        解压临时目录，成功后删除
```

- 归档只接受 `.zip` / `.tar.xz` / `.tar.gz` / `.tgz` / `.tar.zst`；其他 `E5001`；
- 拒绝绝对路径、`..`、盘符、symlink、非普通 tar 条目（`E5002`）；
- 安装后运行 `ffmpeg -version` / `ffprobe -version`（`E5008`）；
- 同一 asset 重复安装幂等；不同 asset 需 `--force`（`E5003`）；
- `current` 是真实目录时拒绝覆盖（`E5004`）；旧目录隔离失败 `E5007`。

示例：[`schemas/examples/install.example.json`](schemas/examples/install.example.json)

---

### 4.10 `env status`（已实现，只读）

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `current_type` | string | directory_junction / symlink / pointer_file / missing |
| `current_target` | string | current 指向的版本化目录 |
| `path_contains_current` | boolean | PATH 是否包含 current/bin |
| `resolved_ffmpeg` | string | 实际解析到的 ffmpeg |
| `variables` | object | 关键变量当前值摘要 |
| `warnings` | array | `OMF-W*` |

实现行为：

- `current_type` 由 `current` symlink / 真实目录 / `current.pointer` / 缺失判定；
- `path_contains_current` 检查进程 PATH 是否包含 `<data-root>/current/bin`（Windows 大小写不敏感）；
- `resolved_ffmpeg` 只在 `current/bin` 可访问时给出；
- `current.pointer` 模式会带 `OMF-W1005`（PATH 无法跟随 pointer）；
- `variables` 中未设置的变量输出 `null`。

示例：[`schemas/examples/env-status.example.json`](schemas/examples/env-status.example.json)

---

### 4.11 `env set` / `env unset` / `env rollback`（已实现，默认 dry-run）

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `dry_run` | boolean | 是否只预览 |
| `applied` | boolean | 是否实际写入 |
| `operations` | array | 操作摘要 |
| `journal_id` | string | 事务 ID |
| `rollback_supported` | boolean | 是否可回滚 |

参数：`--apply`（默认 dry-run）、`--no-path`、`--no-variables`、`--lae-compat`、`--no-shell-rc`、`--system`、`--authorize-system`；`env rollback --journal <id> --apply` 恢复备份，默认 dry-run。Windows 写入后广播 `WM_SETTINGCHANGE`；Linux 维护 `~/.profile` 托管块与 `~/.local/bin` shim，并支持 `--no-path` / `--no-variables` / `--lae-compat` 控制托管块内容。

`--system` 切换到系统级作用域（Windows `HKLM` 系统环境；Linux `/usr/local/bin` shim + `/etc/profile.d/onemoreffmpeg.sh`）。授权规则：进程已提权（root / Administrator）直接放行；未提权时 dry-run 可用 `--authorize-system` 预览，但 **`--apply` 必须真正提权**——`--authorize-system` 只记录同意、不能替代提权，未提权的 `--apply` 返回 `OMF-E6003`（退出 2）。工具绝不自动提权。详见 [`SYSTEM_REQUIREMENTS.md`](SYSTEM_REQUIREMENTS.md)。

`env set|unset|rollback` 的 `--apply` 路径持有全局 data root 事务锁 `<data-root>/.transaction.lock`；被其他事务占用且等待超时返回 `OMF-E6009`（退出 1）。

示例：[`schemas/examples/env-set.example.json`](schemas/examples/env-set.example.json)

---

### 4.12 `doctor`（已实现，只读）

`data` 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `healthy` | boolean | 总体健康 |
| `checks` | array | 每项含 `name` / `ok` / `detail` |
| `warnings` | array | `OMF-W*` |
| `journal_pending` | array | 未提交/未回滚 journal |

检查项：`data_root`、`current_exists`、`trio_valid`、`path_contains_current`、`no_pending_journal`。默认报告整体仍退出 0，脚本必须读取 `healthy`；`journal_pending` 为未完成 journal 的 ID 列表。`--strict` 在 `healthy=false` 时改为返回 `OMF-E6006`（退出 1），完整报告置于错误对象 `details.report`；健康时仍输出正常结果并退出 0。

示例：[`schemas/examples/doctor.example.json`](schemas/examples/doctor.example.json)

---

### 4.13 `uninstall`（已实现，默认 dry-run）

`data` 遵循 `onemoreffmpeg.uninstall/v1`：`root_dir` / `dry_run` / `applied` / `journal_id`。

- `--root <ABS>`：必须是 `<data-root>/apps/` 下的安装目录，否则 `OMF-E6007`；也可用 `--release <json>` 或 `--provider` + `--release-tag` 定位；
- `--apply`：实际删除；不带时为 dry-run；
- 删除后若 `current` 指向该目录，同时移除 current / current.pointer；
- 不删除下载归档、provider、配置或非本项目管理的目录。

---

### 4.14 `capabilities --only <kind>`（已实现）

`data` 遵循 `onemoreffmpeg.capability-list/v1`：`kind` / `entries` / `command` / `installation_id` / `fingerprint` / `warnings`。只执行该类别所需的 FFmpeg 命令，不输出其他类别。`--cache-dir` / `--no-cache` / `--refresh` 同样生效：缓存文件为 `list-<kind>-<fingerprint>.json`，写入失败记录 `OMF-W1007`。不支持的 `--only <kind>` 属于非法枚举值，返回 `OMF-E7005`（退出 2），而非 help topic 的 `OMF-E2005`。

### 4.15 `release list` / `release install`（已实现）

`release list` 的 `data` 遵循 `onemoreffmpeg.release-catalog/v1`：

- `channel=stable`：版本号来自 FFmpeg 官网 release 页，9.0 线展示其最后一个补丁（如 9.0.2），8.1 线展示 8.1.3；下载资产从 provider release URL 解析；`--days` 对 stable 不适用并输出 0，`--count` 会截断排序后的版本；
- `channel=daily`：默认展示最近 7 天的 `autobuild-*` 快照；`--days` 改变窗口，`--count` 限制数量；版本从资产名解析，无法解析时回退为 snapshot tag（覆盖 `master` / `nX.Y-latest` 资产）；
- `release install`：`--version` 或 `--snapshot` 选择（二者互斥），未指定时安装列表第一个；为支持选择旧版本，内部会忽略 `--count` 并扩大查找窗口；默认 dry-run，`--apply` 才安装；nonfree provider 需 `--allow-nonfree`，未显式指定 `--variant` 时自动使用 `nonfree`。

### 4.16 `journal list` / `journal rollback`（已实现）

- `journal list`：`onemoreffmpeg.journal-list/v1`，输出全部 record 摘要与 pending ID；
- `journal rollback --journal <id> --apply`：`onemoreffmpeg.journal-action/v1`，支持 `install` / `uninstall` / `env-set` / `env-unset`；默认 dry-run，只返回预览，预览的 `action` 取自记录本身；
- 校验在 dry-run 与 `--apply` 下都先执行：journal id 强制白名单，拒绝 `..` 与路径分隔符；id 非法、记录不存在或读取失败返回 `OMF-E6005`（退出 1）；记录已回滚（`rolled-back`）或 action 不在支持列表返回 `OMF-E7001`（退出 2）；
- `env rollback --journal <id>` 只接受 `env-set` / `env-unset` 记录；指向 install/uninstall 记录或已回滚记录返回 `OMF-E7001`，id 非法/缺失返回 `OMF-E6005`；
- uninstall 先把安装目录移入 `<data-root>/quarantine/<journal-id>/`，回滚时恢复目录与 current。

---

## 5. 版本与兼容性

> **Phase 6 契约冻结（2026-11-14）**：命令集（18 个 result.command）、`data.schema` 集（17 个，全 `/v1`）与退出码映射已冻结，
> 由 `internal/contracts/freeze_test.go` 在 CI 中守护——任何静默改动都会使测试失败。契约变更必须走主版本升级与 [`VERSIONING.md`](VERSIONING.md)。
> 外部消费者见 [`../examples/external-caller/`](../examples/external-caller/README.md)。

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
