## ADDED Requirements

### Requirement: 人类手动提交工单端点
系统 SHALL 提供 `POST /tickets/submit` 端点，要求 JWT 认证，接受 `api_key_id`（uint64）、`project`（string）、`name`（string）、`args`（[]string），调用 `TicketService.Submit` 走与 AI 提交一致的业务逻辑。

#### Scenario: 成功提交工单
- **WHEN** 已登录用户 POST `/tickets/submit`，body 为 `{"api_key_id": 1, "project": "my-project", "name": "deploy", "args": ["--env", "prod"]}`
- **THEN** 系统调用 `TicketService.Submit`，返回 `{"ticket_id": N, "status": "pending_approval", "dryrun_output": "..."}`

#### Scenario: API Key 不存在或已禁用
- **WHEN** 提交的 `api_key_id` 对应的 Key 不存在或已禁用
- **THEN** 返回 400 错误，提示 "API Key 不存在或已禁用"

#### Scenario: 程序不存在
- **WHEN** 提交的 project+name 找不到已注册且启用的程序
- **THEN** 返回 400 错误（由 TicketService.Submit 返回）

#### Scenario: 未登录
- **WHEN** 未携带有效 JWT Token 访问 `/tickets/submit`
- **THEN** 返回 401 错误

### Requirement: 工单列表页提交入口
工单列表页 SHALL 显示"提交工单"按钮，点击后弹出 Dialog。Dialog 内包含：API Key 下拉选择器（显示 Key 名称）、程序下拉选择器（显示 project/name）、参数输入框（textarea，每行一个 arg）、提交和取消按钮。

#### Scenario: 打开提交对话框
- **WHEN** 用户点击"提交工单"按钮
- **THEN** 弹出 Dialog，下拉列表自动加载当前可用的 API Keys 和已注册程序

#### Scenario: 无可用 API Key
- **WHEN** 系统中无已启用的 API Key
- **THEN** 下拉显示空状态，提示用户前往 API Key 管理页创建

#### Scenario: 无已注册程序
- **WHEN** 系统中无已注册且启用的程序
- **THEN** 下拉显示空状态，提示用户前往程序管理页注册

#### Scenario: 成功提交并刷新列表
- **WHEN** 用户填写完表单点击提交，后端返回成功
- **THEN** Dialog 关闭，工单列表自动刷新，新工单出现在列表中

#### Scenario: 提交失败
- **WHEN** 后端返回错误（如 dry-run 失败）
- **THEN** Dialog 保持打开，显示错误信息
