# Phase 4 实施状态

> 日期：2026-11-12
> 范围：Provider / Resolve / Download
> 决策：D1（latest + release_id/published_at/asset_sha256）、D5（`--allow-nonfree`）、D7（Job Object 推迟 Phase 5）
> 验证：Go 1.27.4 windows/amd64；真实 GitHub API（匿名）与真实资产下载

---

## 1. 完成内容

| 能力 | 位置 | 说明 |
|---|---|---|
| GitHub API 客户端 | `internal/provider/github` | `LatestRelease` / `ReleaseByTag` / `ListReleases`；3 次重试、`Retry-After`、403/429 限流、`len(via)` 重定向上限、4 MiB 响应上限、token 脱敏 |
| 资产匹配 | `internal/provider/assets.go` | 按 target/variant/linkage/line 匹配；stable 取最高 `nX.Y`，daily 取 master；`-shared` 折叠为 linkage；无匹配 `E3003` |
| pin 解析 | `internal/provider/resolve.go` | stable → `latest` + release id + published + sha；daily → 具体 `autobuild-*`；校验源优先 digest，其次 `checksums.sha256`，都没有 `E3005`；两者冲突 `E4002` |
| nonfree 同意 | `internal/provider/consent.go` | 缺失 `--allow-nonfree` → `E7001`；成功 → `W1006`；`redistributable=false` |
| 自定义 provider | `internal/provider/custom.go` | `<data-root>/providers.toml`（`onemoreffmpeg.providers/v1`）；严格校验、原子写入、重复 id 拒绝 |
| 下载 | `internal/download` | 流式 SHA-256、`*.part` 原子落盘、`max_bytes`、重定向上限、`checksums_url` 二次校验、已有文件复用、失败清理 |
| CLI | `internal/cli/{provider,resolve,download}.go` | `provider list` / `provider add` / `resolve` / `download`；`--release` 接受裸 release 或 `resolve --json` 输出 |
| 配置 | `internal/config` | `[network]` / `[download]`；loopback http 仅用于测试/自建镜像 |

---

## 2. 真实环境验证

```text
resolve --provider btbn --channel stable --target win64
  release_tag=latest  release_id=441698707
  published_at=2026-11-11T13:00:32Z
  asset=ffmpeg-n9.0-latest-win64-gpl-9.0.zip
  asset_sha256=20158509057316e2...（GitHub asset digest）

resolve --provider btbn --channel daily --target win64
  release_tag=autobuild-2026-11-11-12-39
  asset=ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip

resolve --provider bnqdzj-nonfree --channel daily --variant nonfree --target win64
  -> OMF-E7001（缺 --allow-nonfree）
  + --allow-nonfree -> autobuild-2026-11-11-19-37 / nonfree.zip / W1006

provider add --id omf-e2e --url <ffmpeg.zip> --sha256 <hash> --license-hint gpl --apply
resolve --provider omf-e2e -> release_tag=direct-3fcaed06e00b, repository=custom
download --release <resolve.json> --dir <tmp>
  -> downloads/3fcaed06e00b94dd/checksums.sha256（5296 bytes）+ artifact.json
```

下载未对 ~195 MB 的 win64 资产做真实全量拉取；该路径由 `internal/download` 的 httptest 用例覆盖（含大小/重定向/取消/摘要不符）。

---

## 3. 测试与构建

```text
gofmt -l .                 无输出
go vet ./...               通过
go test -count=1 ./...     全部通过（新增 provider/resolve/assets/custom/download/CLI 用例）
go build -tags deps ./...  通过
GOOS=windows / linux       go build 通过
```

关键回归用例：

- `TestSelectAsset*`：真实资产名表驱动（master / n8.1 / n9.0 / nonfree / shared / linux64）；
- `TestResolve*`：真实 release payload 形状、digest 与 checksums 双源、冲突、缺失、nonfree 同意、custom；
- `TestFetch*`：复用、摘要不符、checksums 不一致、大小、重定向、5xx、取消、危险文件名；
- `TestProviderAddAndList` / `TestResolveAgainstFakeGitHub` / `TestDownloadOfflineRelease`：CLI 端到端。

---

## 4. DoD 对照

| `PROJECT_PLAN.md` Phase 4 DoD | 状态 |
|---|---|
| BtbN stable / daily 都能解析 | 完成（stable=latest+pin，daily=autobuild tag） |
| BnqDzj nonfree 能解析 | 完成（需 `--allow-nonfree`） |
| 自定义 URL 必须提供 checksum | 完成（`--sha256` 必填，缺失 `E3009`） |
| HTTP 重定向、超时、重试策略明确 | 完成（统一 3 次、`Retry-After`、`len(via)` 上限、可配 timeout） |
| 下载不写目标安装目录 | 完成（只写 `<downloads>/<sha16>/`） |
| 不修改 PATH | 完成（无环境写入代码） |
| 不把 nonfree 二进制加入 Git 或 Release | 完成（下载目录由用户控制；nonfree 仅本机） |

---

## 5. 已知限制

1. `download` 不支持断点续传；失败后重新下载（已有完整文件可复用）。
2. `checksums.sha256` 只识别 `sha256sum` 格式；PGP 签名未处理。
3. custom provider 的 `published_at` 使用解析时间（上游直链没有发布时间），已在 `PROVIDERS.md` 说明。
4. stable 依赖上游 `latest` release 内的 `nX.Y` 资产；若上游改结构，返回 `E3003` 而不是猜测。
5. Windows Job Object 按 D7 推迟到 Phase 5；当前 `taskkill` + `Kill` 为尽力终止。
6. 未实现 `--select` / provider 优先级；`resolve` 每次显式指定 provider。
7. `provider add` 的全链路 `Load → AddCustom → Save` 使用 `providers.toml.lock` 跨进程锁；文件提交为“备份旧文件 + rename + 失败回滚”。锁文件超过 30 秒会被视为陈旧锁并清理，因此极长事务仍可能遇到锁竞争。
8. `download` 复用已有文件时会完整重算 SHA-256，并重新拉取 `checksums_url`（若存在）；195 MB 资产约需一次全量读取。
9. daily 只尝试最新 `autobuild-*`；若该 release 缺目标资产，不会回退到上一个 autobuild（返回 `E3003`）。
10. GitHub asset digest 存在但 `checksums.sha256` 拉取失败时，resolve 降级为仅 digest，并留空 `checksums_url` / `checksums_sha256`；下载阶段因此不再做第二源校验。
11. GitHub release 列表最多读取 100 条（一个请求），未实现分页回溯。
12. `Retry-After` 超过 60s 时直接返回 `E3006`，不阻塞等待。
13. 运行时 stdout 数据按 `onemoreffmpeg.*/v1` 做字段级契约校验；`--release` 离线输入使用同一套规则，非法对象返回 `E7003`。
14. `[network] timeout` 与 `[download] max_redirects` 已贯通到 GitHub API 与 checksums 请求；`max_redirects=0` 表示禁止重定向。
15. 运行时 stdout 契约校验同样覆盖 capabilities 缓存命中路径；非法 data root 覆盖不再被静默忽略。
