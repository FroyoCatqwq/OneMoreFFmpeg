# CLI JSON 示例

> 状态：**Phase 6 契约冻结基准**（2026-11-14）。命令集、`data.schema` 集与退出码映射由
> `internal/contracts/freeze_test.go` 守护。
> 配套：[`../CLI_SCHEMA.md`](../CLI_SCHEMA.md)、[`../ERROR_CODES.md`](../ERROR_CODES.md)、
> [`../archive/PHASE6_STATUS.md`](../archive/PHASE6_STATUS.md)

本目录存放 CLI JSON 的 **v1 示例**，用于：

- 固定 stdout 字段形状；
- 作为 Go struct / validator 的 golden test 基准；
- 作为未来完整 JSON Schema Draft 的输入；
- 供 LAE / VideoDamage 等外部调用方参考。

注意：

- 这些是示例，不是完整 JSON Schema Draft 文件；
- 字段名和类型以 `CLI_SCHEMA.md` 为准；
- 错误码以 `ERROR_CODES.md` 为准；
- **全部 18 个命令均已实现**（含 `env.*` / `doctor` / `uninstall` / `release.list` / `journal.list|rollback` / `capabilities --only`），
  17 个 `data.schema` 全为 `/v1`；命令集与 schema 集已冻结，清单见 `../archive/PHASE6_STATUS.md`。
- `provider-resolve.example.json` 已按 D1 使用真实可达组合：`release_tag=latest` + `release_id` + `published_at` + `asset_sha256`；示例中的 SHA-256 仍是占位值，真实解析必须来自 checksums 或上游 API。
- release / artifact / provider-list / provider-add 示例已由 `internal/contracts` 做字段级校验（HTTPS、SHA-256、RFC 3339、枚举、nonfree 不可再分发）。

文件：

- [`examples/version.example.json`](examples/version.example.json)
- [`examples/detect.example.json`](examples/detect.example.json)
- [`examples/validate.example.json`](examples/validate.example.json)
- [`examples/capabilities.example.json`](examples/capabilities.example.json)
- [`examples/provider-list.example.json`](examples/provider-list.example.json)
- [`examples/provider-add.example.json`](examples/provider-add.example.json)
- [`examples/provider-resolve.example.json`](examples/provider-resolve.example.json)
- [`examples/artifact.example.json`](examples/artifact.example.json)
- [`examples/install.example.json`](examples/install.example.json)
- [`examples/env-status.example.json`](examples/env-status.example.json)
- [`examples/env-set.example.json`](examples/env-set.example.json)
- [`examples/doctor.example.json`](examples/doctor.example.json)
- [`examples/error.example.json`](examples/error.example.json)
- [`examples/uninstall.example.json`](examples/uninstall.example.json)
- [`examples/journal-list.example.json`](examples/journal-list.example.json)
- [`examples/journal-action.example.json`](examples/journal-action.example.json)
- [`examples/capability-list.example.json`](examples/capability-list.example.json)
- [`examples/release-catalog.example.json`](examples/release-catalog.example.json)

17 个 `data.schema` 现在**每个都有对应样例**，`internal/contracts` 的
`TestValidateExamples` 会逐个核对，缺一份即失败。
