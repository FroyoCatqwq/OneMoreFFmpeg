# 错误码注册表 v1

> 状态：Phase 0 契约
> 日期：2026-11-10
> 适用：CLI stdout JSON、日志 `code` 字段、未来 JSON-RPC `error.code`
> 原则：稳定、不可复用、不可静默改语义

---

## 1. 错误对象

所有 CLI 失败输出统一为：

```json
{
  "schema": "onemoreffmpeg.error/v1",
  "ok": false,
  "code": "OMF-E1001",
  "message": "directory does not exist",
  "details": {
    "path": "C:/Tools/FFmpeg"
  },
  "cause": "optional upstream diagnostic"
}
```

字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `schema` | string | 是 | 固定为 `onemoreffmpeg.error/v1` |
| `ok` | boolean | 是 | 固定为 `false` |
| `code` | string | 是 | 稳定错误码，来自本文档 |
| `message` | string | 是 | 短文本，不超过 200 字符 |
| `details` | object | 是 | 机器可读上下文，可为空对象 |
| `cause` | string | 否 | 上游诊断摘要，不作为稳定机器接口 |

日志、stderr、未来插件协议中的 `code` 必须使用同一注册表。

---

## 2. 退出码

| CLI 退出码 | 含义 | 典型 code |
|---:|---|---|
| 0 | 成功 | 无 |
| 1 | 运行期错误 | 除下表列出的用法/取消码之外的 E1xxx–E6xxx、`E7006`、E9xxx |
| 2 | 用法/配置/契约错误 | `E7001–E7005`、`E7007`、`E2005`、`E3001`、`E3004`、`E3008`、`E3009`、`E6003`、`E6004` |
| 130 | 用户取消 | `OMF-E1011`、`OMF-E4006`、`OMF-E9004` |

> 退出码由 `internal/errors.exitFor` 唯一决定；上表两行不重叠：凡是退出 2 的 E2/E3/E6 码都不再计入“运行期错误”。

---

## 3. 编号范围

```text
OMF-E1xxx  发现 / 校验 / FFmpeg 进程
OMF-E2xxx  能力探测
OMF-E3xxx  provider / resolve / GitHub API
OMF-E4xxx  download / checksum
OMF-E5xxx  install / archive / current
OMF-E6xxx  environment / PATH / journal
OMF-E7xxx  CLI / config / TOML / JSON
OMF-E9xxx  internal / cancellation

OMF-Wxxxx  警告码（不导致命令失败）
```

---

## 4. E1xxx：发现 / 校验 / FFmpeg 进程

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E1001` | `DIR_MISSING` | 显式目录不存在；也用于 `--in-detail-usage` 与 `--json` 互斥（stdout 只留 JSON） | 1 |
| `OMF-E1002` | `BINARY_MISSING` | ffmpeg/ffprobe/ffplay 缺失 | 1 |
| `OMF-E1003` | `VERSION_MISMATCH` | 三件套版本核心不一致 | 1 |
| `OMF-E1004` | `INSTALLATION_UNUSABLE` | 安装不可用（缺二进制或校验失败） | 1 |
| `OMF-E1005` | `PERMISSION_DENIED` | 无执行或读取权限 | 1 |
| `OMF-E1006` | `PROCESS_TIMEOUT` | FFmpeg 命令超时 | 1 |
| `OMF-E1007` | `PROCESS_FAILED` | FFmpeg 非零退出 | 1 |
| `OMF-E1008` | `SCAN_LIMIT_EXCEEDED` | 全盘扫描超过深度/数量/时间上限 | 1 |
| `OMF-E1009` | `SCAN_ROOT_INVALID` | 扫描根无效或不可读 | 1 |
| `OMF-E1010` | `SYMLINK_LOOP` | 符号链接循环 | 1 |
| `OMF-E1011` | `CANCELED` | 用户取消 | 130 |
| `OMF-E1012` | `EXEC_FAILED` | 进程无法启动（可执行文件损坏/格式错误） | 1 |

> `ffexec` 对非零退出返回 `Result.ExitCode`，不直接返回 error；由能力探测/运行层按策略映射成 `OMF-E1007`。
> 超时、取消、I/O 未收敛或启动失败时 `Result.ExitCode = -1`（`ffexec.ExitCodeNotStarted`），不会伪装成 0。

保留说明：

- `E1003`（三件套版本不一致）目前不单独返回：版本差异写入 `checks[]` 并由 `status=degraded` 表达，Phase 5 安装校验可复用；
- `E1004`（安装不可用）目前只在“显式目录无有效安装”时由 `validate --dir` / `capabilities --dir` 使用；
- `E1010`（symlink loop）暂不可达：`WalkDir` 不跟随符号链接，循环不会发生；
- `E1011`（用户取消）保留给未来交互式命令；当前 context 取消统一 `E9004`，下载取消使用 `E4006`。

---

## 5. E2xxx：能力探测

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E2001` | `CAPABILITY_COMMAND_FAILED` | 某个能力命令启动失败或超时 | 1 |
| `OMF-E2002` | `CAPABILITY_PARSE_FAILED` | 输出整体无法解析 | 1 |
| `OMF-E2003` | `CAPABILITY_CACHE_READ` | 缓存读取失败，且无法回退 | 1 |
| `OMF-E2004` | `CAPABILITY_CACHE_WRITE` | 缓存写入失败 | 1 |
| `OMF-E2005` | `HELP_TOPIC_REJECTED` | `-h <topic>` 不在 allowlist | 2 |
| `OMF-E2006` | `OUTPUT_LIMIT_EXCEEDED` | stdout 超过上限且无法截断恢复 | 1 |

说明：

- 单个能力命令失败通常应记录 warning，不一定导致整个 `capabilities` 命令失败；
- 只有所有能力命令都失败或缓存/输出不可用时才返回整体错误；
- 当前实现：单个命令失败 → `OMF-W1002`；13 个命令全部失败 → `OMF-E2001` 且不写缓存；`-h <topic>` 未命中 → 该命令 `ok=false` + `OMF-W1002`，整体仍成功；
- `OMF-E2002` / `OMF-E2003` / `OMF-E2004` / `OMF-E2006` 为保留码：当前解析器对未知行降级为 `OMF-W1002`，缓存读取失败降级为 `OMF-W1003`，缓存写入失败降级为 `OMF-W1007`，不单独使用这些码。
- `OMF-E2005` 仅用于 `-h <topic>` 的 help topic 校验；`capabilities --only <kind>` 的 kind 不在支持列表属于非法枚举值，使用 `OMF-E7005`（退出 2），不复用 `E2005`。

---

## 6. E3xxx：Provider / Resolve

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E3001` | `PROVIDER_UNKNOWN` | provider id 未注册 | 2 |
| `OMF-E3002` | `RELEASE_NOT_FOUND` | 指定 release/tag 不存在 | 1 |
| `OMF-E3003` | `ASSET_NOT_FOUND` | 当前 target/variant/linkage 无匹配资产 | 1 |
| `OMF-E3004` | `CHANNEL_UNSUPPORTED` | provider 不支持请求的 channel | 2 |
| `OMF-E3005` | `CHECKSUMS_FILE_MISSING` | 缺少可信 checksums 文件 | 1 |
| `OMF-E3006` | `GITHUB_RATE_LIMITED` | GitHub API 限流或 429/403 | 1 |
| `OMF-E3007` | `GITHUB_API_FAILED` | GitHub API 其他失败 | 1 |
| `OMF-E3008` | `CUSTOM_URL_INVALID` | 自定义 URL 非法或非 https | 2 |
| `OMF-E3009` | `CUSTOM_SHA256_REQUIRED` | 自定义 direct asset 缺少 SHA-256 | 2 |
| `OMF-E3010` | `RELEASE_NOT_PINNED` | `release_tag=latest` 且缺少 `release_id` / `published_at` / `asset_sha256` | 1 |

说明：

- `release_tag=latest` 不是错误；只有缺 pin 字段时才是 `E3010`（D1），离线 release 文件、契约校验与 `resolve` 输出统一使用该码；
- 具体 tag（含 `autobuild-*`）本身就是 pin，不需要 `release_id`；
- nonfree provider 缺少 `--allow-nonfree` 属于用法错误，使用 `OMF-E7001`（退出 2），成功时输出 `OMF-W1006`；不复用 `E3004`；
- `E3005` 只表示「没有任何可信校验来源」；checksums 网络失败 `E4001`、重定向超限 `E4003`、文件超限 `E4004`、摘要不符 `E4002`；
- checksums/下载阶段的 context 取消统一 `E9004`（退出 130），超时用 `E3007` / `E4001`（退出 1）。

---

## 7. E4xxx：Download / Checksum

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E4001` | `DOWNLOAD_FAILED` | 下载失败 | 1 |
| `OMF-E4002` | `CHECKSUM_MISMATCH` | asset SHA-256 不匹配 | 1 |
| `OMF-E4003` | `REDIRECT_LIMIT` | 重定向次数超限 | 1 |
| `OMF-E4004` | `DOWNLOAD_SIZE_LIMIT` | 下载大小超限 | 1 |
| `OMF-E4005` | `TEMP_WRITE_FAILED` | 临时文件写入失败 | 1 |
| `OMF-E4006` | `DOWNLOAD_CANCELED` | 下载被用户取消 | 130 |

---

## 8. E5xxx：Install / Archive / Current

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E5001` | `ARCHIVE_UNSUPPORTED` | 不支持的压缩格式 | 1 |
| `OMF-E5002` | `ARCHIVE_UNSAFE_PATH` | zip slip / 绝对路径 / symlink 逃逸 | 1 |
| `OMF-E5003` | `INSTALL_DIR_CONFLICT` | 目标安装目录冲突 | 1 |
| `OMF-E5004` | `CURRENT_SWITCH_FAILED` | current 指针切换失败 | 1 |
| `OMF-E5005` | `ROLLBACK_FAILED` | 回滚失败 | 1 |
| `OMF-E5006` | `EXECUTABLE_BUSY` | Windows 正在运行的 exe 无法替换（**保留编号，当前无使用点**：`RenamePath` 改为退避重试） | 1 |
| `OMF-E5007` | `QUARANTINE_FAILED` | 旧版本隔离/清理失败 | 1 |
| `OMF-E5008` | `INSTALL_VALIDATION_FAILED` | 安装后三件套校验失败 | 1 |
| `OMF-E5009` | `IO_FAILED` | 文件/目录读写或创建失败（解压、复制、写 install.json、建目录、备份 shell rc） | 1 |

---

## 9. E6xxx：Environment / PATH / Journal

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E6001` | `ENV_READ_FAILED` | 读取环境变量/注册表失败 | 1 |
| `OMF-E6002` | `ENV_WRITE_FAILED` | 写入环境变量/注册表失败 | 1 |
| `OMF-E6003` | `SYSTEM_SCOPE_DENIED` | 系统级作用域未授权：未提权且未传 `--authorize-system` | 2 |
| `OMF-E6004` | `SHELL_RC_DENIED` | shell rc 不在允许范围（**保留编号**：备份/读写这类 IO 失败改用 `E5009`） | 2 |
| `OMF-E6005` | `JOURNAL_FAILED` | journal 写入/读取/提交失败 | 1 |
| `OMF-E6006` | `DOCTOR_UNHEALTHY` | `doctor --strict` 检测到不健康状态 | 1 |
| `OMF-E6007` | `NON_MANAGED_FILE` | 试图删除/覆盖非本项目管理的文件 | 1 |
| `OMF-E6008` | `WM_SETTINGCHANGE_FAILED` | Windows 环境变更广播失败 | 1 |
| `OMF-E6009` | `TRANSACTION_LOCKED` | data root 被另一事务锁定（install/env/uninstall 并发） | 1 |

说明：

- `journal rollback` / `env rollback` 的 journal id 强制白名单（拒绝 `..` 与路径分隔符）；id 非法、记录不存在或读取/解析失败统一 `OMF-E6005`（退出 1），dry-run 预览同样先校验；
- 记录已回滚（`rolled-back`）、action 不在支持列表，或 `env rollback` 指向非 env 记录，属于用法错误 `OMF-E7001`（退出 2），不复用 `E6005`。
- 系统级授权模型：进程已提权（Linux root / Windows Administrator）直接放行；未提权时 dry-run 需 `--authorize-system` 才预览，而 **`--apply` 必须真正提权**（`--authorize-system` 只记录同意、不能替代提权），否则 `OMF-E6003`。工具绝不自动 `sudo`/提权；非管理员被拒绝并告知用 sudo/管理员运行。
- `doctor` 默认退出 0；`--strict` 在 `healthy=false` 时返回 `OMF-E6006`（退出 1），报告置于 `details.report`。
- `OMF-E6009` 为 data root 事务锁竞争（install/env/uninstall/rollback 的 `--apply`）；锁超过 30 秒视为陈旧自动回收，活跃锁等待超过 5 秒返回该码。

---

## 10. E7xxx：CLI / Config / JSON

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E7001` | `CLI_USAGE` | 子命令/参数组合非法 | 2 |
| `OMF-E7002` | `TOML_PARSE_FAILED` | TOML 1.1 语法错误 | 2 |
| `OMF-E7003` | `SCHEMA_MISMATCH` | config/CLI JSON schema 不匹配 | 2 |
| `OMF-E7004` | `UNKNOWN_KEY` | strict mode 拒绝未知字段 | 2 |
| `OMF-E7005` | `INVALID_VALUE` | 类型/范围/枚举非法 | 2 |
| `OMF-E7006` | `JSON_OUTPUT_FAILED` | stdout JSON 序列化或写入失败；失败后不再向 stdout 追加第二个对象，只写 stderr | 1 |
| `OMF-E7007` | `CONFIG_PATH_INVALID` | config / release / providers 文件路径不存在或不可读；`ONE_MORE_FFMPEG_HOME` 非绝对路径 | 2 |

---

## 11. E9xxx：Internal / Cancel

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E9001` | `INTERNAL_INVARIANT` | 内部不变量失败 | 1 |
| `OMF-E9002` | `INTERNAL_PANIC` | panic 被捕获 | 1 |
| `OMF-E9003` | `INTERNAL_UNEXPECTED` | 未分类内部错误 | 1 |
| `OMF-E9004` | `CONTEXT_CANCELED` | context 取消 | 130 |

---

## 12. 警告码

| 编号 | 名称 | 说明 |
|---|---|---|
| `OMF-W1001` | `FFPLAY_MISSING` | ffplay 缺失或不可运行，但不影响 ffmpeg/ffprobe |
| `OMF-W1002` | `CAPABILITY_PARSE_WARNING` | 部分输出行无法解析 |
| `OMF-W1003` | `CACHE_STALE` | 缓存过期、损坏或不可读，已重新探测 |
| `OMF-W1004` | `NETWORK_DRIVE_SKIPPED` | 网络盘/可移动盘默认跳过 |
| `OMF-W1005` | `ENV_NOT_EFFECTIVE_CURRENT_PROCESS` | 当前进程无法立即看到新 PATH |
| `OMF-W1006` | `NONFREE_BINARY` | 使用 nonfree 资产，需合规提示 |
| `OMF-W1007` | `CACHE_WRITE_FAILED` | 能力缓存写入失败，本次结果仍返回 |

警告码不改变退出码；写入 `warnings[]`。

---

## 13. 变更规则

- 编号一旦发布不得复用；
- 不得改变已有编号语义；
- 新增错误码必须更新本文档和 `CLI_SCHEMA.md` 示例；
- 删除/重命名属于破坏性变更，必须提升 schema 主版本；
- 代码中不允许手写未登记的 `OMF-E` / `OMF-W` 字符串；
- 错误消息可改进，编号和语义不能静默变化。
