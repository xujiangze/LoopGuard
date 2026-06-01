## Why

LoopGuard 托管的脚本必须支持 `--only-print` 协议（输出 `DRYRUN-OK` + 请求预览），但目前缺少一份面向脚本开发者的通用改造指南。开发者不知道如何改造现有脚本（Python/Go/Bash）使其符合协议要求，导致注册程序时 dry-run 校验反复失败。需要一个 Claude Code skill，在开发者提出"让脚本支持 LoopGuard"时自动指导改造。

## What Changes

- 新增 `.claude/skills/loopguard-script/SKILL.md`，定义完整的协议合规改造指南
- skill 指导 AI agent 分析任意脚本的危险操作点，注入 `--only-print` 分支逻辑
- skill 定义标准 dry-run 输出格式（Markdown，聚焦打印实际请求：SQL、HTTP 请求、Shell 命令等）
- skill 提供多语言模板片段（Python / Go / Bash）
- skill 包含敏感字段脱敏规则（Token/API Key/密码/Secret）
- skill 包含合规检查清单

## Capabilities

### New Capabilities

- `loopguard-script-protocol`: `--only-print` 协议的完整规范，包括输入参数、输出格式、退出码、脱敏规则、合规检查项
- `script-adaptation-guide`: 多语言脚本改造指南，包括危险操作识别、分支注入、dry-run 输出实现、模板代码

### Modified Capabilities

（无现有 spec 需要修改）

## Impact

- 新增文件：`.claude/skills/loopguard-script/SKILL.md`（单一 skill 文件）
- 不影响现有后端 API、前端、数据库
- 与 `docs/guide.md` 中的协议描述保持一致，但面向不同受众（guide.md 面向 API 使用者，skill 面向脚本开发者）
