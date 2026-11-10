# OneMoreFFmpeg

> 实验性技术项目：统一管理 FFmpeg 的发现、下载、安装、部署、环境变量与能力检查。

仓库：<https://github.com/FroyoCatqwq/OneMoreFFmpeg>

状态：**Phase 0 项目计划与文档阶段，尚未开始实现代码。**

---

## 1. 项目定位

`OneMoreFFmpeg` 是一个独立的 Go 工具链，用来回答和解决下面几个问题：

- 机器上到底有哪些 `ffmpeg` / `ffprobe` / `ffplay`？
- 它们来自哪里：PATH、Scoop、Homebrew、环境变量、指定目录，还是全盘扫描？
- 它们能不能用？三件套是否同目录、同版本、可启动？
- 它们支持哪些 muxer / demuxer / decoder / encoder / filter / pix_fmt / layout / sample_fmt？
- 如何从 BtbN 或 BnqDzj 下载每日构建或稳定版？
- 如何安装、切换、回滚、配置 PATH / 环境变量？
- 如何把以上能力开放给 `LiveArchiveEncoder` 与 `LiveArchiveEncoder-VideoDamage` 调用？

一句话：

> **OneMoreFFmpeg 管“FFmpeg 从哪里来、装到哪里、怎么验证、怎么被发现”；不负责视频损伤算法，也不负责 LAE 业务逻辑。**

---

## 2. 当前范围

首批只做简单但完整的闭环：

| 能力 | 状态 |
|---|---|
| 管理 FFmpeg 路径 | 计划中 |
| 下载/安装/部署 FFmpeg | 计划中 |
| 配置环境变量 / PATH | 计划中 |
| 检查支持的能力列表 | 计划中 |
| Windows + Linux | 计划中 |
| 自定义 FFmpeg URL | 计划中 |
| 内置 BtbN 与 BnqDzj 来源 | 计划中 |
| 每日构建版 / 稳定版 | 计划中 |
| 识别系统已有 FFmpeg | 计划中 |
| 全盘扫描 | 计划中，但必须显式启用 |
| 对外 Go API / CLI / 插件接口 | 计划中 |
| 写视频损伤代码 | 不做 |
| 自构建 FFmpeg | 不做，属于另一条未来路线 |

当前支持的 FFmpeg 能力查看命令范围：

```text
-L
-license
-h <topic>
-version
-muxers
-demuxers
-devices
-decoders
-encoders
-filters
-pix_fmts
-layouts
-sample_fmts
```

详见 [`docs/CAPABILITIES.md`](docs/CAPABILITIES.md)。

---

## 3. 技术决策

| 项 | 决策 |
|---|---|
| 语言 | Go，仅 Go |
| Go 版本 | `go1.27.3`，CI 精确锁定 |
| 平台 | Windows x64 + Linux x86_64 |
| 构建 | `CGO_ENABLED=0` 优先，纯 Go 交叉编译 |
| CLI | `one-more-ffmpeg`，可选短别名 `omf` |
| Go module | `github.com/FroyoCatqwq/OneMoreFFmpeg` |
| 配置 | TOML 1.1；解析库在 Phase 1 评估 |
| API 输出 | JSON，稳定字段名，版本化 |
| 插件 | 优先 JSON-RPC 2.0 over stdio；C ABI 后续评估 |
| 安装源 | 内置 BtbN + BnqDzj，支持自定义 URL |
| nonfree | 只保存元数据与下载入口，不在本仓库/公开 Release 分发二进制 |
| 兼容性 | 实验项目，不为旧配置/旧 API 做长期兼容层 |

本项目是实验性技术项目，允许大胆使用 Go 1.27 已 GA 的语法特性与架构实验；但**公共 API、配置文件 schema、插件协议、安装目录布局必须显式版本化**。

---

## 4. 文档

| 文档 | 内容 |
|---|---|
| [`docs/README.md`](docs/README.md) | 文档索引与权威顺序 |
| [`docs/PROJECT_PLAN.md`](docs/PROJECT_PLAN.md) | 项目计划、阶段、步骤、DoD、技术点 |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | 模块划分、数据模型、依赖方向 |
| [`docs/CAPABILITIES.md`](docs/CAPABILITIES.md) | FFmpeg 能力命令、解析与缓存 |
| [`docs/PROVIDERS.md`](docs/PROVIDERS.md) | BtbN / BnqDzj / 自定义 URL / daily / stable |
| [`docs/DETECTION_AND_VALIDATION.md`](docs/DETECTION_AND_VALIDATION.md) | 系统发现、Scoop/Homebrew、PATH、全盘扫描、可用性校验 |
| [`docs/ENVIRONMENT.md`](docs/ENVIRONMENT.md) | PATH、环境变量、current 链接、回滚 |
| [`docs/API_AND_PLUGIN.md`](docs/API_AND_PLUGIN.md) | Go API、CLI、JSON-RPC、C ABI 后续 |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Phase 0–7 与 M0 门禁 |
| [`docs/TECHNICAL_NOTES.md`](docs/TECHNICAL_NOTES.md) | Go 1.27、并发、安全、测试、许可证、注意点 |

---

## 5. 当前进度

- GitHub 仓库：已创建；
- `go.mod`：已创建，锁定 Go 1.27.3；
- 项目计划与文档：已写入；
- 代码实现：未开始；
- CLI：未实现；
- 下载/安装/环境变量修改：未实现；
- 插件接口：未实现；
- 测试 / CI：未建立。

## 6. 许可证

待定。  
在许可证明确之前，本仓库只提交设计文档，不提交可分发二进制或第三方 FFmpeg 资产。
