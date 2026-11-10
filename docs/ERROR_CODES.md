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
| 1 | 运行期错误 | E1xxx–E6xxx、E9xxx |
| 2 | 用法/配置/契约错误 | `E7xxx`、`E2005`、`E3001`、`E3004`、`E3008`、`E3009`、`E6003`、`E6004` |
| 130 | 用户取消 | `OMF-E1011`、`OMF-E4006`、`OMF-E9004` |

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
| `OMF-E1001` | `DIR_MISSING` | 显式目录不存在 | 1 |
| `OMF-E1002` | `BINARY_MISSING` | ffmpeg/ffprobe/ffplay 缺失 | 1 |
| `OMF-E1003` | `VERSION_MISMATCH` | 三件套版本核心不一致 | 1 |
| `OMF-E1004` | `INSTALLATION_UNUSABLE` | 启动失败/权限错误/损坏 | 1 |
| `OMF-E1005` | `PERMISSION_DENIED` | 无执行或读取权限 | 1 |
| `OMF-E1006` | `PROCESS_TIMEOUT` | FFmpeg 命令超时 | 1 |
| `OMF-E1007` | `PROCESS_FAILED` | FFmpeg 非零退出 | 1 |
| `OMF-E1008` | `SCAN_LIMIT_EXCEEDED` | 全盘扫描超过深度/数量/时间上限 | 1 |
| `OMF-E1009` | `SCAN_ROOT_INVALID` | 扫描根无效或不可读 | 1 |
| `OMF-E1010` | `SYMLINK_LOOP` | 符号链接循环 | 1 |
| `OMF-E1011` | `CANCELED` | 用户取消 | 130 |

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
- 只有所有能力命令都失败或缓存/输出不可用时才返回整体错误。

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
| `OMF-E3010` | `RELEASE_NOT_PINNED` | 解析结果仍是 `latest` 而非具体 tag | 1 |

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
| `OMF-E5006` | `EXECUTABLE_BUSY` | Windows 正在运行的 exe 无法替换 | 1 |
| `OMF-E5007` | `QUARANTINE_FAILED` | 旧版本隔离/清理失败 | 1 |
| `OMF-E5008` | `INSTALL_VALIDATION_FAILED` | 安装后三件套校验失败 | 1 |

---

## 9. E6xxx：Environment / PATH / Journal

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E6001` | `ENV_READ_FAILED` | 读取环境变量/注册表失败 | 1 |
| `OMF-E6002` | `ENV_WRITE_FAILED` | 写入环境变量/注册表失败 | 1 |
| `OMF-E6003` | `SYSTEM_SCOPE_DENIED` | `--system` 被拒绝或未提权 | 2 |
| `OMF-E6004` | `SHELL_RC_DENIED` | shell rc 不在允许范围或备份失败 | 2 |
| `OMF-E6005` | `JOURNAL_FAILED` | journal 写入/读取/提交失败 | 1 |
| `OMF-E6006` | `DOCTOR_UNHEALTHY` | doctor 检测到不可恢复状态 | 1 |
| `OMF-E6007` | `NON_MANAGED_FILE` | 试图删除/覆盖非本项目管理的文件 | 1 |
| `OMF-E6008` | `WM_SETTINGCHANGE_FAILED` | Windows 环境变更广播失败 | 1 |

---

## 10. E7xxx：CLI / Config / JSON

| 编号 | 名称 | 说明 | 退出 |
|---|---|---|---:|
| `OMF-E7001` | `CLI_USAGE` | 子命令/参数组合非法 | 2 |
| `OMF-E7002` | `TOML_PARSE_FAILED` | TOML 1.1 语法错误 | 2 |
| `OMF-E7003` | `SCHEMA_MISMATCH` | config/CLI JSON schema 不匹配 | 2 |
| `OMF-E7004` | `UNKNOWN_KEY` | strict mode 拒绝未知字段 | 2 |
| `OMF-E7005` | `INVALID_VALUE` | 类型/范围/枚举非法 | 2 |
| `OMF-E7006` | `JSON_OUTPUT_FAILED` | stdout JSON 序列化失败 | 1 |
| `OMF-E7007` | `CONFIG_PATH_INVALID` | config 路径不存在或不可读 | 2 |

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
| `OMF-W1001` | `FFPLAY_MISSING` | ffplay 缺失，但不影响 ffmpeg/ffprobe |
| `OMF-W1002` | `CAPABILITY_PARSE_WARNING` | 部分输出行无法解析 |
| `OMF-W1003` | `CACHE_STALE` | 缓存过期或损坏，已重新探测 |
| `OMF-W1004` | `NETWORK_DRIVE_SKIPPED` | 网络盘/可移动盘默认跳过 |
| `OMF-W1005` | `ENV_NOT_EFFECTIVE_CURRENT_PROCESS` | 当前进程无法立即看到新 PATH |
| `OMF-W1006` | `NONFREE_BINARY` | 使用 nonfree 资产，需合规提示 |

警告码不改变退出码；写入 `warnings[]`。

---

## 13. 变更规则

- 编号一旦发布不得复用；
- 不得改变已有编号语义；
- 新增错误码必须更新本文档和 `CLI_SCHEMA.md` 示例；
- 删除/重命名属于破坏性变更，必须提升 schema 主版本；
- 代码中不允许手写未登记的 `OMF-E` / `OMF-W` 字符串；
- 错误消息可改进，编号和语义不能静默变化。
