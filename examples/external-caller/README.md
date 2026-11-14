# 外部调用示例（Phase 6 契约）

下游项目（如 **LiveArchiveEncoder / LiveArchiveEncoder-VideoDamage**）通过**调用 CLI 可执行文件**使用 OneMoreFFmpeg，
而不是 import Go 包或链接 Go runtime。本目录给出两种语言的最小可运行示例：

- [`call_omf.py`](call_omf.py) — Python 3（仅标准库）。
- [`call_omf.ps1`](call_omf.ps1) — PowerShell 7。

运行：

```bash
OMF_BINARY=/path/to/one-more-ffmpeg python call_omf.py
```
```powershell
$env:OMF_BINARY = "C:\path\one-more-ffmpeg.exe"; ./call_omf.ps1
```

## 契约要点（外部消费者必须遵守）

1. **stdout 只有 JSON**：成功是 `onemoreffmpeg.result/v1` 信封，失败是 `onemoreffmpeg.error/v1` 信封；诊断/日志在 stderr。
   加 `--json` 保证机器可读输出。
2. **信封结构稳定**：
   ```json
   { "schema": "onemoreffmpeg.result/v1", "ok": true, "command": "doctor", "data": { "schema": "onemoreffmpeg.doctor/v1", ... } }
   ```
   失败：
   ```json
   { "schema": "onemoreffmpeg.error/v1", "ok": false, "code": "OMF-E6003", "message": "...", "details": {} }
   ```
3. **退出码由错误码唯一决定**：`0` 成功；`2` 用法/配置/契约错误；`1` 运行期错误；`130` 取消。
   消费者按 `code` 分支，不要解析人类文本。完整注册表见 [`docs/ERROR_CODES.md`](../../docs/ERROR_CODES.md)。
4. **schema 主版本冻结**：当前所有 `data.schema` 均为 `/v1`。主版本变化即破坏性变更（见 [`docs/VERSIONING.md`](../../docs/VERSIONING.md)）；
   `internal/contracts/freeze_test.go` 会在 CI 中拒绝命令集/schema 集/退出码的静默改动。
5. **只读 vs 写入**：`detect` / `validate` / `capabilities` / `doctor` / `env status` / `journal list` / `release list` 只读；
   `install` / `uninstall` / `env set|unset|rollback` / `journal rollback` / `provider add` 默认 dry-run，需 `--apply`。
6. **doctor 默认退出 0**：即使 `healthy=false` 也退出 0，必须读 `data.healthy`；需要非零退出时用 `doctor --strict`。

## 典型集成流程

```text
version --json                 # 确认 CLI 可用与版本
doctor --json                  # 读 healthy；不健康则提示用户运行 install/env set
detect --json                  # 找到 installation.root_dir
capabilities --dir <root> --only encoders --no-cache --json
                               # 取所需能力列表，供下游编码决策
```

下游无需关心 data-root 布局、journal、current 指针等内部细节——这些都被 CLI 契约封装。