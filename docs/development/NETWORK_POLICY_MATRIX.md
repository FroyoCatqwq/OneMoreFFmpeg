# 网络策略矩阵（Phase 2 前置，ADR-0004 §6.2）

> 本文件是迁移前的**逐调用点策略真相源**。任何 `netpolicy`/`httpfetch` 迁移都必须逐行对照本表，
> 保持"初始 URL 规则 / redirect URL 规则 / 是否 sticky HTTPS / 回调顺序 / 重定向上限 / 错误码 / client 来源"完全一致。
> 本表以当前 v1.1.2 源码为准（分支 `refactor/modular-monolith`）。

## 1. 矩阵

| 调用点 | 初始 URL 规则 | redirect URL 规则 | Sticky HTTPS（整链） | 回调顺序 | 上限 | 错误码 | client 来源 |
|---|---|---|---|---|---|---|---|
| `config.isHTTPSURL`（`config.go:177`） | 仅 https；拒绝 userinfo；host 必填 | 无（配置校验，不请求） | 不适用 | 不适用 | 不适用 | `E7005`（配置） | 无 |
| `customurl/urlpolicy.ValidateURL`（`urlpolicy.go:26`） | http/https、host 必填、拒绝 userinfo | 同左 | **是**（检查整条 via，https→http 一律拒绝） | 先 Validate，再整链检查；**无调用方回调** | `MaxRedirects`（默认 10，`<=0` 时被替换） | `E3008`/`E3007` | `NewClient` 自建或从 base clone，设 TLS≥1.2/可选 CA |
| `provider/fetch.go fetchURL`（`fetch.go:18`） | http/https、host 必填、拒绝 userinfo（**自建镜像可 http**） | http/https；**仅当初始 scheme 是 https 时**拒绝降级到 http（只看初始，不检查整链） | **否**（仅 sticky 初始 scheme） | 自建 client，无调用方回调 | `maxRedirects<=0 → 5`；`len(via) >= max` 即拒绝 | `E3008`/`E3007`/`E4004` | 自建 client（Timeout 参数） |
| `download.isAllowedURL`（`download.go:517`） | http/https、host 必填、拒绝 userinfo（自托管可 http） | 同左；https 初始拒绝 http 目标；**不检查完整 via** | 部分（仅初始 scheme） | 覆盖 `client.CheckRedirect`，无调用方回调 | `MaxRedirects`；`len(via) >= max` 拒绝 | `E4001`（redirect） | 调用方 `HTTPClient` 或自建；**clone 不修改原对象** |
| `selfupdate`（`selfupdate.go:191`） | HTTPS-only（`selfupdate.go:166` 资产 URL 必须 https） | http/https 检查 + https 初始拒绝降级 | 部分 | 自建 client 覆盖 CheckRedirect | 默认 5 | `E3007`/`E3008` | 自建 |
| `provider/github client`（`github/client.go:114`） | https-only、host 必填、拒绝 userinfo（`client.go:73`） | 同左；拒绝 https→http | 是（控制面 API） | 自建 client，保留 token/rate/retry | 默认 4（go-github 默认） | `E3007` | 自建，带认证/transport |
| `customurl/checksum.Fetch`（`checksum.go:45`） | 由调用方 `client` 决定（当前来自 `urlpolicy.NewClient`） | 由 client.CheckRedirect 决定 | 由 client 决定（整链） | 调用方 client 已配策略 | 由 client 决定 | `E3007`/`E4004` | 调用方注入 |
| `customurl/latest.Read`（`latest.go:26`） | 同上 | 同上 | 同上 | 同上 | 同上 | `E3007`/`E7003`/`E4004` | 调用方注入 |

## 2. 两族策略（行为等价，不得微调）

- **Strict 族**（控制面 / 自更新 API）：`customurl/urlpolicy`、`provider/github`、`selfupdate` 的初始 URL 与 redirect 语义。
  - 仅 https；http 仅 loopback 的配置校验场景单独保留；
  - redirect 检查**整条 via**，任何曾出现 https 后到 http 一律拒绝（含 `http→https→http`）。
- **Custom 族**（用户自托管 / 自定义源）：`provider/fetch.go`、`download`、`customurl/checksum`、`customurl/latest`。
  - 允许 http + https 任意 host（**不做 SSRF**，见 CUSTOM_URL_SPEC）；拒绝 userinfo；
  - redirect 规则**保持现状**：`fetch.go` 仅 sticky 初始 scheme；`download` 仅检查目标 URL + 初始 scheme；
  - `customurl/checksum`/`latest` 的整链检查来自**调用方注入的 client**，不属于 helper 自身职责。
- **禁止**把 `provider/fetch.go` 迁到 Strict：会拒绝用户自建 http 镜像，属行为变更（ADR 明令禁止）。

## 3. netpolicy 封装契约

- `Policy` 为封闭具体类型，字段未导出，零值 fail closed；仅 `StrictPolicy()` / `CustomPolicy()` 两个构造器。
- `Validate(raw string) error` 与 `CheckRedirect(maxRedirects int, original func) (func, error)`：
  - 非法 `maxRedirects`（<=0 或 >Ceiling）返回 `(nil, typed error)`，**不 clamp、不默认**；
  - 回调顺序：`req/URL nil 检查 → 跳数检查 → 前置 Validate（Strict 含整链快照）→ original → 若 nil 则重查最终 URL + 整链`；
  - `original` 返回非 nil（含 `http.ErrUseLastResponse`）**原样返回**，不包装；
  - `via` 为空按 0 跳处理，`len(via)-1` 语义（最大跳数，非请求数）。
- **Strict 整链状态快照**：必须在调用 `original` **之前**遍历 via 记录 `sawHTTPS`（original 可以修改 via 元素）；
  回调返回 nil 后，用该快照 + 最终 req.URL 再判降级。
- `maxRedirects` 配置来源 `[download] max_redirects`，默认由 `config` 拥有；`netpolicy` 只暴露硬上限 `MaxRedirectsCeiling=10`。

## 4. httpfetch 封装契约

- `FetchLimited(ctx, client, rawURL, maxBytes, maxRedirects, policy) ([]byte, error)`：
  - 显式接收全部数值/策略参数，**禁止隐式默认与便利包装**；
  - 前置 `policy.Validate(rawURL)`；clone client 并覆盖 `CheckRedirect`（policy.CheckRedirect），
    保留调用方 Transport/TLS/Timeout/Proxy/Jar，**绝不修改原对象**；
  - 初始 URL 与每次重定向都经过策略验证；`http.ErrUseLastResponse` 原样透传；
  - `maxBytes <= 0` / `> MaxBytesCeiling` → `E4004`（防 `+1` 溢出）；`LimitReader(max+1)` 超限失败不截断；
  - UA `OneMoreFFmpeg`；非 2xx → typed error（`E3007`）；`defer Body.Close()`；
  - 只用于小文本（checksums / latest / 受限 release 元数据），不用于归档下载。
- `MaxBytesCeiling`：由 `httpfetch` 拥有并导出，值 **8 MiB**（基于实测：checksums 4 MiB、latest 文本更小、release JSON < 1 MiB）；
  超过上限需要流式写文件 API，不得无上限提升 `[]byte` 读取。

## 5. 迁移顺序与验收

1. PR-3a：新建 `netpolicy`，补本表两族 + 二次校验 + off-by-one 边界测试；不改任何调用点。
2. PR-4：新建 `httpfetch`，补参数契约 + client clone + redirect 验证测试。
3. PR-4b：按本表迁移 `provider/fetch.go` → Custom 行为、`customurl/checksum.Fetch` → 调用方注入 Custom；
   `download`/`selfupdate`/`github` 的大文件或专属 client **不迁**，仅记录为已知策略入口。
4. 每个 PR 验收：行为等价测试（逐调用点对比旧/新错误码和 URL 接受集合）+ 契约 freeze（golden/schema 不变）。
