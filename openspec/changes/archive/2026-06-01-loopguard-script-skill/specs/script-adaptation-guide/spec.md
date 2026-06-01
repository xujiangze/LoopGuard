## ADDED Requirements

### Requirement: 危险操作识别

skill SHALL 指导 AI agent 识别脚本中的危险操作点，包括以下类别：HTTP 写请求（POST/PUT/PATCH/DELETE）、数据库写操作（INSERT/UPDATE/DELETE/DDL）、文件写入、子进程执行（subprocess/exec/system）。

#### Scenario: 识别 HTTP 写请求

- **WHEN** skill 分析包含 `requests.post`、`requests.put`、`http.Post`、`curl -X POST` 等调用的脚本
- **THEN** skill 将这些调用标记为危险操作，在 dry-run 分支中打印完整请求信息

#### Scenario: 识别数据库写操作

- **WHEN** skill 分析包含 `INSERT INTO`、`UPDATE`、`DELETE FROM`、`cursor.execute(write_sql)` 等调用的脚本
- **THEN** skill 将这些调用标记为危险操作，在 dry-run 分支中打印完整 SQL 语句

#### Scenario: 识别文件写入

- **WHEN** skill 分析包含 `open(file, 'w')`、`os.WriteFile`、`>` 重定向等操作的脚本
- **THEN** skill 将这些操作标记为危险操作，在 dry-run 分支中打印目标文件路径和写入内容摘要

### Requirement: 参数注入模板

skill SHALL 为 Python、Go、Bash 三种语言提供 `--only-print` 参数注入的代码模板。模板 SHALL 包含：参数定义、标志变量获取、dry-run 分支入口点。

#### Scenario: Python 参数注入

- **WHEN** skill 处理 Python 脚本
- **THEN** skill 指导在 argparse 中添加 `--only-print` 参数（`action='store_true'`），并提供分支模板 `if args.only_print: ...`

#### Scenario: Go 参数注入

- **WHEN** skill 处理 Go 脚本
- **THEN** skill 指导使用 `flag.Bool("only-print", false, ...)` 定义参数，并提供分支模板 `if *onlyPrint { ... }`

#### Scenario: Bash 参数注入

- **WHEN** skill 处理 Bash 脚本
- **THEN** skill 指导在参数解析循环中添加 `--only-print)` 分支，设置 `ONLY_PRINT=true` 标志变量

### Requirement: dry-run 输出实现指南

skill SHALL 指导 AI agent 在每个危险操作点前插入 dry-run 输出逻辑。输出 SHALL 打印该操作点将要发出的完整请求内容（SQL、HTTP 详情、命令行等），然后跳过实际执行。

#### Scenario: 单一操作的脚本

- **WHEN** 脚本只有一个危险操作点
- **THEN** skill 指导在操作前加 `if dry_run` 分支，打印请求后 `exit(0)`，正常逻辑放 else 分支

#### Scenario: 多操作的脚本

- **WHEN** 脚本有多个危险操作点
- **THEN** skill 指导收集所有请求到列表中，在 dry-run 分支一次性输出所有请求后 `exit(0)`

### Requirement: 只读查询处理

skill SHALL 指导 AI agent 在 dry-run 模式下合理处理只读查询（数据库 SELECT、HTTP GET）。只读查询可按需执行以构建完整请求预览，但 SHOULD 尽量延迟连接或跳过不必要的外部依赖。

#### Scenario: 查询结果用于构建请求

- **WHEN** 脚本需要通过 SELECT 查询获取 ID 或配置，用于构建后续写请求
- **THEN** dry-run 模式下允许执行该 SELECT 查询，用查询结果构建完整的请求预览

#### Scenario: 查询结果不影响请求内容

- **WHEN** 脚本的查询结果仅用于日志或调试
- **THEN** dry-run 模式下跳过该查询

### Requirement: 合规检查清单

skill SHALL 包含一份合规检查清单。AI agent 改造完成后 SHALL 逐项验证：`--only-print` 参数存在、stdout 含 `DRYRUN-OK`、退出码为 0、敏感字段已脱敏、无写操作执行、无交互阻塞。

#### Scenario: 改造完成验证

- **WHEN** AI agent 完成脚本改造
- **THEN** agent 按清单逐项验证，运行 `script --only-print ...` 检查输出和退出码，报告每项通过或失败

#### Scenario: 合规项不通过

- **WHEN** 验证发现某项不通过（如输出缺少 `DRYRUN-OK`）
- **THEN** agent 修复该问题后重新验证，直到全部通过
