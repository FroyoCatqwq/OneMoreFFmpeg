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

### PR-0a `9d112ac` — Windows 路径断言修复 + e2e 扩充（Phase 2 前置）

**改动清单**
1. **5 个 Windows 长/短名失败**：4 个测试断言改为 `os.SameFile` 文件身份比较（detect×2、platform×2）；第 5 个 `TestInstallRollbackRestoresPreviousVersion` 暴露**真实产品 bug**——`rollback.go` 把 `EvalSymlinks` 产出的 `current_target`（长名）与词法 data-root（8.3 短名）做 containment/相等比较，误报 E6007。新增 `normalizeForCompare`（最长存在前缀 + EvalSymlinks）统一所有 journal 路径形式，并用于 containment 与 `currentTarget == target` 判断。
2. **e2e 扩充（无外网）**：新增 3 个 pytest 文件——`test_cli_directory_provider_e2e.py`（11 测试：directory 模式 latest/时间戳大小写/checksum 边界/nonfree W1006/下载全链）、`test_cli_network_errors_e2e.py`（5 个错误码映射 + stderr 无 JSON）、`test_cli_concurrency_e2e.py`（并行只读 + 缓存并发）；`test_update_e2e.py` 参数化三种互斥组合。
3. **运行方式统一**：mise.toml / CI workflow / remote_linux.py / E2E_TESTING.md 的硬编码四文件改为 `test/e2e/` 目录发现（pytest.ini `testpaths` 已存在），新增 e2e 文件自动纳入；更新 test/README.md 文件表。

**验证**：Windows VM 上 5 个失败测试全绿；本机 e2e 67 passed/4 skipped；Linux VM 69 passed/2 skipped；gofmt/vet/lint(0)/go test 全绿。

## Phase 2：网络策略边界

### PR-3a/PR-4 `pending` — netpolicy + httpfetch + 首个调用点迁移

**前置产物**：[`NETWORK_POLICY_MATRIX.md`](NETWORK_POLICY_MATRIX.md) — 逐调用点策略矩阵（初始 URL / redirect / sticky HTTPS / 回调顺序 / 上限 / 错误码 / client 来源），作为迁移验收真相源。

**改动清单**
1. 新建 `internal/netpolicy`：封闭 `Policy`（未导出 `policyKind` 枚举，零值 fail closed）+ `StrictPolicy()`/`CustomPolicy()`；`Validate` 两族独立；`CheckRedirect` 实现"nil 检查 → `len(via)-1` 跳数 → 前置校验（Strict 先快照 sawHTTPS）→ original → 若 nil 则重查最终 URL + 整链"，`original` 非 nil 原样返回（含 `ErrUseLastResponse`）；非法 `maxRedirects` 返回 `(nil, typed error)` 不 clamp。
2. 新建 `internal/httpfetch`：`FetchLimited(ctx, client, rawURL, maxBytes, maxRedirects, policy)` 显式参数；`MaxBytesCeiling = 8 MiB`；clone client 只覆盖 CheckRedirect（串联调用方原回调，项目策略先执行）；初始 URL 由 `policy.Validate` 校验；上限/非 2xx/超限 typed error。
3. `provider/fetch.go` 迁移到 `netpolicy.CustomPolicy()` 做 URL 与 redirect 校验，**保持历史错误码**（redirect 超限 `E3007`、URL `E3008`、大小 `E4004`）与 sticky 初始 scheme 行为；`checksum.Fetch`/`latest.Read` 不迁（策略由调用方注入 client 决定，迁移会改变行为）；`download`/`selfupdate`/`github` 专属 client 不迁（大文件流 / 认证 / rate 语义不同）。

**验证**：netpolicy 覆盖两族 URL、二次校验、整链快照（original 篡改 via 无效）、off-by-one 0/1/2/Ceiling/Ceiling+1、nil 边界；httpfetch 覆盖参数契约、client 不修改、重定向串联、大小/非 2xx；`provider` 行为等价测试通过。

## Phase 1 退出条件核对

- [x] 依赖图无环（`go list ./...` 无 cycle）
- [x] JSON golden / schema freeze 不变（全部 PASS）
- [x] 重复提取均有真实调用点证据
- [x] 未扩大外部 API 面（`LinkKind` 为 platform 内部行为描述类型，供 install 边界使用；`model` 新增均为纯数据函数）
- [x] otter 有真实生产调用（capabilities L1）
- [x] 可变全局 map 全部函数化并有防御性副本测试
