# Phase 5 实施状态：安装与 current 切换

> 日期：2026-11-13
> 范围：`install`（解压、版本化目录、install.json、current、journal）
> 不在本轮：`env set/status`、`doctor`、`uninstall`、Windows directory junction（归 Phase 6 / 后续）

---

## 1. 完成内容

| 能力 | 位置 | 说明 |
|---|---|---|
| 安全解压 | `internal/archive` | zip / tar.xz / tar.gz / tar.zst；拒绝绝对路径、`..`、盘符、symlink、非普通 tar 条目；条目/大小上限 |
| 版本化安装 | `internal/install` | `<data-root>/apps/<provider>/<tag>/<sha256[:16]>/bin/`；`install.json`；`ffmpeg-trio.lock`（可选） |
| 安装后校验 | `internal/install` | `ffmpeg -version` / `ffprobe -version`，失败 `E5008` 并清理 |
| current 指针 | `internal/install` | 优先 symlink；Windows 无权限时写 `current.pointer`；真实目录拒绝覆盖 `E5004` |
| 事务 journal | `internal/journal` | pending → committed / failed；`<data-root>/journal/<id>.json` |
| 冲突与替换 | `internal/install` | 同 asset 幂等；不同 asset / 已存在目录需 `--force`，旧目录隔离后替换 |
| CLI | `internal/cli/install.go` | `install --release|--provider...`、`--artifact`、`--apply`（默认 dry-run）、`--force`、`--no-current`、`--emit-trio-lock` |

---

## 2. 安全与失败语义

```text
unsupported archive        E5001
zip slip / symlink         E5002
target already exists      E5003
current is a real dir      E5004
quarantine failed          E5007
post-install validation    E5008
archive file missing       E7007
archive sha mismatch       E4002
```

- dry-run 只做路径/冲突/格式检查，不写文件、不下载；
- apply 失败会删除临时目录、回滚已移动的目标目录与隔离目录，并写 failed journal；
- 归档 SHA-256 在解压前重新计算，必须与 release pin 一致；
- nonfree release 仍需 `--allow-nonfree`；
- 不修改 PATH，不修改系统环境变量（Phase 6）。

---

## 3. 验证

```text
go test ./internal/archive ./internal/journal ./internal/install ./internal/cli
  通过：zip/tar.xz 解压、zip slip、symlink、unsupported、dry-run、
        幂等安装、force 替换、journal 状态、current symlink/pointer
go build -tags deps ./... / windows / linux
```

测试使用 `internal/testutil` 的真实 fake 可执行文件打包成 zip，安装后实际执行 `-version`。

---

## 4. 已知限制

1. Windows 只写 symlink 或 `current.pointer`，未实现 directory junction（需 reparse point）；
2. 没有 `uninstall` / `rollback` 子命令；journal 只记录，不提供自动回滚入口；
3. `env` / `doctor` 仍属 Phase 6；
4. 安装只保留 `bin/` 三件套，不复制 LICENSE/README（后续可加 `--keep-docs`）；
5. `current` 切换非严格原子（先删旧链接再建新链接），失败时安装目录会回滚。
