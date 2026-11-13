# Phase 4 代码审查与修复记录

> 日期：2026-11-12
> 范围：`internal/provider`、`internal/provider/github`、`internal/download`、`internal/cli/{provider,resolve,download}.go`、`internal/config`、`internal/model`、契约文档
> 方法：逐文件审查 + 独立只读审计 + 真实 GitHub/真实下载复验

---

## 1. 结论

Phase 4 主流程可用，但首版存在 3 个会直接影响结果正确性的缺陷（daily 回退、资产格式/歧义、checksums 强依赖）与若干安全/契约问题。本轮全部修复并补回归测试。

```text
gofmt / go vet / go test ./...        通过
go build -tags deps / windows / linux 通过
真实 stable / daily / nonfree 解析    通过
真实 custom → resolve → download     通过（小文件）
```

---

## 2. 高优先级（已修复）

### 2.1 daily 静默回退到 stable 资产

`selectAsset` 的 daily 分支在找不到 master 资产时保留全部候选，导致 `channel=daily` 可能 pin 到 `n8.1` / `n9.0` 资产，违反“禁止静默跨 channel”。

修复：daily 必须命中 master，否则 `E3003`；`daily + --release-tag latest` 直接 `E7001`。

### 2.2 资产格式未参与匹配

契约要求 Windows 优先 `.zip`、Linux 优先 `.tar.xz`，但 `parseAssetName` 丢弃扩展名。当前上游每个 target 只发一种格式，问题未暴露；一旦同时存在两种格式会被“歧义”检查误杀。

修复：`parsedAsset.extension` + target 对应格式优先；只有同格式同 line 的重复资产才返回 `E3003`。

### 2.3 digest 可用时仍被 checksums 失败阻断

真实网络复验时 `github.com` 的 checksums 下载超时，导致已有 asset digest 的 stable/daily 解析全部失败。

修复：digest 存在时，checksums 拉取/解析失败降级为“仅 digest”，不写 checksums 字段；digest 缺失时失败仍是 `E3005`；双源不一致仍是 `E4002`。

---

## 3. 中优先级（已修复）

| 问题 | 修复 |
|---|---|
| custom `checksums_url` 从不计算 `checksums_sha256`，违反 PROVIDERS §3 成对要求 | resolveCustom 拉取 checksums、计算 sha256、校验目标行 |
| custom 静默忽略 `--release-tag` / `--release-line` | 显式返回 `E7001` |
| custom URL 可指向目录/无扩展名/带 userinfo | 必须 https + `.zip/.tar.xz/.tar.gz/.7z`，拒绝 userinfo 与不可用文件名 |
| `variant`/`linkage` 无枚举校验，非法值落到 `E3003` | `E7001`，错误信息指出字段 |
| nonfree 同意早于资产匹配，`btbn + nonfree` 先打 `W1006` 再报无资产 | 移到资产匹配之后，只在成功路径产生合规警告 |
| `download` 的 nonfree 检测只看 `license_hint`，离线 release 的 `variant=nonfree` 可绕过 | 同时检查 variant；始终补 `W1006` |
| nonfree 下载结果丢失 `W1006` | `model.Artifact` 新增可选 `warnings`，CLI/artifact.json 都携带 |
| `provider add` 允许 `--max-bytes 0` 被静默当默认值 | CLI 返回 `E7005` |
| `SaveCustomFile` 跳过校验，可写入重复/保留 id 或非法 URL | 保存前逐个 normalize + validate |
| `providers.toml` 同文件重复 id 只取第一条 | 加载时按顺序校验，重复 `E7005` |
| GitHub 资产/checksums URL 允许 loopback http | GitHub 来源强制 https（custom 也在 add 阶段拒绝 http） |
| GitHub API/下载重定向可降级到 http | CheckRedirect 校验目标 URL；非 https 直接报错 |
| `Retry-After` 无上限可挂起 24h | 超过 60s 立即返回 `E3006` |
| `contextError` 用字符串匹配 deadline | 改 `errors.Is` |
| `Registry` 只取前 20 条 release | 提高到 100 条；仍无分页回溯 |
| `Provider.Resolve` 桩方法返回误导性 `E3007` | 从接口移除，解析统一走 `Resolver` |

---

## 4. 新增/更新测试

- daily 无 master → `E3003`；`daily + latest` → `E7001`；
- win64 优先 zip、linux64 优先 tar.xz；同 line 同格式重复 → `E3003`；
- digest + checksums 拉取失败 → 成功且无 checksums 字段；无 digest → `E3005`；
- custom 短 sha / http / userinfo / 无扩展名 → `E3009` / `E3008`；
- custom `--release-tag|line` → `E7001`；custom checksums 不一致 → `E4002`；
- `provider add` 大写/空白归一化；重复 id 文件 → `E7005`；
- GitHub `Retry-After: 3600` 快速失败；
- download 重定向到 http 被拒；损坏旧文件可被替换；nonfree artifact 带 `W1006`；
- checksum deadline → `E4001`。

---

## 5. 第二轮审查（运行时契约与错误语义）

| 问题 | 修复 |
|---|---|
| `resolve` / `download` 输出只校验 `schema`，非法 release 可正常输出 | 新增 `contracts.ValidateDataFields`，`writeResult` 在写 stdout 前调用；`--release` 复用同一套校验 |
| `asset_name` 可做路径穿越、`warnings` 可含未登记码、`checksums_url/sha256` 可不成对、artifact_id/local_path 无约束 | 契约层补齐：裸文件名、已登记警告码、成对校验、`art-<16hex>`、绝对 local_path |
| `[network] timeout` 对 API 客户端无效（`HTTPClient` 预建后未重建） | 新增 `RebuildHTTPClient`，CLI 注入 timeout/max_redirects 后重建；`MaxRedirects=0` 表示禁止重定向 |
| checksums 独立 fetch 硬编码 30s/5 次重定向 | `Resolver.Timeout` / `Resolver.MaxRedirects` 由配置注入 |
| checksums 拉取/解析/大小错误全部报 `E3005` | 网络 `E4001`、重定向 `E4003`、大小 `E4004`；`E3005` 只保留给“无校验来源” |
| checksums 阶段取消报 `E3005` 且退出 1 | 取消 `E9004`（130），超时 `E3007`（1） |
| 流式下载的 deadline 被报成「用户取消」`E4006` | 统一 `contextError`：deadline `E4001`，取消 `E4006` |
| 复用已有文件时跳过 checksums 第二源校验 | checksums 校验移到资产下载前，复用时仍重新校验 |
| checksums 失败删除已验证的完整资产 | 先校验 checksums 再下载，失败时根本不落盘资产 |
| `artifact.json` 写入失败留下「有资产无记录」且可能删掉旧记录 | 提交改为备份旧文件 + rename + 失败回滚；新下载资产在 record 失败时删除，复用资产不删除 |
| `os.Rename` 前无条件 `os.Remove` 会丢失并发进程的已提交数据 | 下载资产、artifact、providers.toml 三处统一使用备份式提交 |
| nonfree provider 用 `--variant gpl` 可绕过同意与 `W1006` | 同意判定加入 provider 身份；`supportsVariant` 拒绝 nonfree-only provider 的自由 variant |
| `download --dir relative` / 非法 sha / 文件不存在错误码错配 | CLI 前置返回 `E7001` / `E7003` / `E7007` |

新增回归：运行时契约非法输入、未登记 warning、checksums 成对、artifact 格式、API timeout 重建、canceled checksums fetch、checksums 复用、artifact 写入失败清理、`max_redirects=0`、nonfree variant 拒绝、相对目录拒绝。

---

## 6. 第三轮审查（全项目交叉问题）

| 问题 | 修复 |
|---|---|
| Windows UNC / `\\?\` 路径下 artifact 落盘成功，但 `writeResult` 的绝对路径校验误报 `E7003` | 契约层接受 `\\` / `//` 前缀；下载层同时拒绝 `\` 形式的文件名，避免 Linux/Windows 判定不一致 |
| `latest` 缺 pin 在契约层返回 `E7003`、离线文件返回 `E3010` | 统一 `E3010`，测试按码断言 |
| 非法 `ONE_MORE_FFMPEG_HOME`（相对路径）被 `loadConfig` 静默吞掉，不同命令行为不一致 | 仅忽略 `E9003`（无法判定 home），`E7007` 明确上抛 |
| `capabilities` / `validate` 被 detect 的 `probe<=timeout` 规则误伤 | `parseDetectOptions` 增加 `enforceTimeoutBudget`，仅 `detect` 传 true |
| 相对 `--cache-dir` 在当前工作目录写缓存 | 要求绝对路径，否则 `E7001` |
| capabilities 缓存命中可输出多余命令键 / 缺字段 | 把示例中的 capabilities 结构校验下沉到 `validateDataFields`，运行时与示例同强度 |
| stdout 写入失败后人类文本与错误对象混写，或写出两个 JSON | 人类写失败映射 `E7006`；`writeFailure` 遇 `E7006` 只写 stderr；JSON 失败降级 stderr |
| `validate --installation-id --roots "   "` 被误判为冲突 | 用解析后的 roots 数量判断，空白视为未提供 |

补充：`context.DeadlineExceeded` 在 detect/capabilities 仍使用 `E1006`（进程/采集超时，退出 1），与 `E9004`（用户取消，130）区分；这是有意语义，已在本节记录。

---

## 7. 保留限制（已写入 PHASE4_STATUS）

1. 只尝试最新 autobuild，不回退旧 autobuild；
2. release 列表最多 100 条，无分页；
3. `provider add` 锁文件有 30 秒陈旧锁清理策略，极长并发事务仍可能竞争；常规单进程 CLI 不受影响；
4. digest 降级后下载不再做第二源校验；
5. Windows Job Object 仍按 D7 推迟 Phase 5。
