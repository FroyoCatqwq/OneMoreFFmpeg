# 归档文档（历史快照）

这里存放的是**历史记录**，不是当前规范。每个文件都保留着它写下时的结论、数字与判断；
其中「未实现」「计划中」「待办」之类的表述，多数已经在后续 Phase 完成。

> **判断当前能力请只看 [`../README.md`](../README.md) §1 与代码**，不要引用本目录的结论。

归档规则：一份文档一旦被后续文档完全取代，或描述的是某个已完成阶段的实施/审查过程，
就移到这里，并在原位置（索引、README）留下指向本目录的链接。

## 阶段状态与代码审查

| 文件 | 内容 |
|---|---|
| [`PHASE1_STATUS.md`](PHASE1_STATUS.md) | Phase 1（骨架 / 契约 / 错误码 / ffexec）实施状态与验证 |
| [`PHASE1_CODE_REVIEW.md`](PHASE1_CODE_REVIEW.md) | 同上阶段的代码审查与残留 tech debt |
| [`PHASE2_STATUS.md`](PHASE2_STATUS.md) | Phase 2（发现 / 三件套校验 / detect / validate）状态 |
| [`PHASE2_CODE_REVIEW.md`](PHASE2_CODE_REVIEW.md) | 同上阶段的审查记录 |
| [`PHASE3_STATUS.md`](PHASE3_STATUS.md) | Phase 3（13 项能力检查 / 缓存 / capabilities）状态 |
| [`PHASE3_CODE_REVIEW.md`](PHASE3_CODE_REVIEW.md) | 同上阶段的审查记录 |
| [`PHASE4_READINESS.md`](PHASE4_READINESS.md) | Phase 4 准入清单、真实上游事实与决策项 D1–D9 |
| [`PHASE4_STATUS.md`](PHASE4_STATUS.md) | Phase 4（provider / resolve / download）状态与已知限制 |
| [`PHASE4_CODE_REVIEW.md`](PHASE4_CODE_REVIEW.md) | 同上阶段的审查记录 |
| [`PHASE5_STATUS.md`](PHASE5_STATUS.md) | Phase 5（install / current / journal / env / uninstall / doctor）状态与双机验证 |
| [`PHASE6_STATUS.md`](PHASE6_STATUS.md) | Phase 6（CLI 契约冻结：18 命令 / 17 schema / 退出码）与解压加固 |

## 全量审查记录

| 文件 | 内容 |
|---|---|
| [`DOCUMENTATION_AUDIT.md`](DOCUMENTATION_AUDIT.md) | 早期文档审查：事实核实、P0/P1/P2 问题清单 |
| [`FULL_AUDIT_2026-11-11.md`](FULL_AUDIT_2026-11-11.md) | 首轮全量审查与缺陷修复 |
| [`FULL_AUDIT_2026-11-13.md`](FULL_AUDIT_2026-11-13.md) | 第二轮：全量审查与二次修复 |
| [`FULL_AUDIT_2026-11-14.md`](FULL_AUDIT_2026-11-14.md) | 第三轮：用户/代码双视角审查（F1–F14） |
| [`FULL_AUDIT_2026-11-14b.md`](FULL_AUDIT_2026-11-14b.md) | 同日第二轮：事务锁语义、journal 路径白名单、只读命令写盘 |
| [`FULL_AUDIT_2026-11-14c.md`](FULL_AUDIT_2026-11-14c.md) | 同日第三轮：shim 回滚白名单、下载共享目录清理 |
| [`FULL_AUDIT_2026-11-14d.md`](FULL_AUDIT_2026-11-14d.md) | 同日第四轮：移除首次提示、`--force` 恢复幂等、锁心跳、`E5009` |

## 已归档但仍有约束力的部分

有些归档文件里包含**决策**（不是状态），例如 `PHASE4_READINESS.md` 的 D1/D5（stable pin、nonfree 同意）
与 `PHASE6_STATUS.md` 的契约冻结集合。这些决策已经**搬进现行文档**（`CLI_SCHEMA.md`、`ERROR_CODES.md`、
`internal/contracts/freeze_test.go`），归档文件只作为「决策当时怎么来的」的证据保留。
