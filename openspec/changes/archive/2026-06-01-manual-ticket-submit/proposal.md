## Why

当前工单只能由 AI Agent 通过 API Key 提交，人类审批人无法在 UI 上手动模拟 AI 的提交流程。这导致：开发调试新注册程序时不方便测试、给新人演示系统时缺乏直观的操作路径、AI 不可用时缺少人工兜底手段。

## What Changes

- 在工单列表页新增"提交工单"按钮和对话框
- 对话框内提供 API Key 下拉选择（显示名称）和程序下拉选择（project/name）
- 支持填写参数（textarea，每行一个 arg）
- 后端新增 JWT 认证的提交端点 `POST /tickets/submit`，接收 `api_key_id` + program + args，内部调用现有 `TicketService.Submit` 走完全一致的 dry-run → pending_approval 流程

## Capabilities

### New Capabilities
- `manual-ticket-submit`: 人类通过 UI 手动提交工单的能力，模拟 AI Agent 的提交行为

### Modified Capabilities

## Impact

- **后端**: `internal/api/human_handler.go` 新增 `Submit` 方法；`internal/api/router.go` 注册新路由
- **前端**: `TicketListPage` 新增提交按钮和 Dialog；`api.ts` 新增提交方法；需加载 API Keys 和 Programs 列表
- **无破坏性变更**: 现有 AI 提交端点和审批流程完全不受影响
