# v1.1.1（2026-11-18）

> Bugfix 版本：修复 v1.1.0 自更新可靠性、进度语义、缓存正确性与全盘扫描权限问题。
> CLI 外部契约保持 19 命令 / 18 个 `/v1` schema。
> 产物：Windows x64 / Linux x64 二进制、RHEL 9+ x86_64 RPM、`SHA256SUMS.txt`、build provenance；Scoop bucket 同步更新到 v1.1.1。

## 修复要点

### P0
- **失败下载不再显示成功**：进度抽象拆为 `Update/Complete/Fail/Abort/Close` 四态，互斥幂等；网络中断、取消、校验失败、提交失败都不输出 100%/completed；未知 Content-Length 不伪造百分比；进度只走 stderr。
- **自更新失败可靠恢复**：回滚错误用 `errors.Join` 合并、不再吞掉；恢复失败时保留唯一备份并在错误中给出路径；新增阶段 journal 与崩溃恢复 `Recover()`；保留原文件权限；备份清理失败记录日志。

### P1
- **Windows 真实替换测试**：文件替换抽象为可注入 `FileOps`，临时文件覆盖成功替换、目标占用/权限失败、恢复成功、恢复失败保留备份、权限保留；Windows CI 不再 Skip 核心替换路径。
- **统一缓存存储**：key 文件名改为 SHA-256 摘要（消除碰撞），entry 保存 key digest/schema 并校验；读写错误传播；`Prune()` 清理过期/损坏文件；capabilities 的 JSON/锁/原子写不再复制实现。
- **严格 SemVer**：只接受 `X.Y.Z`（或 tag `vX.Y.Z`），拒绝缺字段、多字段、空字段、非数字、前导零、溢出、prerelease；数值语义比较。

## 全盘扫描权限

`detect --scan` 与 `detect --dir <root> --recursive` 会遍历整个文件系统根目录，普通权限下会因大量目录不可读而产生不完整结果。v1.1.1 起这类全盘扫描要求 elevated 权限：Linux 需 root 或 sudo，Windows 需 Administrator 或 SYSTEM；未提权时返回 `OMF-E6003` 并给出重新运行提示。普通目录扫描（指定具体路径）不受影响。

## 信任边界（Build Provenance）

更新器通过 GitHub Release asset 的 SHA-256 digest 校验下载内容完整性；这**不等同于完整的发布者身份验证**（attestation 的 repository/workflow identity 验证尚未在更新器内实现）。需要强身份保证的用户可用 `gh attestation verify` 独立校验 Release 产物。

## 安装

```powershell
scoop bucket add omf https://github.com/FroyoCatqwq/OneMoreFFmpeg-bucket
scoop install one-more-ffmpeg
scoop update one-more-ffmpeg
```

```bash
sudo dnf install -y https://github.com/FroyoCatqwq/OneMoreFFmpeg/releases/download/v1.1.1/one-more-ffmpeg-1.1.1-1.el9.x86_64.rpm
```

完整说明见 [`安装指南`](../guides/INSTALL.md)；变更见 [`CHANGELOG.md`](CHANGELOG.md)。

## 验证

gofmt / vet / golangci-lint（含 gosec）/ go test / fuzz / 三平台交叉编译全绿；Linux 与 Windows 的替换、恢复、来源保护、缓存、进度测试均实际执行；Linux race 覆盖 output/cache/selfupdate/download。
