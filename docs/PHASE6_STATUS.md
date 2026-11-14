# Phase 6 实施状态：CLI 外部调用契约

> 日期：2026-11-14
> 范围：CLI 子命令 / `--json` schema / 退出码 / stdout-stderr 分离 / 文件产物稳定化；外部调用示例；契约冻结测试。
> 状态：**Phase 6 完成**。明确不提供 Go API / JSON-RPC / C ABI。

---

## 1. 完成内容

| 能力 | 位置 | 说明 |
|---|---|---|
| 命令集冻结 | `internal/contracts` | 18 个 result.command（`knownCommands`）；`ValidateCommand` 拒绝集合外命令 |
| schema 冻结 | `internal/model` / `internal/contracts` | 17 个 `data.schema` 全为 `/v1`；`ValidateData` 拒绝未注册 schema（含主版本升级） |
| 字段级校验 | `internal/contracts` | `ValidateDataFields` 对每个 schema 做运行时字段校验，stdout 输出前必过 |
| 退出码契约 | `internal/errors` | `exitFor` 唯一裁决：0 成功 / 2 用法 / 1 运行期 / 130 取消；警告码 0 |
| stdout/stderr 分离 | `internal/cli` | stdout 仅 JSON 信封（result/v1 或 error/v1）；诊断/日志/首次提示走 stderr |
| 文件产物 | `internal/install` / `download` | install.json、ffmpeg-trio.lock、artifact.json、SHA256SUMS（release）形状稳定 |
| 契约冻结测试 | `internal/contracts/freeze_test.go` | 快照命令集 / schema 集 / 退出码；主版本升级与未知命令被拒绝 |
| 外部调用示例 | `examples/external-caller/` | Python + PowerShell，仅消费 stdout JSON，不 import Go / 不链接 Go runtime |
| 错误码↔文档双向校验 | `internal/contracts` | 注册表与 `ERROR_CODES.md` 互为覆盖，防止漏登记/漏文档 |

## 2. 退出条件核对

- [x] 外部项目不需要 import OneMoreFFmpeg（示例仅用标准库 subprocess + json）。
- [x] 外部项目不需要链接 Go runtime（消费 published 二进制）。
- [x] 所有机器结果在 stdout JSON 或明确文件中（install.json / artifact.json / lock）。
- [x] CLI schema 主版本变化即拒绝（`ValidateData` + `freeze_test.go`）。

## 3. Phase 6 前置工作（本轮一并完成）

- **P2-5 解压安全边界**（`internal/archive`）：拒绝 NTFS ADS/任意冒号、Windows 保留设备名（NUL/COM1…）、尾随点/空格别名、`.`/空路径分量；新增压缩比上限（1000:1，blunt zip/tar bomb）；zip 解压前磁盘空间预检（`platform.AvailableBytes`，best-effort）；补单元 + zip/tar.xz 负向测试。TOCTOU 由 `O_CREATE|O_EXCL` + 空目标目录 + 拒绝 symlink 条目缓解。
- **文档 reconcile**：把 deepseek 第二轮改动同步到文档——授权模型（`--apply` 必须真正提权，`--authorize-system` 仅 dry-run 预览）、锁 owner nonce + 心跳、`rollback_guard` 路径白名单（E7001）、只读/dry-run 不创建 data root。涉及 SYSTEM_REQUIREMENTS / CLI_SCHEMA / ERROR_CODES / ENVIRONMENT / USAGE / HOWTO / man 页。

## 4. 验证

- `go build ./...` / `go vet ./...`（Windows + Linux 交叉）通过。
- `go test ./...` 全绿（含新增 archive 负向测试与 contracts 冻结测试）。
- 外部调用示例 `examples/external-caller/call_omf.py` 实跑成功：消费 version/doctor/detect/capabilities JSON，无需 Go。
- 本地 Windows + fedora45 + win2025server 全量 e2e（见提交说明）。

## 5. 仍保留 / 下一步（Phase 7）

- P1-4a：`install --apply` 在锁外下载（有意取舍，持锁会阻塞其它进程；失败清理已只动本次文件）。
- Phase 7：CI 加 gofmt / govulncheck 门禁、fuzz（URL/parser）、真实上游资产集成测试、release 自动化 + 签名/provenance。
- 建议项：GitHub releases Link 分页、覆盖率门禁、arm64 或明确仅 amd64、Linux fish/zsh rc。