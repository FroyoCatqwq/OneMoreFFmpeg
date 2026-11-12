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

## 5. 保留限制（已写入 PHASE4_STATUS）

1. 只尝试最新 autobuild，不回退旧 autobuild；
2. release 列表最多 100 条，无分页；
3. `provider add` 无跨进程写锁；
4. digest 降级后下载不再做第二源校验；
5. Windows Job Object 仍按 D7 推迟 Phase 5。
