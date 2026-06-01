## Context

LoopGuard 的工单提交通道只有 AI Agent 端（`POST /api/v1/tickets`，API Key 认证），人类审批人通过 JWT 登录后只能查看和审批工单，无法主动提交。需要在工单列表页增加手动提交入口，让人类模拟 AI 的提交流程。

现有架构：
- AI 提交：`AIHandler.Submit` → 从 context 取 `api_key_id`（中间件写入）→ `TicketService.Submit`
- 人类查看：`HumanHandler.ListMine` → JWT 认证 → 按 approver_id 查询
- 程序注册时已绑定 approver_id，提交工单通过 project+name 定位程序

## Goals / Non-Goals

**Goals:**
- 人类通过 UI 手动提交工单，走与 AI 完全一致的业务逻辑（dry-run → pending_approval）
- 工单关联用户选择的 API Key，可追溯
- 改动最小化，复用现有 `TicketService.Submit`

**Non-Goals:**
- 不修改现有 AI 提交端点
- 不改变工单状态机
- 不支持批量提交

## Decisions

### 1. 新增 JWT 认证的桥接端点

**决定**: 在 `HumanHandler` 新增 `POST /tickets/submit`（JWT 认证），请求体带 `api_key_id`，内部调用 `TicketService.Submit`。

**替代方案**: 前端直接存原始 API Key 调用 AI 端点 — 放弃，因为 API Key 只在创建时展示一次，存储不安全且交互不自然。

**理由**: 复用 `TicketService.Submit` 确保逻辑一致；JWT 认证保证只有登录用户可用；`api_key_id` 从下拉选择比手动粘贴 Key 更友好。

### 2. 前端用 Dialog + 下拉选择

**决定**: 使用 shadcn/ui Dialog + Select 组件。API Key 下拉显示名称，程序下拉显示 `project/name`，参数用 textarea 每行一个 arg。

**理由**: 项目已用 shadcn/ui，保持一致；下拉比手动输入减少出错。

### 3. 提交后留在列表页刷新

**决定**: 提交成功后刷新工单列表，新工单出现在列表顶部。

**理由**: 用户可能连续提交多个工单，跳转详情页会打断流程。

## Risks / Trade-offs

- [用户需预先创建 API Key] → 提交对话框中若无可选 Key，显示提示引导用户前往 API Key 管理页创建
- [用户需有已注册程序] → 同上，无可选程序时提示引导
