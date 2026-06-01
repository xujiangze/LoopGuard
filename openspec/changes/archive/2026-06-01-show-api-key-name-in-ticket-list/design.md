## Context

工单的 `submitted_by` 字段已存储 API Key ID，API 响应中已包含该字段。`/api-keys` 管理接口已存在，返回 `id` 和 `name`。后端无需改动，纯前端变更。

## Goals / Non-Goals

**Goals:**
- 审批人在工单列表和详情页能看到工单来自哪个 API Key（显示名称）

**Non-Goals:**
- 后端改动（不加字段、不改 API）
- 处理 API Key 被删除后的映射缺失（显示 ID 即可）
- AI 侧轮询接口的展示改动

## Decisions

**前端查表映射方案**

页面加载时并行请求 `/api-keys` 和 `/tickets`，用 `Map<id, name>` 做映射。列表显示 name，找不到映射时 fallback 显示 `Key #id`。

理由：后端零改动，利用已有接口。API Key 列表通常很短（几十条），性能无影响。

## Risks / Trade-offs

- [API Key 被删除后映射不到] → fallback 显示 `Key #id`，可接受
- [多一次 API 调用] → API Key 列表很小，开销可忽略
