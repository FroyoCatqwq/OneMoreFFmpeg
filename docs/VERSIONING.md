# 版本与发布规范

> 本项目遵循 [Semantic Versioning 2.0.0](https://semver.org/)；发布 tag 使用 `v` 前缀，例如 `v0.6.0`。

## 1. 版本号含义

- **MAJOR**：不兼容的 CLI 参数、退出码、JSON schema 或 safe default 变化；
- **MINOR**：向后兼容的新命令、新字段、新平台、新 provider 能力；
- **PATCH**：向后兼容的 bug 修复、文档修正、测试与构建修复。

`0.y.z` 阶段表示项目仍在快速演进；但同一 minor 内仍保证 JSON schema `onemoreffmpeg.*/v1` 的字段只增不改，破坏性变化必须升 minor。

## 2. 发布流程

```text
1. 全量测试：go test ./...、Windows pytest/Robot、Linux 服务器 pytest/Robot/Bats
2. 更新 docs/CHANGELOG 或 docs/releases/release-vX.Y.Z.md
3. 提交代码与文档
4. 创建 annotated tag：git tag -a vX.Y.Z -m "vX.Y.Z"
5. 推送 main 与 tag（走 127.0.0.1:10808 代理）
6. 构建多平台二进制并生成 SHA256SUMS.txt
7. gh release create vX.Y.Z --title ... --notes-file ... <artifacts>
8. 校验 release 资产与 checksum
```

## 3. 产物命名

```text
one-more-ffmpeg-vX.Y.Z-windows-amd64.exe
one-more-ffmpeg-vX.Y.Z-linux-amd64
SHA256SUMS.txt
```

## 4. 兼容性

- CLI 人类输出可调整；`--json` 字段以 `docs/CLI_SCHEMA.md` 为准；
- 错误码编号不复用、不改变语义；新增错误码必须更新注册表与文档；
- 数据目录布局与 journal schema 变化必须升 minor，并提供迁移/回滚说明。

## 5. 历史

| 版本 | 日期 | 说明 |
|---|---|---|
| v0.6.0 | 2026-11-13 | 首个公开 release：Phase 1–5、审查修复、e2e |
| v0.7.0 | 2026-11-14 | 全局事务锁、系统级作用域与显式提权授权、doctor --strict、Windows symlink 回退 |
