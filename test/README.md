# 测试目录

本项目测试按运行方式与用途分类。

## 1. Go 单元测试（`*_test.go`）—— 按包分布

Go 工具链要求 `*_test.go` 与被测包**同目录**（内部测试 `package xxx` 需要访问未导出符号，`go test ./...` 按包发现测试），因此它们不放在本目录，而是跟随各自源码包：

| 位置 | 覆盖 |
|---|---|
| `internal/<pkg>/*_test.go` | 对应包的单元测试（archive/cache/capabilities/cli/config/contracts/detect/env/ffexec/install/journal/lock/output/provider/selfupdate 等） |
| `internal/<pkg>/*_internal_test.go` | 需要访问包内未导出符号的白盒测试 |
| `internal/<pkg>/*_fuzz_test.go` | fuzz 目标 |
| `cmd/one-more-ffmpeg/script_test.go` + `testdata/*.txtar` | testscript CLI 脚本测试 |

运行：`mise exec -- go test ./...`、`mise run fuzz`、`mise run quality`。

## 2. 端到端测试（`test/e2e/`）—— 按框架分类

| 路径 | 用途 |
|---|---|
| `test/e2e/test_cli_e2e.py` | CLI 基础命令（version/provider/download/install/env/journal/doctor） |
| `test/e2e/test_cli_full_e2e.py` | 全命令 JSON 信封、capabilities 字段级解析、参数互斥 |
| `test/e2e/test_env_real_e2e.py` | 环境变量真实修改（Linux shim/profile、Windows 注册表） |
| `test/e2e/test_update_e2e.py` | 自更新命令（默认 dry-run 检查） |
| `test/e2e/robot/cli.robot` | Robot Framework CLI 流程 |
| `test/e2e/bats/cli.bats` | Bats（Linux shell）流程 |
| `test/e2e/conftest.py` / `pytest.ini` / `requirements.txt` | pytest 配置与 fixture |

运行：`mise run e2e-pytest`、`mise run e2e-robot`、`mise run e2e-bats`。

## 3. 测试基础设施（`test/infra/`）

| 路径 | 用途 |
|---|---|
| `test/infra/remote_linux.py` | 通过 SSH 把仓库部署到 Linux CI 服务器并运行全套 e2e（含 Docker 全新环境） |

运行：`mise run e2e-linux-server`（需 `OMF_LINUX_SERVER_INFO` 或本地 `Linux-Server-Info.txt`）。

## 4. 运行产物

`test/e2e/results/`、`__pycache__/`、`.pytest_cache/` 是运行时产物，已在 `.gitignore` 忽略，不提交。

详细 e2e 说明见 [`docs/development/E2E_TESTING.md`](../docs/development/E2E_TESTING.md)。
