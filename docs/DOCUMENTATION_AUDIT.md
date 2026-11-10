# OneMoreFFmpeg 文档审查与评估

> 日期：2026-11-10
> 范围：`README.md`、`docs/*.md`、`go.mod`
> 方式：逐文档阅读 + 交叉引用检查 + 外部事实核对 + 本地命令验证
> 状态：**编码前最终审查；P0/P1 已修复，P2 剩余非阻塞项列出**

---

## 0. 第二轮更新（2026-11-10）

依据本报告，已完成第二轮文档修订：

| 审计项 | 第二轮状态 |
|---|---|
| P1-8 错误码注册表 | **已补**：新增 [`ERROR_CODES.md`](ERROR_CODES.md) |
| P1-9 LICENSE | **已解决**：用户选择 GPL-3.0-only；根目录 `LICENSE` 已添加，[`LICENSE_DECISION.md`](LICENSE_DECISION.md) 已更新 |
| P1-10 CLI JSON schema | **已补**：新增 [`CLI_SCHEMA.md`](CLI_SCHEMA.md) 与 [`schemas/`](schemas/) 示例 |
| P1-11 Windows PATH / 环境变量细节 | **已补**：新增 [`PLATFORM_ENVIRONMENT.md`](PLATFORM_ENVIRONMENT.md) |
| P1-12 provider resolver 细节 | **已补**：新增 [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md) |
| P2-4 未来 API 文档误读 | **已缓解**：`API_AND_PLUGIN.md` 明确未来内容不进入 Phase 1 |
| P2-5 channel 支持矩阵 | **已补**：`PROVIDERS.md` 与 `PROVIDER_RESOLVER.md` |
| P2-6 Phase 0 历史动作 | **已改**：路线图改为“仓库已建立” |
| P2-7 计划命令标注 | **已补**：README 明确当前没有可运行 CLI，命令均为计划 |
| P2-8 go.sum / 依赖锁定 | **已补**：README 与 Phase 1 明确 Phase 1 生成并提交 |

P0/P1 文档级阻塞项已关闭。P2 剩余项不阻塞 Phase 1。

技术文档层面已具备进入 Phase 1 的条件；代码、CI、测试、真实 provider 下载仍未实现。

第二轮验证结果：

- Markdown 相对链接：OK；
- Linter 诊断：0；
- `git diff --check`：通过；
- Markdown/JSON 尾随空格：OK；
- 13 个 CLI JSON 示例：全部解析通过；
- 排除本报告中的“原文引用”后，当前文档无旧表述残留；
- 工作区仍有未提交改动。

### 0.1 编码前 P0/P1 修复（2026-11-10）

第二轮独立审查发现 5 项 P0，已全部修复：

| 编号 | 问题 | 修复 |
|---|---|---|
| P0-1 | CAPABILITIES / DETECTION 示例放在顶层，没有统一信封 | 改为 `onemoreffmpeg.result/v1` + `data`，与 `CLI_SCHEMA.md` 一致 |
| P0-2 | `data.schema` 或顶层 `schema` 二义 | 改为“所有命令的 `data` 必须含 `schema`”；version 示例已补 |
| P0-3 | `env status` 字段集冲突 | 统一为 `current_type` / `current_target` 等平台字段 |
| P0-4 | README / docs README 仍写“许可证待决” | 改为“GPL-3.0-only” |
| P0-5 | resolve / install 缺 `repository` / `license_hint` / `redistributable` | schema 与示例均已补齐 |

P1 修复：

- 二级命令 `command` 统一为点号：`provider.list` / `provider.add` / `env.status` / `env.set`；
- `PROJECT_PLAN` 区分发现实现顺序与发现优先级；
- Phase 1 DoD 不再依赖 Phase 7 CI，改为本地 Go 1.27.3 构建/测试；
- `THIRD_PARTY_NOTICES` 统一为 Phase 1 产出；
- 修正 GPLv3 与 AGPLv3 网络条款混淆；
- 移除无构建决策支撑的 `cpu_baseline` 字段；
- C1 stable line 不再硬编码 `n9.0`，改为 resolver 解析结果；
- 补全错误码退出码映射；
- 依赖表补 `golang.org/x/time`；
- 补 `install --emit-trio-lock` 说明；
- 补 provider-add / env-set 示例与 CLI_SCHEMA 示例链接；
- 统一 `current_type` / `current_pointer` 为 `directory_junction` 枚举；
- rollback 失败错误码改为 `OMF-E5005`；
- release / install 的 `repository` / `license_hint` / `redistributable` 同步到 `PROVIDER_RESOLVER.md` 与 `PROVIDERS.md`。

P2 剩余：未来清单在 README / ROADMAP / API_AND_PLUGIN 三处重复维护；非阻塞，Phase 1 可收敛到 `docs/README.md` 单一权威。

---

## 1. 总体结论

文档整体质量较好：

- 范围收敛清晰；
- CLI-first、无包管理器、无 API/插件当前范围已经统一；
- provider、能力命令、发现、环境变量、未来 API 各自有独立文档；
- 依赖库有版本/时间/拒绝理由；
- 当前没有代码，文档没有假装代码已完成。

但**还不能称为“实现级冻结文档”**。

主要问题不是方向，而是：

```text
阶段边界有 1 处矛盾
依赖状态有 1 处误写
Windows current 指针过于简化
环境变量 CLI 参数不完整
错误码只有“稳定”说法，没有注册表
CLI JSON schema 尚未冻结
```

评级：

| 维度 | 评价 |
|---|---|
| 架构方向 | 良好 |
| 文档结构 | 良好 |
| 事实准确性 | 中上；已修正若干问题 |
| 阶段一致性 | 中；M0/Phase 4 已修正 |
| 实现可执行性 | 中；CLI schema、错误码、Windows 细节不足 |
| 对外可集成性 | 中；CLI-first 明确，但 JSON 契约不完整 |
| 可交付状态 | 计划文档可用；不能直接据此写完整实现 |

结论：

> **文档方向正确，可以作为 Phase 1 起点；第一轮指出的 LICENSE、错误码、CLI JSON schema、Windows current/PATH 细节已在第二轮补齐；剩余工作主要是实现、测试与第三方许可证清单。**

---

## 2. 已核实的外部事实

### 2.1 Go 版本

通过 GitHub API 核对 `golang/go` 标签：

```text
go1.27.0
go1.27.1
go1.27.2
go1.27.3
go1.27rc1
go1.27rc2
go1.27rc3
```

结论：

- `go1.27.3` 是当前能找到的最新 1.27 稳定 tag；
- 文档与 `go.mod` 锁定 `go1.27.3`；
- 未发现 `go1.27.4`。

限制：

- 本地没有 Go 命令；
- 未实际执行 `go build` / `go test`；
- `go.mod` 语法未通过真实 Go 工具链验证。

### 2.2 TOML 1.1

通过官方 GitHub README 核对 `github.com/pelletier/go-toml`：

```text
This library supports TOML v1.1.0.
Import: github.com/pelletier/go-toml/v2
License: MIT
```

结论：

- `GO_LIBRARY_EVALUATION.md` 中“支持 TOML 1.1.0”有官方依据；
- 采用 `pelletier/go-toml/v2` 合理；
- `BurntSushi/toml` 的拒绝理由中“TOML 1.1 支持不如明确”应理解为工程判断，不是对上游的绝对否定。

### 2.3 FFmpeg `-L` 与 `-license`

本地 FFmpeg 实测：

```text
ffmpeg -hide_banner -L
ffmpeg -hide_banner -license
```

本机输出均只显示 nonfree license 提示，两者没有“短 license / 完整 license”的差异。

结论：

- 原 `CAPABILITIES.md` 的“短 license / 完整 license”描述不准确；
- 已改为两个命令都输出 `license.raw`，说明为 `show license`；
- 更完整的 license 文本可能因 FFmpeg 构建和参数不同而变化，不能硬编码“长短”。

### 2.4 Provider 仓库

通过 GitHub API 核对：

```text
FFmpeg-Builds/FFmpeg-Builds         可访问
BnqDzj/FFmpeg-Builds-nonfree        可访问
```

结论：

- 两个内置 provider 仓库路径有效；
- 文档中“BtbN 当前不提供 nonfree 资产、BnqDzj 提供 nonfree”有历史依据；
- 已在 `PROVIDERS.md` 增加“截至 2026-11-10”和“以 resolver 查询为准”，避免把动态事实写成永久事实。

### 2.5 依赖版本

通过 Go module proxy `@latest` 核对主要依赖，均记录在 `GO_LIBRARY_EVALUATION.md`。

重点结论：

- `urfave/cli/v3` v3.14.0，2026-10-02；
- `pelletier/go-toml/v2` v2.4.3，2026-07-05；
- `klauspost/compress` v1.20.4，2026-11-09；
- `ulikunitz/xz` v0.5.17，2026-09-19；
- `stretchr/testify` v1.12.1，2026-08-17；
- `spf13/cobra` v1.10.2，2025-12-03，按 6 个月标准拒绝；
- `hashicorp/go-plugin` v1.8.0，2026-04-29，差约 11 天，当前不采用。

---

## 3. 已修正的问题

### P1-1：M0 包含 provider resolve，但 Phase 4 才实现 provider

原表述：

```text
M0 = 发现 + 校验 + 能力检查 + provider resolve
```

但 `Phase 4` 才是 `Provider / Resolve / Download`。这是阶段顺序矛盾。

已修正：

- `PROJECT_PLAN.md` M0 删除 provider resolve；
- `ROADMAP.md` M0 删除 provider resolve；
- 明确“provider resolve / download 属于 Phase 4”。

### P1-2：“当前采用”与 `go.mod` 没有依赖不一致

原 `GO_LIBRARY_EVALUATION.md` 标题是“当前采用”，但 `go.mod` 只有：

```go
module github.com/FroyoCatqwq/OneMoreFFmpeg

go 1.27.3
```

没有 `require`，没有 `go.sum`。

已修正：

- 改为“建议采用（Phase 1 写入 go.mod）”；
- 结论改为“建议采用”；
- 保留“当前仓库只提交 Go 版本，不提前写入依赖”的说明。

### P1-3：`TECHNICAL_NOTES.md` 写入本机环境状态

原句：

```text
本机当前没有安装 Go 命令；本阶段只提交文档和 go.mod，不编译。
```

这是会话环境事实，不是项目长期文档事实。

已修正：

```text
本阶段只提交文档和 go.mod，不提交 Go 代码；编译验证在 Phase 1 进行。
```

### P1-4：Windows `current` 只写 symlink

原 `ENVIRONMENT.md` 和 `ARCHITECTURE.md` 使用：

```text
current -> apps/<...>
```

Windows symlink 需要开发者模式或提权，普通用户不一定可用。

已修正：

```text
current（symlink / Windows junction / directory pointer） -> apps/<...>
```

仍需在实现文档中明确：

- Windows 优先 junction 还是 directory copy；
- junction 创建失败时如何回退；
- 卸载/切换时如何识别本项目创建的指针；
- 不删除用户自建同名链接。

### P1-5：环境变量文档提到 CLI 参数，但没有命令面

原 `ENVIRONMENT.md` 提到：

```text
--overwrite
--system
--prepend
--shell-rc
--lae-compat
```

但命令列表没有这些参数。

已修正：

- 在 `env set` 命令说明下新增参数表；
- 明确 `--dry-run` / `--yes` / `--overwrite` / `--prepend` / `--system` / `--shell-rc` / `--lae-compat`。

仍需：

- 为 `--system` 定义提权失败、回滚、审计日志；
- 为 `--shell-rc` 定义只允许的文件、备份、diff、失败恢复；
- 为 `--prepend` 定义重复项去重规则。

### P1-6：`-L` / `-license` 描述不准确

原 `CAPABILITIES.md`：

```text
-L        短 license
-license  完整 license
```

本地实测两者输出相同 license 提示。

已修正：

```text
-L         show license
-license   show license
```

### P1-7：provider 动态事实没有时间边界

原 `PROVIDERS.md` 把“BtbN 当前不提供 nonfree”写成无时间条件的事实。

已修正：

```text
截至 2026-11-10 ... 实际以 resolver 查询为准。
```

### P1-8：错误码声称“稳定”，但没有注册表（已解决）

原 `ARCHITECTURE.md` 写：

```text
稳定字符串错误码，例如 OMF-E1001
```

但仓库没有任何错误码列表。

已修正：

- 新增 [`ERROR_CODES.md`](ERROR_CODES.md)；
- 定义 E1xxx–E9xxx 与 W1xxx；
- 定义退出码 0/1/2/130；
- 定义错误对象 `onemoreffmpeg.error/v1`；
- 增加编号不可复用、变更需同步 CLI schema 的规则。

---

## 4. 第一轮 P1 记录（第二轮已处理）

> 本节保留第一轮问题记录；解决状态以第 0 节为准。

### P1-9：没有 LICENSE（已解决）

已选择 **GPL-3.0-only**，根目录 `LICENSE` 已添加，`README.md` 与 [`LICENSE_DECISION.md`](LICENSE_DECISION.md) 已更新。

### P1-10：CLI JSON schema 未冻结

当前只有：

- `detect` 有一个示例；
- `capabilities` 有一个顶层示例；
- `artifact` / `install` / `journal` 有片段；
- `provider resolve` / `install` / `env status` / `doctor` 没有完整 schema。

外部项目只通过 CLI 调用，因此 CLI JSON 就是 API。

建议在 Phase 1 前补：

```text
detect.json
validate.json
capabilities.json
provider-list.json
provider-resolve.json
artifact.json
install.json
env-status.json
doctor.json
error.json
```

每个 schema 明确：

- 必填字段；
- 可选字段；
- 版本字段；
- 错误对象；
- 是否允许未知字段；
- 示例。

### P1-11：Windows PATH / 环境变量写入细节不足

需要明确：

- user PATH 与 machine PATH 的读取/写入 API；
- `REG_SZ` / `REG_EXPAND_SZ` 保留；
- 写入失败时如何恢复；
- 广播 `WM_SETTINGCHANGE` 失败是否回滚；
- `--system` 的 UAC 行为；
- 不允许静默提权；
- 路径去重、大小写、尾随反斜杠；
- 重启/重新登录提示。

### P1-12：provider resolver 细节不足

需要明确：

- GitHub API 匿名/Token 限流；
- 重定向、超时、重试；
- 资产匹配规则；
- `checksums.sha256` 格式；
- release tag 与 channel 的映射；
- 找不到稳定版时返回什么错误；
- 自定义 URL 的 checksum 强制规则；
- 非 GitHub 自定义 URL 的语义。

---

## 5. P2 级问题

### P2-1：能力命令“13 类”表述可更精确（已修正）

文档原写“13 项能力命令”，但其中包含：

- `-L`；
- `-license`；
- `-h <topic>`；
- `-version`；
- 9 个列表命令。

已统一为“13 个 FFmpeg 参数 / 能力入口”，避免“能力类别”歧义。

### P2-2：`DETECTION_AND_VALIDATION.md` 的 common dirs 未在 README 摘要列出（已修正）

README 原说来源是 PATH / 环境变量 / 指定目录 / 全盘扫描，但检测文档还包括 `common_dir`。

已统一为：

```text
PATH / env / 指定目录 / 常见目录 / 全盘扫描
```

### P2-3：`GO_LIBRARY_EVALUATION.md` 的 `klauspost/compress` 用途可更精确（已修正）

`klauspost/compress` 主要用于 gzip/zstd/deflate 等；zip 容器可直接用标准库 `archive/zip`。

已改成：

```text
archive/zip + klauspost/compress（deflate/gzip/zstd）+ ulikunitz/xz
```

避免让人以为所有 zip 都必须用第三方库。

### P2-4：`API_AND_PLUGIN.md` 对未来 API 描述过多

当前已经明确不实现，但文件仍有 Go interface、JSON-RPC 方法、manifest、C ABI 细节。

这不是错误，但会增加“当前是否已实现”的误读风险。

建议：

- 标题已标 DEFERRED；
- 在文件顶部加一句“以下内容不进入 Phase 1”；
- 或在未来重新开放时再展开。

### P2-5：`PROVIDERS.md` 的 `stable` 对 BnqDzj 的支持边界可更明确

文档写了“若上游没有稳定 release line，则返回 unsupported”，但没有在 channel 表中直接标出。

建议在 channel 表加一列：

```text
btbn: stable/daily
bnqdzj-nonfree: daily；stable 视上游
custom: 由 manifest 声明
```

### P2-6：`ROADMAP.md` 的 Phase 0 “新建 GitHub repo”是历史动作

这不是错误，但路线图应面向未来，建议改为“仓库已建立”或移除。

### P2-7：README 缺少“当前没有代码”的显式说明

README 状态已写“尚未开始实现”，但仍列出未来 CLI 命令。

建议在 CLI 示例前加：

```text
以下命令为计划命令，尚未实现
```

### P2-8：`go.mod` 没有依赖，也没有 `go.sum`

这是当前正确状态，但未来 Phase 1 要明确：

- 依赖只在 Phase 1 写入；
- `go.sum` 必须提交；
- 不允许无锁 `go get` 后直接发布。

---

## 6. 文档优点

应保留：

1. 明确 CLI-first，不再假装有 Go SDK/插件；
2. 明确当前不做包管理器，避免第一阶段复杂化；
3. provider 文档把 stable/daily 与 release pin 分开；
4. 能力命令有独立文档；
5. 发现/校验与安装/环境变量分开；
6. 依赖库有版本时间、最低新鲜度标准和拒绝理由；
7. 明确 nonfree 不进 public release；
8. 明确 `current` 指针与 journal/rollback 方向；
9. 未来 API/插件单独隔离，不污染当前范围；
10. 文档没有声称代码已完成。

---

## 7. 建议的改进顺序

### 进入 Phase 1 前（第二轮已完成）

1. [已完成] 决定许可证为 GPL-3.0-only，并添加 `LICENSE`；
2. [已完成] 写 `docs/ERROR_CODES.md`；
3. [已完成] 写 `docs/CLI_SCHEMA.md` 与 `docs/schemas/*.json` 示例；
4. [已完成] 明确 Windows `current` 用 junction / symlink / pointer_file；
5. [已完成] 明确 `--system` / `--shell-rc` 的安全边界；
6. [已完成] 统一 README / 检测文档的“来源”列表。

Phase 1 产出：`THIRD_PARTY_NOTICES` 与依赖许可证清单。

### Phase 1 中

1. 锁定依赖版本并生成 `go.sum`；
2. `go test` / `go vet` / 交叉编译；
3. 实现 CLI skeleton；
4. 冻结错误码；
5. 产出 detect/validate/capabilities 的 JSON 示例；
6. 加入 fake ffmpeg 测试。

### Phase 2 前

1. 确认 provider resolver 细节；
2. 确认 GitHub API 限流策略；
3. 确认 `checksums.sha256` 解析；
4. 确认全盘扫描的时间/数量上限；
5. 确认不误删非本项目文件。

---

## 8. 最终评价

> **文档已达到 Phase 1 可开工水平；P0/P1 文档阻塞项已关闭。**

它不是“所有功能实现细节都已冻结”的终态，因为 provider/download/install 仍需要 Phase 4/5 的真实实现验证；但 CLI JSON、错误码、平台环境、provider resolver 与许可证都已足够支撑 Phase 1。

第二轮之后，剩余工作是实现与交付，不是文档契约缺口。Phase 1 已完成其中的骨架部分：

```text
[已完成] Go CLI 骨架
[已完成] go.sum 与依赖锁定
[已完成] THIRD_PARTY_NOTICES 初稿
[已完成] 本地单元测试 / vet / 双平台交叉编译
[待办] Phase 2–5 功能实现
[待办] GitHub Actions / 发布 CI
[待办] 真实 BtbN / BnqDzj resolve/download 验证
```

这些完成后，项目才达到完整交付级；文档层面已不再阻塞 Phase 1。
