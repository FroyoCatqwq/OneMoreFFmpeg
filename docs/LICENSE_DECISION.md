# 许可证决策记录

> 状态：**DECIDED / GPL-3.0-only**
> 日期：2026-11-10
> 仓库：`github.com/FroyoCatqwq/OneMoreFFmpeg`
> 决定：项目代码与文档使用 **GPL-3.0-only**

---

## 1. 最终决定

OneMoreFFmpeg 采用：

```text
GPL-3.0-only
```

根目录已添加：

```text
LICENSE
```

该文件为 GNU General Public License version 3 全文。
README 中的许可证标识使用 `GPL-3.0-only`，与 `LiveArchiveEncoder-VideoDamage` 保持一致。

---

## 2. 为什么选择 GPL-3.0-only

原因：

1. 与 `LiveArchiveEncoder-VideoDamage` 主项目保持一致；
2. 本项目是 LAE 生态的实验性工具链，优先保证生态内许可证一致；
3. 与 GPL FFmpeg 构建的使用边界更清晰；
4. 避免“文档 GPL、代码 MIT”这类早期不必要的双许可证复杂度；
5. 当前阶段没有闭源 SDK 或商业嵌入承诺；
6. 未来若需要更宽松的 SDK/宿主授权，再单独评估双许可证或例外条款。

---

## 3. GPL-3.0-only 的实际含义

简要说明，不构成法律意见：

- 允许运行、研究、修改、分发；
- 分发修改版或衍生作品时，必须遵守 GPL-3.0 的源码提供与同许可证要求；
- 必须保留版权与许可证声明；
- 不提供担保；
- 不能把 GPL 代码并入闭源专有程序后闭源分发，除非有额外授权；
- 分发（convey）目标代码时须提供 Corresponding Source；GPLv3 本身没有 AGPLv3 式网络条款。

完整条款以根目录 `LICENSE` 为准。

---

## 4. 与 FFmpeg 资产的关系

GPL-3.0-only 只覆盖 OneMoreFFmpeg 自己的代码与文档，不自动覆盖第三方 FFmpeg 二进制。

规则不变：

- BtbN GPL 资产：按其 GPL 条款使用；
- BtbN LGPL 资产：按其 LGPL 条款使用；
- BnqDzj nonfree 资产：不可公开再分发；
- nonfree 二进制不进入 public GitHub Release；
- 不把 FFmpeg 二进制提交到 Git；
- 不把 nonfree 资产上传为公开 CI artifact；
- `install.json` 继续记录 `redistributable` 与 `license_hint`；
- Phase 1 生成 `THIRD_PARTY_NOTICES` 与依赖许可证清单。

---

## 5. 对依赖库的影响

当前建议依赖：

```text
urfave/cli/v3
pelletier/go-toml/v2
lmittmann/tint
stretchr/testify
golang.org/x/sync
golang.org/x/sys
golang.org/x/term
golang.org/x/time
mattn/go-isatty
dustin/go-humanize
klauspost/compress
ulikunitz/xz
sethvargo/go-retry
```

Phase 1 必须：

1. 生成依赖许可证清单；
2. 复核 GPL-3.0-only 兼容性；
3. 对不兼容依赖拒绝或替换；
4. 在 `THIRD_PARTY_NOTICES` 中记录；
5. CI 加许可证检查；
6. 不把 nonfree FFmpeg 当作项目依赖打包。

---

## 6. 仍需完成

- `THIRD_PARTY_NOTICES`；
- 依赖许可证清单；
- `go-licenses` / 等价工具检查；
- 公开 Release 资产许可证说明；
- 若未来做 Go SDK / C ABI，重新评估 GPL 对宿主的影响；
- 若用户改变许可证策略，必须写 ADR 并更新 `LICENSE`、README、审计报告。

---

## 7. 当前状态

| 项 | 状态 |
|---|---|
| LICENSE 文件 | 已添加 GPL-3.0 全文 |
| README 许可证章节 | 已改为 GPL-3.0-only |
| 代码许可证 | GPL-3.0-only |
| 文档许可证 | GPL-3.0-only |
| 第三方依赖清单 | Phase 1 生成 |
| nonfree 分发 | 明确禁止 |
| 审计 P1-9 | **已关闭** |

---

## 8. 结论

> OneMoreFFmpeg 采用 GPL-3.0-only。
>
> 根目录 `LICENSE` 已存在。
>
> 后续 Phase 1 只需完成第三方依赖许可证清单与 `THIRD_PARTY_NOTICES`，不再存在“许可证未选择”的阻塞项。
