# OneMoreFFmpeg 项目计划

> 状态：Phase 0 计划文档
> 日期：2026-11-10
> 原则：从简单闭环开始，质量优先，不做大而全的第一步

---

## 1. 项目目标

OneMoreFFmpeg 要解决的是“FFmpeg 三件套的生命周期前段”：

```text
发现 → 识别 → 校验 → 能力检查 → 解析来源 → 下载 → 安装 → 配置环境 → 切换/回滚 → 对外服务
```

本项目不负责：

- 视频编码/解码算法；
- damage 算法；
- LAE 转录/归档业务逻辑；
- FFmpeg 自构建；
- GUI / TUI / CUI；
- 第三方二进制的公开再分发。

---

## 2. 首批需求原文映射

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
| Scoop / Homebrew / 环境变量 / 指定目录 / 全盘扫描 | `DETECTION_AND_VALIDATION.md` |
| 验证可用性 | `DETECTION_AND_VALIDATION.md` |
| Go 1.27 | `README.md` + `go.mod` + `TECHNICAL_NOTES.md` |
| 对外 API / 插件接口 | `API_AND_PLUGIN.md` |
| LAE / VideoDamage 调用 | `API_AND_PLUGIN.md` |
| 实验性、可大胆使用架构和语法糖 | `TECHNICAL_NOTES.md` |

---

## 3. 总体设计原则

1. **先只读，后写入**  
   M0 只做发现、校验、能力检查，不改 PATH、不安装、不删除。

2. **先外部进程，后嵌入式 API**  
   FFmpeg 是外部可执行文件；Go 侧只管理路径、进程和元数据。

3. **先 JSON/TOML 契约，后 Go 结构体优化**  
   对外接口先稳定字段，内部可以用 Go 1.27 的泛型/迭代器做语法糖。

4. **pin 后才算可复现**  
   `latest` / `daily` 只作为解析入口，不直接成为安装记录；必须记录 release tag、asset name、SHA-256。

5. **默认不修改系统状态**  
   PATH、shell profile、注册表、安装目录都必须显式参数触发，并支持 dry-run / rollback。

6. **nonfree 与公开分发隔离**  
   内置来源可以列元数据；二进制不进入 public GitHub Release。

7. **实验项目不做长期兼容包袱**  
   schema / API / 插件协议显式版本化；不兼容时升主版本，不静默兼容。

---

## 4. 阶段划分

### Phase 0：项目计划与文档

状态：本阶段。

产出：

- README；
- `go.mod`；
- 项目计划；
- 架构；
- 能力命令；
- provider；
- 发现/校验；
- 环境变量；
- API/插件；
- 路线图；
- 技术注意点。

DoD：

- 文档互相引用一致；
- 所有链接有效；
- 决策已记录；
- Go 版本已锁定；
- 代码尚未开始。

---

### Phase 1：仓库骨架与只读 CLI

目标：能编译、能输出版本、能 dry-run 发现。

步骤：

1. 建立 Go workspace / module；
2. `cmd/one-more-ffmpeg` 空 CLI；
3. `pkg/onemore` 公共类型；
4. `internal/config`：TOML 1.1 解析库评估与 schema；
5. `internal/log`：`log/slog` 结构化日志；
6. `internal/ffexec`：统一 `exec.CommandContext`、超时、stdout/stderr 限制；
7. `one-more-ffmpeg version`；
8. `one-more-ffmpeg detect --json --dry-run`；
9. 单元测试用 fake ffmpeg shell 脚本；
10. Windows/Linux 交叉编译验证。

DoD：

- `CGO_ENABLED=0 GOOS=windows/linux` 都能编译；
- CLI `--help` / `version` 可用；
- `detect` 不写磁盘、不改 PATH；
- 错误码稳定；
- Go 1.27.3 CI 跑通。

---

### Phase 2：发现与可用性校验

目标：找到系统已有 FFmpeg，并判断能不能用。

步骤：

1. `PATH` 发现：`exec.LookPath`；
2. 环境变量发现：`ONE_MORE_FFMPEG_DIR` / `FFMPEG_DIR` / `LAE_FFMPEG_DIR`；
3. Scoop 发现；
4. Homebrew 发现；
5. 指定目录发现；
6. 显式 `--scan` 全盘扫描；
7. 去重：canonical path + binary SHA-256；
8. 三件套同目录校验；
9. `-version` / `-L` 启动校验；
10. 输出 `Installation` + `ValidationReport`。

DoD：

- Windows Scoop 样例通过；
- Linux Homebrew 样例通过；
- 指定目录、PATH、环境变量样例通过；
- 全盘扫描有超时、排除目录、symlink loop 保护；
- 不执行任何安装或 PATH 修改；
- fake ffmpeg 覆盖损坏/超时/权限错误。

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
7. `-version` 解析 version/buildconf 关键字段；
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
9. 下载到临时目录，原子移动；
10. 不自动安装，只产出 `Artifact`。

DoD：

- BtbN stable / daily 都能解析；
- BnqDzj nonfree 能解析；
- 自定义 URL 必须提供 checksum；
- HTTP 重定向、超时、断点续传策略明确；
- 下载不写目标安装目录；
- 不修改 PATH；
- 不把 nonfree 二进制加入 Git 或 Release。

---

### Phase 5：安装、环境变量、切换、回滚

目标：完成部署闭环。

步骤：

1. 解压器：zip / tar.xz，防路径穿越；
2. 版本化安装目录；
3. `current` 链接/目录指针；
4. Windows `PATH` user scope；
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

### Phase 6：对外 API 与插件接口

目标：让 LAE / VideoDamage / 其他语言调用。

步骤：

1. `pkg/onemore` Go API v1；
2. CLI JSON 输出稳定化；
3. JSON-RPC 2.0 over stdio；
4. plugin manifest；
5. capability / permission 模型；
6. 进程插件超时、取消、日志、进度；
7. 示例：LAE 调用 `detect` + `capabilities`；
8. 示例：VideoDamage 调用 `install` / `env`；
9. C ABI / `-buildmode=c-shared` 评估；
10. 明确不采用 Go `plugin` 包。

DoD：

- Go API 有版本；
- JSON-RPC 有 `initialize` / `shutdown` / 错误码；
- 所有跨语言输出无 Go 内部类型；
- 插件权限显式；
- 不向宿主暴露网络/安装权限给低权限插件；
- C ABI 结论写入 ADR。

---

### Phase 7：质量、CI、发布

目标：把实验项目变成可重复构建工具。

步骤：

1. `go test ./...`；
2. `go vet`；
3. `gofmt`；
4. `golangci-lint` 评估；
5. Windows/Linux GitHub Actions；
6. fake ffmpeg 单测；
7. 真实 BtbN / BnqDzj 资产集成测试；
8. fuzz URL / parser；
9. checksum / provenance 输出；
10. release 只发布源码和校验和，不发布 nonfree 二进制。

DoD：

- 双平台 CI；
- Go 1.27.3 精确锁定；
- 文档/代码同步；
- 发布资产 SHA-256；
- `doctor` 可审计；
- 公开 Release 不含 nonfree 二进制。

---

## 5. M0 硬门禁

M0 定义为：

```text
只读发现 + 可用性校验 + 13 项能力检查 + provider resolve（不下载/不安装/不改环境）
```

通过条件：

- Windows + Linux；
- PATH / env / Scoop / Homebrew / 指定目录 / 显式扫描；
- 至少一个真实 FFmpeg 安装通过；
- 损坏、超时、权限错误、非 trio 目录都有明确结果；
- 13 项能力命令输出稳定 JSON；
- 无 PATH / 环境变量写入；
- 无下载/安装/删除行为；
- `go test` 与 `go vet` 通过。

M0 不通过，不进入下载/安装/环境变量写入实现。

---

## 6. 技术点

- `context.Context` 全链路取消；
- `exec.CommandContext` 统一进程超时；
- `io.LimitReader` 限制输出；
- `log/slog` 结构化日志；
- 泛型 + 迭代器用于结果流；
- `errors.Join` 聚合多来源错误；
- `net/http` 下载与重定向限制；
- `debug/buildinfo` / runtime version 写入 `--version`；
- `os.Root` / `io/fs` 风格限制解压目录；
- `encoding/json` 稳定输出；
- 原子写 + journal + rollback；
- 能力缓存按 installation fingerprint；
- 安全解压：zip slip / symlink / 绝对路径 / 权限位；
- Windows 注册表 / `WM_SETTINGCHANGE`；
- Linux shim 优先于 shell rc 修改。

详见 [`TECHNICAL_NOTES.md`](TECHNICAL_NOTES.md)。

---

## 7. 注意点

1. 不要一开始就写大型插件系统；先做只读 CLI 和 Go API。
2. 不要静默修改 PATH；所有写入都要显式、可回滚。
3. 不要把 `latest` 当安装记录；必须 pin release tag + asset SHA-256。
4. 不要在公开 GitHub Release 放 BnqDzj nonfree 二进制。
5. 不要用 Go `plugin` 包；Windows 不支持，部署和 ABI 都脆弱。
6. 不要依赖 shell；直接调用可执行文件，避免注入。
7. 不要把 ffmpeg 输出当稳定 API；解析器必须容错并保存 raw hash。
8. 不要无边界全盘扫描；必须显式 `--scan`、排除目录、超时、最大深度。
9. 不要在同一路径覆盖正在运行的 ffmpeg.exe；需要 quarantine/rollback。
10. 不要为实验项目背旧配置兼容包袱；schema/API 主版本直接升级。

---

## 8. 当前状态

- Phase 0 文档完成；
- 未写 Go 代码；
- 未下载任何 FFmpeg；
- 未安装任何 FFmpeg；
- 未修改 PATH；
- 未选择 LICENSE；
- 未提交 CI。
