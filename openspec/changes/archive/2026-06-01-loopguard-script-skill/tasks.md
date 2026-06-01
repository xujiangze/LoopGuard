## 1. Skill 文件创建

- [x] 1.1 创建 `.claude/skills/loopguard-script/SKILL.md`，填写 frontmatter（name、description、metadata）
- [x] 1.2 编写「协议定义」章节：`--only-print` 参数行为、`DRYRUN-OK` 标记、退出码规则、约束条件
- [x] 1.3 编写「脱敏规则」章节：敏感字段类型、前后各 4 位脱敏算法、短字符串处理
- [x] 1.4 编写「输出格式」章节：Markdown 模板结构（DRYRUN-OK 标记 + 请求列表，覆盖 SQL/HTTP/Shell/文件操作）

## 2. 改造指南与模板

- [x] 2.1 编写「危险操作识别」章节：按类型列出检查项（HTTP 写请求、DB 写操作、文件写入、子进程执行）
- [x] 2.2 编写「改造步骤」章节：分析脚本 → 注入参数 → 添加 dry-run 分支 → 实现输出 → 移除交互
- [x] 2.3 编写 Python 模板片段（argparse 添加参数 + if dry_run 分支）
- [x] 2.4 编写 Go 模板片段（flag.Bool 添加参数 + if dryRun 分支）
- [x] 2.5 编写 Bash 模板片段（参数解析循环添加 --only-print + if $ONLY_PRINT 分支）
- [x] 2.6 编写「只读查询处理」指引：何时允许、何时跳过、延迟连接建议

## 3. 合规验证

- [x] 3.1 编写「合规检查清单」：6 项验证（参数存在、DRYRUN-OK、退出码 0、脱敏、无写操作、无交互）
- [x] 3.2 端到端验证：用一个示例通用脚本运行 skill 流程，确认输出符合协议
