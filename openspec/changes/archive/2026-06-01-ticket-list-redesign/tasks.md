## 1. 后端 Store 层

- [x] 1.1 在 `internal/store/store.go` 中新增 `GetProgramsByIDs(ids []uint64) (map[uint64]model.Program, error)` 方法
- [x] 1.2 在 `internal/store/store.go` 中新增 `GetAPIKeysByIDs(ids []uint64) (map[uint64]model.APIKey, error)` 方法

## 2. 后端 API 层

- [x] 2.1 在 `internal/api/human_handler.go` 中定义 `ticketListItem` 响应 DTO 结构体
- [x] 2.2 改造 `ListMine` handler：批量收集 program_ids 和 api_key_ids，调用批量查询，组装 `[]ticketListItem` 返回
- [x] 2.3 确认 `GET /tickets/:id` 详情接口仍返回完整 `model.Ticket`（含 dryrun_output/exec_output），不改动

## 3. 前端类型与数据层

- [x] 3.1 更新 `web/src/types/index.ts`：Ticket 类型新增 `program_project`、`program_name`、`submitted_by_name`，移除列表不用的 `dryrun_output`/`exec_output`（详情页单独类型或复用）
- [x] 3.2 重写 `TicketListPage` 的 `useEffect`：移除 `/api-keys` 请求和 `apiKeyMap` 状态，改为单次 `GET /tickets` 请求

## 4. 前端表格 UI

- [x] 4.1 重新定义表格列：程序（project/name 两行）、参数预览（args join 截断）、提交来源（submitted_by_name）、状态 Badge、提交时间、操作按钮
- [x] 4.2 实现状态排序：`exec_failed/dryrun_failed` > `pending_approval` > `executing/approved` > `done` > `rejected`，同优先级按 `created_at` 倒序
- [x] 4.3 实现行左边框颜色标记：红=失败、黄=待审批、绿=完成、灰=其他

## 5. 前端 Tab 计数

- [x] 5.1 改为一次拉全部工单（不传 status 参数），前端本地计算各状态计数并显示在 Tab 标签上
- [x] 5.2 Tab 切换时前端本地过滤，不发网络请求

## 6. 验证

- [x] 6.1 用普通用户（非 admin）登录，确认工单列表正常加载，不再 403
- [x] 6.2 确认程序显示为 `project/name` 格式，提交来源显示 API Key 名称
- [x] 6.3 确认失败工单排在最前且有红色左边框
- [x] 6.4 确认详情页（dryrun_output/exec_output）功能不受影响
