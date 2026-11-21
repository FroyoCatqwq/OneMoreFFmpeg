# 重构进度记录

> 分支：`refactor/modular-monolith`；基线 commit：`6c2824e`。
> ADR：[`adr/0004-modular-architecture.md`](adr/0004-modular-architecture.md)；基线：[`REFACTOR_BASELINE.md`](REFACTOR_BASELINE.md)。

## Phase 1：低风险叶子包和契约安全

### PR-1 `fa7c06b` — 共享 checksum 原语与死代码清理

**改动清单**
- 新增 `internal/checksum/primitives.go`：`SumHex` / `IsLowerHex64` / `IsHex64`（`TrimDigestPrefix` 因无调用方在 PR-3 删除）
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

### PR-3 — otter 真正接入 + 全局 map 函数化 + checksum 彻底收口（按用户四项要求）

**改动清单**
1. **otter 真正使用**：`internal/cache.Store` 加上 otter 的 `ExpiryCreating` TTL（修复内存层绕过过期的缺陷）；`internal/capabilities` 新增 `memory_cache.go`，把 otter L1 真正接入 `Collect`/`CollectList`（key 含 cachePath+文件身份+fingerprint+installationID+topic/kind），磁盘仍为权威 L2，`--no-cache` 不分配缓存、不解析 data-root。此前 `cache.Store` 只有测试调用的问题已消除。
2. **全局 map 函数化**：`model.KnownDataSchemas` → `func KnownDataSchemas() map[string]struct{}`（返回副本）+ `IsKnownDataSchema` 热路径；`contracts.CommandDataSchemas` → `func CommandDataSchemas() map[string][]string`（深拷贝内部 slices）；`knownCommands` 改为从 `commandDataSchemas` 派生，**消除双真相源**；新增防御性副本测试 `registry_immutability_test.go`。
3. **checksum 彻底收口**：`download`/`provider`/`provider/custom`/`custom_resolve`/`contracts` 的 64-hex regexp 全部替换为 `checksum.IsLowerHex64`；`checksum` 包内部 `isLowerHex` 也委托给共享原语；`install.checksumOfString` 委托 `checksum.SumHex`；删除无调用方的 `TrimDigestPrefix`。
4. **死注入清理**：删除 `download.Options.Now`（与 PR-1 删除 `selfupdate.Options.Now` 一致，审计确认无使用）。

**依赖图变化（相关边）**

| 边 | As-Is | To-Be |
|---|---|---|
| `capabilities → cache`（otter Store） | 无 | 新增（L1 真正接入） |
| `contracts → checksum` | 无 | 新增（hex 校验收敛） |
| `provider → checksum` | 无 | 新增（custom.go 校验收敛） |
| `cache → checksum` | 无 | 保持不新增（缓存键哈希与摘要验证语义不同，伪重复不抽取） |

**验证**：gofmt / vet / go test ./... / golangci-lint 0 issues；契约 freeze（Frozen/Golden/Validate/JSON Schema）全部 PASS；`go list ./...` 无循环依赖。

## Phase 1 退出条件核对

- [x] 依赖图无环（`go list ./...` 无 cycle）
- [x] JSON golden / schema freeze 不变（全部 PASS）
- [x] 重复提取均有真实调用点证据
- [x] 未扩大外部 API 面（`LinkKind` 为 platform 内部行为描述类型，供 install 边界使用；`model` 新增均为纯数据函数）
- [x] otter 有真实生产调用（capabilities L1）
- [x] 可变全局 map 全部函数化并有防御性副本测试
