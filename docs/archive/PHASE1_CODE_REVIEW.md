# Phase 1 代码审查与修复记录

> 日期：2026-11-10
> 范围：`cmd/`、`internal/`、`go.mod`、`go.sum`、`config.example.toml`、`THIRD_PARTY_NOTICES.md`、Phase 1 文档
> 方式：逐文件代码审查 + 独立只读审查 + 测试复现 + 修复后回归
> 状态：**P0 / P1 已修复；Phase 1 可冻结**

---

## 1. 结论

Phase 1 初版存在两类地基问题：

1. `ffexec` 超时可能因孙进程持有 stdout/stderr 管道而长时间挂起；
2. `--json` 成功路径与失败路径使用两套判定，`--json=1` 等场景会破坏 JSON 契约。

加上若干 P1 契约/实现偏差，初版**不能原样冻结**。

本轮已修复 P0/P1，并增加高风险回归测试。

当前判定：

> **Phase 1 可冻结，可以进入 Phase 2。**

---

## 2. P0 修复

### P0-1：ffexec 超时挂起、无进程树回收、超时丢输出

原实现：

```go
runCtx, cancel := context.WithTimeout(ctx, timeout)
cmd := commandFor(runCtx, executable, args)
err := cmd.Run()
```

问题：

- `exec.CommandContext` 默认只杀直接子进程；
- 孙进程仍持有 stdout/stderr 管道时，`Wait` 可能远超 Timeout；
- 没有 Unix `Setpgid` / Windows `taskkill /T` 进程树；
- 超时直接返回空 `Result`，丢失 stderr 证据。

修复：

- `cmd.WaitDelay = 2s`（可配置 `Runner.WaitDelay`）；
- Unix：`Setpgid` + `cmd.Cancel` 杀进程组；
- Windows：`cmd.Cancel` 调用 `taskkill /F /T /PID`；
- 超时/取消返回**已捕获的部分 Result**；
- `exec.ErrWaitDelay` 归类为 `OMF-E1006`；
- 增加孙进程持管道测试与超时部分输出测试；
- 删除不安全的 `.bat/.cmd` 特殊分支，避免 `cmd.exe` 二次解析注入面。

相关文件：

- `internal/ffexec/runner.go`
- `internal/ffexec/proc_unix.go`
- `internal/ffexec/proc_windows.go`
- `internal/ffexec/runner_test.go`
- `internal/testutil/fake_ffmpeg.go`

### P0-2：`--json` 双轨判定

原实现：

- 成功用 `jsonEnabled(cmd)`；
- 失败用 `wantsJSON(opts.Args)`；
- `--json=1` 解析为 true，但 argv 扫描不匹配；
- `--config --json` 会把 `--json` 当 config 值，却仍触发 JSON。

修复：

- `wantsJSON` 支持 `--json=1` / `--json=true` / `--json=false`；
- 跳过 `--config` / `-c` / `--log-level` 的值；
- 遇到 `--` 停止扫描；
- 失败路径同样能得到正确 JSON 模式；
- 增加 `--json=1`、`--json=false`、`--json version` 回归测试。

相关文件：

- `internal/cli/json.go`
- `internal/cli/root_test.go`

---

## 3. P1 修复

### P1-1：usage 错误污染 stdout

原行为：

```text
version --bogus --json
```

stdout 出现 `Incorrect Usage` + help 文本 + 错误 JSON，且退出码不稳定。

修复：

- 给所有子命令继承 root 的 `OnUsageError` 与 `ExitErrHandler`；
- usage 错误统一返回 `OMF-E7001`；
- JSON 模式下 stdout 只输出单个错误对象；
- 增加 `TestUsageErrorJSON`。

验证：

```text
version --bogus --json
exit=2
stdout={"schema":"onemoreffmpeg.error/v1",...,"code":"OMF-E7001",...}
stderr 为空
```

### P1-2：logging 未接线

修复：

- `Before` 中写入 `logger.Debug("configuration loaded", ...)`；
- 默认 info 级无噪声；`--log-level debug` 可看到日志；
- `PHASE1_STATUS` 明确“基础设施已接线，业务日志仍少”。

### P1-3：detect 占位不可机判

修复：

- `DetectData` 增加 `placeholder`；
- Phase 1 `detect --json` 输出 `"placeholder": true`；
- `detect.example.json` 与 `CLI_SCHEMA.md` 同步；
- 消费者不再把占位误判成“本机没有 FFmpeg”。

### P1-4：`OMF-E7006` 退出码冲突

原：

- 文档表格写 1；
- `exitFor` 前缀规则把所有 E7xxx 判为 2。

修复：

- 代码显式列出 E7xxx 映射；
- `E7006` 固定为 1；
- 文档退出码总表改为 `E7001–E7005` / `E7007` 为 2；
- 增加 `TestExitCodeMapping` 覆盖 E7006。

### P1-5：`E1007` 归属不清

修复：

- 在 `ERROR_CODES.md` 明确：
  - `ffexec` 对非零退出返回 `Result.ExitCode`；
  - 不直接返回 error；
  - 由能力探测/运行层按策略映射成 `OMF-E1007`。

### P1-6：contracts 校验不完整

修复：

- `result` 必须 `ok=true`；
- `command` 必须在白名单；
- `data.schema` 必须在 `model.KnownDataSchemas` 白名单；
- `error` 必须 `ok=false`、code 在注册表、message 1..200 字符、details 为 object；
- `ValidateErrorRegistryDocs` 增加反向测试 `ValidateRegistryCoveredByDocs`；
- 13 个示例数量由精确测试锁定；
- 增加非法 result/error 测试。

### P1-7：TOML 1.1 声明

独立审查曾怀疑 `go-toml/v2` 只支持 TOML 1.0。实际核对模块缓存 README：

```text
This library supports TOML v1.1.0.
```

结论：**TOML 1.1 声明正确，不改文档。**

### P1-8：`Env` 只能追加

修复：

- `ffexec.Runner` 增加 `CleanEnv` 与 `UnsetEnv`；
- 支持覆盖、清理继承环境、unset；
- Windows 环境键大小写不敏感；
- 增加 env 覆盖 / clean / unset 测试。

### P1-9：高风险测试缺失

新增测试：

- ffexec 超时部分输出；
- 孙进程持管道；
- ctx 取消；
- 可执行文件缺失 / 空路径；
- env 覆盖 / clean / unset；
- output total bytes；
- config 缺 schema / 目录 / 类型错误 / 示例解析；
- usage 错误 JSON；
- `--json=1` / `--json=false` / `--json` 前置；
- 契约非法 result/error；
- 错误码双向文档覆盖；
- E7006 / From(nil)。

### P1-10：`.bat/.cmd` 注入面

修复：

- 删除 `.bat/.cmd` 特殊处理；
- 只执行真实可执行文件，避免 `cmd.exe` 二次解析；
- Phase 1 的 ffmpeg 三件套是 exe，不需要 batch wrapper。

---

## 4. P2 已处理

- `From(nil)` panic 已修；
- `--config <目录>` 返回 E7007；
- TOML 类型错误返回 E7005；
- config 缺 `schema` 返回 E7003；
- `config.example.toml` 补合法枚举注释，并加解析测试；
- `version` 的 `build_time/git_commit` 开发构建允许 `unknown`，文档已注明；
- `--help` 明确为人类输出例外，`--json --help` 不作为机器结果；
- `omf` 别名文档改为“未来候选，当前未提供”；
- `model` 补齐 CLI schema 常量白名单；
- `testutil` 使用 `testing.TB`、清理 `GOOS/GOARCH/GOFLAGS/CGO_ENABLED`、支持孙进程与 env 输出；
- `main` 注册 `signal.NotifyContext(os.Interrupt)`；
- `Run` 增加 panic 恢复，映射 `OMF-E9002`。

---

## 5. 仍保留的 P2 / Tech Debt

以下不阻塞 Phase 1，登记到 Phase 2 之前处理：

- `ffexec` 非零退出仍以 `Result.ExitCode` 表达，E1007 由调用方映射；
- Unix 信号退出归一化为 `ExitCode=1` + `Signal` 字符串，不暴露具体 signal 编号；
- `model.Installation.Sources/Warnings` 若未来直接构造为 nil，会序列化 `null`；Phase 2 用构造函数保证 `[]`；
- `New` 对未知错误码仍降级为 E9001，测试构建可再加 strict panic；
- warning 码不走 error 通道；
- fake ffmpeg 每个测试重新 `go build`，测试耗时约 12s；
- `go test -race` 尚未纳入 CI；
- 未来 API/插件仍不实现；
- 包管理器识别仍不实现。

---

## 6. 验证结果

```text
gofmt -l .                         无输出
go mod verify                      all modules verified
go build ./...                     通过
go test ./...                      通过
go build -tags deps ./...          通过
go vet ./...                       通过
GOOS=windows GOARCH=amd64 go build 通过
GOOS=linux   GOARCH=amd64 go build 通过
```

CLI 冒烟：

- `version --json`：正确；
- `detect --json`：`placeholder=true`；
- `--json version`：正确；
- `--json=1 --config missing version`：JSON E7007，exit 2；
- `--json=false version`：人类输出；
- `version --bogus --json`：单 JSON E7001，exit 2，stderr 为空；
- `--help`：人类帮助，作为文档化例外。

---

## 7. 最终判定

> **Phase 1 的 CLI JSON 契约、错误码、进程执行器三块地基已修复并通过回归。**
>
> **可以冻结 Phase 1，进入 Phase 2。**
>
> Phase 2 第一项应是真实 FFmpeg 发现与校验；届时应继续使用 `ffexec`，不要在 Phase 2 绕过进程树/超时/输出限制。
