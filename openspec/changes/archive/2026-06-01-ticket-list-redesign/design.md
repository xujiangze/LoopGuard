## Context

当前 `GET /api/v1/tickets` 返回原始 `model.Ticket`，前端需要额外请求 `/api-keys`（AdminOnly，普通用户 403）和 `/programs` 来展示程序名称和提交来源。前端用 `.catch(() => [])` 静默吞掉 403，导致普通用户永远看到 `Key #3` 这种无意义文本。

工单列表表格只显示裸 `program_id` 数字，没有 args 预览，失败工单无视觉突出。

## Goals / Non-Goals

**Goals:**
- 普通用户（审批人）能一次请求获取所有列表展示所需字段
- 程序显示为 `project/name` 可读格式
- 失败工单在视觉上突出，默认排在最前
- Tab 显示各状态计数

**Non-Goals:**
- 不改 Ticket 数据模型（不加 DB 列）
- 不改路由结构或中间件
- 不改详情页（详情页已有完整的 dryrun/exec output）
- 不做分页（工单量在审批人视角下很小）

## Decisions

### 1. 后端组装富字段响应，而非前端多请求 join

**选择**: `ListMine` handler 返回 `ticketListItem` DTO，批量查 programs 和 api-keys 后组装。

**替代方案**: 前端额外调 `/programs` 做映射 — 额外网络请求 + 前端 join 逻辑复杂，且 `/api-keys` 对普通用户不可用。

**理由**: 一次请求返回所有展示数据，前端零 join 逻辑。store 用 `WHERE id IN (...)` 批量查询，性能无问题。

### 2. 列表响应去掉 dryrun_output / exec_output

**选择**: `ticketListItem` 不含这两个大文本字段。

**理由**: 列表页不展示它们。单个工单的 dryrun output 可达数 KB，列表 50 条就是几百 KB 无用数据。详情页已有 `GET /tickets/:id` 返回完整字段。

### 3. Tab 计数前端本地计算

**选择**: 一次拉全部工单（不带 status 参数），前端从结果中统计各状态数量并做 tab 过滤。

**替代方案**: 后端返回 counts 对象 — 增加接口复杂度。

**理由**: 审批人视角下工单量很小（几十到几百条），一次全拉 + 前端过滤更简单，且 tab 切换无需网络请求。

### 4. 状态排序前端实现

**选择**: 前端按优先级排序：`exec_failed/dryrun_failed` > `pending_approval` > `executing/approved` > `done` > `rejected`。

**理由**: 纯展示逻辑，不涉及后端。后端保持 `ORDER BY id desc` 不变。

### 5. 视觉突出用行左边框颜色

**选择**: 根据 status 给 TableRow 加 `border-left: 3px solid <color>`：红=失败，黄=待审，绿=完成，灰=其他。

**替代方案**: 整行背景色 — 面积太大，在深色/浅色主题下颜色不好调。

**理由**: 左边框是微妙的视觉暗示，不干扰阅读，在两种主题下都清晰。

## Risks / Trade-offs

- **[列表响应字段减少]** → 详情页已有独立接口，无影响。前端 `Ticket` 类型需拆分为 `TicketListItem` 和 `TicketDetail` 两种类型。
- **[批量查询内存]** → 审批人视角工单量小，collect IDs + IN 查询完全可控。如果未来量级增长再加分页。
- **[前端全量拉取]** → 审批人视角下数据量小（几十条），如果未来增长到上万条需要改为后端过滤 + 分页。
