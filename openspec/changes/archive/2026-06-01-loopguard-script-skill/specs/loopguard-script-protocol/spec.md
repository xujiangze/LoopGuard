## ADDED Requirements

### Requirement: --only-print 参数支持

脚本 SHALL 接受 `--only-print` 命令行参数。当此参数存在时，脚本 SHALL 进入 dry-run 模式，不执行任何写操作。

#### Scenario: 带 --only-print 执行

- **WHEN** 脚本以 `--only-print` 参数启动
- **THEN** 脚本不执行任何写操作（HTTP 写请求、DB 写操作、文件写入、Shell 命令），只打印将要执行的请求内容，退出码为 0

#### Scenario: 不带 --only-print 执行

- **WHEN** 脚本不带 `--only-print` 参数启动
- **THEN** 脚本正常执行所有操作，行为与改造前一致

### Requirement: DRYRUN-OK 标记输出

dry-run 模式下，脚本 SHALL 在 stdout 的第一行输出 `DRYRUN-OK` 字符串。

#### Scenario: dry-run 成功输出标记

- **WHEN** 脚本以 `--only-print` 执行且参数合法
- **THEN** stdout 第一行为 `DRYRUN-OK`，退出码为 0

#### Scenario: dry-run 参数校验失败

- **WHEN** 脚本以 `--only-print` 执行但参数不合法（缺少必需参数、格式错误等）
- **THEN** stdout 不包含 `DRYRUN-OK`，退出码非 0，stderr 输出错误信息

### Requirement: Markdown 格式请求预览

dry-run 模式下，`DRYRUN-OK` 标记之后 SHALL 输出 Markdown 格式的请求预览。预览内容 SHALL 聚焦于脚本即将发出的实际请求（SQL 语句、HTTP 请求、Shell 命令、文件操作等）。

#### Scenario: 包含 SQL 操作的脚本

- **WHEN** dry-run 模式下脚本将要执行 SQL 语句
- **THEN** 输出中包含 `### 请求 N: SQL` 段落，在代码块中打印完整 SQL 语句，参数值以内联或占位方式展示

#### Scenario: 包含 HTTP 请求的脚本

- **WHEN** dry-run 模式下脚本将要发送 HTTP 请求
- **THEN** 输出中包含 `### 请求 N: HTTP <METHOD>` 段落，打印 URL、Headers（脱敏后）、请求体

#### Scenario: 包含 Shell 命令的脚本

- **WHEN** dry-run 模式下脚本将要执行 Shell 命令
- **THEN** 输出中包含 `### 请求 N: Shell` 段落，打印完整命令

### Requirement: 敏感字段脱敏

dry-run 输出中，以下类型的字段 MUST 进行脱敏处理：Token、API Key、密码、Secret。脱敏规则：保留前 4 位和后 4 位，中间用 `****` 替代；长度不足 8 位的全部替换为 `****`。

#### Scenario: Token 脱敏

- **WHEN** dry-run 输出包含 Authorization header 或 token 字段
- **THEN** token 值被脱敏为 `abcd****efgh` 格式（前 4 + **** + 后 4）

#### Scenario: 密码脱敏

- **WHEN** dry-run 输出包含数据库密码或配置中的 secret 字段
- **THEN** 密码值被脱敏为 `****` 或 `myse****word` 格式

#### Scenario: 短字符串脱敏

- **WHEN** 敏感字段长度不足 8 位
- **THEN** 整个值被替换为 `****`

### Requirement: --help 参数支持

脚本 SHALL 支持 `--help` 参数，输出帮助信息后以退出码 0 退出。帮助信息中 SHALL 包含 `--only-print` 参数的说明。

#### Scenario: 执行 --help

- **WHEN** 脚本以 `--help` 参数启动
- **THEN** stdout 输出包含所有参数说明（含 `--only-print`），退出码为 0

### Requirement: dry-run 无交互阻塞

dry-run 模式下，脚本 SHALL NOT 等待用户交互输入（如 `input()`、`confirm()`、`read -p` 等）。所有输出 SHALL 一次性完成到 stdout。

#### Scenario: 原脚本有交互确认

- **WHEN** 脚本原有交互式确认逻辑（如 "确认执行？y/n"）
- **THEN** dry-run 模式下跳过交互，直接输出请求预览
