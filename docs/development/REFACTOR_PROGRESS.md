# 重构进度记录

> 分支：`refactor/modular-monolith`；基线 commit：`6c2824e`。
> ADR：[`adr/0004-modular-architecture.md`](adr/0004-modular-architecture.md)；基线：[`REFACTOR_BASELINE.md`](REFACTOR_BASELINE.md)。

## Phase 1：低风险叶子包和契约安全

### PR-1 `fa7c06b` — 共享 checksum 原语与死代码清理

**改动清单**
- 新增 `internal/checksum/primitives.go`：`SumHex` / `IsLowerHex64` / `IsHex64` / `TrimDigestPrefix`
- 替换本地重复实现：`ffexec`、`capabilities`、`detect` 的 `sha256Hex`；`official`、`customurl/checksum` 的 `isLowerHex`；`selfupdate` 的 `isHex64`
- 删除无引用的 `internal/deps` 占位包
- 删除 `selfupdate.Options.Now` 死注入

**依赖图变化（相关边）**

| 边 | As-Is | To-Be |
|---|---|---|
| `ffexec → checksum` | 无 | 新增 |
| `capabilities → checksum` | 无 | 新增 |
| `detect → checksum` | 无 | 新增 |
| `official → checksum` | 无 | 新增 |
| `customurl/checksum → checksum` | 无 | 新增 |
| `selfupdate → checksum` | 无 | 新增 |
| `internal/deps` | 存在（空包） | 删除 |

**验证**：gofmt / vet / go test ./... / golangci-lint 0 issues（Windows 本机 + Linux GOOS）

### PR-2 `4bf3b7e` — 移除反向依赖与 contracts 同包拆分

**改动清单**
- `contracts.go` 按职责同包拆分：`contracts.go`（公共 API + 注册表）、`validators.go`（私有 `validate*` 系列）；**校验行为不变**
- 新增 `internal/model/capability_data.go`：能力 kind 与 full command keys 的权威数据（零依赖）
- `contracts` 改用 `model.IsCapabilityKind` / `model.CapabilityFullCommandKeys`，**移除 `contracts → capabilities`**
- `capabilities.IsCapabilityKind` / `CapabilityKinds` 委托 model（保持调用方 API）
- `platform` 定义自己的 `LinkKind` 类型与常量；`ReplaceLink` 返回 `LinkKind`；`install` 在边界映射为 journal 字符串；**移除 `platform → model`**
- 删除无生产调用的 `platform.LinkKind()`

**依赖图变化（相关边）**

| 边 | As-Is | To-Be |
|---|---|---|
| `contracts → capabilities` | 存在 | **删除** |
| `contracts → model` | 存在 | 存在（保留） |
| `capabilities → model` | 存在 | 存在（保留） |
| `platform → model` | 存在 | **删除** |
| `platform → errors` | 存在 | 存在（允许） |
| `install → platform` | 存在 | 存在（保留） |
| `model → capability_data`（同包） | 无 | 新增文件 |

**验证**
- gofmt / vet / go test ./... / golangci-lint 0 issues
- freeze：`TestFrozen*`、`TestGolden*`、`TestValidate*`、JSON Schema 校验全部 PASS（18 命令 / 17 schema）
- 零内部依赖不变量：`errors=0`、`model=0`、`output=0`、`buildinfo=0`；`platform=1`（仅 `errors`，允许）

## 尚未处理（Phase 1 显式跳过，需在后续 Phase 评估）

- `cache.Store`（otter + 磁盘持久化）仍无生产调用。用户曾明确要求引入并使用 otter，直接删除会与该要求冲突；应在 Phase 1/2 以"真正接入 capabilities 缓存"或"独立提案删除"二选一决定。
- `model.KnownDataSchemas` / `contracts.CommandDataSchemas` 仍为导出可变 map；当前生产代码只读，测试只读，未发现真实写入风险；改为函数会扩大测试改动面，留待 Phase 4/6 与契约 freeze 一起评估。
- Windows 8.3 短名测试断言失败（5 个）仍未修，属测试假设问题，留 Phase 6 跨平台回归前处理。

## Phase 1 退出条件核对

- [x] 依赖图无环（`go list ./...` 无 cycle）
- [x] JSON golden / schema freeze 不变（全部 PASS）
- [x] 重复提取均有真实调用点证据
- [x] 未扩大外部 API 面（`LinkKind` 为 platform 内部行为描述类型，供 install 边界使用；`model` 新增均为纯数据函数）
