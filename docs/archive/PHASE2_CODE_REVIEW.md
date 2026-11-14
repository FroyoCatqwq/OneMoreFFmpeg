# Phase 2 代码审查与修复记录

> 日期：2026-11-11
> 范围：`internal/detect/`、`internal/cli/`、`internal/model/`、`internal/testutil/`
> 状态：P0/P1 已修复，Phase 2 可冻结

---

## 1. 结论

Phase 2 初版发现并修复了 1 个扫描范围缺陷、若干契约/可测试性问题；没有发现与 Phase 1 契约冲突的 P0。

当前判定：

> **Phase 2 可以冻结，进入 Phase 3：能力检查。**

---

## 2. 已修复问题

### P1-1：扫描排除规则会误伤任意 `tmp` 目录

原实现：

```go
case ".git", "node_modules", "$recycle.bin", "system volume information", "winsxs", "proc", "sys", "dev", "run", "tmp":
    return true
```

问题：

- `C:\Tools\FFmpeg\tmp`、`/home/user/tmp` 也会被跳过；
- 真实安装可能被漏掉。

修复：

- 仅排除绝对系统路径 `/proc`、`/sys`、`/dev`、`/run`、`/tmp`、`/var/lib/docker`；
- 仍排除 `.git`、`node_modules`、`$Recycle.Bin`、`System Volume Information`、`WinSxS`。

### P1-2：`detect` 仍是占位语义

Phase 1 的 `detect --json` 返回空 `installations` 和 `placeholder=true`，真实消费者无法区分“未实现”和“本机未安装”。

修复：

- 实现 `internal/detect`；
- `detect` 做真实来源发现、hash、版本、license 校验；
- 移除 `placeholder` 输出；
- `detect.example.json` 更新；
- `CLI_SCHEMA.md` 标注 `placeholder` 仅 Phase 1 使用、Phase 2 起省略。

### P1-3：`validate` 输入方式不明确

原契约只有 `validate data.installation_id`，没有定义如何得到一个 installation 进行校验。

修复：

```text
validate --dir <PATH>
validate --installation-id <ID>
```

- 二者必须二选一；
- `--installation-id` 重新做默认发现（不启用 `--scan`）并匹配 ID；
- `--dir` 只校验显式目录；
- 已写入 `CLI_SCHEMA.md` 与 `DETECTION_AND_VALIDATION.md`。

### P1-4：扫描错误去重键不稳定

`joinDetails` 原实现按 map 随机顺序拼接，同一错误可能重复；非字符串 details 被丢弃。

修复：

- 对 details key 排序；
- 用 `fmt.Sprint` 覆盖非字符串值；
- `dedupeErrors` 与 `scanRoots` 使用稳定键。

### P1-5：`Installation` / `ValidateData` 的 nil slice 会输出 `null`

修复：

- `DetectData.MarshalJSON` 保证 `installations` / `errors` 为 `[]`；
- `Installation.MarshalJSON` 保证 `sources` / `warnings` 为 `[]`；
- `ValidateData.MarshalJSON` 保证 `checks` / `warnings` 为 `[]`。

### P1-6：fake ffmpeg 无法模拟不同程序版本

修复：

- fake 进程根据自身 basename 读取 `OMF_FAKE_VERSION_FFMPEG` / `OMF_FAKE_VERSION_FFPROBE` / `OMF_FAKE_VERSION_FFPLAY`；
- 支持版本一致、版本不一致、ffplay 缺失、超时、权限错误等测试。

### P1-7：CLI `--roots` 与 `--scan` 关系未校验

修复：

- `--roots` 但没有 `--scan` 直接返回 `OMF-E7001`；
- `validate` 未提供 `--dir` / `--installation-id` 返回 `OMF-E7001`。

### P1-8：扫描大小写/重复根

修复：

- canonical root + Windows 大小写不敏感 key；
- 同一路径被多个 source 发现时合并 `sources`；
- 父根已扫描时跳过子根。

---

## 3. 测试覆盖

新增或扩展：

- 显式目录 healthy；
- 环境变量发现；
- PATH 发现；
- `--scan` 指定根；
- `--max-files` 限制；
- `validate --dir`；
- `validate --installation-id`；
- ffplay 缺失 degraded；
- 版本不一致 unusable；
- `ProbeTimeout` 超时 unusable；
- POSIX 权限错误 unusable；
- CLI `--roots` 无 `--scan`；
- CLI `validate` 无输入；
- detect 不再输出 placeholder；
- validate nil slice 序列化。

---

## 4. 验证结果

```text
gofmt -l .                         无输出
go mod verify                      all modules verified
go build ./...                     通过
go test -count=1 ./...             全部通过
go build -tags deps ./...          通过
go vet ./...                       通过
GOOS=windows GOARCH=amd64 go build 通过
GOOS=linux   GOARCH=amd64 go build 通过
Markdown 相对链接                  OK
Markdown/JSON/Go 尾随空格          OK
```

---

## 5. 保留的 Tech Debt

- `--roots` 用逗号分隔，暂不支持路径中本身含逗号；
- `--max-files` 是每个扫描根的上限，不是全局上限；
- `validate --installation-id` 依赖当前发现环境，非持久化注册表；
- 扫描后如果整体 timeout 到期，后续校验可能被截断；
- 版本解析失败时更倾向 `unusable`，`unknown` 主要留给 `--no-validate`；
- `FFplay` 缺失不影响整体可用性，但会降级为 `degraded`。

这些不阻塞 Phase 3。

---

## 6. 最终判定

> **Phase 2 的真实发现、三件套校验、detect/validate 输出与测试已达到可冻结水平。**
>
> 可以进入 Phase 3，但 Phase 3 必须复用 `internal/detect` 的 `Installation` 和 Phase 1 的 `ffexec`，不要另起一套进程/超时/错误码逻辑。
