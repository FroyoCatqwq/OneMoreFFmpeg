# 外部集成指南

下游项目（如 **LiveArchiveEncoder / LiveArchiveEncoder-VideoDamage**）通过**调用 CLI 可执行文件**使用 OneMoreFFmpeg，
而不是 import Go 包或链接 Go runtime。可运行示例见仓库 [`examples/external-caller/`](../../examples/external-caller/)：

- [`call_omf.py`](../../examples/external-caller/call_omf.py) — Python 3（仅标准库）；
- [`call_omf.ps1`](../../examples/external-caller/call_omf.ps1) — PowerShell 7。

```bash
OMF_BINARY=/path/to/one-more-ffmpeg python examples/external-caller/call_omf.py
```

## 契约要点（必须遵守）

1. **stdout 只有 JSON**：成功是 `onemoreffmpeg.result/v1`，失败是 `onemoreffmpeg.error/v1`；诊断日志在 stderr。始终加 `--json`。
2. **信封稳定**：成功 `{schema, ok:true, command, data:{schema,...}}`；失败 `{schema:"onemoreffmpeg.error/v1", ok:false, code, message, details, cause?}`。
3. **退出码由错误码决定**：`0` 成功 / `2` 用法与契约错误 / `1` 运行期错误 / `130` 取消。按 `code` 分支，不要解析人类文本，见 [`ERROR_CODES.md`](../reference/ERROR_CODES.md)。
4. **schema 主版本冻结**：全部 `data.schema` 为 `/v1`；`internal/contracts/freeze_test.go` 在 CI 中守护命令集 / schema 集 / 退出码。
5. **只读 vs 写入**：`detect`/`validate`/`capabilities`/`doctor`/`env status`/`journal list`/`release list`/`update`（默认检查）只读；`install`/`uninstall`/`env set|unset|rollback`/`journal rollback`/`provider add`/`update --apply` 默认 dry-run，需 `--apply` 才落盘。
6. **doctor 默认退出 0**：即使 `data.healthy=false` 也退出 0；需非零退出用 `doctor --strict`。
7. **进度与日志不污染 stdout**：进度条按 TTY/CI/Pipe 自适应，只写 stderr；`--json` 完全不渲染动画。下游可安全解析 stdout 为单个 JSON 信封。
8. **自更新**：`update --json` 返回 `onemoreffmpeg.update/v1`（`update_available`、`applied` 等）；外部程序可据此提示用户，是否执行更新由用户决定（`update --apply`）。

## 典型流程

```text
version --json                                          # 确认可用与版本
doctor --json                                           # 读 data.healthy；不健康提示用户 install/env set
detect --json                                           # 找 installation.root_dir
capabilities --dir <root> --only encoders --no-cache --json   # 取所需能力供编码决策
update --json                                                # 可选：检查 CLI 是否有新版
```

下游无需关心 data-root 布局、journal、current 指针等内部细节——它们都被 CLI 契约封装。

完整字段见 [`CLI_SCHEMA.md`](../reference/CLI_SCHEMA.md)；版本兼容规则见 [`VERSIONING.md`](../reference/VERSIONING.md)。
