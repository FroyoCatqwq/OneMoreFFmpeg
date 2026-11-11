# OneMoreFFmpeg 项目计划

> 状态：Phase 0 计划文档
> 日期：2026-11-10
> 原则：从简单闭环开始，质量优先，不做大而全的第一步

---

## 1. 项目目标

OneMoreFFmpeg 要解决的是“FFmpeg 三件套的生命周期前段”：

```text
发现 → 识别 → 校验 → 能力检查 → 解析来源 → 下载 → 安装 → 配置环境 → 切换/回滚
```

当前外部项目只通过**调用本项目产物可执行文件**集成：

```text
CLI 参数 + stdout JSON / 文件产物 + stderr 日志 + 退出码
```

本项目不负责：

- 视频编码/解码算法；
- damage 算法；
- LAE 转录/归档业务逻辑；
- FFmpeg 自构建；
- GUI / TUI / CUI；
- 第三方二进制的公开再分发；
- 当前阶段不提供 Go API / 插件接口；
- 当前阶段不提供包管理器识别。

---

## 2. 最新范围调整

| 调整 | 结果 |
|---|---|
| 取消 Scoop / Homebrew / 包管理器识别 | 当前不做，纳入未来考虑 |
| 取消对外开放 Go API | 当前不做，纳入未来考虑 |
| 取消插件接口 / JSON-RPC / gRPC / C ABI | 当前不做，纳入未来考虑 |
| 其他项目调用方式 | 只调用 CLI 可执行文件 |
| 依赖库策略 | 标准库为主 + 6 个月内活跃第三方库，详见 `GO_LIBRARY_EVALUATION.md` |

原因：

> 程序功能还不完善，过早开放 API/插件会冻结不成熟接口；包管理器差异会显著增加平台复杂度。

---

## 3. 首批需求原文映射

| 需求 | 落地位置 |
|---|---|
| 管理 FFmpeg 路径 | `DETECTION_AND_VALIDATION.md` + `ENVIRONMENT.md` |
| 安装包 | `PROVIDERS.md` + `ARCHITECTURE.md` 的 downloader/installer |
| 环境变量 | `ENVIRONMENT.md` |
| 检查可支持编解码器、滤镜 | `CAPABILITIES.md` |
| 13 个 FFmpeg 能力命令 | `CAPABILITIES.md` |
| Windows + Linux | `ARCHITECTURE.md` + `TECHNICAL_NOTES.md` |
| 自定义 FFmpeg URL | `PROVIDERS.md` |
| 内置 BtbN / BnqDzj | `PROVIDERS.md` |
| 每日构建 / 稳定版 | `PROVIDERS.md` 的 channel 模型 |
| 系统已有 FFmpeg 识别 | `DETECTION_AND_VALIDATION.md` |
| 环境变量 / 指定目录 / 全盘扫描 | `DETECTION_AND_VALIDATION.md` |
| 包管理器识别 | 未来，不在当前 Phase |
| 验证可用性 | `DETECTION_AND_VALIDATION.md` |
| Go 1.27 | `README.md` + `go.mod` + `TECHNICAL_NOTES.md` |
| 非标准库评估 | `GO_LIBRARY_EVALUATION.md` |
| 对外 API / 插件 | 未来，不在当前 Phase |
| LAE / VideoDamage 调用 | 当前只调用 CLI 可执行文件 |
| 实验性、可大胆使用架构和语法糖 | `TECHNICAL_NOTES.md` |

---

## 4. 总体设计原则

1. **先只读，后写入**
   M0 只做发现、校验、能力检查，不改 PATH、不安装、不删除。

2. **CLI-first**
   当前唯一对外形态是可执行文件；不暴露 Go package 作为公共 SDK，不提供插件协议。

3. **先外部进程，后嵌入式 API**
   FFmpeg 是外部可执行文件；Go 侧只管理路径、进程和元数据。

4. **先 JSON/TOML 契约，后 Go 结构体优化**
   CLI 输出和 TOML 配置先稳定字段，内部可以用 Go 1.27 泛型/迭代器做语法糖。

5. **pin 后才算可复现**
   `latest` / `daily` 只作为解析入口，不直接成为安装记录；必须记录 release tag、asset name、SHA-256。

6. **默认不修改系统状态**
   PATH、shell profile、注册表、安装目录都必须显式参数触发，并支持 dry-run / rollback。

7. **nonfree 与公开分发隔离**
   内置来源可以列元数据；二进制不进入 public GitHub Release。

8. **实验项目不做长期兼容包袱**
   CLI 输出 schema / TOML schema / 安装目录布局显式版本化；不兼容时升主版本，不静默兼容。

---

## 5. 阶段划分

### Phase 0：项目计划与文档

状态：已完成文档初稿。

产出：

- README；
- `go.mod`；
- 项目计划；
- Go 依赖库评估；
- [`CLI_SCHEMA.md`](CLI_SCHEMA.md) 与 [`schemas/`](schemas/)；
- [`ERROR_CODES.md`](ERROR_CODES.md)；
- 架构；
- 能力命令；
- provider 总原则；
- [`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)；
- 发现/校验；
- 环境变量总原则；
- [`PLATFORM_ENVIRONMENT.md`](PLATFORM_ENVIRONMENT.md)；
- [`LICENSE_DECISION.md`](LICENSE_DECISION.md)；
- 未来 API/插件设计；
- 路线图；
- 技术注意点；
- 文档审查报告。

DoD：

- 文档互相引用一致；
- 所有链接有效；
- 决策已记录；
- Go 版本已锁定；
- 依赖库版本新鲜度已核对；
- CLI JSON 契约、错误码、平台环境细节、provider resolver 细节已补齐；
- 许可证已确定为 GPL-3.0-only；
- Phase 0 结束时代码尚未开始；Phase 1 骨架已完成，见 [`PHASE1_STATUS.md`](PHASE1_STATUS.md)。

### Phase 0 契约引用

- CLI 输出：[`CLI_SCHEMA.md`](CLI_SCHEMA.md) 与 [`schemas/`](schemas/)；
- 错误码：[`ERROR_CODES.md`](ERROR_CODES.md)；
- 平台环境：[`PLATFORM_ENVIRONMENT.md`](PLATFORM_ENVIRONMENT.md)；
- Provider resolver：[`PROVIDER_RESOLVER.md`](PROVIDER_RESOLVER.md)；
- 许可证决策：[`LICENSE_DECISION.md`](LICENSE_DECISION.md)。

---

### Phase 1：仓库骨架与只读 CLI（已完成）

目标：能编译、能输出 `version --json`、能让 `detect --json` 返回只读占位结果。

步骤：

1. `cmd/one-more-ffmpeg` 空 CLI；
2. `internal/model` 内部数据模型；
3. `internal/config`：`pelletier/go-toml/v2` strict mode + TOML 1.1 schema；`internal/errors`：错误码注册表 v1；
4. `internal/log`：`log/slog` + `lmittmann/tint`；
5. CLI 框架：`urfave/cli/v3`；
6. `internal/ffexec`：统一 `exec.CommandContext`、超时、stdout/stderr 限制；
7. `one-more-ffmpeg version`；
8. `one-more-ffmpeg detect --json` 占位（detect 默认只读，不需要额外 dry-run）；
9. fake ffmpeg 测试框架；
10. `stretchr/testify` + `golang.org/x/sync`；
11. Windows/Linux 交叉编译验证；
12. `THIRD_PARTY_NOTICES` 与依赖许可证清单初稿。

DoD：

- `CGO_ENABLED=0 GOOS=windows/linux` 都能编译；
- CLI `--help` / `version` 可用；
- `detect` 不写磁盘、不改 PATH；
- 错误码稳定；
- 本地 Go 1.27.3 构建/测试通过；CI 在 Phase 7 建立。

---

### Phase 2：发现与可用性校验（已完成）

> 实施与验证结果见 [`PHASE2_STATUS.md`](PHASE2_STATUS.md)。

目标：找到系统已有 FFmpeg，并判断能不能用。

> 下列是实现顺序，不是发现优先级；优先级以 [`DETECTION_AND_VALIDATION.md`](DETECTION_AND_VALIDATION.md) 为准。

步骤：

1. `PATH` 发现：`exec.LookPath`；
2. 环境变量发现：`ONE_MORE_FFMPEG_DIR` / `FFMPEG_DIR` / `LAE_FFMPEG_DIR`；
3. 指定目录发现；
4. 常见目录发现；
5. 显式 `--scan` 全盘扫描；
6. 去重：canonical path + binary SHA-256；
7. 三件套同目录校验；
8. `-version` / `-L` 启动校验；
9. 输出 `Installation` + `ValidationReport`。

当前不做：

- Scoop；
- Homebrew；
- winget / choco；
- apt / dnf / pacman / snap / flatpak 专门适配。

DoD：

- PATH、环境变量、指定目录、全盘扫描样例通过；
- 全盘扫描有超时、排除目录、symlink loop 保护；
- 不执行任何安装或 PATH 修改；
- fake ffmpeg 覆盖损坏/超时/权限错误；
- 不因包管理器路径缺失而失败。

---

### Phase 3：能力检查

目标：结构化输出用户列出的 13 项能力。

步骤：

1. 实现 `CAPABILITIES.md` 的 13 个命令适配；
2. `-h <topic>` 做 topic allowlist；
3. 每个命令设置独立 timeout；
4. 保存 raw stdout hash、退出码、stderr 摘要；
5. 解析 `decoders` / `encoders` / `filters` / `muxers` / `demuxers` / `devices` / `pix_fmts` / `layouts` / `sample_fmts`；
6. `-L` / `-license` 解析 license 关键行；
7. `-version` 解析 version 关键字段；
8. 能力缓存按 installation fingerprint；
9. `capabilities --json`；
10. golden 解析器测试。

DoD：

- 13 个命令全部有测试；
- 输出 JSON 字段稳定；
- 不因未知行格式直接崩溃；
- 缓存命中/失效可验证；
- `ffplay` 缺失时降级为 warning，不误判整个 trio 不可用。

---

### Phase 4：Provider / Resolve / Download

目标：解析 BtbN、BnqDzj、自定义 URL，并安全下载。

步骤：

1. `Provider` 接口；
2. 内置 `btbn`；
3. 内置 `bnqdzj-nonfree`；
4. `custom-url` provider；
5. GitHub Release API + 资产过滤；
6. `stable` / `daily` channel 解析；
7. 具体 release tag + asset SHA-256 pin；
8. `checksums.sha256` 校验；
9. `net/http` + `sethvargo/go-retry` 下载到临时目录；
10. 不自动安装，只产出 `Artifact`。

DoD：

- BtbN stable / daily 都能解析；
- BnqDzj nonfree 能解析；
- 自定义 URL 必须提供 checksum；
- HTTP 重定向、超时、重试策略明确；
- 下载不写目标安装目录；
- 不修改 PATH；
- 不把 nonfree 二进制加入 Git 或 Release。

---

### Phase 5：安装、环境变量、切换、回滚

目标：完成部署闭环。

步骤：

1. `klauspost/compress` + `ulikunitz/xz` 安全解压；
2. 版本化安装目录；
3. `current` 链接/目录指针；
4. Windows `PATH` user scope（`x/sys/windows/registry`）；
5. Linux `~/.local/bin` shim；
6. `ONE_MORE_FFMPEG_DIR` / `FFMPEG_DIR` 管理；
7. journal 与 rollback；
8. uninstall；
9. `doctor` 检查路径、环境变量、current、缓存；
10. `env status` / `env set` / `env unset`。

DoD：

- 默认 dry-run；
- 写入前备份旧状态；
- 安装失败自动清理临时目录；
- 切换失败可回滚；
- Windows 不默认改 machine PATH；
- Linux 不默认改 `/etc/profile`；
- 不会删除未由本项目管理的 FFmpeg。

---

### Phase 6：CLI 外部调用契约

目标：让 LAE / VideoDamage / 脚本通过调用可执行文件稳定集成。

步骤：

1. CLI 子命令稳定化；
2. `--json` 输出 schema 稳定化；
3. stdout / stderr 分离；
4. 退出码稳定化；
5. 文件产物路径稳定化；
6. `version --json` 暴露构建信息；
7. `doctor --json` 暴露健康状态；
8. 输出 `install.json` / `artifact.json` / 可选 `ffmpeg-trio.lock`；
9. 写 LAE / VideoDamage 调用 CLI 的示例；
10. 明确不提供 Go API / JSON-RPC / C ABI。

DoD：

- 外部项目不需要 import OneMoreFFmpeg；
- 外部项目不需要链接 Go runtime；
- 所有机器可读结果都在 stdout JSON 或明确文件中；
- 错误信息通过稳定 `code` 字段；
- CLI schema 主版本变化即拒绝，不静默迁移。

### Phase 6 未来，不是当前任务

- 对外开放 Go API；
- JSON-RPC / gRPC / Connect RPC 插件；
- C ABI / `-buildmode=c-shared`；
- `pkg/onemore` 公共 SDK；
- 包管理器识别；
- 自动更新服务。

---

### Phase 7：质量、CI、发布

步骤：

1. `go test ./...`；
2. `go vet`；
3. `gofmt`；
4. `govulncheck`；
5. Windows/Linux GitHub Actions；
6. fake ffmpeg 单测；
7. 真实 BtbN / BnqDzj 资产集成测试；
8. fuzz URL / parser；
9. checksum / provenance 输出；
10. release 只发布源码和校验和，不发布 nonfree 二进制。

---

## 6. M0 硬门禁

M0 定义为：

```text
只读发现 + 可用性校验 + 13 项能力检查（不下载/不安装/不改环境）
```

> provider resolve / download 属于 Phase 4，不属于 M0；M0 不依赖 provider 实现。

通过条件：

- Windows + Linux；
- PATH / env / 指定目录 / 显式扫描；
- 至少一个真实 FFmpeg 安装通过；
- 损坏、超时、权限错误、非 trio 目录都有明确结果；
- 13 项能力命令输出稳定 JSON；
- 无 PATH / 环境变量写入；
- 无下载/安装/删除行为；
- 无包管理器依赖；
- 无 Go API / 插件依赖；
- `go test` 与 `go vet` 通过。

M0 不通过，不进入下载/安装/环境变量写入实现。

---

## 7. 技术点

- `context.Context` 全链路取消；
- `exec.CommandContext` 统一进程超时；
- `io.LimitReader` 限制输出；
- `log/slog` + `tint` 结构化日志；
- `urfave/cli/v3` CLI；
- `pelletier/go-toml/v2` TOML 1.1 strict mode；
- `golang.org/x/sync/errgroup` 并发；
- `golang.org/x/sys` 系统 API；
- `golang.org/x/term` / `go-isatty` 终端判断；
- `klauspost/compress` + `ulikunitz/xz` 解压；
- `sethvargo/go-retry` 下载重试；
- `dustin/go-humanize` 日志可读大小；
- `stretchr/testify` 测试；
- `encoding/json` 稳定输出；
- 原子写 + journal + rollback；
- 能力缓存按 installation fingerprint；
- 安全解压：zip slip / symlink / 绝对路径 / 权限位。

详见 [`GO_LIBRARY_EVALUATION.md`](GO_LIBRARY_EVALUATION.md) 与 [`TECHNICAL_NOTES.md`](TECHNICAL_NOTES.md)。

---

## 8. 注意点

1. 不要一开始就写 Go SDK 或插件系统；当前只做 CLI 可执行文件。
2. 不要现在做 Scoop / Homebrew / winget / apt 适配。
3. 不要静默修改 PATH；所有写入都要显式、可回滚。
4. 不要把 `latest` 当安装记录；必须 pin release tag + asset SHA-256。
5. 不要在公开 GitHub Release 放 BnqDzj nonfree 二进制。
6. 不要用 Go `plugin` 包；Windows 不支持，部署和 ABI 都脆弱。
7. 不要依赖 shell；直接调用可执行文件，避免注入。
8. 不要把 ffmpeg 输出当稳定 API；解析器必须容错并保存 raw hash。
9. 不要无边界全盘扫描；必须显式 `--scan`、排除目录、超时、最大深度。
10. 不要在同一路径覆盖正在运行的 ffmpeg.exe；需要 quarantine/rollback。
11. 不要为实验项目背旧配置兼容包袱；schema/CLI 输出主版本直接升级。
12. 不要引入超过 6 个月未发版的第三方库；即使流行也按硬标准拒绝。

---

## 9. 当前状态

- Phase 0 文档完成；
- Go 依赖评估完成；
- CLI JSON 契约与示例完成；
- 错误码注册表完成；
- 平台环境变量细则完成；
- Provider resolver 细则完成；
- 许可证已确定为 GPL-3.0-only，根目录 `LICENSE` 已添加；
- Phase 1 Go 骨架完成，详见 [`PHASE1_STATUS.md`](PHASE1_STATUS.md)；
- Phase 1 代码审查与 P0/P1 修复完成，详见 [`PHASE1_CODE_REVIEW.md`](PHASE1_CODE_REVIEW.md)；
- Phase 2 真实发现与校验完成，详见 [`PHASE2_STATUS.md`](PHASE2_STATUS.md)；
- `go.sum` 已生成；
- `version --json` / `detect --json` / `validate --json` / `--help` 已验证；
- 本地单测、`go vet`、Windows/Linux 交叉编译通过；
- `THIRD_PARTY_NOTICES.md` 初稿完成；
- 未下载任何 FFmpeg；
- 未安装任何 FFmpeg；
- 未修改 PATH；
- 未实现能力探测、provider resolve、install/env；
- 未实现包管理器识别；
- 未实现 Go API / 插件；
- 未提交 GitHub Actions / 发布 CI。
