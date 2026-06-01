## Context

LoopGuard 平台要求被托管的命令行程序支持 `--only-print` 协议。当前 `docs/guide.md` 描述了协议的行为要求，但缺少一份面向开发者的实操改造指南。开发者需要手动参照 guide.md 中的示例脚本来改造自己的脚本，效率低且容易遗漏关键点。

本项目已有 `.claude/skills/` 目录结构，存放了 openspec 系列 skill。新 skill 将遵循相同的目录和格式约定。

## Goals / Non-Goals

**Goals:**

- 创建一个 Claude Code skill，当开发者要求"让脚本支持 LoopGuard"或"加 --only-print"时自动触发
- skill 指导 AI agent 完成脚本改造的全流程：分析危险操作 → 注入参数和分支 → 实现 dry-run 输出 → 验证合规
- skill 通用化，适用于 Python、Go、Bash 三种语言的脚本
- dry-run 输出聚焦于**打印实际请求**（SQL 语句、HTTP 请求、Shell 命令、文件操作等），不做抽象描述
- 敏感字段（Token/API Key/密码/Secret）在 dry-run 输出中必须脱敏

**Non-Goals:**

- 不创建独立的 Python SDK 或客户端库
- 不修改 LoopGuard 后端代码或前端代码
- 不实现 dry-run 输出的结构化解析（LoopGuard 只检查 `DRYRUN-OK` 字符串存在性）
- 不覆盖 entry_ipban.py 等具体脚本的改造（skill 是通用指南）

## Decisions

### D1: Skill 触发方式 — 关键词匹配

skill 通过 `description` 字段中的关键词触发（`loopguard`、`only-print`、`dry-run`、`DRYRUN-OK`）。不做主动拦截——开发者明确要求时才激活。

**替代方案**: hook 监听 Bash 命令自动拦截危险操作。否决原因：这属于 LoopGuard agent skill 的范畴，与脚本开发者指南的定位不同。

### D2: dry-run 输出格式 — Markdown

dry-run 输出使用 Markdown 格式（`##` 标题、表格、代码块）。理由：

1. LoopGuard 审批页面可直接渲染 Markdown，审批人可读性好
2. 开发者本地调试时终端也能看懂
3. SQL/HTTP 请求体可用 ` ``` ` 代码块包裹，格式清晰

输出结构：`DRYRUN-OK` 标记行 + 空行 + Markdown 格式的请求列表。

**替代方案**: 纯文本 / JSON。否决原因：纯文本缺结构，JSON 对审批人不友好。

### D3: 脱敏策略 — 前后各保留 4 位

Token/API Key/密码类字段保留前 4 位和后 4 位，中间用 `****` 替代。短于 8 位的字符串全部替换为 `****`。这是最常见的脱敏方式，平衡了可辨识性和安全性。

### D4: dry-run 深度 — 按需混合

dry-run 时允许执行只读查询（如数据库 SELECT、API GET），因为构建完整请求预览可能依赖查询结果。但禁止所有写操作（INSERT/UPDATE/DELETE、POST/PUT/PATCH 请求、文件写入、Shell 执行）。

### D5: 多语言模板 — 嵌入 SKILL.md

Python/Go/Bash 的参数注入和分支逻辑模板直接嵌入 SKILL.md，不拆分成独立文件。理由：skill 体积小，单文件方便引用和阅读。

### D6: 合规检查 — 可执行清单

skill 末尾附合规检查清单，AI agent 在改造完成后逐项验证。清单包含：`--only-print` 参数存在、stdout 含 `DRYRUN-OK`、退出码为 0、敏感字段已脱敏、无写操作执行、无交互阻塞。

## Risks / Trade-offs

- [脚本多样性] 不同脚本的"危险操作"识别无固定规则，AI agent 需要根据代码语义判断 → skill 提供常见危险操作类型的检查清单（HTTP 写请求、DB 写操作、文件写入、子进程执行），agent 按清单逐项检查
- [只读查询副作用] dry-run 允许的只读查询可能产生意外副作用（如触发数据库连接） → skill 建议尽量延迟外部连接，只在确实需要时才连接
- [Markdown 渲染差异] 不同终端对 Markdown 的渲染效果不同 → 输出格式保持简单（标题 + 列表 + 代码块），避免复杂格式
