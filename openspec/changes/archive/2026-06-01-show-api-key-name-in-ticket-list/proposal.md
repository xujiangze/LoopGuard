## Why

审批人在工单列表和详情页无法识别工单来自哪个 AI Agent。`submitted_by` 字段已存储 API Key ID，但前端未展示，审批人只看到一个无意义的数字。

## What Changes

- 工单列表页（TicketListPage）新增"提交来源"列，显示 API Key 名称
- 工单详情页（TicketDetailPage）基本信息卡新增"提交来源"行，显示 API Key 名称
- 前端加载时额外请求 `/api-keys` 列表，建立 `id → name` 映射

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `ticket-approval`: 工单列表和详情页展示提交来源（API Key 名称），使用前端查表映射方案

## Impact

- 前端：`TicketListPage.tsx`、`TicketDetailPage.tsx` 需修改
- 后端：无改动
- API：无改动（`/api-keys` 接口已存在，`submitted_by` 字段已返回）
