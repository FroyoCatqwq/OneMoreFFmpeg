# CLI JSON 示例

> 状态：Phase 2 契约形状基准
> 配套：[`../CLI_SCHEMA.md`](../CLI_SCHEMA.md)、[`../ERROR_CODES.md`](../ERROR_CODES.md)、[`../PHASE2_STATUS.md`](../PHASE2_STATUS.md)

本目录存放 CLI JSON 的 **v1 示例**，用于：

- 固定 stdout 字段形状；
- 作为 Go struct / validator 的 golden test 基准；
- 作为未来完整 JSON Schema Draft 的输入；
- 供 LAE / VideoDamage 等外部调用方参考。

注意：

- 这些是示例，不是完整 JSON Schema Draft 文件；
- 字段名和类型以 `CLI_SCHEMA.md` 为准；
- 错误码以 `ERROR_CODES.md` 为准；
- Phase 2 已实现 `version --json` / `detect --json` / `validate --json`；其余示例仍代表计划契约。

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
