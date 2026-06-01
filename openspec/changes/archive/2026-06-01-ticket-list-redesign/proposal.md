## Why

普通用户（审批人）查看工单列表时，前端调 `/api-keys` 接口获取提交来源名称，但该接口是 AdminOnly，导致 403。同时工单列表只显示裸 `program_id` 数字，无法识别程序；失败工单没有视觉突出，需要切 tab 才能发现。

## What Changes

- **后端**：`GET /tickets` 列表接口返回富字段（`program_project`、`program_name`、`submitted_by_name`），不再返回大段的 `dryrun_output`/`exec_output`（由详情接口提供）
- **后端**：store 层新增 `GetProgramsByIDs`、`GetAPIKeysByIDs` 批量查询方法
- **前端**：移除 `/api-keys` 请求，直接使用后端返回的富字段渲染
- **前端**：表格列重新设计 — 程序显示 `project/name`，新增 args 预览列
- **前端**：失败工单视觉突出（行左边框颜色标记），默认按紧急度排序（失败 > 待审批 > 其他）
- **前端**：Tab 显示各状态计数（前端本地计算）

## Capabilities

### New Capabilities

- `ticket-list-enrichment`: 工单列表 API 返回富字段（程序名称、提交来源名称），一次请求包含所有展示信息

### Modified Capabilities

- `ticket-approval`: 工单列表前端展示改造 — 新表格列、状态视觉分层、默认排序

## Impact

- **API**: `GET /api/v1/tickets` 响应结构变化（新增字段，移除大文本字段）
- **Store**: 2 个新增批量查询方法
- **前端**: `TicketListPage.tsx` 重写，`types/index.ts` Ticket 类型更新
- **不涉及**: 数据模型、路由、中间件、其他页面
